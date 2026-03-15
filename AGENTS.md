# Agent Instructions

`spotpilot` is a Go CLI that enables AI agents to control Spotify playback. The v1 functional spec lives in `docs/functional/v1-spec.md`. The technical baseline lives in `docs/technical/`.

Before making architecture-affecting changes, read:

- `docs/technical/README.md`
- `docs/technical/baseline.md`
- `docs/technical/architecture.md`
- `docs/technical/repository-layout.md`
- any specialized file relevant to the change, especially `configuration.md`, `output.md`, `error-model.md`, `runtime.md`, `tooling.md`, `testing.md`, and `open-questions.md`

`docs/technical/` is the source of truth. Update it before or alongside any technical convention change, then keep `.github/copilot-instructions.md` aligned with it.
