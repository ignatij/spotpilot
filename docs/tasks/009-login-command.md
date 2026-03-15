# Task 009: Login Command

## Goal

Implement the `spotpilot login` command with browser-based session/cookie import from Chrome/Chromium.

## Prerequisites

- Task 002 (output system)
- Task 003 (error handling)
- Task 006 (session storage)
- Task 007 (browser detection)

## Scope

- [ ] Implement login use case in `internal/app`:
  - Check for existing valid session → return success without reopening login
  - If no valid session: launch Chrome/Chromium, open Spotify login
  - Wait for user authentication with bounded timeout
  - Import browser session/cookies
  - Save session via credential storage port
- [ ] Implement cookie/session import from Chrome/Chromium browser profiles:
  - Read Chrome cookie database
  - Extract Spotify session cookies
  - Handle encrypted cookies (macOS Keychain, Linux wallet)
- [ ] Wire `login` Cobra command in `cmd/spotpilot`
- [ ] Return proper envelope: `ok: true, command: "login", state: "playing"` or appropriate state
- [ ] Handle timeout expiry with clear error
- [ ] Add tests for the login use case with faked dependencies

## References

- `docs/functional/v1-spec.md` §5.1, §8.1
- `docs/technical/architecture.md` (workflow composition)

## Acceptance Criteria

- Idempotent: re-running login with valid session returns success
- Times out with clear error if user doesn't complete login
- Session is persisted via credential storage
- Uses correct browser preference (Chrome > Chromium)
- Output follows standard envelope
