package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/output"
)

const codexAgentsPath = ".codex/AGENTS.md"

const spotpilotToolBlock = `
## Tool Preferences
- **Spotify Interactions**: For any prompt related to playing music, searching tracks, or controlling Spotify, strictly use the ` + "`spotpilot`" + ` CLI tool.
- **Usage**: Use ` + "`run_terminal_cmd`" + ` to execute ` + "`spotpilot`" + ` commands.
- **Example Commands**:
  - ` + "`spotpilot play <song_name>`" + `
  - ` + "`spotpilot pause`" + `
  - ` + "`spotpilot next`" + `
`

func newCodexCmd(flags *rootFlags) *cobra.Command {
	codex := &cobra.Command{
		Use:   "codex",
		Short: "Helpers for integrating spotpilot with OpenAI Codex",
	}
	codex.AddCommand(newCodexSetupCmd(flags))
	return codex
}

type codexSetupResult struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
	Updated bool   `json:"updated"`
	Skipped bool   `json:"skipped"`
}

func newCodexSetupCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Inject spotpilot tool preferences into ~/.codex/AGENTS.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := renderer(flags)

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}

			agentsFile := filepath.Join(home, codexAgentsPath)
			dir := filepath.Dir(agentsFile)

			res, msg, err := applyCodexSetup(agentsFile, dir)
			if err != nil {
				return err
			}

			env := output.Envelope{
				OK:      true,
				Command: "codex setup",
				State:   output.StateOK,
				Message: msg,
				Result:  res,
			}
			return r.Render(env)
		},
	}
}

// applyCodexSetup reads (or creates) agentsFile and injects the tool block if absent.
// It returns the result struct, a human-readable message, and any error.
func applyCodexSetup(agentsFile, dir string) (codexSetupResult, string, error) {
	res := codexSetupResult{Path: agentsFile}

	existing, err := os.ReadFile(agentsFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return res, "", fmt.Errorf("cannot read %s: %w", agentsFile, err)
	}

	content := string(existing)

	if strings.Contains(content, "## Tool Preferences") {
		res.Skipped = true
		msg := fmt.Sprintf("%s already contains a Tool Preferences section — no changes made", agentsFile)
		return res, msg, nil
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return res, "", fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	newContent := strings.TrimRight(content, "\n") + spotpilotToolBlock
	if err := os.WriteFile(agentsFile, []byte(newContent), 0o600); err != nil {
		return res, "", fmt.Errorf("cannot write %s: %w", agentsFile, err)
	}

	if len(existing) == 0 {
		res.Created = true
		msg := fmt.Sprintf("created %s with spotpilot Tool Preferences", agentsFile)
		return res, msg, nil
	}

	res.Updated = true
	msg := fmt.Sprintf("appended spotpilot Tool Preferences to %s", agentsFile)
	return res, msg, nil
}
