package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/output"
)

const codexAgentsPath = ".codex/AGENTS.md"

const codexToolBlock = `
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

			res, msg, err := applyInstructionSetup(instructionSetupSpec{
				targetFile:    agentsFile,
				targetDir:     dir,
				commandName:   "codex setup",
				targetLabel:   agentsFile,
				presentNeedle: "**Spotify Interactions**",
				block:         codexToolBlock,
			})
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
