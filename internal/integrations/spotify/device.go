package spotify

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/domain"
)

// DeviceDetector implements the app.DeviceDetector port by polling the Spotify
// API until a local device appears or the context deadline is exceeded.
type DeviceDetector struct {
	client   app.SpotifyClient
	hostname string
	interval time.Duration
}

// NewDeviceDetector creates a DeviceDetector.
// hostname is used to identify the local machine's device.
func NewDeviceDetector(client app.SpotifyClient, hostname string) *DeviceDetector {
	return &DeviceDetector{client: client, hostname: hostname, interval: 1 * time.Second}
}

// WaitForLocalDevice polls until a local device is found or ctx is cancelled.
func (d *DeviceDetector) WaitForLocalDevice(ctx context.Context) (*domain.Device, error) {
	var lastDevices []domain.Device
	var lastListErr error

	for {
		devices, err := d.client.ListDevices(ctx)
		if err == nil {
			lastDevices = devices
			var fallbackSpotpilot *domain.Device
			for i := range devices {
				if isLocal(&devices[i], d.hostname) {
					if isSpotpilotDevice(&devices[i]) {
						if fallbackSpotpilot == nil {
							fallbackSpotpilot = &devices[i]
						}
						continue
					}
					return &devices[i], nil
				}
			}
			if fallbackSpotpilot != nil {
				return fallbackSpotpilot, nil
			}
		} else {
			lastListErr = err
		}

		select {
		case <-ctx.Done():
			parts := []string{fmt.Sprintf("waiting for local device: %v", ctx.Err())}
			if len(lastDevices) > 0 {
				parts = append(parts, fmt.Sprintf("seen devices=%s", summarizeDevices(lastDevices, d.hostname)))
			}
			if lastListErr != nil {
				parts = append(parts, fmt.Sprintf("last list_devices error=%v", lastListErr))
			}
			return nil, errors.New(strings.Join(parts, "; "))
		case <-time.After(d.interval):
		}
	}
}

func isSpotpilotDevice(d *domain.Device) bool {
	return strings.EqualFold(strings.TrimSpace(d.Name), "spotpilot")
}

func summarizeDevices(devices []domain.Device, hostname string) string {
	if len(devices) == 0 {
		return "[]"
	}
	const maxItems = 6
	items := make([]string, 0, len(devices))
	for _, device := range devices {
		tags := make([]string, 0, 3)
		if device.IsLocal {
			tags = append(tags, "local")
		}
		if device.IsActive {
			tags = append(tags, "active")
		}
		if hostname != "" && device.Name == hostname {
			tags = append(tags, "hostname")
		}
		descriptor := fmt.Sprintf("%s(type=%s", device.Name, device.Type)
		if len(tags) > 0 {
			descriptor += "," + strings.Join(tags, ",")
		}
		descriptor += ")"
		items = append(items, descriptor)
	}
	if len(items) > maxItems {
		remaining := len(items) - maxItems
		items = append(items[:maxItems], fmt.Sprintf("...+%d more", remaining))
	}
	return "[" + strings.Join(items, "; ") + "]"
}

// isLocal returns true if the device appears to be on the local machine.
func isLocal(d *domain.Device, hostname string) bool {
	if hostname != "" && strings.EqualFold(strings.TrimSpace(d.Name), strings.TrimSpace(hostname)) {
		return true
	}
	if d.IsLocal {
		return true
	}
	// Spotify desktop apps report type "Computer".
	return strings.EqualFold(strings.TrimSpace(d.Type), "Computer")
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
