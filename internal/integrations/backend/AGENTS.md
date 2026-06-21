# Backend Integration

- Keep external API contracts typed at the boundary. Parse JSON into structs before passing to callers.
- Use `context.Context` as the first argument for any operation that can block.
- Configure network clients with explicit timeouts; never rely on the default `http.Client`.
- Test with local fakes or `httptest.Server`; never call external services in unit tests.
- Do not add third-party HTTP or JSON dependencies unless the plan explicitly changes the dependency budget.
