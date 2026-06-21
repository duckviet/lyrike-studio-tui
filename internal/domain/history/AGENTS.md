# History / Command Pattern

- Keep the command manager independent from Bubble Tea, HTTP clients, filesystem paths, and process state.
- Commands must be reversible: applying a command returns both the new document and its inverse.
- Preserve enhanced word timings when mutating line text or timestamps.
- Validate indices and document state before applying; return typed errors.
- Tap-sync accepts the authoritative playback position as a typed value.
- Table tests for command edges; sequential tests for undo/redo stack behavior.
