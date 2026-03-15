# Runtime

## Process Behavior

- Default to deterministic command behavior
- Do not introduce interactive prompts unless a command explicitly opts into interactive behavior
- Keep startup cheap and avoid network calls or heavy initialization before command execution
- Use injected runtime abstractions or writers instead of writing directly to global process I/O in command logic
- Use aliases and shorthand flags conservatively because they expand the compatibility surface
- Mark experimental or hidden features explicitly in both code and docs
- For destructive operations, prefer explicit confirmation flags such as `--yes` in non-interactive usage

## Filesystem Conventions

- Use XDG-style or otherwise platform-standard directories for config, cache, and state
- Keep application files under clearly named `spotpilot` subdirectories
- Default config file name: `config.yaml`
- The durable local storage mechanism beyond config/cache is still undecided
- Prefer explicit path flags over implicit current-directory behavior unless the working directory is clearly part of the contract
- Normalize and validate path inputs at the boundary
- Filesystem mutations should follow the same dry-run and change-summary conventions as other mutating commands

## Time and Randomness

- Do not introduce abstractions for time or randomness until they are needed
- If they become necessary, wrap them behind small helpers so tests can stay deterministic

## Runtime vs Output Responsibilities

- Runtime/process behavior owns command execution flow and exit-code mapping
- `internal/output` owns envelope models and rendering into JSON, YAML, or text
- Introduce a dedicated runtime package only when concrete cross-cutting runtime concerns justify it
- Defer standardizing progress reporting until real long-running workflows require it
