# Lyrike Studio TUI ULW Goals

1. Initialize workspace, module, repo hygiene, local rules, and CLI version smoke.
2. Implement LRC domain parser and renderer with standard/enhanced LRC tests.
3. Define playback contracts and deterministic fake player test seam.
4. Define backend client contracts and fixtures for the existing FastAPI local API.
5. Define atomic draft persistence contracts with injected XDG paths.
6. Add lyric edit actions, undo/redo history, and tap-sync behavior.
7. Implement mpv Unix IPC playback adapter and missing-mpv guidance.
8. Implement backend HTTP client for fetch, audio, peaks, SSE, challenge, and publish.
9. Implement XDG atomic draft storage.
10. Fix the existing backend peaks-cache bug in `/home/duckviet/lrclib-upload/backend/routes/local_api.py` with a regression test.
11. Build three-panel TUI model, routing, focus, and terminal layout.
12. Add ASCII waveform and transport controls.
13. Add lyrics editor interactions for synced/plain/meta tabs, keyboard edits, undo/redo, and tap-sync.
14. Add publish flow panel with deterministic Validate -> PoW -> Publish -> Done states.
15. Add end-to-end tmux TUI QA harness against fake backend and fake player.
16. Write docs and operator guidance for install, mpv, backend URL, keybindings, drafts, and troubleshooting.
17. Run final verification, review, cleanup, and handoff with evidence.

