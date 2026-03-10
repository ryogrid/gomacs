# PRD: Fix C-SPC (Set Mark) Not Working

## Introduction

The `C-SPC` (Ctrl+Space) keybinding for setting the mark in goomacs does not work on Windows Terminal under WSL2. The code correctly handles `tcell.KeyCtrlSpace`, but the terminal may not deliver the expected NUL byte (0x00) for Ctrl+Space, or tcell may not map it correctly in all environments. This fix ensures set-mark works reliably across terminals.

## Goals

- Make C-SPC reliably trigger set-mark on Windows Terminal / WSL2
- Maintain backward compatibility with terminals where it already works
- Minimal, focused fix with no unnecessary changes

## User Stories

### US-001: Diagnose C-SPC Key Event
**Description:** As a developer, I need to understand what key event (if any) the terminal actually sends for Ctrl+Space so I can handle it correctly.

**Acceptance Criteria:**
- [ ] Add temporary debug logging or test to capture the actual `ev.Key()` and `ev.Modifiers()` values when pressing Ctrl+Space in Windows Terminal on WSL2
- [ ] Document the finding (what key code the terminal sends vs. `tcell.KeyCtrlSpace`)
- [ ] Remove debug code after diagnosis

### US-002: Fix Set-Mark Keybinding
**Description:** As a user, I want to press C-SPC to set the mark so that I can select text regions.

**Acceptance Criteria:**
- [ ] Pressing Ctrl+Space in Windows Terminal on WSL2 sets the mark and shows "Mark set" in the message bar
- [ ] Mark position matches current cursor position
- [ ] Region highlighting works when moving cursor after setting mark
- [ ] C-w (kill region) and M-w (copy region) work after setting mark with C-SPC
- [ ] Existing terminals where C-SPC already works are not broken
- [ ] Typecheck/lint passes (`go vet ./...`)
- [ ] All existing tests pass (`go test ./...`)

## Functional Requirements

- FR-1: The set-mark command must be triggered by Ctrl+Space across all common terminal emulators (Windows Terminal, VSCode terminal, xterm, etc.)
- FR-2: If `tcell.KeyCtrlSpace` does not match the actual key event, add handling for the actual key code sent by the terminal (e.g., `KeyNUL`, `Key(0)`, or `KeyRune` with rune 0)
- FR-3: The "Mark set" message must appear in the status bar when the mark is successfully set

## Non-Goals

- No configurable keybinding system
- No changes to other keybindings
- No general terminal compatibility audit beyond C-SPC

## Technical Considerations

- `tcell.KeyCtrlSpace` is defined as `iota + 64 = 64` in tcell v2
- `tcell.KeyNUL` is defined as `iota = 0` — these are different constants
- Ctrl+Space sends NUL (0x00) in most terminals, but some terminals may send a different code or not send it at all
- tcell's `NewEventKey` maps raw byte 0x00 to `KeyCtrlSpace` via `KeyCtrlSpace + Key(r)`, but this path may not be taken if the terminal/tty layer filters NUL bytes
- Windows Terminal on WSL2 may have specific input handling quirks

## Success Metrics

- C-SPC correctly sets the mark in Windows Terminal on WSL2
- All existing tests continue to pass
- No regressions in other keybindings

## Open Questions

- Does Windows Terminal send NUL (0x00) for Ctrl+Space, or a different byte sequence?
- Is the NUL byte being filtered by the tty read layer or UTF-8 decoder before reaching tcell's input processor?
- Would handling `KeyNUL` (value 0) as a fallback be sufficient, or is a different approach needed?
