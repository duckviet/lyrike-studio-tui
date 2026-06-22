# ULW tui-fetch-input planning draft

Status: awaiting-approval
Pending action: write/finish .omo/plans/tui-fetch-input.md (done) and await user start-work or high-accuracy review.

User request: TUI lacks URL/video ID input; "New project ID" input not reasonable. Example URL https://www.youtube.com/watch?v=P0N0h_EOS-c.

Decisions (from user + Metis):
- Fetch modal triggered by Ctrl-O (not Ctrl-F; Ctrl-F is editor snap-to-playing-line).
- 'n' in project picker opens the same modal; manual "New project ID" removed.
- If a draft exists for the parsed video ID, load it; else create project with projectID=videoID.
- Ctrl-S with no project opens fetch modal.
- TUI parser extracts video ID locally; media fetch stays backend call.
- memoryDraftStore.Load must return not-found error to match FileStore.
- Remove all projectPickerCreate references.
- Reset trackName/artistName/albumName on new-project path.
- Plan path: .omo/plans/tui-fetch-input.md (7 todos + final wave).
- Metis blockers folded into plan.
