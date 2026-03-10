# PRD: goomacs - Minimal Emacs-like CLI Editor

## Introduction

goomacs is a lightweight Emacs-like text editor for the terminal, written in pure Go. It provides a familiar Emacs keybinding experience for quick file edits without the complexity of a full Emacs installation. It uses `tcell` for terminal handling and supports single-file editing with essential Emacs keybindings including cursor movement, kill/yank, search, mark/region, and undo.

## Goals

- Provide a lightweight, fast-starting terminal editor for quick file edits
- Support core Emacs keybindings that experienced users expect
- Display a status line showing filename, modified state, and cursor position
- Run on any terminal that supports standard escape sequences
- Keep the codebase simple and maintainable with zero dependencies beyond `tcell`

## User Stories

### US-001: Project Setup and Terminal Initialization
**Description:** As a developer, I need the Go project scaffolding and terminal initialization so that the editor can take over the terminal screen.

**Acceptance Criteria:**
- [ ] `go.mod` initialized with module name `goomacs`
- [ ] `tcell` added as dependency
- [ ] Main function initializes a `tcell.Screen`, enters raw mode, and cleans up on exit
- [ ] Running `go run .` shows a blank terminal screen and exits cleanly on `C-c`
- [ ] `go build` produces a working binary
- [ ] Typecheck and build pass

### US-002: Buffer Data Structure
**Description:** As a developer, I need an in-memory buffer data structure to hold text content so that editing operations can be performed efficiently.

**Acceptance Criteria:**
- [ ] Buffer struct stores lines as a slice of strings (or rune slices)
- [ ] Buffer tracks cursor position (row, column)
- [ ] Buffer supports inserting a character at cursor position
- [ ] Buffer supports deleting a character at cursor position (backspace)
- [ ] Buffer supports inserting a newline (splitting a line)
- [ ] Buffer supports deleting at beginning of line (joining with previous line)
- [ ] Unit tests pass for all buffer operations
- [ ] Typecheck passes

### US-003: File Open and Display
**Description:** As a user, I want to open a file from the command line so that I can view its contents in the editor.

**Acceptance Criteria:**
- [ ] Running `goomacs filename.txt` opens and displays the file content
- [ ] Running `goomacs` with no arguments opens an empty buffer
- [ ] File content is rendered correctly in the terminal
- [ ] Long lines are truncated at screen width (no wrapping for now)
- [ ] Lines beyond the screen height are not displayed (scrolling comes later)
- [ ] Typecheck passes

### US-004: Basic Cursor Movement
**Description:** As a user, I want to move the cursor using Emacs keybindings so that I can navigate within the file.

**Acceptance Criteria:**
- [ ] `C-f` (Ctrl+F) moves cursor forward one character
- [ ] `C-b` (Ctrl+B) moves cursor backward one character
- [ ] `C-n` (Ctrl+N) moves cursor down one line
- [ ] `C-p` (Ctrl+P) moves cursor up one line
- [ ] `C-a` (Ctrl+A) moves cursor to beginning of line
- [ ] `C-e` (Ctrl+E) moves cursor to end of line
- [ ] Arrow keys also work for movement
- [ ] Cursor does not move beyond buffer boundaries
- [ ] Cursor is visually displayed at the correct position
- [ ] Typecheck passes

### US-005: Scrolling / Viewport
**Description:** As a user, I want the view to scroll when the cursor moves beyond the visible area so that I can edit files larger than the screen.

**Acceptance Criteria:**
- [ ] View scrolls down when cursor moves below the last visible line
- [ ] View scrolls up when cursor moves above the first visible line
- [ ] `M-v` (Alt+V or Esc then V) scrolls up one page
- [ ] `C-v` (Ctrl+V) scrolls down one page
- [ ] `M-<` (Alt+Shift+<) moves to beginning of buffer
- [ ] `M->` (Alt+Shift+>) moves to end of buffer
- [ ] Scroll offset is tracked and rendering adjusts accordingly
- [ ] Typecheck passes

### US-006: Text Insertion and Basic Editing
**Description:** As a user, I want to type characters and perform basic edits so that I can modify the file content.

**Acceptance Criteria:**
- [ ] Printable characters are inserted at cursor position
- [ ] `Enter` inserts a new line and moves cursor to the beginning of the next line
- [ ] `Backspace` deletes the character before the cursor
- [ ] `C-d` deletes the character at the cursor (forward delete)
- [ ] `C-d` at end of line joins the next line to the current line
- [ ] Backspace at beginning of line joins the current line to the previous line
- [ ] Screen re-renders correctly after each edit
- [ ] Typecheck passes

### US-007: Status Line
**Description:** As a user, I want to see a status line at the bottom of the screen so that I know which file I'm editing and where my cursor is.

**Acceptance Criteria:**
- [ ] Status line is displayed on the second-to-last row of the terminal
- [ ] Status line shows the filename (or `[No Name]` for new buffers)
- [ ] Status line shows `[Modified]` when the buffer has unsaved changes
- [ ] Status line shows cursor position as `Line X, Col Y`
- [ ] Status line has a visually distinct background (reverse video)
- [ ] Status line updates on every cursor move or edit
- [ ] Typecheck passes

### US-008: File Save
**Description:** As a user, I want to save the file so that my changes are persisted to disk.

**Acceptance Criteria:**
- [ ] `C-x C-s` saves the current buffer to the file it was opened from
- [ ] After saving, the `[Modified]` indicator is cleared
- [ ] A "Saved filename.txt" message is briefly shown in the status line or message area
- [ ] If buffer has no filename (new buffer), saving is skipped with a message (write-to not implemented)
- [ ] File is written atomically (write to temp, then rename) or safely
- [ ] Typecheck passes

### US-009: Quit
**Description:** As a user, I want to quit the editor cleanly so that the terminal is restored to its normal state.

**Acceptance Criteria:**
- [ ] `C-x C-c` quits the editor
- [ ] Terminal is fully restored on exit (cursor visible, normal mode)
- [ ] If buffer is modified, a warning message is shown and quit is cancelled
- [ ] Repeating `C-x C-c` when modified forces quit without saving
- [ ] Typecheck passes

### US-010: Kill and Yank (Cut and Paste)
**Description:** As a user, I want to cut and paste text using Emacs kill/yank so that I can move text around efficiently.

**Acceptance Criteria:**
- [ ] `C-k` kills (cuts) text from cursor to end of line into the kill ring
- [ ] `C-k` on an empty line kills the newline (joins with next line)
- [ ] Consecutive `C-k` commands append to the kill ring entry
- [ ] `C-y` yanks (pastes) the last killed text at cursor position
- [ ] Yanked text is inserted correctly, handling multi-line content
- [ ] Typecheck passes

### US-011: Mark and Region
**Description:** As a user, I want to select a region of text using mark so that I can kill or copy specific sections.

**Acceptance Criteria:**
- [ ] `C-SPC` (Ctrl+Space) sets the mark at the current cursor position
- [ ] The region between mark and cursor (point) is visually highlighted
- [ ] `C-w` kills (cuts) the region and stores it in the kill ring
- [ ] `M-w` (Alt+W) copies the region to the kill ring without deleting
- [ ] `C-g` deactivates the mark (cancels selection)
- [ ] After kill or copy, the mark is deactivated
- [ ] Typecheck passes

### US-012: Undo
**Description:** As a user, I want to undo changes so that I can recover from mistakes.

**Acceptance Criteria:**
- [ ] `C-/` (Ctrl+/) undoes the last editing operation
- [ ] `C-_` (Ctrl+Shift+-) also triggers undo (alternative binding)
- [ ] Undo supports multiple levels (at least 100 undo steps)
- [ ] Undo restores both content and cursor position
- [ ] Undo operations can be undone themselves (redo by undoing the undo)
- [ ] Typecheck passes

### US-013: Incremental Search
**Description:** As a user, I want to search for text incrementally so that I can quickly find content in the file.

**Acceptance Criteria:**
- [ ] `C-s` starts incremental search forward
- [ ] Characters typed during search update the search query and jump to the next match
- [ ] `C-s` again during search jumps to the next match
- [ ] `C-r` starts incremental search backward
- [ ] `Enter` or `C-g` exits search mode (`C-g` restores original cursor position)
- [ ] The current match is visually highlighted
- [ ] The search query is shown in the status line or a message area
- [ ] Typecheck passes

## Functional Requirements

- FR-1: The editor must initialize the terminal using `tcell`, entering raw/alternate screen mode
- FR-2: The editor must accept an optional filename as a command-line argument
- FR-3: The editor must store file content in a line-based buffer data structure
- FR-4: The editor must render the buffer content to the terminal, respecting viewport offset
- FR-5: The editor must handle all specified Emacs keybindings (C-f/b/n/p/a/e/d/k/y/w/s/v and M-v/w/</>)
- FR-6: The editor must maintain a kill ring for cut/copy/paste operations
- FR-7: The editor must maintain an undo history of at least 100 operations
- FR-8: The editor must display a status line with filename, modified state, and cursor position
- FR-9: The editor must save files with `C-x C-s` and quit with `C-x C-c`
- FR-10: The editor must support multi-key command sequences (C-x prefix)
- FR-11: The editor must restore the terminal to its original state on exit
- FR-12: The editor must handle terminal resize events

## Non-Goals

- No multiple buffer / multi-file support
- No split windows or frames
- No syntax highlighting
- No configuration file or customization
- No macro recording/playback
- No minibuffer with command input (M-x)
- No line wrapping (long lines are truncated)
- No mouse support
- No clipboard integration (system clipboard)
- No file browser or directory listing
- No auto-save or backup files

## Technical Considerations

- **Language:** Go (1.21+)
- **Terminal library:** `github.com/gdamore/tcell/v2`
- **Architecture:** Keep it simple — a main loop that reads events, updates buffer state, and re-renders. No need for complex abstractions at this stage.
- **Buffer representation:** Slice of `[]rune` lines is simple and sufficient for single-file editing
- **Undo:** Store snapshots of changed lines or use a command pattern with inverse operations
- **Key sequences:** `C-x` prefix requires a simple state machine (waiting for second key)

## Success Metrics

- Editor starts in under 100ms
- Can open and edit files of 10,000+ lines without lag
- All specified keybindings work as documented
- Clean exit with no terminal corruption in all cases
- `go build` with zero warnings

## Open Questions

- Should `C-x C-f` (find-file) be included for opening a different file, or is that out of scope for single-file mode?
- Should tab key insert a tab character or spaces?
- What should happen with binary files or files with very long lines?
