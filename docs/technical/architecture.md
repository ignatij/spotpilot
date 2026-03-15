# Architecture

`spotpilot` is a fast, agent-friendly Go CLI with a thin Cobra command layer over an application layer, a focused domain layer, and explicit adapters for external effects.

## Architectural Shape

- Keep the executable entrypoint thin
- Keep Cobra command builders thin and focused on CLI concerns
- Put use-case orchestration in `internal/app`
- Keep domain concepts and domain rules in `internal/domain`
- Keep external service adapters in `internal/integrations`
- Keep configuration, rendering, and runtime/process concerns in dedicated boundary packages
- Avoid `pkg/` unless the repository intentionally needs a reusable public package

## Dependency Direction

The intended dependency direction is:

`cmd -> internal/app -> internal/domain`

Adapters depend inward on consumer-owned ports exposed by the consuming layer.

- `cmd` depends on `internal/app`, `internal/config`, `internal/output`, and runtime wiring
- `internal/app` owns use-case orchestration, use-case models, and consumer-owned ports for external dependencies
- `internal/domain` contains business concepts, domain rules, and domain-level errors where needed
- `internal/integrations` implements ports owned by `internal/app`
- Boundary layers translate application/domain results and errors into user-facing output and exit behavior

## Structural Conventions

- Keep packages few and cohesive; split only when responsibilities clearly diverge
- Organize `internal/app` by use case or feature
- Organize `internal/domain` by domain area as the model grows
- Organize `internal/integrations` by external system or adapter
- Keep shared command helpers minimal rather than building a heavy internal command framework
- Prefer explicit constructor wiring over a DI framework
- Use `context.Context` at command and external integration boundaries from the start
- Keep exports minimal; prefer unexported types and functions unless cross-package use requires otherwise
- Prefer singular, verb-first, feature-oriented package names
- Keep command and app package names aligned where practical
- Return concrete types from constructors by default; use interfaces at consumer boundaries
- Keep interfaces small and focused

## Ports and Adapters

- Use small abstractions for filesystem and external service access
- Define ports deliberately where external effects exist
- Ports/interfaces are owned by the consuming layer, not by the adapter package
- Do not let adapter-driven interfaces shape the core design
- Wrap third-party clients behind repository-owned adapters
- Do not leak vendor client or response types across the application boundary
- Map integration data into app-owned or domain-owned models before wider use
- Keep vendor-specific auth and config translation inside integration packages

### v1 Integration Boundaries

- **Spotify Web API** — all playback control and search goes through the Spotify Web API, not through direct desktop-app control
- **OAuth / Token Storage** — handles the Spotify OAuth PKCE flow, stores tokens in OS keychain with protected-file fallback
- **Browser Launch** — opens the OS default browser for OAuth; abstracted behind a port so tests can substitute it

### Workflow Composition

- Commands like `play` may auto-trigger `login` when no valid session exists; this is app-layer workflow composition, not domain logic
- Keep auto-trigger logic in `internal/app` use cases, not in Cobra command wiring or domain packages

## Execution Flow

Commands should follow a consistent shape:

`load config -> validate inputs -> run use case -> render result -> map error to exit code`

Also:

- Keep startup cheap and side-effect free
- Defer expensive work until the selected command actually runs
- Separate parsing and validation from effectful execution
- Prefer immutable input structs and explicit returns where practical
- Map flags and resolved config into explicit input structs before calling the application layer
- Let commands and the application layer compose workflows; do not push CLI workflow composition into domain packages
- Centralize dependency wiring at the root command and initialize heavier dependencies lazily where possible
