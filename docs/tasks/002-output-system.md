# Task 002: Output System

## Goal

Implement the standard JSON envelope, plain-text rendering, and the `--plain` global flag so all subsequent commands can use a consistent output layer.

## Prerequisites

- Task 001 (project scaffolding)

## Scope

- [ ] Define envelope struct in `internal/output` with fields: `ok`, `command`, `state`, `message`, `result` (optional)
- [ ] Implement JSON renderer (default)
- [ ] Implement plain-text renderer (single concise line)
- [ ] Wire `--plain` as a persistent global flag on the root command
- [ ] Ensure all output goes to `stdout`; diagnostics to `stderr`
- [ ] Add unit tests for both renderers
- [ ] Add unit tests for envelope construction

## References

- `docs/technical/output.md`
- `docs/functional/v1-spec.md` §6

## Acceptance Criteria

- Envelope struct matches v1 spec JSON shape
- JSON output is valid and deterministic
- `--plain` produces a single line of text
- Tests pass for both formats
