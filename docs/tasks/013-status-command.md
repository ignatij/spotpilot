# Task 013: Status Command

## Goal

Implement `spotpilot status` — a side-effect-free read-only command.

## Prerequisites

- Task 002 (output system)
- Task 003 (error handling)
- Task 006 (session storage)
- Task 008 (Spotify API client)

## Scope

- [ ] Implement status use case in `internal/app`:
  - Check session validity
  - If not logged in: return `state: "not_logged_in"` — do NOT trigger login
  - If logged in: query current playback state via Spotify API
  - Report: login state, active device, current track
- [ ] Handle status states:
  - `playing` — track is playing, include track info
  - `paused` — track is paused, include track info
  - `idle` — logged in but nothing playing (valid state, not error)
  - `not_logged_in` — no valid session
- [ ] Wire `status` Cobra command in `cmd/spotpilot`
- [ ] Return standard envelope; `result` includes device and track info where applicable
- [ ] Status must NOT: trigger login, launch browser, launch Spotify, attempt recovery
- [ ] Add tests for each status state with faked dependencies

## References

- `docs/functional/v1-spec.md` §5.7

## Acceptance Criteria

- Completely side-effect free
- Never triggers login or launches anything
- Reports correct state for all four outcomes
- `idle` is treated as success (`ok: true`), not as error
- `not_logged_in` uses `ok: false` with appropriate state
- Output follows standard envelope
