# Error Model

## Error Handling Principles

- Use a small, stable set of user-facing error categories
- Keep error messages concise and machine-parseable by default
- Provide richer debugging context only in verbose or debug modes
- Wrap errors with context using `%w`
- Only introduce custom error types or sentinels when callers need branching behavior
- Treat no-op and idempotent outcomes as success, not as errors

## Initial Error Categories

- `validation`
- `auth`
- `not_found`

These categories should map to stable, documented exit codes from the start.

## Boundary Responsibilities

- Domain and application layers can express typed errors or categorized errors where it helps model behavior
- `internal/output` renders structured error envelopes
- Runtime/process boundary code maps categorized errors to process exit codes
- Keep shared error taxonomy close to `internal/domain` and boundary layers for now; do not add a separate `internal/errs` package yet

## Validation Errors

- Treat user input and configuration validation as first-class validation errors
- Prefer failing before effectful execution whenever possible
- Normalize and validate path inputs at the boundary before effectful code uses them
