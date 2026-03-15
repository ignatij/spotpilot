# Task 007: Browser Detection and Launch

## Goal

Implement Chrome/Chromium detection and browser launch for the login flow, abstracted behind a port.

## Prerequisites

- Task 001 (project scaffolding)

## Scope

- [ ] Define a browser-launch port in `internal/app`
- [ ] Implement browser adapter in `internal/integrations/browser`:
  - Detect Google Chrome installation (macOS + Linux paths)
  - Detect Chromium installation (macOS + Linux paths)
  - Preference order: Chrome first, then Chromium
  - Fail clearly if neither is installed
- [ ] Implement launch function to open a URL in the detected browser
- [ ] Separate browser launch for login (Chrome/Chromium only) from web player fallback (default browser acceptable)
- [ ] Add unit tests with fakes for the browser port

## References

- `docs/technical/architecture.md` (browser launch as integration boundary)
- `docs/technical/runtime.md` (external process interactions)
- `docs/functional/v1-spec.md` §9

## Acceptance Criteria

- Chrome is preferred over Chromium
- Clear error if neither browser is found
- Browser port is substitutable in tests
- Works on both macOS and Linux
