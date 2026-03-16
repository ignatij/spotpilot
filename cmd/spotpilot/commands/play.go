package commands

import (
	"fmt"
	"strings"

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
			ctx, cancel := withPlaybackTimeout(ctx, cfg)
			defer cancel()

			loginUC := app.NewLogin(deps.store, deps.loginPerformer)
			playUC := app.NewPlay(loginUC, deps.spotify, deps.devices, deps.appLauncher, deps.browser)

			res, err := playUC.Run(ctx, app.PlayInput{Query: args[0]})
			if err != nil {
				return runErr(flags, "play", err)
			}

			r := renderer(flags)
			pr := buildPlayResult(res.Match, args[0])
			msg := buildPlayMessage(res.Match, args[0])

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

func buildPlayResult(match *domain.MatchResult, query string) playResult {
	pr := playResult{MatchType: string(match.Type)}
	switch match.Type {
	case domain.MatchTypeTrack:
		pr.Title = firstNonEmpty(match.Track.Title, query)
		pr.Artist = match.Track.Artist
	case domain.MatchTypeAlbum:
		pr.Title = firstNonEmpty(match.Album.Name, query)
		pr.Artist = match.Album.Artist
	case domain.MatchTypeArtist:
		pr.Title = firstNonEmpty(match.Artist.Name, query)
	}
	return pr
}

func buildPlayMessage(match *domain.MatchResult, query string) string {
	switch match.Type {
	case domain.MatchTypeTrack:
		title := firstNonEmpty(match.Track.Title, query)
		artist := strings.TrimSpace(match.Track.Artist)
		if artist == "" {
			return fmt.Sprintf("Playing: %s", title)
		}
		return fmt.Sprintf("Playing: %s — %s", title, artist)
	case domain.MatchTypeAlbum:
		title := firstNonEmpty(match.Album.Name, query)
		artist := strings.TrimSpace(match.Album.Artist)
		if artist == "" {
			return fmt.Sprintf("Playing album: %s", title)
		}
		return fmt.Sprintf("Playing album: %s — %s", title, artist)
	case domain.MatchTypeArtist:
		return fmt.Sprintf("Playing artist: %s", firstNonEmpty(match.Artist.Name, query))
	}
	return "Playing"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "unknown"
}
