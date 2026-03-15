# Task 006: Session and Credential Storage

## Goal

Implement the hybrid credential storage layer: OS keychain with protected-file fallback.

## Prerequisites

- Task 005 (domain models)

## Scope

- [ ] Define a session/credential storage port in `internal/app`
- [ ] Implement keychain adapter in `internal/integrations/auth`:
  - macOS Keychain support
  - Linux secret-service / keyring support
- [ ] Implement protected-file fallback adapter:
  - Restricted file permissions (0600)
  - XDG-style state directory
- [ ] Implement session validity check (is stored session still usable?)
- [ ] Implement save, load, and clear operations
- [ ] Auto-detect keychain availability and fall back gracefully
- [ ] Add unit tests with fakes for the storage port

## References

- `docs/technical/runtime.md` (credential storage)
- `docs/technical/open-questions.md` (resolved: hybrid storage)
- `docs/functional/v1-spec.md` §8

## Acceptance Criteria

- Credentials are stored securely when keychain is available
- Falls back to protected file when keychain is unavailable
- Session load/save/clear works on both macOS and Linux
- Storage port is testable with fakes
