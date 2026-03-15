# Tooling

## Primary Tooling

- Build toolchain: Go `1.26`
- Linting: `golangci-lint` with a small curated ruleset
- Release tooling: `goreleaser`
- CI baseline: GitHub Actions running formatting checks, tests, and lint on push and pull request

## Build Metadata

- Inject version metadata at build or release time
- The baseline metadata set is: version, commit, and build date
- The `version` command should expose this metadata through the standard output system
- Use Go linker flags (`-ldflags`) as the standard injection mechanism
- Local development builds should use sensible defaults such as `dev` plus commit or build date when available
- Keep build metadata wiring near `cmd/spotpilot` until real complexity justifies extracting it

## Commands

Use the standard Go toolchain as the source of truth. A thin `Makefile` is acceptable as a convenience layer, not as the real implementation.

```sh
# Build the CLI
go build ./...

# Run the CLI during development
go run ./cmd/spotpilot <args>

# Run all tests
go test ./...

# Run a single test
go test ./path/to/package -run '^TestName$'

# Run lint checks
golangci-lint run

# Create release artifacts
goreleaser release --snapshot --clean
```

## Generation Policy

- Generated artifacts such as shell completions, man pages, or derived docs should normally be produced in build or release workflows
- Do not commit generated artifacts by default unless there is a strong reason
