# Keybindings

Implemented keybindings in the current TUI:

| Key | Scope | Action |
| --- | --- | --- |
| `Tab` | Global | Move focus to the next panel. |
| `Shift+Tab` | Global | Move focus to the previous panel. |
| `Ctrl-S` | Global | Save the current project draft. |
| `Ctrl-O` | Global | Open the centered fetch overlay to enter a YouTube URL or video ID. |
| `Ctrl-L` | Global | Open the project selector overlay to load a project draft. |
| `?` | Global | Open the scrollable interactive keybindings help menu. |
| `Space` | Global outside edit | Play or pause playback. |
| `Left` / `Right` | Global outside edit | Seek playback backward or forward by one second. |
| `q` / `Esc` | Global outside edit | Quit the application / Close active overlay modal. |
| `j` / `Down` | Lyrics editor | Select the next lyric line. |
| `k` / `Up` | Lyrics editor | Select the previous lyric line. |
| `Enter` | Lyrics editor | Edit the selected lyric line text. |
| `t` | Lyrics editor | Tap-sync the selected line to the current playback position. |
| `Ctrl+Z` | Lyrics editor | Undo the last lyrics edit command. |
| `Ctrl+Y` | Lyrics editor | Redo the last undone lyrics edit command. |
| `Ctrl+E` | Lyrics editor | Edit project metadata (Track, Artist, Album). |

### Key Priority & Overlay Handling
When any centered overlay modal is active (e.g., Project Selector, Fetch Overlay, Help Menu, Confirm Dialog), keyboard input is captured exclusively by the active overlay:
- **Project Selector / Fetch Overlay**: Keypresses route to the search field. `Enter` accepts, `Esc` cancels.
- **Confirm Dialog**: `y` or `Enter` confirms, `n` or `Esc` cancels.
- **Help Menu**: `↑`/`↓` or `j`/`k` scroll the keybindings columns, `?` or `Esc` closes.
- **Footer Pane**: Displays context-specific key hints dynamically updated based on the focused panel and active overlay.
