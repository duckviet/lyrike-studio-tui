# Implementation Notes

The executable implementation follows `.omo/plans/lyrike-studio-tui.md` and `.omo/plans/ku-ui-integration.md`.

Task 1 establishes the module, command entrypoint, ownership rules, and version smoke surface. Later tasks add the lyrics domain, playback adapters, backend integration, storage, and TUI workflow.

## Centered Overlay & Modal UI Layer

The TUI uses a centralized, semantic theme system and centered overlay modals ported from `ku`:
- **Theme System**: Defines precomputed styles (PaneActive, PaneInactive, ModalBorder, ModalTitle, etc.) built from a semantic color palette.
- **Overlay Compositor**: Intercepts focus and renders helper views (Fuzzy Selector, Confirm Dialog, Help Keybindings) centered on top of the main body without replacing the entire screen state.
- **Context-Aware Footer**: Displays context-specific key hints and right-aligned status messages at the bottom row of the screen.
- **Fuzzy Selector**: Handles live-filtering, subsequence scoring, and list pagination for selecting projects or templates.
