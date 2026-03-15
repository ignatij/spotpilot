# Task 005: Domain Models

## Goal

Define the core domain types that represent Spotify concepts used across the application.

## Prerequisites

- Task 001 (project scaffolding)

## Scope

- [ ] Define domain types in `internal/domain`:
  - `Track` (title, artist, album)
  - `Album` (name, artist)
  - `Artist` (name)
  - `Device` (name, type, is_local)
  - `MatchResult` with match type (track, album, artist) and resolved entity
  - `PlaybackState` (playing, paused, idle)
  - `Session` (validity check, storage metadata)
- [ ] Define domain-level error types where they model real domain conditions (e.g., `ErrNotFound`)
- [ ] Keep all types free of transport/rendering tags
- [ ] Add unit tests for any domain logic or validation rules

## References

- `docs/technical/architecture.md`
- `docs/technical/repository-layout.md` (domain package responsibilities)
- `docs/functional/v1-spec.md` §5.2 (matching policy), §5.7 (status states)

## Acceptance Criteria

- Domain types are pure — no JSON tags, no infrastructure imports
- Match resolution priority (track > album > artist > not_found) is expressible through the types
- Types are sufficient for all v1 commands
