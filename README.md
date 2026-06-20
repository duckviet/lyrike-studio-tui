# Lyrike Studio TUI

Terminal-first lyrics studio for local LRC editing, playback sync, and publish workflows.

## Status

Implementation has started from `.omo/plans/lyrike-studio-tui.md`. The current executable surface is the CLI version smoke:

```bash
go run ./cmd/lyrike-studio-tui --version
```

## Planned Surfaces

- Go module: `github.com/duckviet/lyrike-studio-tui`
- TUI entrypoint: `cmd/lyrike-studio-tui`
- Domain code: `internal/domain`
- Terminal UI: `internal/tui`
- Playback adapters: `internal/playback`
- Existing backend integration: `internal/integrations`

See `.omo/plans/lyrike-studio-tui.md` for the full implementation plan and evidence requirements.

