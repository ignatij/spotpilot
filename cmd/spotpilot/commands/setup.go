package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type instructionSetupResult struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
	Updated bool   `json:"updated"`
	Skipped bool   `json:"skipped"`
}

type instructionSetupSpec struct {
	targetFile    string
	targetDir     string
	commandName   string
	targetLabel   string
	presentNeedle string
	block         string
}

func applyInstructionSetup(spec instructionSetupSpec) (instructionSetupResult, string, error) {
	res := instructionSetupResult{Path: spec.targetFile}

	existing, err := os.ReadFile(spec.targetFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return res, "", fmt.Errorf("cannot read %s: %w", spec.targetFile, err)
	}

	content := string(existing)
	if strings.Contains(content, spec.presentNeedle) {
		res.Skipped = true
		msg := fmt.Sprintf("%s already contains spotpilot %s instructions — no changes made", spec.targetLabel, spec.commandName)
		return res, msg, nil
	}

	if err := os.MkdirAll(spec.targetDir, 0o700); err != nil {
		return res, "", fmt.Errorf("cannot create directory %s: %w", spec.targetDir, err)
	}

	newContent := strings.TrimRight(content, "\n") + spec.block
	if err := os.WriteFile(spec.targetFile, []byte(newContent), 0o600); err != nil {
		return res, "", fmt.Errorf("cannot write %s: %w", spec.targetFile, err)
	}

	if len(existing) == 0 {
		res.Created = true
		msg := fmt.Sprintf("created %s with spotpilot %s instructions", spec.targetLabel, spec.commandName)
		return res, msg, nil
	}

	res.Updated = true
	msg := fmt.Sprintf("appended spotpilot %s instructions to %s", spec.commandName, spec.targetLabel)
	return res, msg, nil
}
