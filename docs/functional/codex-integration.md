# Integrating Spotpilot with OpenAI Codex CLI

This guide shows how to let Codex use `spotpilot` as its music-control tool.
The objective is simple: when a user asks for music actions in natural language,
Codex translates that intent into deterministic `spotpilot` CLI commands.

## Why this works well

`spotpilot` is JSON-first and non-interactive by default, which is exactly what
an agent needs:

- predictable command surface
- machine-parseable responses
- stable error categories and exit codes

That means Codex can execute a command, inspect JSON output, and decide what to
say or do next.

---

## Prerequisites

1. Install `spotpilot`:

```sh
brew install ignatij/spotpilot/spotpilot
```

2. Verify installation:

```sh
spotpilot version
```

3. Authenticate once (opens browser flow):

```sh
spotpilot login
```

4. Confirm Codex CLI can execute shell commands in your environment.

---

## Recommended intent-to-command mapping

Use this as your stable contract between natural language and CLI:

| User intent | Command |
|---|---|
| Play a track/query | `spotpilot play "<query>"` |
| Pause playback | `spotpilot pause` |
| Resume playback | `spotpilot resume` |
| Skip to next track | `spotpilot next` |
| Go to previous track | `spotpilot previous` |
| Show currently playing status | `spotpilot status` |
| Show CLI version/build | `spotpilot version` |

Notes:

- Keep query text quoted.
- Do not add extra words to command args.
- Prefer one command per user intent unless recovery is required.

---

## Suggested Codex system instructions

Add the following to your Codex system prompt/profile for this workspace:

```text
When user intent is Spotify playback control, use the `spotpilot` CLI.

Command mapping:
- play music X -> spotpilot play "X"
- pause -> spotpilot pause
- resume -> spotpilot resume
- next -> spotpilot next
- previous/back -> spotpilot previous
- what is playing/status -> spotpilot status

Execution and response policy:
- Execute exactly one mapped command unless command fails.
- Parse JSON output from stdout as source of truth.
- If ok=true, summarize result in one concise sentence.
- If ok=false or non-zero exit code, summarize the error and suggest one recovery action.
- Never fabricate playback state; only report fields present in command output.
- Keep responses short and operational.
```

---

## Example end-to-end flow

### User asks

> Please play Master of Puppets on Spotify.

### Codex executes

```sh
spotpilot play "Master of Puppets"
```

### Typical machine output

```json
{"ok":true,"command":"play","state":"ok","message":"Playing: Metallica — Master of Puppets","result":{"track":"Master of Puppets","artist":"Metallica","device":"MacBook Pro"}}
```

### Codex replies to user

> Playing “Master of Puppets” by Metallica on your active Spotify device.

---

## Error-handling behavior for Codex

If command fails, Codex should stay deterministic:

1. Report concise reason (from JSON `message` or stderr).
2. Suggest one likely fix.
3. Optionally run one diagnostic command if user asked for help.

Examples:

- Not logged in/auth expired → ask user to run `spotpilot login`
- No active device → ask user to open Spotify on a device, then retry
- Query not found → ask user for a narrower search string

---

## Optional: command wrappers (if your Codex setup supports tools)

If your Codex environment supports named tools, define wrappers like:

- `spotify_play(query)` -> `spotpilot play "{query}"`
- `spotify_pause()` -> `spotpilot pause`
- `spotify_status()` -> `spotpilot status`

This keeps prompt logic cleaner while preserving the same CLI contract.

---

## Relationship to repository agent docs

This repo already includes [AGENTS.md](../../AGENTS.md), which describes how
coding agents should work with the codebase.  Keep this Codex integration guide
focused on **runtime command routing** (user intent -> CLI command), while
`AGENTS.md` remains focused on **development workflow**.
