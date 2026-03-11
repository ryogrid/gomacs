# PRD: End-to-End Testing Framework with tmux PTY Emulation

## Introduction

Add an end-to-end testing framework to goomacs that uses tmux as a pseudo-terminal harness. Tests programmatically spawn goomacs inside a tmux session, send keystrokes via `tmux send-keys`, and verify screen output via `tmux capture-pane`. This approach tests the real binary against a real terminal, catching bugs that unit tests miss — rendering glitches, escape sequence issues, mode interaction bugs, and race conditions. The framework lives inside the goomacs repo and runs in GitHub Actions (Linux).

## Goals

- Provide a Go test helper (`e2e/` package) that manages tmux session lifecycle (create, send keys, capture, destroy)
- Support both full-screen snapshot assertions and targeted line/region assertions
- Cover all major goomacs features: editing, navigation, search, kill/yank, undo, buffers, windows, syntax highlighting
- Run reliably in CI (GitHub Actions, Linux, no display required)
- Keep test authoring simple — each test should read like a script of user actions and expected outcomes

## User Stories

### US-001: tmux Test Harness Core
**Description:** As a developer, I want a Go test helper that manages a tmux session so that I can write E2E tests that send keys and capture screen output.

**Acceptance Criteria:**
- [ ] `e2e/harness.go` provides a `Harness` struct with methods: `Start(binary string, args ...string)`, `SendKeys(keys string)`, `Capture() []string`, `CapturePane() string`, `Close()`
- [ ] `Start()` creates a detached tmux session with a fixed geometry (e.g., 80x24), runs the goomacs binary inside it
- [ ] `SendKeys(keys string)` calls `tmux send-keys -t <session> <keys>` — supports literal text, control keys (C-x, C-s, etc.), and special keys (Enter, Escape, Up, Down, Left, Right, BSpace)
- [ ] `Capture() []string` calls `tmux capture-pane -t <session> -p` and returns the screen as a slice of strings (one per row)
- [ ] `CapturePane() string` returns the full screen as a single string (rows joined by newlines)
- [ ] `Close()` kills the tmux session and cleans up
- [ ] Each test gets a unique session name to allow parallel execution
- [ ] A `WaitForContent(substring string, timeout time.Duration) error` helper polls `Capture()` until the substring appears or timeout expires
- [ ] `go build ./e2e/...` succeeds
- [ ] `go vet ./e2e/...` passes

### US-002: Assertion Helpers
**Description:** As a developer, I want assertion helpers for screen content so that I can write readable, maintainable E2E tests.

**Acceptance Criteria:**
- [ ] `AssertLineContains(t, row int, substr string)` captures the screen and checks that the given row contains the substring
- [ ] `AssertLineEquals(t, row int, expected string)` checks exact match (trailing whitespace trimmed)
- [ ] `AssertStatusBar(t, substr string)` checks the status line row (second-to-last visible row) contains the substring
- [ ] `AssertMessageLine(t, substr string)` checks the last row contains the substring
- [ ] `AssertScreenContains(t, substr string)` checks that the substring appears anywhere on screen
- [ ] `AssertScreenSnapshot(t, name string)` captures the full screen and compares against a golden file in `e2e/testdata/<name>.golden`; on first run or with `UPDATE_GOLDEN=1`, writes the golden file
- [ ] `AssertCursorAt(t, row, col int)` verifies the cursor position using `tmux display-message -p '#{cursor_x} #{cursor_y}'`
- [ ] All assertion helpers include clear failure messages showing expected vs actual
- [ ] `go build ./e2e/...` succeeds
- [ ] `go vet ./e2e/...` passes

### US-003: Test Infrastructure and CI Setup
**Description:** As a developer, I want the E2E tests to run in GitHub Actions so that regressions are caught automatically.

**Acceptance Criteria:**
- [ ] `e2e/e2e_test.go` uses `TestMain` to build the goomacs binary once (`go build -o <tmpdir>/goomacs .`) before running tests
- [ ] Tests are skipped with a clear message if `tmux` is not found in PATH
- [ ] A helper creates temporary test files in `t.TempDir()` for each test that needs file content
- [ ] `.github/workflows/e2e.yml` (or added to existing CI workflow) installs tmux (`apt-get install -y tmux`), builds goomacs, and runs `go test ./e2e/ -v -timeout 120s`
- [ ] Tests use a short sleep/poll interval (100ms) to keep total suite time under 60 seconds
- [ ] `go test ./e2e/ -v` passes locally when tmux is installed
- [ ] `go vet ./e2e/...` passes

### US-004: Basic Editing Tests
**Description:** As a developer, I want E2E tests covering basic editing so that text insertion, deletion, and file save are verified against the real terminal.

**Acceptance Criteria:**
- [ ] Test: open goomacs with no args, verify `*scratch*` appears in status bar
- [ ] Test: type "Hello, World!", verify text appears on screen at line 1
- [ ] Test: Backspace deletes the last character (type "abc", Backspace, verify "ab")
- [ ] Test: C-d deletes character at cursor (type "abc", C-a, C-d, verify "bc")
- [ ] Test: Enter inserts a newline (type "line1", Enter, "line2", verify both lines)
- [ ] Test: C-x C-s saves to a file, verify file content on disk matches buffer content
- [ ] Test: C-x C-c quits the editor (tmux pane exits)
- [ ] All tests pass with `go test ./e2e/ -v -run TestBasicEditing`

### US-005: Navigation Tests
**Description:** As a developer, I want E2E tests covering cursor movement and scrolling so that navigation commands are verified.

**Acceptance Criteria:**
- [ ] Test: C-f / C-b moves cursor forward/backward (verify cursor position changes)
- [ ] Test: C-n / C-p moves cursor down/up across lines
- [ ] Test: C-a moves to beginning of line, C-e moves to end of line (verify cursor column)
- [ ] Test: C-v scrolls down one page, M-v scrolls up one page (verify different content visible)
- [ ] Test: M-< jumps to beginning of buffer, M-> jumps to end
- [ ] Test: C-l goto-line — type line number, verify cursor moves to correct line and status bar updates
- [ ] All tests pass with `go test ./e2e/ -v -run TestNavigation`

### US-006: Search Tests
**Description:** As a developer, I want E2E tests for incremental search so that C-s and C-r behavior is verified.

**Acceptance Criteria:**
- [ ] Test: C-s followed by typing a query highlights and moves to the first match (verify message line shows "I-search: <query>")
- [ ] Test: C-s C-s advances to next match
- [ ] Test: C-r searches backward (verify message line shows "I-search backward: <query>")
- [ ] Test: Enter exits search with cursor at match position
- [ ] Test: C-g cancels search and restores original cursor position
- [ ] Test: Backspace in search removes last character and re-searches
- [ ] All tests pass with `go test ./e2e/ -v -run TestSearch`

### US-007: Kill, Yank, and Undo Tests
**Description:** As a developer, I want E2E tests for kill/yank and undo so that clipboard and history operations are verified.

**Acceptance Criteria:**
- [ ] Test: C-k kills to end of line, verify line content changes
- [ ] Test: C-y yanks the killed text back (kill then yank at different position)
- [ ] Test: C-SPC sets mark, move cursor, C-w kills region (verify region removed)
- [ ] Test: M-w copies region without deleting, C-y pastes it
- [ ] Test: C-_ (undo) reverses the last edit
- [ ] Test: consecutive C-k calls accumulate in kill ring (kill two lines, yank gets both)
- [ ] All tests pass with `go test ./e2e/ -v -run TestKillYankUndo`

### US-008: Buffer Management Tests
**Description:** As a developer, I want E2E tests for multi-buffer operations so that file opening, switching, and killing buffers are verified.

**Acceptance Criteria:**
- [ ] Test: C-x C-f opens a file, verify file content displayed and filename in status bar
- [ ] Test: C-x b switches to a different buffer by name (verify status bar changes)
- [ ] Test: C-x C-b shows buffer list with all open buffers
- [ ] Test: C-x k kills a buffer (verify it's removed from buffer list)
- [ ] Test: Tab completion in Find file prompt completes filenames
- [ ] All tests pass with `go test ./e2e/ -v -run TestBufferManagement`

### US-009: Window Splitting Tests
**Description:** As a developer, I want E2E tests for window splitting so that vertical and horizontal splits are verified visually.

**Acceptance Criteria:**
- [ ] Test: C-x 2 splits vertically — verify two status bars visible on screen
- [ ] Test: C-x 3 splits horizontally — verify vertical separator character '│' visible
- [ ] Test: C-x o switches focus between windows (verify active status bar marker changes from `==` to `--`)
- [ ] Test: C-x 0 closes current window, C-x 1 closes all other windows
- [ ] Test: each window independently displays its buffer content
- [ ] All tests pass with `go test ./e2e/ -v -run TestWindowSplitting`

### US-010: Syntax Highlighting Smoke Test
**Description:** As a developer, I want a basic E2E test that verifies syntax highlighting is active for recognized file types.

**Acceptance Criteria:**
- [ ] Test: open a `.go` file containing `package main` and `func main()`, verify the screen output differs from a plain text file (at minimum: capture-pane in ANSI mode shows escape sequences, or verify colored output via `tmux capture-pane -e` which includes escape codes)
- [ ] Test: open a `.txt` file, verify no ANSI color escapes in capture output
- [ ] All tests pass with `go test ./e2e/ -v -run TestSyntaxHighlighting`

## Functional Requirements

- FR-1: The `e2e/` package must manage tmux sessions with unique names per test
- FR-2: `SendKeys` must correctly translate Emacs-style key notation to tmux send-keys syntax (C-x → `C-x`, M-v → `Escape v`, etc.)
- FR-3: Screen capture must return stable output (poll/wait for editor to finish rendering before asserting)
- FR-4: Golden file snapshots must support an update mode (`UPDATE_GOLDEN=1 go test ./e2e/...`)
- FR-5: Tests must clean up tmux sessions even on test failure (use `t.Cleanup`)
- FR-6: The binary must be built once per test run, not per test case
- FR-7: All tests must use temporary directories for test files to avoid polluting the repo
- FR-8: The test suite must complete in under 60 seconds in CI

## Non-Goals

- No GNU Screen support (tmux only)
- No cross-platform support (Linux only; macOS may work but is not tested)
- No performance benchmarking via E2E tests
- No mouse interaction testing
- No automated visual regression testing with image comparison
- No test coverage measurement for E2E tests (they complement unit tests, not replace them)

## Design Considerations

### tmux Key Mapping

Emacs keys need translation to tmux `send-keys` format:

| Emacs Notation | tmux send-keys |
|---------------|----------------|
| `C-x` | `C-x` |
| `C-g` | `C-g` |
| `C-SPC` | `C-Space` |
| `M-v` | `Escape v` (send Escape, then v) |
| `M-<` | `Escape <` |
| `M->` | `Escape >` |
| `M-w` | `Escape w` |
| `Enter` | `Enter` |
| `Backspace` | `BSpace` |
| `C-_` | `C-_` |
| Literal text | quoted string |

### Timing Strategy

Terminal rendering is asynchronous. The framework uses a poll-based approach:
1. Send keys
2. Poll `capture-pane` every 100ms
3. Check if expected content has appeared
4. Timeout after 5 seconds (configurable per assertion)

This avoids fragile `time.Sleep` calls while keeping tests fast.

### File Layout

```
e2e/
├── harness.go          # Harness struct, tmux session management
├── assertions.go       # AssertLineContains, AssertScreenSnapshot, etc.
├── keys.go             # Key notation translation helpers
├── e2e_test.go         # TestMain (build binary), test helpers
├── basic_test.go       # US-004: Basic editing tests
├── navigation_test.go  # US-005: Navigation tests
├── search_test.go      # US-006: Search tests
├── killyank_test.go    # US-007: Kill/yank/undo tests
├── buffer_test.go      # US-008: Buffer management tests
├── window_test.go      # US-009: Window splitting tests
├── highlight_test.go   # US-010: Syntax highlighting tests
└── testdata/           # Golden files for snapshot tests
    └── *.golden
```

## Technical Considerations

- tmux must be installed in the CI environment (`apt-get install -y tmux`)
- tmux sessions are created with `tmux new-session -d -s <name> -x 80 -y 24` for consistent geometry
- `tmux capture-pane -t <session> -p` returns plain text; `-e` flag includes ANSI escape codes (useful for syntax highlighting tests)
- `tmux display-message -t <session> -p '#{cursor_x} #{cursor_y}'` returns cursor position
- Each test should use `t.Parallel()` where possible, since unique session names prevent conflicts
- The goomacs binary path is set via `TestMain` and stored in a package-level variable
- Tests that modify files must use `t.TempDir()` to avoid interference

## Success Metrics

- All 10 user stories have passing tests
- CI runs complete in under 60 seconds
- At least 30 individual test cases covering the major feature areas
- Zero flaky tests (polling-based assertions with reasonable timeouts)

## Open Questions

- Should the framework support recording and replaying test sessions for debugging?
- Should we add a `-e2e` build tag to separate E2E tests from unit tests in `go test ./...`?
