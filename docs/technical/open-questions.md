# Open Questions

## Active Decisions

- Durable local state storage beyond config/cache is still undecided
- Additional error categories beyond `validation`, `auth`, and `not_found` should be added only when concrete use cases require them
- Any future cross-cutting runtime or infrastructure package should be introduced only when concrete adapters justify it
- Progress-reporting conventions should be defined only when real long-running workflows appear
