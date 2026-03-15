# Task 011: Play Command

## Goal

Implement the `spotpilot play "<query>"` command — the primary command that ties together login, search, match resolution, device resolution, and playback.

## Prerequisites

- Task 009 (login command)
- Task 008 (Spotify API client)
- Task 010 (device resolution)

## Scope

- [ ] Implement play use case in `internal/app`:
  1. Ensure valid session (auto-trigger login if needed)
  2. Search Spotify with raw query string
  3. Resolve match: track → album → artist → not_found
  4. Select top result automatically (no disambiguation)
  5. Resolve local playback device (full recovery flow)
  6. Start playback on resolved device
  7. Return structured result
- [ ] Wire `play` Cobra command with positional `query` argument
- [ ] Return envelope with `match_type`, `title`, `artist` in `result` field
- [ ] Handle not_found: return `ok: false, state: "not_found"`
- [ ] Handle login auto-trigger: if login succeeds, resume play flow transparently
- [ ] Plain mode: `Playing: <title> — <artist>`
- [ ] Add tests for the play use case with faked search/playback/device dependencies

## References

- `docs/functional/v1-spec.md` §5.2
- `docs/technical/architecture.md` (workflow composition)

## Acceptance Criteria

- Query is passed as-is to Spotify search (no cleanup)
- Match priority: track > album > artist > not_found
- Top result selected automatically
- Auto-login works transparently
- Device resolution uses full recovery flow
- One-shot playback replaces current playback
- Output follows standard envelope
