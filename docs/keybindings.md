# Keybindings

Implemented keybindings in the current TUI:

| Key | Scope | Action |
| --- | --- | --- |
| `Tab` | Global | Move focus to the next panel. |
| `Shift+Tab` | Global | Move focus to the previous panel. |
| `j` / `Down` | Lyrics editor panel | Select the next lyric line. |
| `k` / `Up` | Lyrics editor panel | Select the previous lyric line. |
| `e` | Lyrics editor panel | Replace the selected line text with the deterministic edit text used by the QA harness. |
| `t` | Lyrics editor panel | Tap-sync the selected line to the current playback position held by the panel. |
| `Ctrl+Z` | Lyrics editor panel | Undo the last lyrics edit command. |
| `Ctrl+Y` | Lyrics editor panel | Redo the last undone lyrics edit command. |

The media panel displays planned transport hints for play/pause, seek, and loop controls. Those hints are visual only until full playback wiring is connected to the root TUI model.
