# Draft Domain

- Keep draft domain independent from Bubble Tea, HTTP clients, filesystem paths, and process state.
- Represent draft identity, metadata, and snapshots as distinct typed primitives.
- Validate untrusted input at package boundaries and pass typed values internally.
- Use table tests for constructor and snapshot edge cases.
