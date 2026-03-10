# Terminal Backend (term package)

The `term/` package provides a pure Go terminal I/O layer using ANSI/VT100 escape sequences and Linux syscalls, with 256-color rendering support.

## Package Structure

```mermaid
graph TB
    subgraph "term/screen.go — Interfaces & Types"
        SI[Screen Interface]
        EI[Event Interface]
        KE[KeyEvent]
        RE[ResizeEvent]
        CO[Color Type]
        ST[Style Struct]
        KC[KeyCode Constants]
        MM[ModMask Constants]
    end

    subgraph "term/terminal.go — Implementation"
        TM[Terminal Struct]
        RM[Raw Mode<br/>termios syscalls]
        SB[Screen Buffer<br/>cell diffing]
        WS[writeStyledCell<br/>256-color ANSI]
        IP[Input Parser<br/>control/ANSI/UTF-8]
        SH[SIGWINCH Handler]
        ES[Event System<br/>channels]
    end

    SI -.->|implemented by| TM
    EI -.->|implemented by| KE
    EI -.->|implemented by| RE
    TM --- RM
    TM --- SB
    TM --- WS
    TM --- IP
    TM --- SH
    TM --- ES
```

## Screen Interface

```go
type Screen interface {
    Init() error
    Fini()
    Size() (width, height int)
    PollEvent() Event
    PostEvent(Event)
    Clear()
    SetContent(x, y int, ch rune, style Style)
    Show()
    ShowCursor(x, y int)
    Sync()
}
```

This is the only abstraction boundary in the terminal layer. All rendering and input code in `main.go` depends on this interface, not on the concrete `Terminal` struct.

## Key Types

### Event Hierarchy

```mermaid
classDiagram
    class Event {
        <<interface>>
        +isEvent()
    }
    class KeyEvent {
        -key KeyCode
        -ch rune
        -mod ModMask
        +Key() KeyCode
        +Rune() rune
        +Modifiers() ModMask
    }
    class ResizeEvent {
        -width int
        -height int
        +Size() (int, int)
    }
    Event <|.. KeyEvent
    Event <|.. ResizeEvent
```

### Color and Style

The `Style` type is a struct supporting 256-color foreground/background, reverse video, and bold:

```mermaid
classDiagram
    class Color {
        <<int16>>
    }

    class Style {
        -fg Color
        -bg Color
        -reverse bool
        -bold bool
        +Foreground(Color) Style
        +Background(Color) Style
        +Bold(bool) Style
        +Reverse(bool) Style
        +Fg() Color
        +Bg() Color
        +IsBold() bool
        +IsReverse() bool
    }

    Style --> Color : fg, bg

    note for Color "ColorDefault = -1\n0-255 = palette index"
    note for Style "StyleDefault = {fg: -1, bg: -1}\nAll methods return new Style (builder pattern)"
```

- `Color` is `int16`. `-1` (`ColorDefault`) means use the terminal's default color. Values 0-255 map to the 256-color ANSI palette.
- `Style` is a value type (struct). All methods return a new `Style` (builder pattern).
- `StyleDefault` is a `var` (not `const`, since Go structs can't be const): `Style{fg: ColorDefault, bg: ColorDefault}`.
- Go struct `==` comparison works for all fields, so the cell diffing in `Show()` works without custom logic.

### KeyCode Constants

Key codes start at 256 to leave 0-255 for ASCII. Notable constants:

| Constant | Value Range | Input |
|----------|-------------|-------|
| `KeyRune` | 256 | printable chars |
| `KeyCtrlA`..`KeyCtrlZ` | 259-284 | 0x01-0x1A |
| `KeyCtrlSpace` | 285 | 0x00 |
| `KeyCtrlUnderscore` | 286 | 0x1F |
| `KeyEnter` | 287 | 0x0D |
| `KeyBackspace` / `KeyBackspace2` | 288-289 | 0x08 / 0x7F |
| `KeyEsc` | 290 | 0x1B |
| `KeyUp/Down/Left/Right` | 291-294 | ESC [ A/B/C/D |
| `KeyTab` | 295 | 0x09 |

## Terminal Initialization

```mermaid
sequenceDiagram
    participant App as main.go
    participant Term as Terminal
    participant Kernel as Linux Kernel
    participant Stdout

    App->>Term: NewTerminal()
    App->>Term: Init()
    Term->>Kernel: ioctl(TCGETS) — save original termios
    Term->>Kernel: ioctl(TCSETS) — set raw mode
    Note over Kernel: Clear: ECHO, ICANON, ISIG,<br/>IEXTEN, ICRNL, IXON, OPOST<br/>Set: CS8, VMIN=1, VTIME=0
    Term->>Stdout: ESC[?1049h — enter alternate screen
    Term->>Stdout: ESC[?25l — hide cursor
    Term->>Term: allocate cell buffers (width x height)
    Term->>Term: mark all prev cells dirty (ch = -1)
    Term->>Term: start readInput() goroutine
    Term->>Term: start handleSigwinch() goroutine
    App->>Term: ... editor runs ...
    App->>Term: Fini()
    Term->>Term: close stopSig channel
    Term->>Kernel: signal.Stop(sigwinch)
    Term->>Stdout: ESC[?25h — show cursor
    Term->>Stdout: ESC[?1049l — exit alternate screen
    Term->>Kernel: ioctl(TCSETS) — restore original termios
```

### Raw Mode Flags

| Category | Flags Cleared | Purpose |
|----------|--------------|---------|
| Input (`Iflag`) | `BRKINT`, `ICRNL`, `INPCK`, `ISTRIP`, `IXON` | Disable break, CR-to-NL, parity, stripping, flow control |
| Output (`Oflag`) | `OPOST` | Disable output processing |
| Control (`Cflag`) | -- (sets `CS8`) | 8-bit characters |
| Local (`Lflag`) | `ECHO`, `ICANON`, `IEXTEN`, `ISIG` | Disable echo, canonical mode, extended input, signals |

## Screen Rendering

### Cell Buffer Architecture

```
cells[height][width]  ← current frame
prev[height][width]   ← previous frame (for diffing)

Each cell: { ch rune, style Style }
```

### 256-Color ANSI Rendering

The `writeStyledCell()` method on `Terminal` is the single source of truth for converting a `Style` into ANSI escape sequences:

```mermaid
flowchart TD
    A["writeStyledCell(ch, style)"] --> B{style == StyleDefault?}
    B -->|yes| C["write ch (no escapes)"]
    B -->|no| D["build ANSI sequence parts"]
    D --> E{bold?}
    E -->|yes| F["add '1' (bold)"]
    E -->|no| G{fg != ColorDefault?}
    F --> G
    G -->|yes| H["add '38;5;N' (fg color)"]
    G -->|no| I{bg != ColorDefault?}
    H --> I
    I -->|yes| J["add '48;5;N' (bg color)"]
    I -->|no| K{reverse?}
    J --> K
    K -->|yes| L["add '7' (reverse video)"]
    K -->|no| M["emit ESC[parts...m"]
    L --> M
    M --> N["write ch"]
    N --> O["emit ESC[0m (reset)"]
```

All attributes are combined into a single `\033[...m` sequence with semicolon separators. After the character, `\033[0m` resets all attributes. Default-styled cells emit no escape sequences.

### ANSI Escape Sequence Summary

| Sequence | Purpose |
|----------|---------|
| `ESC[?1049h` / `ESC[?1049l` | Enter / exit alternate screen buffer |
| `ESC[?25h` / `ESC[?25l` | Show / hide cursor |
| `ESC[row;colH` | Position cursor (1-based coordinates) |
| `ESC[1m` | Bold |
| `ESC[7m` | Reverse video |
| `ESC[38;5;Nm` | Set foreground to 256-color palette index N |
| `ESC[48;5;Nm` | Set background to 256-color palette index N |
| `ESC[0m` | Reset all attributes |

### Show() Diff Pipeline

```mermaid
flowchart TD
    A["Show()"] --> B{Size changed?}
    B -->|yes| C[resize: reallocate buffers]
    B -->|no| D[Diff Loop]
    C --> D
    D --> E{"cells[r][c] != prev[r][c]?"}
    E -->|no| F[skip cell]
    E -->|yes| G["write ESC[row;colH (position)"]
    G --> H["writeStyledCell(ch, style)"]
    H --> I["prev[r][c] = cells[r][c]"]
    I --> E
    F --> E
    D --> J["position cursor: ESC[row;colH"]
    J --> K["show cursor: ESC[?25h"]
    K --> L["flush bufio.Writer"]
```

**Key optimization**: Only changed cells produce output. The `prev` buffer tracks what was last rendered. On `Sync()`, all `prev` cells are set to `ch = -1` (sentinel), forcing a full redraw on the next `Show()`.

## Keyboard Input Parsing

### Parser Architecture

```mermaid
flowchart TD
    A[readInput goroutine] -->|read bytes| B[parseInput]
    B --> C{first byte?}
    C -->|0x1B ESC| D[parseEscSequence]
    C -->|0x00| E[KeyCtrlSpace]
    C -->|0x01-0x1A| F["KeyCtrlA + (byte - 1)"]
    C -->|0x09| G[KeyTab]
    C -->|0x0D| H[KeyEnter]
    C -->|0x1F| I[KeyCtrlUnderscore]
    C -->|0x7F| J[KeyBackspace2]
    C -->|0x20-0x7E| K["KeyRune + rune"]
    C -->|0x80+| L["UTF-8 decode -> KeyRune"]

    D --> M{next byte?}
    M -->|"0x5B '&#91;'"| N[CSI Sequence]
    M -->|0x20-0x7E| O["Alt+key (ModAlt)"]
    M -->|0x01-0x1A| P["Alt+Ctrl (ModAlt)"]
    M -->|nothing in 50ms| Q[bare KeyEsc]

    N --> R{third byte?}
    R -->|A| S[KeyUp]
    R -->|B| T[KeyDown]
    R -->|C| U[KeyRight]
    R -->|D| V[KeyLeft]
```

### Escape Key Timeout

Distinguishing a bare Escape press from an Alt+key or ANSI sequence:

```mermaid
sequenceDiagram
    participant Stdin
    participant Parser as parseInput()
    participant Timer as 50ms Timer
    participant Events as Event Channel

    Stdin->>Parser: 0x1B (ESC byte)
    Note over Parser: No more bytes in buffer
    Parser->>Timer: start 50ms timeout
    alt More bytes arrive within 50ms
        Stdin->>Parser: additional bytes
        Parser->>Parser: parseEscSequence()
        Parser->>Events: Alt+key or arrow key
    else Timeout expires
        Timer->>Parser: timeout
        Parser->>Events: bare KeyEsc
    end
```

The timeout is set to 50ms (`escTimeout`), which works well for local and SSH sessions.

## Event System

### Channel Architecture

```mermaid
graph LR
    subgraph "Producers"
        RI[readInput goroutine]
        SW[handleSigwinch goroutine]
        PE[PostEvent caller]
    end

    subgraph "Channels (cap: 64)"
        EC[events channel]
        PC[posted channel]
    end

    subgraph "Consumer"
        PO[PollEvent]
    end

    RI -->|KeyEvent| EC
    SW -->|ResizeEvent| EC
    PE -->|any Event| PC

    PC -->|priority| PO
    EC -->|fallback| PO
```

### PollEvent Priority

```go
func (t *Terminal) PollEvent() Event {
    // Fast path: check posted channel (non-blocking)
    select {
    case ev := <-t.posted:
        return ev
    default:
    }
    // Slow path: wait for either channel
    select {
    case ev := <-t.posted:
        return ev
    case ev := <-t.events:
        return ev
    }
}
```

Posted events (from `PostEvent()`) always take priority over stdin/resize events. This is used by the search mode exit logic in `main.go`, which re-posts the key event that triggered the exit so it can be processed as a normal command.

## Resize Handling

```mermaid
sequenceDiagram
    participant Kernel
    participant SigHandler as handleSigwinch()
    participant EventChan as events channel
    participant EventLoop as main.go

    Kernel->>SigHandler: SIGWINCH signal
    SigHandler->>SigHandler: Size() via TIOCGWINSZ ioctl
    SigHandler->>EventChan: ResizeEvent{width, height}
    EventChan->>EventLoop: PollEvent() returns ResizeEvent
    EventLoop->>EventLoop: recalcWindows()
    EventLoop->>EventLoop: Sync() + redraw
```

- `SIGWINCH` is caught via `os/signal.Notify()`
- The handler goroutine queries the new terminal size using the `TIOCGWINSZ` ioctl
- A `ResizeEvent` is posted to the events channel
- The event loop calls `recalcWindows()` to redistribute window heights, then `Sync()` (marks all cells dirty) and redraws

## Test Infrastructure

Tests use dependency injection to avoid real terminal I/O:

- **`newTestTerminal()`** -- Creates a `Terminal` with pre-allocated channels and buffers, no `Init()` syscall needed.
- **`Terminal.in` field** -- Accepts any `io.Reader`, allowing tests to inject `bytes.Reader` instead of `os.Stdin`.
- **`showForTest()`** -- Test-only version of `Show()` that skips the `Size()` ioctl call. Uses `writeStyledCell()` for consistent rendering logic.
- **`drainEvents()`** -- Non-blocking drain of all queued events for assertion.
- **`parseInput()`** -- Can be called directly (no goroutine needed) for unit testing input parsing.

26 tests cover:
- Control character parsing (Ctrl-A through Ctrl-Z, Ctrl-Space, Ctrl-Underscore)
- Special keys (Enter, Backspace, Tab, Escape)
- ANSI arrow key sequences (Up, Down, Left, Right)
- UTF-8 multi-byte rune decoding
- Alt+key modifier detection
- Screen buffer operations (SetContent, Clear)
- Cell diffing and selective output
- 256-color foreground/background ANSI output
- Bold attribute rendering
- Reverse video style
- Combined style attributes (bold + fg + bg + reverse in one sequence)
- PostEvent priority over stdin events
- Sync dirty marking for full redraw
