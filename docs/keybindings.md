# Keybindings

Implemented keybindings in the current TUI:

| Key | Scope | Action |
| --- | --- | --- |
| `Tab` | Global | Move focus to the next panel. |
| `Shift+Tab` | Global | Move focus to the previous panel. |
| `Ctrl-S` | Global | Save the current project draft. |
| `Ctrl-O` | Global | Open the fetch modal to enter a YouTube URL or video ID. |
| `Ctrl-P` | Global | Open the project picker to load a project draft; press `n` to create from URL. |
| `j` / `Down` | Lyrics editor panel | Select the next lyric line. |
| `k` / `Up` | Lyrics editor panel | Select the previous lyric line. |
| `Space` | Global outside text edit | Play or pause playback. |
| `Left` / `Right` | Global outside text edit | Seek playback backward or forward by one second. |
| `q` | Global outside text edit | Quit the application. |
| `e` | Lyrics editor panel | Replace the selected line text with the deterministic edit text used by the QA harness. |
| `t` | Lyrics editor panel | Tap-sync the selected line to the current playback position held by the panel. |
| `Ctrl+Z` | Lyrics editor panel | Undo the last lyrics edit command. |
| `Ctrl+Y` | Lyrics editor panel | Redo the last undone lyrics edit command. |

Key priority is global focus, save, and project picker first, then editor text-edit keys when lyrics text is being edited, then global playback/quit keys, then the focused panel. This means `Space`, `Left`, `Right`, and `q` edit text while the lyrics editor is in text edit mode instead of controlling playback or quitting.
