# PRD: Multi-Buffer Management & Window Splitting

## Introduction

Add Emacs-like multi-buffer management and window splitting to goomacs. Currently the editor supports only a single file/buffer per session. This feature enables users to open multiple files simultaneously, switch between buffers, split the screen to view multiple buffers at once, and close buffers when no longer needed.

## Goals

- Hold multiple buffers (files) simultaneously and switch between them freely
- Open new files via command-line arguments or from within the editor
- Split the screen vertically (top/bottom) to display multiple buffers simultaneously
- Close unneeded buffers with unsaved-change warnings
- All existing single-buffer operations (editing, search, save, undo, etc.) continue to work unchanged

## User Stories

### US-001: Internal Multi-Buffer Management
**Description:** As a developer, I need goomacs to internally manage multiple buffers so that users can have several files open at once.

**Acceptance Criteria:**
- [ ] `main.go` holds a buffer list (`[]*Buffer`) and an active buffer index
- [ ] Multiple files can be specified via command-line arguments (`goomacs file1.go file2.go`)
- [ ] When no arguments are given, starts with a single empty buffer (preserves existing behavior)
- [ ] Each buffer maintains independent cursor position, scroll offset, modified flag, and undo history
- [ ] Typecheck passes (`go vet ./...`)

### US-002: Switch Buffer via Minibuffer (C-x b)
**Description:** As a user, I want to switch buffers by typing a buffer name so that I can quickly jump to a known file.

**Acceptance Criteria:**
- [ ] `C-x b` shows "Switch to buffer: " prompt in the message line
- [ ] Typing a buffer name and pressing Enter switches to that buffer
- [ ] The previous buffer name is shown as default; pressing Enter without input switches to it
- [ ] Entering a non-existent buffer name creates a new empty buffer with that name
- [ ] `C-g` cancels the operation
- [ ] Typecheck passes

### US-003: Buffer List Display (C-x C-b)
**Description:** As a user, I want to see a list of all open buffers so that I can choose which one to edit.

**Acceptance Criteria:**
- [ ] `C-x C-b` creates and displays a special `*Buffer List*` buffer
- [ ] The list shows each buffer's name, modified flag (`*`), and file path
- [ ] The currently active buffer is marked with `>`
- [ ] The buffer list is read-only (editing is disabled)
- [ ] Typecheck passes

### US-004: Open File (C-x C-f)
**Description:** As a user, I want to open a new file from within the editor so that I don't have to restart goomacs.

**Acceptance Criteria:**
- [ ] `C-x C-f` shows "Find file: " prompt in the message line
- [ ] Typing a file path and pressing Enter loads the file into a new buffer and makes it active
- [ ] If the file is already open, switches to the existing buffer (no duplicates)
- [ ] If the path does not exist, creates a new empty buffer with that filename
- [ ] `C-g` cancels the operation
- [ ] Typecheck passes

### US-005: Close Buffer (C-x k)
**Description:** As a user, I want to close a buffer I no longer need so that my buffer list stays manageable.

**Acceptance Criteria:**
- [ ] `C-x k` shows "Kill buffer: (default CURRENT_BUFFER_NAME)" prompt in the message line
- [ ] Pressing Enter closes the current buffer; typing a name closes the specified buffer
- [ ] If the buffer has unsaved changes, prompts "Buffer modified; kill anyway? (y/n)"
- [ ] The last remaining buffer cannot be closed (replaced with a new empty buffer instead)
- [ ] If the closed buffer was displayed in a window, that window switches to another buffer
- [ ] `C-g` cancels the operation
- [ ] Typecheck passes

### US-006: Split Window Vertically (C-x 2)
**Description:** As a user, I want to split the screen into top and bottom panes so that I can view two buffers (or two positions in the same buffer) simultaneously.

**Acceptance Criteria:**
- [ ] `C-x 2` splits the current window into two vertically stacked windows
- [ ] Both windows initially display the same buffer
- [ ] Each window has an independent scroll offset
- [ ] Each window has its own status line at the bottom of its region; the active window's status line is visually distinct
- [ ] Multiple splits are possible (3+ windows)
- [ ] Typecheck passes

### US-007: Move Focus Between Windows (C-x o)
**Description:** As a user, I want to move focus between split windows so that I can edit in different panes.

**Acceptance Criteria:**
- [ ] `C-x o` moves focus to the next window (cycling)
- [ ] The active window's status line is visually distinguishable (e.g., highlighted vs dimmed)
- [ ] The cursor is displayed at the active window's buffer cursor position
- [ ] Typecheck passes

### US-008: Close Windows (C-x 0 / C-x 1)
**Description:** As a user, I want to close windows to return to a simpler layout.

**Acceptance Criteria:**
- [ ] `C-x 0` closes the current window (buffer remains open). No-op if only one window exists
- [ ] `C-x 1` closes all windows except the current one (buffers remain open)
- [ ] After closing windows, remaining windows redistribute the screen height evenly
- [ ] Typecheck passes

### US-009: Per-Window Status Line
**Description:** As a user, I want each window to show its own status line so that I always know which buffer each window is displaying.

**Acceptance Criteria:**
- [ ] Each window has a status line showing the buffer name (preserves existing display format)
- [ ] In multi-window mode, each window has its own status line
- [ ] The active window's status line is visually distinct from inactive windows
- [ ] Typecheck passes

### US-010: Quit Confirmation for All Unsaved Buffers
**Description:** As a user, I want goomacs to warn me about all unsaved buffers when quitting so that I don't lose work.

**Acceptance Criteria:**
- [ ] `C-x C-c` checks all buffers for unsaved changes
- [ ] If any buffer has unsaved changes, displays "Modified buffers exist; exit anyway? (y/n)"
- [ ] `y` exits, `n` cancels
- [ ] If no buffers have unsaved changes, exits immediately
- [ ] Typecheck passes

## Functional Requirements

- FR-1: Manage zero or more buffers via a buffer list (`[]*Buffer`) with active buffer tracking
- FR-2: Accept multiple file arguments on the command line (first file is active)
- FR-3: `C-x b` performs minibuffer-input buffer switching
- FR-4: `C-x C-b` displays all buffers in a special read-only `*Buffer List*` buffer
- FR-5: `C-x C-f` performs minibuffer-input file opening
- FR-6: `C-x k` performs minibuffer-input buffer closing (with unsaved-change confirmation)
- FR-7: `C-x 2` splits the current window vertically (top/bottom)
- FR-8: `C-x o` cycles focus between windows
- FR-9: `C-x 0` closes the current window; `C-x 1` closes all other windows
- FR-10: Each window has its own status line; active window is visually distinguished
- FR-11: Window regions are evenly distributed across screen height (each window = text area + 1 status line)
- FR-12: The message line occupies the bottom row of the screen and is shared across all windows
- FR-13: Implement a generic minibuffer input mode (shared by buffer switch, file open, and buffer close) with `C-g` cancel support
- FR-14: On quit, check all buffers for unsaved changes and prompt for confirmation if any exist
- FR-15: `C-x C-s` saves the buffer displayed in the currently active window

## Non-Goals

- Horizontal (left/right) window splitting is not included (vertical split only)
- Tab bar UI is not included
- Buffer name completion (Tab completion) is not included
- File path completion is not included
- Directory browser (dired equivalent) is not included
- Frames (multiple terminal windows) are not included

## Design Considerations

- Minibuffer input should follow the same pattern as the existing search mode (`searchMode`): a mode flag, an input buffer, prompt display, and `C-g` cancel
- Windows should be defined as a `Window` struct holding a pointer to the displayed buffer, a scroll offset, and the screen row range
- Active vs inactive window distinction can use status line styling: reverse video for active, plain text (or `---` vs `===` delimiters) for inactive

## Technical Considerations

- Introduce a `Window` struct. Each window holds a buffer pointer and its own scroll offset. The buffer's cursor position remains on the `Buffer` struct (as in Emacs, where multiple windows showing the same buffer share one cursor/point)
- Modify the rendering pipeline to loop over all windows and draw each window's region
- Keep changes to `term/screen.go` (Screen interface) and `term/terminal.go` minimal. Rendering logic changes should be in `main.go`
- Build the minibuffer input mode as a generic mechanism reusable across buffer switch, file open, and buffer close
- `Buffer.ScrollOffset` currently lives on the Buffer struct but must move to the `Window` struct for per-window independent scrolling
- Recalculate window row ranges on resize events

## Success Metrics

- `goomacs file1.go file2.go` launches and `C-x b` / `C-x C-b` can switch between buffers
- `C-x C-f` opens new files from within the editor
- `C-x 2` splits the screen, `C-x o` moves between windows, each window edits independently
- `C-x 0` / `C-x 1` close windows and the layout redistributes correctly
- `C-x k` closes buffers with unsaved-change confirmation
- All existing features (cursor movement, editing, search, undo, save) work correctly in a multi-buffer, multi-window environment

## Open Questions

- When the same buffer is displayed in multiple windows, should each window have its own independent cursor position, or share the buffer's single cursor? (Emacs uses per-window independent `point`, but this adds implementation complexity)
- Should the initial empty buffer have a name like `*scratch*`?
- Should pressing Enter in the `*Buffer List*` navigate to the selected buffer?
