package commands

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/config"
	"github.com/ignatij/spotpilot/internal/output"
)

type playbackRunner interface {
	Run(ctx context.Context, in app.PlaybackInput) (*app.PlaybackResult, error)
}

func makeSimplePlayback(
	command string,
	state output.State,
	msg string,
	flags *rootFlags,
	cfgFor func() (config.Config, error),
	ucFn func(*deps) playbackRunner,
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := cfgFor()
		if err != nil {
			return runErr(flags, command, err)
		}
		d, err := buildDeps(cfg)
		if err != nil {
			return runErr(flags, command, err)
		}

		uc := ucFn(d)
		ctx, cancel := withPlaybackTimeout(cmd.Context(), cfg)
		defer cancel()

		if _, err := uc.Run(ctx, app.PlaybackInput{}); err != nil {
			return runErr(flags, command, err)
		}
		return renderer(flags).Render(output.Envelope{
			OK:      true,
			Command: command,
			State:   state,
			Message: msg,
		})
	}
}

func newPauseCmd(flags *rootFlags, cfgFor func() (config.Config, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause Spotify playback",
		RunE: makeSimplePlayback("pause", output.StatePaused, "Playback paused", flags, cfgFor, func(d *deps) playbackRunner {
			return app.NewPause(app.NewLogin(d.store, d.loginPerformer), d.spotify, d.devices, d.appLauncher, d.browser)
		}),
	}
}

func newResumeCmd(flags *rootFlags, cfgFor func() (config.Config, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume Spotify playback",
		RunE: makeSimplePlayback("resume", output.StatePlaying, "Playback resumed", flags, cfgFor, func(d *deps) playbackRunner {
			return app.NewResume(app.NewLogin(d.store, d.loginPerformer), d.spotify, d.devices, d.appLauncher, d.browser)
		}),
	}
}

func newNextCmd(flags *rootFlags, cfgFor func() (config.Config, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Skip to next track",
		RunE: makeSimplePlayback("next", output.StatePlaying, "Skipped to next track", flags, cfgFor, func(d *deps) playbackRunner {
			return app.NewNext(app.NewLogin(d.store, d.loginPerformer), d.spotify, d.devices, d.appLauncher, d.browser)
		}),
	}
}

func newPreviousCmd(flags *rootFlags, cfgFor func() (config.Config, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "previous",
		Short: "Go to previous track",
		RunE: makeSimplePlayback("previous", output.StatePlaying, "Went to previous track", flags, cfgFor, func(d *deps) playbackRunner {
			return app.NewPrevious(app.NewLogin(d.store, d.loginPerformer), d.spotify, d.devices, d.appLauncher, d.browser)
		}),
	}
}
