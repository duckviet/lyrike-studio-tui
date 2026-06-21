# MPV Playback Adapter

- Treat mpv `time-pos` property observation as the authoritative playback clock.
- Use Unix domain sockets only; no Windows named-pipe support.
- Send JSON IPC commands with monotonic request IDs and match responses.
- Surface actionable errors for missing mpv binary or missing IPC socket.
- Test with a local fake mpv Unix socket server; never require real mpv in unit tests.
