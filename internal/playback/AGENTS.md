# Playback

- Treat mpv IPC as the authoritative playback clock when real playback is active.
- Keep fake playback implementations deterministic and sleep-free for tests.
- Do not add Windows named-pipe support unless the plan is updated.

