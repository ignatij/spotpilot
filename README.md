# spotpilot

Control Spotify from the command line — designed for AI agents.

## What it is

`spotpilot` is a Go CLI that exposes deterministic playback commands and
JSON-first output so agents (including Codex-style CLI agents) can reliably
execute and reason about Spotify actions.

## Install

### Homebrew (recommended)

```sh
brew install ignatij/spotpilot/spotpilot
```

### Build from source

```sh
go build ./...
```

## Quick start

```sh
spotpilot login
spotpilot play "Master of Puppets"
spotpilot status
```

## Core commands

- `spotpilot login`
- `spotpilot play "<query>"`
- `spotpilot pause`
- `spotpilot resume`
- `spotpilot next`
- `spotpilot previous`
- `spotpilot status`
- `spotpilot version`

## Documentation

### Functional docs

- [Functional spec](docs/functional/v1-spec.md)
- [Business overview](docs/functional/business-overview.md)
- [Homebrew release guide](docs/functional/releasing-homebrew.md)
- [Codex integration guide](docs/functional/codex-integration.md)

### Technical docs

- [Technical README](docs/technical/README.md)
- [Architecture baseline](docs/technical/architecture.md)
- [Architecture walkthrough](docs/technical/architecture-walkthrough.md)

## Development

```sh
# full local CI parity
make ci

# run tests
go test ./...

# run the CLI
go run ./cmd/spotpilot status
```
