# Task 003: Error Handling and Exit Codes

## Goal

Implement the error model with typed error categories and stable exit-code mapping so commands can consistently signal failures.

## Prerequisites

- Task 002 (output system)

## Scope

- [ ] Define error categories: `validation`, `auth`, `not_found`
- [ ] Map each category to a stable non-zero exit code
- [ ] Define user-facing states: `playing`, `paused`, `idle`, `not_logged_in`, `not_found`, `error`
- [ ] Implement error-to-envelope mapping (produces `ok: false` JSON with correct `state` and `message`)
- [ ] Implement exit-code resolver at the process boundary
- [ ] Keep internal error detail out of user-facing `message` field
- [ ] Add unit tests for category-to-exit-code and error-to-envelope mapping

## References

- `docs/technical/error-model.md`
- `docs/functional/v1-spec.md` §7, §12

## Acceptance Criteria

- Each error category maps to a distinct non-zero exit code
- Error envelopes contain `ok: false` with the correct `state`
- Unknown errors default to the generic `error` state
- Tests cover all categories
