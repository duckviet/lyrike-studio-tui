# Domain Packages

- Keep domain packages independent from Bubble Tea, HTTP clients, filesystem paths, and process state.
- Represent distinct semantic primitives with distinct Go types.
- Validate untrusted input at package boundaries and pass typed values internally.
- Use table tests for parser and command edge cases.

