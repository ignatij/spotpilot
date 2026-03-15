# Error Model

## Error Handling Principles

- Use a small, stable set of user-facing error categories
- Keep error messages concise and machine-parseable by default
- Provide richer debugging context only in verbose or debug modes
- Wrap errors with context using `%w`
- Only introduce custom error types or sentinels when callers need branching behavior
- Treat no-op and idempotent outcomes as success, not as errors

## User-Facing States

v1 commands express outcomes through these states in the `state` field:

- `playing` — playback started or continuing
- `paused` — playback paused
- `idle` — logged in but nothing playing; valid state, not an error
- `not_found` — no matching track, album, or artist
- `not_logged_in` — no valid session exists
- `error` — generic failure; internal detail minimized in user-facing output

## Error Categories and Exit Codes

The initial error categories map to stable, documented exit codes:

- `validation` — invalid input, missing required arguments
- `auth` — authentication failures, expired sessions, `not_logged_in` state
- `not_found` — no matching result for the given query

These categories should map to stable exit codes from the start.

## Boundary Responsibilities

- Domain and application layers can express typed errors or categorized errors where it helps model behavior
- `internal/output` renders structured error envelopes with `ok: false` and the appropriate `state`
- Runtime/process boundary code maps categorized errors to process exit codes
- Keep shared error taxonomy close to `internal/domain` and boundary layers for now; do not add a separate `internal/errs` package yet

## Validation Errors

- Treat user input and configuration validation as first-class validation errors
- Prefer failing before effectful execution whenever possible
- Normalize and validate path inputs at the boundary before effectful code uses them

## Internal Detail Minimization

- Avoid exposing backend detail in user-facing responses
- Keep the `message` field concise and actionable
- Reserve verbose/debug modes for richer diagnostic context
