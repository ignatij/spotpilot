# Open Questions

## Active Decisions

- Any future cross-cutting runtime or infrastructure package should be introduced only when concrete adapters justify it
- Progress-reporting conventions should be defined only when real long-running workflows appear
- YAML output support is deferred beyond v1; reconsider if user or agent demand emerges
- Exact exit-code numbering for each error category should be finalized during implementation

## Resolved Decisions

- **Durable local state storage**: credentials stored in OS keychain (macOS Keychain / Linux secret-service) with a protected-file fallback; resolved as part of v1 spec alignment
