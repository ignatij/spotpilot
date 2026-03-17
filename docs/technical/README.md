# Technical Foundations

This directory is the source of truth for `spotpilot`'s technical baseline while product requirements are still evolving.

Agents and contributors must read this directory before making architecture-affecting changes. When a technical convention changes, update these documents before or alongside implementation.

## Reading Order

1. `baseline.md`
2. `architecture.md`
3. `repository-layout.md`
4. `configuration.md`
5. `output.md`
6. `error-model.md`
7. `runtime.md`
8. `tooling.md`
9. `testing.md`
10. `open-questions.md`

## Documentation Rules

- `docs/technical/` is the project bible for technical conventions
- Keep `baseline.md` short and principle-focused
- Keep operational detail in the specialized files
- Record still-open design questions in `open-questions.md`
- Keep `AGENTS.md` and `.github/copilot-instructions.md` aligned with this directory

## Companion Guides

- `architecture-walkthrough.md` — practical runtime explanation with architecture and flow diagrams
- `../functional/business-overview.md` — business-level explanation of product goals, value, and user workflow
