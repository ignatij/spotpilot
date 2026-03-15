# Task 012: Playback Control Commands

## Goal

Implement `spotpilot pause`, `spotpilot resume`, `spotpilot next`, and `spotpilot previous`.

## Prerequisites

- Task 010 (device resolution)
- Task 009 (login command)
- Task 008 (Spotify API client)

## Scope

- [ ] Implement use cases in `internal/app` for:
  - `pause` — pause current playback on local device
  - `resume` — resume paused playback on local device
  - `next` — skip to next track on local device
  - `previous` — go to previous track on local device
- [ ] All four commands share:
  - Same login/session handling as `play` (auto-trigger login)
  - Same device recovery flow as `play`
  - Local device targeting only
- [ ] Wire Cobra commands in `cmd/spotpilot`
- [ ] Return standard envelope with appropriate `state` (`paused`, `playing`)
- [ ] Add tests for each use case with faked dependencies

## References

- `docs/functional/v1-spec.md` §5.3–§5.6

## Acceptance Criteria

- All four commands auto-trigger login if needed
- All four commands use full device recovery flow
- All four commands target local device only
- Output follows standard envelope with correct `state`
- Tests cover success and no-device-found paths
