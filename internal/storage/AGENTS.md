# Storage

- Keep storage independent from Bubble Tea, HTTP clients, and process state.
- Validate untrusted input at package boundaries and return typed errors.
- Atomic writes only: temp file + rename. Do not re-read production writes as verification.
- Use injected paths in tests; never write to the real user home or XDG directories.
- Table tests for edge cases: corrupt JSON, partial temp files, missing drafts.
