# Output

## Output Contract

- Standard command results must be stable and machine-readable on `stdout`
- JSON is the primary stability target
- YAML and text are convenience formats derived from the same logical result model
- Human-oriented diagnostics belong on `stderr`
- Omit incidental timestamps and other environment-dependent noise unless they are part of the real business result
- Prefer `snake_case` field names in structured output

## Output Formats

- Support `--output json|yaml|text`
- The `version` command should use the same output system and support structured formats too
- `version` output should expose build-time metadata including version, commit, and build date
- Keep default text output plain, compact, and concise
- Add detail through `--verbose` and `--debug`
- YAML is primarily a human-friendly structured view; JSON is the canonical automation target

## Standard Envelope

Use a consistent envelope across commands.

- Success shape: `{"ok": true, "data": ...}`
- Error shape: `{"ok": false, "error": ...}`

Keep shared metadata minimal at first. Add common metadata fields only when a concrete need appears.

For mutating commands:

- Prefer dry-run support when the command can make changes
- Dry-run output should use the same envelope and output formats as real execution
- No-op or idempotent success should use exit code `0` and structured success details such as `changed: false` or `result: "no_change"`
- When relevant, include a small standard change summary describing created, updated, or deleted effects

## Mapping Rules

- Command and application results should stay presentation-free
- `internal/output` owns envelope models and rendering
- Map domain/application results into output DTOs at the boundary
- Do not put transport or rendering concerns into domain objects
- Text output should be rendered from the same underlying result model as JSON and YAML
