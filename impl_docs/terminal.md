# Terminal Backend (term package)

The `term/` package provides a pure Go terminal I/O layer using ANSI/VT100 escape sequences and Linux syscalls. It replaces the previous `tcell` dependency, achieving zero external dependencies.

## Package Structure

```mermaid
graph TB
    subgraph "term/screen.go — Interfaces & Types"
        SI[Screen Interface]
        EI[Event Interface]
        KE[KeyEvent]
        RE[ResizeEvent]
        ST[Style]
        KC[KeyCode Constants]
        MM[ModMask Constants]
    end

    subgraph "term/terminal.go — Implementation"
        TM[Terminal Struct]
        RM[Raw Mode<br/>termios syscalls]
        SB[Screen Buffer<br/>cell diffing]
        IP[Input Parser<br/>control/ANSI/UTF-8]
        SH[SIGWINCH Handler]
        ES[Event System<br/>channels]
    end

    SI -.->|implemented by| TM
    EI -.->|implemented by| KE
    EI -.->|implemented by| RE
    TM --- RM
    TM --- SB
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

### Style

A `uint8` bitmask. Currently only supports reverse video (bit 0).

```go
StyleDefault = Style(0)       // normal text
style.Reverse(true)           // reverse video on
style.Reverse(false)          // reverse video off
style.IsReverse() bool        // check flag
```

### KeyCode Constants

Key codes start at 256 to leave 0-255 for ASCII. Notable constants:

| Constant | Value | Byte(s) |
|----------|-------|---------|
| `KeyRune` | 256 | printable chars |
| `KeyNUL` | 257 | 0x00 |
| `KeyCtrlA`..`KeyCtrlZ` | 259-284 | 0x01-0x1A |
| `KeyCtrlSpace` | 285 | 0x00 (alias) |
| `KeyCtrlUnderscore` | 286 | 0x1F |
| `KeyEnter` | 287 | 0x0D |
| `KeyBackspace` | 288 | 0x08 |
| `KeyBackspace2` | 289 | 0x7F |
| `KeyEsc` | 290 | 0x1B |
| `KeyTab` | 291 | 0x09 |
| `KeyUp/Down/Left/Right` | 292-295 | ESC [ A/B/C/D |

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
| Control (`Cflag`) | — (sets `CS8`) | 8-bit characters |
| Local (`Lflag`) | `ECHO`, `ICANON`, `IEXTEN`, `ISIG` | Disable echo, canonical mode, extended input, signals |

## Screen Rendering

### Cell Buffer Architecture

```
cells[height][width]  ← current frame
prev[height][width]   ← previous frame (for diffing)

Each cell: { ch rune, style Style }
```

### Rendering Pipeline

```mermaid
flowchart TD
    A[Application calls Show] --> B{Size changed?}
    B -->|yes| C[resize: reallocate buffers]
    B -->|no| D[Diff Loop]
    C --> D
    D --> E{cells i,j != prev i,j ?}
    E -->|no| F[skip cell]
    E -->|yes| G["write ESC[row;colH"]
    G --> H{style.IsReverse?}
    H -->|yes| I["write ESC[7m + char + ESC[0m"]
    H -->|no| J[write char]
    I --> K[update prev]
    J --> K
    K --> E
    F --> E
    D --> L["position cursor: ESC[row;colH"]
    L --> M["show cursor: ESC[?25h"]
    M --> N[flush bufio.Writer]
```

**Key optimization**: Only changed cells produce output. The `prev` buffer tracks what was last rendered. On `Sync()`, all `prev` cells are set to `ch = -1` (sentinel), forcing a full redraw on the next `Show()`.

### ANSI Escape Sequences Used

| Sequence | Purpose |
|----------|---------|
| `ESC[?1049h` / `ESC[?1049l` | Enter / exit alternate screen buffer |
| `ESC[?25h` / `ESC[?25l` | Show / hide cursor |
| `ESC[row;colH` | Position cursor (1-based coordinates) |
| `ESC[7m` | Enable reverse video |
| `ESC[0m` | Reset all attributes |

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
    C -->|0x80+| L["UTF-8 decode → KeyRune"]

    D --> M{next byte?}
    M -->|"0x5B '['"| N[CSI Sequence]
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
    EventLoop->>EventLoop: Sync() + redraw
```

- `SIGWINCH` is caught via `os/signal.Notify()`
- The handler goroutine queries the new terminal size using the `TIOCGWINSZ` ioctl (`0x5413`)
- A `ResizeEvent` is posted to the events channel
- The event loop calls `Sync()` (marks all cells dirty) then redraws

## Test Infrastructure

Tests use dependency injection to avoid real terminal I/O:

- **`newTestTerminal()`** -- Creates a `Terminal` with pre-allocated channels and buffers, no `Init()` syscall needed.
- **`Terminal.in` field** -- Accepts any `io.Reader`, allowing tests to inject `bytes.Reader` instead of `os.Stdin`.
- **`showForTest()`** -- Test-only version of `Show()` that skips the `Size()` ioctl call.
- **`drainEvents()`** -- Non-blocking drain of all queued events for assertion.
- **`parseInput()`** -- Can be called directly (no goroutine needed) for unit testing input parsing.

16 tests cover: control character parsing, special keys, ANSI arrow sequences, UTF-8 multi-byte runes, Alt+key modifier, screen buffer operations, cell diffing, reverse video style, PostEvent priority, and Sync dirty marking.
