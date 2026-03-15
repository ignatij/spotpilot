# Task 010: Device Resolution

## Goal

Implement the local device resolution and recovery flow: detect local Spotify device, launch desktop app, fall back to web player.

## Prerequisites

- Task 008 (Spotify API client)
- Task 007 (browser detection)

## Scope

- [ ] Implement device resolution logic in `internal/app`:
  1. Check for existing local Spotify device via API
  2. If not found: launch Spotify desktop app
  3. Wait for local desktop device with bounded timeout
  4. If still not found: open Spotify web player in default browser
  5. Wait for browser device with bounded timeout
  6. If still not found: fail with clear error
- [ ] Define what "local device" means (match by hostname or device type)
- [ ] Implement Spotify desktop app launch (macOS `open -a Spotify`, Linux equivalent)
- [ ] Ensure web player fallback uses default browser (not restricted to Chrome/Chromium)
- [ ] Never target remote devices (phones, speakers, other computers)
- [ ] Make timeout values configurable
- [ ] Add tests for each step of the recovery flow with faked dependencies

## References

- `docs/functional/v1-spec.md` §10, §11
- `docs/technical/runtime.md` (external process interactions)

## Acceptance Criteria

- Recovery order: existing device → desktop app → web player → fail
- Only local devices are targeted
- All waits are bounded
- Timeout values come from config
- Each fallback step is testable independently
