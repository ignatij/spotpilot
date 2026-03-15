package main

// BuildInfo holds version metadata injected at build time via -ldflags.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// buildInfo returns the build metadata, defaulting to "dev" values for local builds.
func buildInfo() BuildInfo {
	v := version
	if v == "" {
		v = "dev"
	}
	c := commit
	if c == "" {
		c = "unknown"
	}
	d := buildDate
	if d == "" {
		d = "unknown"
	}
	return BuildInfo{Version: v, Commit: c, BuildDate: d}
}

// These variables are injected via -ldflags at build/release time.
var (
	version   string
	commit    string
	buildDate string
)
