package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/output"
)

const copilotInstructionsPath = ".copilot/copilot-instructions.md"

const copilotToolBlock = `
## Tool Preferences
- For prompts about Spotify playback, track search, or music control, use the ` + "`spotpilot`" + ` CLI.
- Prefer these mappings:
  - play music X -> ` + "`spotpilot play \"X\"`" + `
  - pause -> ` + "`spotpilot pause`" + `
  - resume -> ` + "`spotpilot resume`" + `
  - next -> ` + "`spotpilot next`" + `
  - previous -> ` + "`spotpilot previous`" + `
  - status -> ` + "`spotpilot status`" + `
- Parse JSON stdout as the source of truth.
- Do not invent playback state that is not present in command output.
- If authentication is missing or expired, tell the user to run ` + "`spotpilot login`" + `.
`

func newCopilotCmd(flags *rootFlags) *cobra.Command {
	copilot := &cobra.Command{
		Use:   "copilot",
		Short: "Helpers for integrating spotpilot with GitHub Copilot",
	}
	copilot.AddCommand(newCopilotSetupCmd(flags))
	return copilot
}

func newCopilotSetupCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Inject spotpilot tool preferences into ~/.copilot/copilot-instructions.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := renderer(flags)

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}

			instructionsFile := filepath.Join(home, copilotInstructionsPath)
			dir := filepath.Dir(instructionsFile)

			res, msg, err := applyInstructionSetup(instructionSetupSpec{
				targetFile:    instructionsFile,
				targetDir:     dir,
				commandName:   "copilot setup",
				targetLabel:   instructionsFile,
				presentNeedle: "use the `spotpilot` CLI",
				block:         copilotToolBlock,
			})
			if err != nil {
				return err
			}

			env := output.Envelope{
				OK:      true,
				Command: "copilot setup",
				State:   output.StateOK,
				Message: msg,
				Result:  res,
			}
			return r.Render(env)
		},
	}
}
