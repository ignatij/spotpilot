package spotify

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/ignatij/spotpilot/internal/domain"
)

// DeviceDetector implements the app.DeviceDetector port by polling the Spotify
// API until a local device appears or the context deadline is exceeded.
type DeviceDetector struct {
	client   *Client
	hostname string
	interval time.Duration
}

// NewDeviceDetector creates a DeviceDetector.
// hostname is used to identify the local machine's device.
func NewDeviceDetector(client *Client, hostname string) *DeviceDetector {
	return &DeviceDetector{client: client, hostname: hostname, interval: 1 * time.Second}
}

// WaitForLocalDevice polls until a local device is found or ctx is cancelled.
func (d *DeviceDetector) WaitForLocalDevice(ctx context.Context) (*domain.Device, error) {
	for {
		devices, err := d.client.ListDevices(ctx)
		if err == nil {
			for i := range devices {
				if isLocal(&devices[i], d.hostname) {
					return &devices[i], nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for local device: %w", ctx.Err())
		case <-time.After(d.interval):
		}
	}
}

// isLocal returns true if the device appears to be on the local machine.
func isLocal(d *domain.Device, hostname string) bool {
	if hostname != "" && d.Name == hostname {
		return true
	}
	// Spotify desktop apps report type "Computer".
	return d.Type == "Computer"
}

// AppLauncher implements the app.AppLauncher port for launching Spotify desktop.
type AppLauncher struct{}

// NewAppLauncher creates an AppLauncher.
func NewAppLauncher() *AppLauncher { return &AppLauncher{} }

// LaunchSpotify starts the local Spotify desktop application.
func (a *AppLauncher) LaunchSpotify(ctx context.Context) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "open", "-a", "Spotify").Start()
	default:
		// Linux: try common Spotify binary names.
		for _, bin := range []string{"spotify", "spotify-client"} {
			if path, err := exec.LookPath(bin); err == nil {
				return exec.CommandContext(ctx, path).Start()
			}
		}
		return fmt.Errorf("spotify desktop app not found")
	}
}
