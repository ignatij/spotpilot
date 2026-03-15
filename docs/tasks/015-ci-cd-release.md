# Task 015: CI/CD and Release Pipeline

## Goal

Set up GitHub Actions CI and goreleaser for automated builds, tests, and Homebrew-installable releases.

## Prerequisites

- Task 001 (project scaffolding)
- All other tasks should be substantially complete before cutting a release

## Scope

- [ ] Create GitHub Actions CI workflow (`.github/workflows/ci.yml`):
  - Trigger on push and pull request
  - Run `go build ./...`
  - Run `go test ./...`
  - Run `golangci-lint run`
  - Run `go vet ./...`
  - Check formatting with `gofmt`
- [ ] Set up `golangci-lint` config (`.golangci.yml`) with curated ruleset
- [ ] Create `goreleaser` config (`.goreleaser.yml`):
  - Build single binary for macOS (amd64, arm64) and Linux (amd64, arm64)
  - Inject version/commit/date via `-ldflags`
  - Generate checksums
  - Homebrew tap formula
- [ ] Create release workflow (`.github/workflows/release.yml`):
  - Trigger on version tag push
  - Run goreleaser
- [ ] Create or reference Homebrew tap repository
- [ ] Verify end-to-end: tag → release → `brew install` works

## References

- `docs/technical/tooling.md`
- `docs/technical/baseline.md` (macOS + Linux)
- `docs/functional/v1-spec.md` §2 (Homebrew-installable)

## Acceptance Criteria

- CI runs on every push and PR
- Lint, test, and build all pass in CI
- Tagged releases produce downloadable binaries
- Homebrew formula is generated and installable
