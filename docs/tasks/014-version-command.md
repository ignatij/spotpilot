# Task 014: Version Command

## Goal

Finalize the `spotpilot version` command with build metadata injection via `-ldflags`.

## Prerequisites

- Task 001 (project scaffolding — stub already exists)
- Task 002 (output system)

## Scope

- [ ] Define version variables in `cmd/spotpilot` (version, commit, build date)
- [ ] Wire `-ldflags` injection in `Makefile` and document in `goreleaser` config
- [ ] Return version info through the standard envelope:
  ```json
  {
    "ok": true,
    "command": "version",
    "state": "ok",
    "message": "spotpilot v0.1.0",
    "result": {
      "version": "0.1.0",
      "commit": "abc1234",
      "build_date": "2026-03-15"
    }
  }
  ```
- [ ] Default to `dev` for local builds without ldflags
- [ ] `--plain` outputs: `spotpilot v0.1.0 (abc1234, 2026-03-15)`
- [ ] `--version` flag is NOT required in v1
- [ ] Add test for version output rendering

## References

- `docs/technical/tooling.md` (build metadata)
- `docs/functional/v1-spec.md` §5.8

## Acceptance Criteria

- Version command uses standard envelope
- Local dev builds show `dev` version
- Release builds show injected metadata
- Works with both JSON and `--plain`
