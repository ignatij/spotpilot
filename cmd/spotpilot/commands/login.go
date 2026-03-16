package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/config"
	"github.com/ignatij/spotpilot/internal/output"
)

func newLoginCmd(flags *rootFlags, cfgFor func() (config.Config, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Spotify",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cfgFor()
			if err != nil {
				return runErr(flags, "login", err)
			}

			deps, err := buildDeps(cfg)
			if err != nil {
				return runErr(flags, "login", err)
			}

			ctx := cmd.Context()
			ctx, cancel := withLoginTimeout(ctx, cfg)
			defer cancel()

			loginUC := app.NewLogin(deps.store, deps.loginPerformer)
			res, err := loginUC.Run(ctx, app.LoginInput{})
			if err != nil {
				return runErr(flags, "login", err)
			}

			r := renderer(flags)
			msg := "Logged in to Spotify"
			if res.AlreadyLoggedIn {
				msg = "Already logged in"
			}
			return r.Render(output.Envelope{
				OK:      true,
				Command: "login",
				State:   output.StateOK,
				Message: msg,
			})
		},
	}
}

// runErr renders the error envelope and returns a non-zero exit via os.Exit.
// Returning a non-nil error from RunE causes Cobra to print usage; instead we
// handle output ourselves and exit directly.
func runErr(flags *rootFlags, command string, err error) error {
	r := renderer(flags)
	env := output.ErrorEnvelope(command, err)
	if renderErr := r.Render(env); renderErr != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", renderErr)
	}
	os.Exit(output.ExitCode(err))
	return nil
}
