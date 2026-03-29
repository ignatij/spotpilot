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
- `spotpilot codex setup`

## Codex / AI agent setup

Run once after install to register spotpilot as the tool of choice for Spotify
interactions inside [OpenAI Codex](https://github.com/openai/codex) (and any
agent that reads `~/.codex/AGENTS.md`):

```sh
spotpilot codex setup
```

This creates or updates `~/.codex/AGENTS.md` with a **Tool Preferences** block
that instructs the agent to use `spotpilot` for all music-related tasks:

```markdown
## Tool Preferences
- **Spotify Interactions**: For any prompt related to playing music, searching
  tracks, or controlling Spotify, strictly use the `spotpilot` CLI tool.
- **Usage**: Use `run_terminal_cmd` to execute `spotpilot` commands.
- **Example Commands**:
  - `spotpilot play <song_name>`
  - `spotpilot pause`
  - `spotpilot next`
```

The command is **idempotent** — if the section already exists it does nothing.
Output follows the standard JSON envelope; use `--plain` for a single summary line.

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
