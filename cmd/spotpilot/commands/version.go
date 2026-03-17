package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/output"
)

// versionResult is the structured payload for the version command.
type versionResult struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

func newVersionCmd(info BuildInfo, flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the spotpilot version",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := renderer(flags)
			msg := fmt.Sprintf("spotpilot %s (%s, %s)", info.Version, info.Commit, info.BuildDate)
			env := output.Envelope{
				OK:      true,
				Command: "version",
				State:   output.StateOK,
				Message: msg,
				Result:  versionResult(info),
			}
			return r.Render(env)
		},
	}
}
