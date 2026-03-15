# Testing

## Default Testing Approach

- Use Go's built-in `testing` package
- Prefer table-driven tests
- Keep helpers minimal and local unless they become broadly reusable
- Favor deterministic tests

## Test Structure

- Keep unit tests close to the packages they verify
- Prefer small mocks or fakes for external dependencies
- Add a limited number of opt-in integration tests once real integrations exist
- Keep concurrency out of tests unless the code truly requires it
- Prefer deterministic assertions on stable result models rather than command-specific ad hoc text assembly

## Architectural Testing Rules

- Use injected writers/runtime abstractions instead of global process I/O
- If time or randomness becomes necessary, wrap it behind small helpers so tests can control it
- Test use cases independently from Cobra command wiring where possible
- Test boundary normalization and validation close to the relevant input types
- Prefer verifying explicit result structs and rendered envelopes over deep inspection of Cobra internals
