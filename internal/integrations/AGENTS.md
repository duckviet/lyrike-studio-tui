# Integrations

- Keep external API contracts typed at the boundary.
- Use `context.Context` as the first argument for operations that can block.
- Configure network clients with explicit timeouts.
- Test integrations with local fakes or `httptest.Server`; do not call external services in unit tests.

