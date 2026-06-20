# Command Entrypoints

- Keep command packages thin. Parse CLI flags, wire dependencies, and delegate behavior to internal packages.
- Do not place domain logic in `cmd/`.
- User-facing command output must be stable enough for smoke tests.

