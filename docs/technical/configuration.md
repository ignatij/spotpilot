# Configuration

## Sources and Precedence

- Configuration sources: flags, environment variables, and an optional config file
- Config file format: YAML
- Precedence: flags override environment variables, environment variables override the config file, and the config file overrides defaults
- Use one centralized config loader that produces a typed config object
- Keep defaults conservative and minimal; fail clearly when important inputs are missing

## File and Directory Conventions

- Use XDG-style or platform-standard directories
- Store config under the standard `spotpilot` config directory
- Default config file name: `config.yaml`
- A missing or empty default config file is acceptable when flags or environment variables are sufficient
- If the user explicitly points to a config path, invalid paths should error clearly

## Validation Rules

- Reject unknown YAML fields by default
- Treat configuration validation failures as first-class validation errors
- Keep `internal/config` narrowly focused on loading, merging, validation, and typed exposure
- Keep validation logic close to the relevant input or config type

## Environment Variables

- Reserve the `SPOTPILOT_*` namespace
- Map environment variables predictably from config keys using uppercase underscore-separated names
- Nested keys should follow the same predictable mapping pattern
- Prefer `snake_case` for config keys so YAML, env mapping, and structured output stay consistent

## Global Flags

The initial shared global flag surface is:

- `--output`
- `--config`
- `--verbose`
- `--debug`

Commands should resolve config before handing off to the application layer, so use cases receive explicit input structs instead of performing config lookups themselves.
