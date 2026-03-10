# PRD: Pure Go Terminal Backend (Eliminate tcell Dependency)

## Introduction

Replace the `github.com/gdamore/tcell/v2` dependency with a pure Go terminal I/O implementation using ANSI/VT100 escape sequences. This eliminates all external dependencies, making gomacs a zero-dependency Go binary. The implementation targets Linux/WSL2 POSIX terminals only and uses only the styling features currently needed (reverse video).

A parallel migration strategy is used: build the pure Go backend as a new package alongside tcell, validate it works, then switch `main.go` to use it and remove tcell.

## Goals

- Eliminate the `tcell` external dependency entirely (zero `require` lines in go.mod)
- Implement terminal I/O using only Go standard library and ANSI/VT100 escape sequences
- Maintain identical editor behavior and appearance after migration
- Keep the implementation minimal -- only what gomacs currently uses

## User Stories

### US-001: Raw Mode Terminal Setup and Teardown
**Description:** As a developer, I need to put the terminal into raw mode on startup and restore it on exit so that the editor can receive individual key presses and control screen output.

**Acceptance Criteria:**
- [ ] Implement `Terminal` struct with `Init()` and `Fini()` methods
- [ ] `Init()` saves original termios, sets raw mode via `golang.org/x/sys/unix` syscalls (or pure Go equivalent using `syscall` package)
- [ ] `Init()` sends ANSI sequences to enter alternate screen buffer (`\033[?1049h`) and hide cursor initially
- [ ] `Fini()` restores original termios, exits alternate screen buffer (`\033[?1049l`), and shows cursor (`\033[?25h`)
- [ ] Terminal is properly restored even on panic (using defer)
- [ ] No external dependencies used -- only Go `syscall` or `golang.org/x/sys/unix` (already indirect dep, or replicate with raw syscall)
- [ ] `go build` succeeds
- [ ] Manual test: run and exit, terminal is restored to normal

### US-002: Screen Size Query and Resize Detection
**Description:** As a developer, I need to query the terminal size and detect resize events so that the editor can adapt its layout.

**Acceptance Criteria:**
- [ ] Implement `Size() (width, height int)` method using `TIOCGWINSZ` ioctl
- [ ] Detect terminal resize via `SIGWINCH` signal
- [ ] Resize events are delivered through the same event channel as key events
- [ ] `go build` succeeds
- [ ] Manual test: resize terminal window, editor redraws correctly

### US-003: Screen Rendering with ANSI Escape Sequences
**Description:** As a developer, I need to render characters to the screen so that the editor can display buffer content, status line, and messages.

**Acceptance Criteria:**
- [ ] Implement cell-based screen buffer (`[][]Cell` where `Cell` has `rune` and `style`)
- [ ] Implement `Clear()` to reset all cells to space
- [ ] Implement `SetContent(x, y int, ch rune, style Style)` to set a cell
- [ ] Implement `Show()` that diffs current buffer vs. previous buffer and writes only changed cells using ANSI cursor positioning (`\033[row;colH`) and character output
- [ ] Implement `ShowCursor(x, y int)` to position the hardware cursor (`\033[row;colH` + show cursor)
- [ ] Implement `Sync()` to force full screen redraw (used after resize)
- [ ] Support reverse video style (`\033[7m` / `\033[0m`)
- [ ] Output is buffered (use `bufio.Writer`) and flushed once per `Show()` call for performance
- [ ] `go build` succeeds
- [ ] Manual test: editor displays text, status line, and messages correctly

### US-004: Keyboard Input Parsing
**Description:** As a developer, I need to read and parse keyboard input so that all existing Emacs keybindings work correctly.

**Acceptance Criteria:**
- [ ] Implement `PollEvent()` that blocks and returns an `Event` interface
- [ ] Parse single-byte control characters: C-a through C-z (0x01-0x1A), C-SPC/NUL (0x00), Enter (0x0D), Backspace (0x7F), Escape (0x1B), Tab (0x09)
- [ ] Parse ANSI escape sequences for arrow keys: `\033[A` (Up), `\033[B` (Down), `\033[C` (Right), `\033[D` (Left)
- [ ] Parse Alt+key combinations: Esc followed by a character (with timeout to distinguish bare Esc)
- [ ] Handle C-_ / C-/ for undo (byte 0x1F)
- [ ] Implement `PostEvent(ev Event)` to re-queue an event (used when exiting search mode)
- [ ] Define `KeyEvent` struct with `Key() KeyCode`, `Rune() rune`, `Modifiers() ModMask` matching the interface main.go expects
- [ ] `go build` succeeds
- [ ] All existing keybindings work: C-f/b/n/p/a/e/v/d/k/y/w/g/s/r/x, C-SPC, C-_, arrow keys, M-v/w/</>, Enter, Backspace, printable runes

### US-005: Define Terminal Abstraction Interface
**Description:** As a developer, I need a clean interface that both the tcell backend and the new pure Go backend can satisfy, so that `main.go` can switch between them.

**Acceptance Criteria:**
- [ ] Define a `Screen` interface in a new file (e.g., `term/screen.go`) with methods: `Init() error`, `Fini()`, `Size() (int, int)`, `PollEvent() Event`, `PostEvent(Event)`, `Clear()`, `SetContent(x, y int, ch rune, style Style)`, `Show()`, `ShowCursor(x, y int)`, `Sync()`
- [ ] Define `Event` interface, `KeyEvent` struct, `ResizeEvent` struct
- [ ] Define `Style` type with `Reverse(bool) Style` method
- [ ] Define key code constants matching all keys used in main.go
- [ ] The pure Go implementation satisfies this interface
- [ ] `go build` succeeds

### US-006: Integrate Pure Go Backend into main.go
**Description:** As a user, I want gomacs to use the pure Go terminal backend so that it has zero external dependencies.

**Acceptance Criteria:**
- [ ] Update `main.go` to import and use the new `term` package instead of `tcell`
- [ ] Replace all `tcell.Key*` constants with equivalent `term.Key*` constants
- [ ] Replace all `tcell.Style*` usage with `term.Style*` equivalents
- [ ] Replace `tcell.EventKey` / `tcell.EventResize` with `term.KeyEvent` / `term.ResizeEvent`
- [ ] Remove `github.com/gdamore/tcell/v2` from `go.mod` and `go.sum`
- [ ] Run `go mod tidy` -- only standard library remains
- [ ] `go build` succeeds with zero external dependencies
- [ ] All existing keybindings work identically
- [ ] Manual test: open file, edit, save, search, undo, mark/region, quit

### US-007: Automated Tests for Terminal Backend
**Description:** As a developer, I need tests for the pure Go terminal backend to verify correctness without a real terminal.

**Acceptance Criteria:**
- [ ] Test keyboard input parsing: control characters, escape sequences, Alt+key, printable runes
- [ ] Test screen buffer: SetContent, Clear, cell diffing logic
- [ ] Test style: reverse video flag
- [ ] Test event queue: PostEvent re-queues correctly
- [ ] `go test ./...` passes

## Functional Requirements

- FR-1: The terminal must be set to raw mode on `Init()` and restored on `Fini()`
- FR-2: The alternate screen buffer must be used (`\033[?1049h` / `\033[?1049l`)
- FR-3: Screen rendering must use ANSI cursor positioning (`\033[row;colH`) and cell diff to minimize output
- FR-4: Reverse video must be supported via `\033[7m` (on) and `\033[0m` (reset)
- FR-5: Hardware cursor must be positioned with `\033[row;colH` on each `Show()` call
- FR-6: Keyboard input must handle: NUL (0x00), control chars (0x01-0x1A), Escape (0x1B), C-_ (0x1F), DEL/Backspace (0x7F), printable UTF-8 runes, ANSI arrow key sequences, and Alt+key (Esc prefix)
- FR-7: An Esc-key timeout (~50ms) must distinguish bare Escape from Esc-prefixed sequences
- FR-8: Terminal resize must be detected via `SIGWINCH` signal and delivered as a `ResizeEvent`
- FR-9: `PostEvent()` must allow re-queuing events for later processing
- FR-10: All output must be buffered and flushed once per `Show()` call

## Non-Goals

- No support for Windows native console (conhost/Windows Terminal without ANSI)
- No support for macOS-specific terminal quirks
- No terminfo/termcap database lookup
- No color support beyond reverse video
- No mouse event support
- No Unicode combining character / wide character (CJK) width handling
- No clipboard integration

## Technical Considerations

- The new terminal code should live in a `term/` subdirectory as an internal package
- Use `syscall` package for `TIOCGWINSZ` ioctl and termios manipulation to avoid any external dependency (including `golang.org/x/sys`)
- UTF-8 input parsing must handle multi-byte rune sequences
- The screen diff algorithm should compare previous and current cell buffers to minimize write syscalls
- The Esc-prefix timeout for Alt+key detection can use `select` with `time.After`
- Signal handling for `SIGWINCH` should use `os/signal.Notify`

## Success Metrics

- `go.mod` contains zero `require` directives (only `module` and `go` lines)
- `go build` produces a working binary
- `go test ./...` passes all existing and new tests
- All keybindings listed in README.md work identically to the tcell-based version
- Binary size is smaller than or equal to the tcell-based version

## Open Questions

- Should `golang.org/x/sys/unix` be acceptable as a dependency for termios, or must we use raw `syscall`? (Current decision: use `syscall` for zero deps)
- What Esc-prefix timeout value works best across SSH connections? (Starting with 50ms)
- Should we support `$TERM` detection for minimal capability checking, or assume ANSI/VT100 universally?
