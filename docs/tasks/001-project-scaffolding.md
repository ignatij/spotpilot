# Task 001: Project Scaffolding

## Goal

Initialize the Go module, create the directory structure, wire up the Cobra root command, and establish the basic build pipeline.

## Prerequisites

None — this is the first task.

## Scope

- [ ] Initialize Go module (`github.com/ignatij/spotpilot`, Go 1.26)
- [ ] Create directory skeleton:
  - `cmd/spotpilot/main.go`
  - `internal/app/`
  - `internal/domain/`
  - `internal/config/`
  - `internal/output/`
  - `internal/integrations/`
- [ ] Add Cobra dependency and wire root command with centralized command registration
- [ ] Add `version` subcommand stub (prints `dev` until build metadata injection is set up)
- [ ] Create a thin `Makefile` with `build`, `test`, `lint`, and `run` targets
- [ ] Verify `go build ./...` and `go test ./...` pass with zero tests
- [ ] Commit the skeleton

## References

- `docs/technical/baseline.md`
- `docs/technical/repository-layout.md`
- `docs/technical/tooling.md`

## Acceptance Criteria

- `go build ./...` succeeds
- `go run ./cmd/spotpilot version` prints version info
- Directory layout matches `docs/technical/repository-layout.md`
