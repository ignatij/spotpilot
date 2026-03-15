package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/config"
	"github.com/ignatij/spotpilot/internal/domain"
	"github.com/ignatij/spotpilot/internal/output"
)

// playResult is the structured payload for the play command.
type playResult struct {
	MatchType string `json:"match_type"`
	Title     string `json:"title,omitempty"`
	Artist    string `json:"artist,omitempty"`
}

func newPlayCmd(flags *rootFlags, cfgFor func() (config.Config, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "play <query>",
		Short: "Search Spotify and play the top result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cfgFor()
			if err != nil {
				return runErr(flags, "play", err)
			}

			deps, err := buildDeps(cfg)
			if err != nil {
				return runErr(flags, "play", err)
			}

			ctx := cmd.Context()
			loginUC := app.NewLogin(deps.store, deps.loginPerformer)
			playUC := app.NewPlay(loginUC, deps.spotify, deps.devices, deps.appLauncher, deps.browser)

			res, err := playUC.Run(ctx, app.PlayInput{Query: args[0]})
			if err != nil {
				return runErr(flags, "play", err)
			}

			r := renderer(flags)
			pr := buildPlayResult(res.Match)
			msg := buildPlayMessage(res.Match)

			return r.Render(output.Envelope{
				OK:      true,
				Command: "play",
				State:   output.StatePlaying,
				Message: msg,
				Result:  pr,
			})
		},
	}
}

func buildPlayResult(match *domain.MatchResult) playResult {
	pr := playResult{MatchType: string(match.Type)}
	switch match.Type {
	case domain.MatchTypeTrack:
		pr.Title = match.Track.Title
		pr.Artist = match.Track.Artist
	case domain.MatchTypeAlbum:
		pr.Title = match.Album.Name
		pr.Artist = match.Album.Artist
	case domain.MatchTypeArtist:
		pr.Title = match.Artist.Name
	}
	return pr
}

func buildPlayMessage(match *domain.MatchResult) string {
	switch match.Type {
	case domain.MatchTypeTrack:
		return fmt.Sprintf("Playing: %s — %s", match.Track.Title, match.Track.Artist)
	case domain.MatchTypeAlbum:
		return fmt.Sprintf("Playing album: %s — %s", match.Album.Name, match.Album.Artist)
	case domain.MatchTypeArtist:
		return fmt.Sprintf("Playing artist: %s", match.Artist.Name)
	}
	return "Playing"
}
