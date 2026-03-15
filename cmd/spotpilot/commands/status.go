package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/config"
	"github.com/ignatij/spotpilot/internal/domain"
	"github.com/ignatij/spotpilot/internal/output"
)

// statusResult is the structured payload for the status command.
type statusResult struct {
	LoggedIn bool        `json:"logged_in"`
	Device   *deviceInfo `json:"device,omitempty"`
	Track    *trackInfo  `json:"track,omitempty"`
}

type deviceInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type trackInfo struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

func newStatusCmd(flags *rootFlags, cfgFor func() (config.Config, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current Spotify playback status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cfgFor()
			if err != nil {
				return runErr(flags, "status", err)
			}

			d, err := buildDeps(cfg)
			if err != nil {
				return runErr(flags, "status", err)
			}

			uc := app.NewStatus(d.store, d.spotify)
			res, err := uc.Run(cmd.Context(), app.StatusInput{})
			if err != nil {
				return runErr(flags, "status", err)
			}

			return renderer(flags).Render(buildStatusEnvelope(res))
		},
	}
}

func buildStatusEnvelope(res *app.StatusResult) output.Envelope {
	if !res.LoggedIn {
		return output.Envelope{
			OK:      false,
			Command: "status",
			State:   output.StateNotLoggedIn,
			Message: "Not logged in",
		}
	}

	sr := statusResult{LoggedIn: true}
	state := output.StateIdle
	msg := "Idle — nothing playing"

	if res.Playback != nil {
		if res.Playback.Device != nil {
			sr.Device = &deviceInfo{
				Name: res.Playback.Device.Name,
				Type: res.Playback.Device.Type,
			}
		}
		if res.Playback.Track != nil {
			sr.Track = &trackInfo{
				Title:  res.Playback.Track.Title,
				Artist: res.Playback.Track.Artist,
			}
		}
		switch res.Playback.State {
		case domain.PlaybackStatePlaying:
			state = output.StatePlaying
			msg = buildNowPlayingMsg(res.Playback)
		case domain.PlaybackStatePaused:
			state = output.StatePaused
			msg = fmt.Sprintf("Paused: %s", buildNowPlayingMsg(res.Playback))
		}
	}

	return output.Envelope{
		OK:      true,
		Command: "status",
		State:   state,
		Message: msg,
		Result:  sr,
	}
}

func buildNowPlayingMsg(pb *domain.CurrentPlayback) string {
	if pb.Track == nil {
		return "Playing"
	}
	if pb.Track.Artist != "" {
		return fmt.Sprintf("Playing: %s — %s", pb.Track.Title, pb.Track.Artist)
	}
	return fmt.Sprintf("Playing: %s", pb.Track.Title)
}
