// Package commands registers and wires all Cobra commands for spotpilot.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/config"
	"github.com/ignatij/spotpilot/internal/output"
)

// BuildInfo holds version metadata.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// rootFlags holds the parsed global flags.
type rootFlags struct {
	plain      bool
	configPath string
	verbose    bool
	debug      bool
}

// NewRoot builds and returns the root Cobra command with all subcommands registered.
func NewRoot(info BuildInfo) *cobra.Command {
	var flags rootFlags

	root := &cobra.Command{
		Use:           "spotpilot",
		Short:         "Control Spotify from the command line — built for AI agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&flags.plain, "plain", false, "output a single human-readable line instead of JSON")
	root.PersistentFlags().StringVar(&flags.configPath, "config", "", "path to config file (default: $XDG_CONFIG_HOME/spotpilot/config.yaml)")
	root.PersistentFlags().BoolVar(&flags.verbose, "verbose", false, "enable verbose output")
	root.PersistentFlags().BoolVar(&flags.debug, "debug", false, "enable debug output")

	// cfgFor returns a resolved Config for use in subcommands.
	cfgFor := func() (config.Config, error) {
		cfg, err := config.Load(flags.configPath)
		if err != nil {
			return cfg, err
		}
		cfg.Plain = flags.plain
		cfg.Verbose = flags.verbose
		cfg.Debug = flags.debug
		return cfg, nil
	}

	root.AddCommand(newVersionCmd(info, &flags))
	root.AddCommand(newLoginCmd(&flags, cfgFor))
	root.AddCommand(newPlayCmd(&flags, cfgFor))
	root.AddCommand(newPauseCmd(&flags, cfgFor))
	root.AddCommand(newResumeCmd(&flags, cfgFor))
	root.AddCommand(newNextCmd(&flags, cfgFor))
	root.AddCommand(newPreviousCmd(&flags, cfgFor))
	root.AddCommand(newStatusCmd(&flags, cfgFor))

	return root
}

// renderer builds an output.Renderer from the global flags.
func renderer(flags *rootFlags) *output.Renderer {
	return output.NewRenderer(os.Stdout, flags.plain)
}

// handleErr writes the error envelope and returns the exit code.
func handleErr(r *output.Renderer, command string, err error) int {
	env := output.ErrorEnvelope(command, err)
	if renderErr := r.Render(env); renderErr != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", renderErr)
	}
	return output.ExitCode(err)
}
