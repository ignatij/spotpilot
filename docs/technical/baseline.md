# Project Baseline

- Module path: `github.com/ignatij/spotpilot`
- Minimum Go version: `1.26`
- Supported platforms: macOS and Linux
- Primary project type: single-binary CLI under `cmd/spotpilot`
- CLI framework: `cobra`
- Dependency philosophy: standard library first; third-party dependencies must clearly justify themselves
- Performance philosophy: simple and sequential by default; add complexity only for proven pain points
- Compatibility philosophy: command names, flags, exit codes, and structured JSON output are compatibility-sensitive once shipped
- Telemetry policy: disabled by default; any future telemetry must be explicit opt-in
