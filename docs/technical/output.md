# Output

## Output Contract

- JSON is the **default** output format because Spotpilot is primarily designed for agentic usage
- Standard command results must be stable and machine-readable on `stdout`
- Human-oriented diagnostics belong on `stderr`
- Omit incidental timestamps and other environment-dependent noise unless they are part of the real business result
- Prefer `snake_case` field names in structured output

## Output Formats

- JSON by default
- `--plain` produces a single concise line of human-readable output
- YAML support is deferred beyond v1
- The `version` command should use the same output system
- `version` output should expose build-time metadata including version, commit, and build date
- Add detail through `--verbose` and `--debug`

## Standard Envelope

Every command returns the same top-level JSON structure.

### Required top-level fields

- `ok` — boolean success indicator
- `command` — the command that was executed
- `state` — the resulting state (e.g., `playing`, `paused`, `idle`, `not_logged_in`, `not_found`, `error`)
- `message` — concise human-readable summary

### Optional fields

- `result` — command-specific payload with additional structured data

### Success example

```json
{
  "ok": true,
  "command": "play",
  "state": "playing",
  "message": "Playing Master of Puppets",
  "result": {
    "match_type": "track",
    "title": "Master of Puppets",
    "artist": "Metallica"
  }
}
```

### Error example

```json
{
  "ok": false,
  "command": "play",
  "state": "not_found",
  "message": "No matching track, album, or artist found"
}
```

### Plain mode example

```
Playing: Master of Puppets — Metallica
```

### Minimal payload principle

Keep JSON payloads minimal. Do not include extra Spotify URLs/URIs unless needed later.

For mutating commands:

- Prefer dry-run support when the command can make changes
- Dry-run output should use the same envelope as real execution
- No-op or idempotent success should use exit code `0` with structured details such as `state: "idle"`
- When relevant, include a small standard change summary describing effects

## Mapping Rules

- Command and application results should stay presentation-free
- `internal/output` owns envelope models and rendering
- Map domain/application results into output DTOs at the boundary
- Do not put transport or rendering concerns into domain objects
- Plain text output should be rendered from the same underlying result model as JSON
