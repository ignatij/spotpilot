# Repository Layout

This repository should start with a small, explicit package set and only grow when new responsibilities clearly justify it.

## Initial Layout

- `cmd/spotpilot`
- `internal/app`
- `internal/domain`
- `internal/config`
- `internal/output`
- `internal/integrations`

## Package Responsibilities

### `cmd/spotpilot`

- Root binary entrypoint
- Root command tree and centralized command registration
- Feature-oriented command packages or files with small builder functions
- Flag wiring and handoff into application use cases
- Map flags and resolved config into explicit input structs
- Centralize dependency wiring and pass resolved dependencies into feature command builders
- Minimal logic only

### `internal/app`

- Use-case orchestration
- Feature/use-case organization
- Application input and result models
- Consumer-owned ports/interfaces for external dependencies
- Explicit result structs even when a use case currently returns only success/failure
- No rendering or process-exit policy

### `internal/domain`

- Core business concepts
- Domain rules and invariants
- Domain-level errors when they model real domain conditions
- No transport/rendering tags
- No config loading, orchestration, or infrastructure concerns

### `internal/config`

- Load, merge, and validate config
- Apply precedence between defaults, config file, environment variables, and flags
- Expose typed configuration to the rest of the system
- Do not become a service locator or dependency container

### `internal/output`

- Success/error envelope models
- Rendering into JSON, YAML, and text
- Mapping application-facing results into presentation-facing DTOs
- No process-exit decisions

### `internal/integrations`

- Adapters for external systems
- Organized by external service or integration concern
- Implement ports owned by `internal/app`
- Wrap vendor clients and translate vendor-specific auth, config, and response data into repository-owned models

## Growth Rules

- Add packages only when responsibilities clearly separate
- Prefer cohesive feature organization over prematurely generic shared packages
- Do not add `pkg/` without an intentional need for a public reusable package
- Decide the exact placement of any cross-command helper package only when real code justifies it
