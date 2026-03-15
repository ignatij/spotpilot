# Task 004: Configuration System

## Goal

Implement the typed config loader with support for YAML file, environment variables, and flag precedence.

## Prerequisites

- Task 001 (project scaffolding)

## Scope

- [ ] Implement `internal/config` with typed config struct
- [ ] Support loading from YAML file (`config.yaml` in XDG-style `spotpilot` directory)
- [ ] Support `SPOTPILOT_*` environment variables
- [ ] Implement precedence: flags > env vars > config file > defaults
- [ ] Reject unknown YAML fields
- [ ] Wire `--config` global flag for custom config file path
- [ ] Wire `--verbose` and `--debug` global flags
- [ ] Add configurable timeout values (login, desktop device, web player device)
- [ ] Add unit tests for precedence and validation

## References

- `docs/technical/configuration.md`
- `docs/functional/v1-spec.md` §11

## Acceptance Criteria

- Config loads correctly from all sources with correct precedence
- Unknown YAML keys cause a clear error
- Default timeout values are conservative and documented
- Tests cover merge logic, env var mapping, and validation
