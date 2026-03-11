# goomacs

A lightweight, fast-starting Emacs-like terminal text editor written in pure Go.

goomacs provides a familiar Emacs keybinding experience for quick file editing without the overhead of a full Emacs installation. It is designed to be minimal, portable, and easy to use.

<img width="883" height="706" alt="image" src="https://github.com/user-attachments/assets/bf947672-a6aa-488d-8064-d9e099baa6d0" />


## Features

- **Emacs keybindings** -- Navigate and edit with standard Emacs key sequences
- **Incremental search** -- Forward and backward search with wraparound (C-s / C-r)
- **Mark and region** -- Set mark, select regions, kill/copy/yank (C-SPC, C-w, M-w, C-y)
- **Kill ring** -- Consecutive kills accumulate; yank pastes the last kill
- **Undo** -- Up to 100 levels of undo history (C-_ / C-/)
- **Tab support** -- Displays tabs with 8-column tab stops
- **Multiple buffers** -- Open and switch between multiple files (C-x b, C-x C-f, C-x C-b)
- **Window splitting** -- Split the screen vertically (C-x 2) or horizontally side-by-side (C-x 3) to view multiple buffers at once
- **Syntax highlighting** -- Automatic syntax coloring for source code files using Chroma (monokai theme, 256-color)
- **M-x command palette** -- Execute commands by name with tab completion (M-x)
- **Comment/uncomment region** -- Toggle comments on selected code with automatic language detection (30+ languages)
- **Tab completion** -- Filename and command name completion in prompts (Tab key)
- **Goto line** -- Jump to any line by number (C-l)
- **Minimal dependencies** -- Pure Go implementation using ANSI/VT100 escape sequences

## Installation

Requires Go 1.24 or later.

```bash
go build -o goomacs .
```

## Usage

```bash
./goomacs                          # Open with an empty *scratch* buffer
./goomacs filename.txt             # Open a single file
./goomacs file1.go file2.go        # Open multiple files (first is active)
```

## Keybindings

### Cursor Movement

| Key | Action |
|-----|--------|
| C-f / Right | Forward one character |
| C-b / Left | Backward one character |
| C-n / Down | Next line |
| C-p / Up | Previous line |
| C-a | Beginning of line |
| C-e | End of line |
| M-< | Beginning of buffer |
| M-> | End of buffer |
| C-v | Scroll down one page |
| M-v | Scroll up one page |
| C-l | Goto line (prompts for line number) |

### Editing

| Key | Action |
|-----|--------|
| C-h | Backspace (delete character before cursor) |
| C-d | Delete character at cursor |
| C-j | Newline (same as Enter) |
| C-k | Kill line (cut to end of line) |
| C-SPC | Set mark |
| C-w | Kill region (cut) |
| M-w | Copy region |
| C-y | Yank (paste) |
| C-_ / C-/ | Undo |
| C-g | Cancel / deactivate mark |

### Search

| Key | Action |
|-----|--------|
| C-s | Incremental search forward |
| C-r | Incremental search backward |

### File / Buffer Operations

| Key | Action |
|-----|--------|
| C-x C-s | Save file |
| C-x C-f | Open file (find-file) |
| C-x b | Switch buffer by name |
| C-x C-b | List all open buffers |
| C-x k | Close (kill) buffer |
| C-x C-c | Quit (warns on unsaved changes) |

### Window Management

| Key | Action |
|-----|--------|
| C-x 2 | Split window vertically (top/bottom) |
| C-x 3 | Split window horizontally (side-by-side) |
| C-x o | Move focus to next window |
| C-x 0 | Close current window |
| C-x 1 | Close all other windows |

### Command Palette

| Key | Action |
|-----|--------|
| M-x | Open command prompt (type command name, Tab to complete, Enter to execute) |

**Available commands:**

| Command | Description |
|---------|-------------|
| comment-region | Comment out the selected region (requires active mark) |
| uncomment-region | Remove comments from the selected region |

Language detection is automatic (powered by go-enry). Supported languages include Go, Python, JavaScript, TypeScript, Rust, C, C++, Java, Ruby, Shell, HTML, CSS, SQL, Haskell, Lisp, and 15+ more. For unrecognized languages, `#` is used as the default comment prefix.

## Status Bar

The status bar at the bottom of the screen shows:
- Filename (or `[No Name]` for unsaved buffers)
- `[Modified]` indicator when the buffer has unsaved changes
- Current line / total lines and column number (e.g., `Line 42/100, Col 15`)

## Project Structure

```
goomacs/
├── main.go              # Event loop, keybinding dispatch, and UI rendering
├── buffer.go            # Buffer data structure and editing operations
├── command.go           # Command registry, M-x commands, comment/uncomment logic
├── highlight.go         # Chroma-based syntax highlighting module
├── buffer_test.go       # Buffer unit tests (69 tests)
├── main_test.go         # Main package tests
├── go.mod               # Go module definition
├── term/                # Pure Go terminal backend
│   ├── screen.go        # Screen interface, Event types, Style, KeyCode constants
│   ├── terminal.go      # Terminal implementation (raw mode, ANSI rendering, input parsing)
│   └── terminal_test.go # Terminal backend tests (26 tests)
├── e2e/                 # End-to-end tests (tmux-based)
│   ├── harness.go       # tmux session management and screen capture
│   ├── assertions.go    # Screen assertion helpers and golden file support
│   ├── e2e_test.go      # TestMain, test helpers, smoke test
│   ├── basic_test.go    # Basic editing tests
│   ├── navigation_test.go # Cursor movement and scrolling tests
│   ├── search_test.go   # Incremental search tests
│   ├── killyank_test.go # Kill, yank, and undo tests
│   ├── buffer_test.go   # Buffer management tests
│   ├── window_test.go   # Window splitting tests
│   ├── highlight_test.go # Syntax highlighting smoke tests
│   └── testdata/        # Golden files for snapshot assertions
└── impl_docs/           # Implementation documentation
    ├── architecture.md  # Architecture overview with diagrams
    ├── buffer.md        # Buffer data structure documentation
    ├── terminal.md      # Terminal backend documentation
    └── event-loop.md    # Event loop and rendering documentation
```

## Dependencies

- [github.com/alecthomas/chroma/v2](https://github.com/alecthomas/chroma) -- Syntax highlighting (lexers and themes)
- [github.com/go-enry/go-enry/v2](https://github.com/go-enry/go-enry) -- Programming language detection for comment style lookup
- [github.com/mattn/go-runewidth](https://github.com/mattn/go-runewidth) -- Unicode character width calculation (East Asian wide characters, etc.)

All terminal handling uses only the Go standard library (`syscall`, `os`, `bufio`, `unicode/utf8`, etc.) via ANSI/VT100 escape sequences.

## Testing

### Unit Tests

```bash
go test ./...
```

### End-to-End Tests

E2E tests use [tmux](https://github.com/tmux/tmux) to spawn goomacs in a pseudo-terminal, send keystrokes, and verify screen output. This catches rendering and interaction bugs that unit tests cannot.

**Prerequisites:**

```bash
# Install tmux (Ubuntu/Debian)
sudo apt-get install -y tmux

# Or on macOS
brew install tmux
```

**Run E2E tests:**

```bash
go test ./e2e/ -v -timeout 120s
```

The tests automatically build the goomacs binary before running. If tmux is not installed, the tests are skipped.

**Update golden files** (snapshot assertions):

```bash
UPDATE_GOLDEN=1 go test ./e2e/ -v -timeout 120s
```

## License

This project is licensed under the MIT License. See [LICENSE.txt](LICENSE.txt) for details.
