# Integrating Spotpilot with GitHub Copilot CLI

This guide shows how to let GitHub Copilot CLI use `spotpilot` as its
music-control tool.

## Quick setup

Run:

```sh
spotpilot copilot setup
```

This creates or updates:

```text
~/.copilot/copilot-instructions.md
```

with a spotpilot-specific instructions block that tells Copilot to:

- use `spotpilot` for Spotify playback requests
- map natural-language playback intents to deterministic CLI commands
- trust JSON stdout as the source of truth
- send users to `spotpilot login` when authentication is missing or expired

The command is idempotent, so rerunning it will not duplicate the spotpilot
instructions block.
