package commands

import (
	"context"
	"time"

	"github.com/ignatij/spotpilot/internal/config"
)

func noopCancel() {}

func withLoginTimeout(ctx context.Context, cfg config.Config) (context.Context, context.CancelFunc) {
	return withTimeoutOrNoop(ctx, cfg.Timeouts.Login)
}

func withPlaybackTimeout(ctx context.Context, cfg config.Config) (context.Context, context.CancelFunc) {
	total := cfg.Timeouts.DesktopDevice + cfg.Timeouts.DesktopDevice + cfg.Timeouts.WebDevice + 10*time.Second
	if cfg.Timeouts.Login > 0 {
		total += cfg.Timeouts.Login
	}
	return withTimeoutOrNoop(ctx, total)
}

func withTimeoutOrNoop(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, noopCancel
	}
	return context.WithTimeout(ctx, timeout)
}
