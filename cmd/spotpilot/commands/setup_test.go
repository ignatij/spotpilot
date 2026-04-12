package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyInstructionSetupCreatesFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, ".copilot", "copilot-instructions.md")

	res, msg, err := applyInstructionSetup(instructionSetupSpec{
		targetFile:    target,
		targetDir:     filepath.Dir(target),
		commandName:   "copilot setup",
		targetLabel:   target,
		presentNeedle: "use the `spotpilot` CLI",
		block:         copilotToolBlock,
	})
	if err != nil {
		t.Fatalf("applyInstructionSetup returned error: %v", err)
	}
	if !res.Created || res.Updated || res.Skipped {
		t.Fatalf("unexpected result: %+v", res)
	}
	if !strings.Contains(msg, "created") {
		t.Fatalf("unexpected message: %q", msg)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading created file: %v", err)
	}
	if string(data) != strings.TrimLeft(copilotToolBlock, "\n") && string(data) != copilotToolBlock {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestApplyInstructionSetupAppendsToExistingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, ".codex", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatalf("creating parent dir: %v", err)
	}
	const existing = "# Existing instructions\n"
	if err := os.WriteFile(target, []byte(existing), 0o600); err != nil {
		t.Fatalf("writing existing file: %v", err)
	}

	res, _, err := applyInstructionSetup(instructionSetupSpec{
		targetFile:    target,
		targetDir:     filepath.Dir(target),
		commandName:   "codex setup",
		targetLabel:   target,
		presentNeedle: "**Spotify Interactions**",
		block:         codexToolBlock,
	})
	if err != nil {
		t.Fatalf("applyInstructionSetup returned error: %v", err)
	}
	if !res.Updated || res.Created || res.Skipped {
		t.Fatalf("unexpected result: %+v", res)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading updated file: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, strings.TrimRight(existing, "\n")) {
		t.Fatalf("expected existing content to be preserved, got: %q", content)
	}
	if !strings.Contains(content, "**Spotify Interactions**") {
		t.Fatalf("expected codex block to be appended, got: %q", content)
	}
}

func TestApplyInstructionSetupSkipsExistingSpotpilotBlock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, ".copilot", "copilot-instructions.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatalf("creating parent dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(copilotToolBlock), 0o600); err != nil {
		t.Fatalf("writing existing spotpilot block: %v", err)
	}

	res, _, err := applyInstructionSetup(instructionSetupSpec{
		targetFile:    target,
		targetDir:     filepath.Dir(target),
		commandName:   "copilot setup",
		targetLabel:   target,
		presentNeedle: "use the `spotpilot` CLI",
		block:         copilotToolBlock,
	})
	if err != nil {
		t.Fatalf("applyInstructionSetup returned error: %v", err)
	}
	if !res.Skipped || res.Created || res.Updated {
		t.Fatalf("unexpected result: %+v", res)
	}
}
