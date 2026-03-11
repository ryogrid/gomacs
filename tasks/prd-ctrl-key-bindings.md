# PRD: Additional Ctrl Key Bindings (C-h, C-j, C-l)

## Introduction

Add three missing Ctrl key bindings that improve editing efficiency: C-h as an alternative backspace, C-j as an alternative Enter, and C-l as a goto-line command using the minibuffer. These are standard Emacs keybindings that goomacs currently lacks.

## Goals

- Provide C-h as an alternative to Backspace for users who prefer keeping hands on the home row
- Provide C-j as an alternative to Enter for newline insertion
- Add goto-line functionality (C-l) using the existing minibuffer prompt system
- Ensure all three bindings work consistently across modes (normal editing, search, minibuffer)

## User Stories

### US-001: C-h as Backspace
**Description:** As a user, I want C-h to delete the character before the cursor so that I can backspace without leaving the home row.

**Acceptance Criteria:**
- [ ] In normal editing mode, C-h behaves identically to Backspace (calls `buf.SaveUndo()` then `buf.Backspace()`)
- [ ] In search mode, C-h removes the last character from the search query (same as Backspace)
- [ ] In minibuffer mode, C-h removes the last character from the minibuffer input (same as Backspace)
- [ ] `go build .` succeeds
- [ ] `go test ./...` passes

### US-002: C-j as Enter
**Description:** As a user, I want C-j to insert a newline so that I can press Enter without leaving the home row.

**Acceptance Criteria:**
- [ ] In normal editing mode, C-j behaves identically to Enter (calls `buf.SaveUndo()` then `buf.InsertNewline()`, with Buffer List special handling)
- [ ] In search mode, C-j exits search and accepts the current match (same as Enter)
- [ ] In minibuffer mode, C-j accepts the input and calls the callback (same as Enter)
- [ ] In confirm mode, C-j is ignored (does not act as "yes" — only 'y'/'n' are valid)
- [ ] `go build .` succeeds
- [ ] `go test ./...` passes

### US-003: C-l Goto Line via Minibuffer
**Description:** As a user, I want to press C-l to be prompted for a line number and jump directly to that line, so that I can navigate large files quickly.

**Acceptance Criteria:**
- [ ] C-l activates the minibuffer with prompt "Goto line: "
- [ ] Entering a valid number (e.g., "42") moves the cursor to that line (1-based: line 1 = first line)
- [ ] The cursor column is set to 0 (beginning of the target line)
- [ ] If the number exceeds total lines, the cursor moves to the last line
- [ ] If the number is less than 1 or not a valid integer, a message "Invalid line number" is shown and the cursor does not move
- [ ] After jumping, the message line shows "Line N" (where N is the actual line jumped to)
- [ ] The window scroll is adjusted to make the target line visible (via existing `AdjustScroll`)
- [ ] C-g cancels the goto-line prompt without moving the cursor
- [ ] `go build .` succeeds
- [ ] `go test ./...` passes

## Functional Requirements

- FR-1: C-h (KeyCtrlH) must trigger backspace in normal, search, and minibuffer modes
- FR-2: C-j (KeyCtrlJ) must trigger newline/accept in normal, search, and minibuffer modes
- FR-3: C-l (KeyCtrlL) must open a minibuffer prompt "Goto line: " and navigate to the entered line number
- FR-4: Goto-line uses 1-based line numbering (line 1 = first line of the buffer)
- FR-5: Invalid goto-line input (non-numeric, zero, negative) shows "Invalid line number" message
- FR-6: Line numbers exceeding buffer length clamp to the last line

## Non-Goals

- No `goto-char` (character offset navigation)
- No line number display in the buffer content area (line numbers in the gutter)
- No recenter-screen behavior (Emacs C-l default) — C-l is repurposed for goto-line
- C-h does not open a help system (Emacs default) — it acts as backspace

## Technical Considerations

- `term.KeyCtrlH`, `term.KeyCtrlJ`, `term.KeyCtrlL` are already defined in `term/screen.go`
- C-h and C-j can be added by extending existing `case` arms (adding to `KeyBackspace`/`KeyEnter` cases)
- C-l goto-line reuses the existing minibuffer infrastructure (`minibufferMode`, `minibufferPrompt`, `minibufferInput`, `minibufferCallback`)
- The goto-line callback needs `strconv.Atoi` to parse the input
- After setting `CursorR`, the next `redraw()` call will invoke `AdjustScroll()` on the active window, so no explicit scroll logic is needed

## Success Metrics

- All three keybindings work in their respective modes without regressions
- Existing Backspace and Enter behavior is unchanged
- Goto-line navigates correctly for edge cases (first line, last line, out of range)

## Open Questions

- None
