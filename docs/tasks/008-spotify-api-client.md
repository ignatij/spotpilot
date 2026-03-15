# Task 008: Spotify API Client Adapter

## Goal

Implement the Spotify Web API client adapter for search, playback control, and device listing.

## Prerequisites

- Task 005 (domain models)

## Scope

- [ ] Define Spotify API port(s) in `internal/app` covering:
  - Search (tracks, albums, artists)
  - Start playback (by track/album/artist URI, on specific device)
  - Pause playback
  - Resume playback
  - Next track
  - Previous track
  - Get current playback state
  - List available devices
- [ ] Implement adapter in `internal/integrations/spotify`:
  - HTTP client wrapping Spotify Web API
  - Map Spotify API responses to domain-owned models
  - Handle auth headers (session/token injection)
  - Conservative timeouts on API calls
  - Do not leak Spotify API types across the boundary
- [ ] Add unit tests for response mapping logic
- [ ] Add test fakes for the Spotify API port

## References

- `docs/technical/architecture.md` (ports and adapters, v1 integration boundaries)
- `docs/technical/repository-layout.md` (integrations/spotify)
- `docs/functional/v1-spec.md` §5.2

## Acceptance Criteria

- All v1 playback and search operations are covered
- Spotify API response types do not leak into `internal/app` or `internal/domain`
- Adapter is testable with fakes
- Timeouts are configurable
