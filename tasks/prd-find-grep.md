# PRD: find-grep Command

## Introduction

Add an Emacs-style `find-grep` command to goomacs, invocable via `M-x find-grep`. This command lets users search across files using the system's `find` and `grep` binaries without leaving the editor. Results are displayed in a special `*grep*` buffer with dedicated navigation keybindings for jumping to matches, skipping between files, and refreshing results.

## Goals

- Allow users to search across files from within the editor using familiar grep/find syntax
- Display results in a navigable `*grep*` buffer with grep-style output (`filepath:line_number:matched_line`)
- Provide efficient result navigation: line-by-line (n/p), file-by-file (M-n/M-p), and jump-to-source (RET)
- Run searches asynchronously to avoid freezing the UI
- Introduce a reusable buffer-local keymap mechanism for special-purpose buffers

## User Stories

### US-001: Grep output parser
**Description:** As a developer, I need a testable parser for grep output so that result lines can be reliably split into file path, line number, and matched text.

**Acceptance Criteria:**
- [ ] Create `grep.go` with a `GrepResult` struct containing `File` (string), `Line` (int), and `Text` (string) fields
- [ ] Implement `ParseGrepLine(line string) (GrepResult, bool)` that parses a single `filepath:linenum:text` line and returns the result plus a success boolean
- [ ] Correctly handles file paths containing colons on Windows-style drives (e.g., `C:\foo:10:text`) — not required, but must not panic
- [ ] Lines that don't match the `path:number:text` pattern return `false` (e.g., blank lines, header lines, binary file notices)
- [ ] Implement `ParseGrepOutput(output string) []GrepResult` that splits output by newlines and parses each line, skipping unparseable lines
- [ ] Unit tests in `grep_test.go` cover: standard lines, lines with colons in the text portion, empty lines, non-matching lines, lines with spaces in file paths
- [ ] Typecheck passes (`go vet ./...`)

### US-002: Buffer-local keymap mechanism
**Description:** As a developer, I need a way to attach buffer-specific keybindings so that special buffers like `*grep*` can have their own key handling without polluting the global keymap.

**Acceptance Criteria:**
- [ ] Add a `Mode` field (string) to the Buffer struct (empty string means normal editing mode)
- [ ] Add a `ReadOnly` field (bool) to the Buffer struct
- [ ] In the main event loop, before processing normal keybindings, check `buf.Mode` — if non-empty, dispatch to a mode-specific handler function
- [ ] Implement a mode dispatch map: `var modeHandlers map[string]func(ev *term.KeyEvent, buf *Buffer, message *string) bool` where the handler returns `true` if the key was consumed
- [ ] If the mode handler returns `false` (key not consumed), fall through to normal keybinding processing
- [ ] When `buf.ReadOnly` is true, character insertion and editing keys (delete, backspace, kill, yank, etc.) are ignored with message "Buffer is read-only"
- [ ] Typecheck passes (`go vet ./...`)

### US-003: find-grep command registration and prompt
**Description:** As a user, I want to invoke `M-x find-grep` and be prompted for a grep command string so that I can search across files.

**Acceptance Criteria:**
- [ ] Register a `find-grep` command via `RegisterCommand` in an `init()` function
- [ ] When invoked, the command opens a minibuffer prompt with text `Run find-grep: `
- [ ] The minibuffer input is pre-populated with `find . -type f -exec grep -nH -e '' {} +`
- [ ] The cursor is positioned between the single quotes (after `-e '`) so the user can type their search pattern immediately
- [ ] Pressing Enter submits the full command string for execution
- [ ] Pressing Escape or C-g cancels the prompt
- [ ] Typecheck passes (`go vet ./...`)

### US-004: Command execution and *grep* buffer creation
**Description:** As a user, I want the grep command to execute and show results in a `*grep*` buffer so that I can browse matches.

**Acceptance Criteria:**
- [ ] The submitted command string is executed via `os/exec` using `sh -c` for shell interpretation
- [ ] Execution runs asynchronously in a goroutine; the message line shows "Searching..." while running
- [ ] On completion, results are parsed via `ParseGrepOutput` and displayed in a `*grep*` buffer
- [ ] Each result line is displayed as `filepath:linenum:text` (the raw grep output)
- [ ] The `*grep*` buffer has `Mode = "grep"`, `ReadOnly = true`, and `Filename = "*grep*"`
- [ ] If a `*grep*` buffer already exists, it is reused (cleared and repopulated) rather than creating a new one
- [ ] The `*grep*` buffer is added to the buffer list and made the active buffer
- [ ] If the command produces no output, display message "No matches found"
- [ ] If the command fails (non-zero exit with stderr), display stderr content as an error message
- [ ] The last-executed command string is stored for re-execution via `g`
- [ ] Typecheck passes (`go vet ./...`)

### US-005: *grep* buffer keybindings — basic navigation (n, p, q)
**Description:** As a user, I want to navigate grep results with `n`/`p` and close the buffer with `q`.

**Acceptance Criteria:**
- [ ] Register a "grep" mode handler in `modeHandlers`
- [ ] `n` moves cursor to the next result line (skips blank or non-result lines); shows "No more results" and stays in place at the last result
- [ ] `p` moves cursor to the previous result line; shows "No more results" and stays in place at the first result
- [ ] `q` closes the `*grep*` buffer (removes from buffer list) and switches to the previous buffer
- [ ] Normal cursor movement keys (C-n, C-p, C-f, C-b, C-v, M-v, arrow keys) still work for scrolling through results
- [ ] Character input keys are blocked (read-only)
- [ ] Typecheck passes (`go vet ./...`)

### US-006: *grep* buffer keybindings — jump to source (RET)
**Description:** As a user, I want to press Enter on a grep result to jump to the matching file and line.

**Acceptance Criteria:**
- [ ] `RET` (Enter) on a result line parses the line to extract file path and line number
- [ ] If the file is already open in an existing buffer, that buffer is reused (switched to) rather than opening a new one
- [ ] If the file is not open, it is loaded via `NewBufferFromFile` and added to the buffer list
- [ ] If the file does not exist, display message "File not found: <path>" and stay in the *grep* buffer
- [ ] After opening/switching to the buffer, the cursor is moved to the target line number
- [ ] The window scrolls so the target line is visible
- [ ] Typecheck passes (`go vet ./...`)

### US-007: *grep* buffer keybindings — file navigation (M-n, M-p) and refresh (g)
**Description:** As a user, I want to skip between files in grep results and refresh the search.

**Acceptance Criteria:**
- [ ] `M-n` (Alt+N) moves cursor to the first result line of the next file (different file path from current line)
- [ ] `M-p` (Alt+P) moves cursor to the first result line of the previous file
- [ ] At the last file's results, `M-n` shows "No more files" and stays in place
- [ ] At the first file's results, `M-p` shows "No more files" and stays in place
- [ ] `g` re-executes the last find-grep command string and refreshes the `*grep*` buffer with new results
- [ ] After `g`, the cursor is reset to the first result line
- [ ] Typecheck passes (`go vet ./...`)

### US-008: E2E tests — basic find-grep flow
**Description:** As a developer, I want E2E tests covering the core find-grep workflow.

**Acceptance Criteria:**
- [ ] Create `e2e/grep_test.go` with test functions in a `TestFindGrep` group
- [ ] Set up a temporary directory tree with multiple files containing known content across nested directories
- [ ] Test invoke: `M-x find-grep Enter` (with default command modified to search for a known string), verify `*grep*` buffer appears with expected results
- [ ] Test RET: navigate to a result, press Enter, verify the correct file opens at the correct line
- [ ] Test n/p: verify cursor moves to next/previous result lines
- [ ] Test q: verify `*grep*` buffer closes and previous buffer is restored
- [ ] Test no matches: search for a string not in any file, verify "No matches found" message
- [ ] Test malformed command: enter an invalid command string, verify error message appears
- [ ] All tests pass with `go test ./e2e/ -v -timeout 120s -run TestFindGrep`
- [ ] Typecheck passes (`go vet ./...`)

### US-009: E2E tests — file navigation and refresh
**Description:** As a developer, I want E2E tests for M-n/M-p file navigation and g refresh.

**Acceptance Criteria:**
- [ ] Set up fixtures with matches spanning at least 3 different files
- [ ] Test M-n: verify cursor jumps from file1's results to file2's first result, then to file3's
- [ ] Test M-p: verify cursor jumps back from file3 to file2, then to file1
- [ ] Test M-n at last file: verify "No more files" message and cursor stays
- [ ] Test M-p at first file: verify "No more files" message and cursor stays
- [ ] Test g refresh: run grep, modify a fixture file (add a new matching line), press g, verify new result appears in refreshed buffer
- [ ] Test n at last result: verify "No more results" message and cursor stays
- [ ] Test p at first result: verify "No more results" message and cursor stays
- [ ] All tests pass with `go test ./e2e/ -v -timeout 120s -run TestFindGrep`
- [ ] Typecheck passes (`go vet ./...`)

### US-010: README update
**Description:** As a user, I want the README to document the find-grep feature so I know how to use it.

**Acceptance Criteria:**
- [ ] Add `find-grep` to the "Available commands" table in the Command Palette section
- [ ] Add a "Find-Grep" subsection under Keybindings documenting the `*grep*` buffer keybindings (RET, n, p, M-n, M-p, g, q)
- [ ] Add `find-grep` to the Features list
- [ ] Add `grep.go` to the Project Structure section
- [ ] Typecheck passes (`go vet ./...`)

## Functional Requirements

- FR-1: `find-grep` must be invocable via `M-x find-grep`
- FR-2: The prompt must pre-populate with `find . -type f -exec grep -nH -e '' {} +` as the default command
- FR-3: The command must be executed via `sh -c` using `os/exec`
- FR-4: Execution must be asynchronous with a "Searching..." indicator
- FR-5: Results must be displayed in a `*grep*` buffer with `Mode = "grep"` and `ReadOnly = true`
- FR-6: The `*grep*` buffer must be reused on subsequent invocations
- FR-7: `RET` must jump to the source file and line, reusing existing buffers when possible
- FR-8: `n`/`p` must navigate between result lines, showing "No more results" at boundaries
- FR-9: `M-n`/`M-p` must navigate between files, showing "No more files" at boundaries
- FR-10: `g` must re-execute the last command and refresh results
- FR-11: `q` must close the `*grep*` buffer and return to the previous buffer
- FR-12: No-match and error cases must display appropriate messages
- FR-13: The grep output parser must be independently unit-testable
- FR-14: The buffer-local keymap mechanism must be reusable for other special buffers
- FR-15: When `ReadOnly` is true, editing operations must be blocked with a message

## Non-Goals

- No embedded grep implementation; always delegate to system `find`/`grep`
- No syntax highlighting of grep results
- No inline result preview or context lines
- No persistent grep history across editor sessions
- No Windows `cmd.exe` support (assumes POSIX `sh -c`)
- No streaming/incremental display of results (results shown after command completes, despite async execution)

## Technical Considerations

- **New file:** `grep.go` for `GrepResult`, `ParseGrepLine`, `ParseGrepOutput`, and the find-grep command handler
- **New test file:** `grep_test.go` for parser unit tests
- **Buffer struct changes:** Add `Mode` (string) and `ReadOnly` (bool) fields to `buffer.go`
- **Event loop changes:** Add mode dispatch before normal key handling in `main.go`
- **Command handler signature:** The existing `func(*Buffer, *string)` signature is insufficient for find-grep since it needs access to the buffer list and active buffer index. Consider passing a broader context or using package-level variables (which are already used for `buffers`, `activeBufferIdx`, etc. in main.go)
- **Async execution:** Use a goroutine with a channel or callback to deliver results back to the main event loop. The terminal's event system may need a mechanism to wake the main loop when async work completes
- **Pre-populated minibuffer input:** The current minibuffer supports setting `minibufferInput` to a pre-populated value. Set cursor position within the input by adjusting `minibufferInput` content and relying on the cursor being at the end

## Success Metrics

- User can run `M-x find-grep`, search for a pattern, and navigate to matching lines across multiple files
- File navigation (M-n/M-p) correctly skips between file boundaries
- Refresh (g) reflects changes made to files since the last search
- All existing tests continue to pass (`go test ./...`)
- E2E tests cover the full interaction flow including edge cases

## Open Questions

- Should the find-grep prompt support command history (up/down arrow to recall previous searches)? (Deferred to future enhancement)
- Should results include context lines (`grep -C`)? (Out of scope for initial version)
- How should the async result delivery integrate with the terminal event loop? Options: (a) use a channel polled in the event loop, (b) use a signal mechanism to inject a synthetic event
