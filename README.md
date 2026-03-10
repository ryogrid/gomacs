# Gomacs

A lightweight, fast-starting Emacs-like terminal text editor written in pure Go.

Gomacs provides a familiar Emacs keybinding experience for quick file editing without the overhead of a full Emacs installation. It is designed to be minimal, portable, and easy to use.

<img width="887" height="715" alt="image" src="https://github.com/user-attachments/assets/fa801524-46ab-4a04-806d-6a5437bcbfb7" />

## Features

- **Emacs keybindings** -- Navigate and edit with standard Emacs key sequences
- **Incremental search** -- Forward and backward search with wraparound (C-s / C-r)
- **Mark and region** -- Set mark, select regions, kill/copy/yank (C-SPC, C-w, M-w, C-y)
- **Kill ring** -- Consecutive kills accumulate; yank pastes the last kill
- **Undo** -- Up to 100 levels of undo history (C-_ / C-/)
- **Tab support** -- Displays tabs with 8-column tab stops
- **Zero dependencies** -- Pure Go implementation using ANSI/VT100 escape sequences, no external libraries

## Installation

Requires Go 1.24 or later.

```bash
go build -o gomacs .
```

## Usage

```bash
./gomacs                # Open with an empty buffer
./gomacs filename.txt   # Open an existing file
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

### Editing

| Key | Action |
|-----|--------|
| C-d | Delete character at cursor |
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

### File Operations

| Key | Action |
|-----|--------|
| C-x C-s | Save file |
| C-x C-c | Quit (warns on unsaved changes) |

## Status Bar

The status bar at the bottom of the screen shows:
- Filename (or `[No Name]` for unsaved buffers)
- `[Modified]` indicator when the buffer has unsaved changes
- Current line / total lines and column number (e.g., `Line 42/100, Col 15`)

## Project Structure

```
gomacs/
├── main.go              # Event loop, keybinding dispatch, and UI rendering
├── buffer.go            # Buffer data structure and editing operations
├── buffer_test.go       # Buffer unit tests (69 tests)
├── main_test.go         # Main package tests
├── go.mod               # Go module definition (zero dependencies)
├── term/                # Pure Go terminal backend
│   ├── screen.go        # Screen interface, Event types, Style, KeyCode constants
│   ├── terminal.go      # Terminal implementation (raw mode, ANSI rendering, input parsing)
│   └── terminal_test.go # Terminal backend tests (16 tests)
└── impl_docs/           # Implementation documentation
    ├── architecture.md  # Architecture overview with diagrams
    ├── buffer.md        # Buffer data structure documentation
    ├── terminal.md      # Terminal backend documentation
    └── event-loop.md    # Event loop and rendering documentation
```

## Dependencies

No external dependencies. Gomacs uses only the Go standard library (`syscall`, `os`, `bufio`, `unicode/utf8`, etc.) for terminal handling via ANSI/VT100 escape sequences.

## Testing

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See [LICENSE.txt](LICENSE.txt) for details.
