# Event Loop and UI Rendering

The event loop in `main.go` is the central coordinator of goomacs. It polls terminal events, dispatches them to buffer operations, and triggers screen rendering.

## Event Loop State Machine

The event loop operates as a state machine with three modes: **Normal**, **Search**, and **C-x Prefix**.

```mermaid
stateDiagram-v2
    [*] --> Normal

    Normal --> CxPrefix : C-x pressed
    Normal --> SearchForward : C-s pressed
    Normal --> SearchBackward : C-r pressed
    Normal --> Normal : editing / movement keys

    CxPrefix --> Normal : C-s (save) or C-c (quit)
    CxPrefix --> Normal : any other key

    SearchForward --> Normal : Enter (accept)
    SearchForward --> Normal : C-g (cancel)
    SearchForward --> Normal : unhandled key (repost)
    SearchForward --> SearchForward : rune / C-s / Backspace
    SearchForward --> SearchBackward : C-r

    SearchBackward --> Normal : Enter (accept)
    SearchBackward --> Normal : C-g (cancel)
    SearchBackward --> Normal : unhandled key (repost)
    SearchBackward --> SearchBackward : rune / C-r / Backspace
    SearchBackward --> SearchForward : C-s
```

### State Variables

| Variable | Type | Purpose |
|----------|------|---------|
| `prefixCx` | `bool` | True when waiting for second key after C-x |
| `searchMode` | `bool` | True when in incremental search |
| `searchForward` | `bool` | True for forward search, false for backward |
| `searchQuery` | `[]rune` | Characters typed in search mode |
| `searchOrigR/C` | `int` | Cursor position before search started (for C-g cancel) |
| `searchMatchR/C` | `int` | Position of current match (for highlighting) |
| `searchHasMatch` | `bool` | Whether current query matches any text |
| `quitWarned` | `bool` | True after first C-x C-c on unsaved buffer |
| `message` | `string` | Message displayed on the bottom line |

## Main Loop Structure

```mermaid
flowchart TD
    A[PollEvent] --> B{Event Type?}

    B -->|KeyEvent| C{searchMode?}
    C -->|yes| D[Handle Search Keys]
    C -->|no| E{prefixCx?}
    E -->|yes| F[Handle C-x Second Key]
    E -->|no| G[Handle Normal Keys]

    B -->|ResizeEvent| H[Sync + Redraw]

    D --> I[redraw]
    F --> I
    G --> I
    H --> I

    I --> A
```

## Search Mode

When the user presses C-s or C-r, the editor enters incremental search mode. Each keystroke updates the search query and immediately finds the next match.

```mermaid
flowchart TD
    START[Enter search mode<br/>save cursor position] --> WAIT[Wait for key]

    WAIT --> |rune| ADD[Append to query<br/>search from cursor]
    WAIT --> |C-s| FWD[Search forward<br/>from cursor+1]
    WAIT --> |C-r| BWD[Search backward<br/>from cursor]
    WAIT --> |Backspace| DEL[Remove last char<br/>re-search from original pos]
    WAIT --> |Enter| ACCEPT[Exit search mode<br/>cursor stays at match]
    WAIT --> |C-g| CANCEL[Exit search mode<br/>restore original cursor]
    WAIT --> |other key| REPOST[Exit search mode<br/>PostEvent the key]

    ADD --> |found| UPDATE[Move cursor to match<br/>update highlight]
    ADD --> |not found| FAIL["Show 'Failing I-search'"]
    FWD --> |found| UPDATE
    FWD --> |not found| FAIL
    BWD --> |found| UPDATE
    BWD --> |not found| FAIL
    DEL --> |query not empty| RESEARCH[Re-search from<br/>original position]
    DEL --> |query empty| RESTORE[Restore cursor to<br/>original position]

    UPDATE --> WAIT
    FAIL --> WAIT
    RESEARCH --> WAIT
    RESTORE --> WAIT
```

**Key behavior**: When an unrecognized key is pressed during search, the editor exits search mode and re-posts the event via `screen.PostEvent(ev)` so it gets handled as a normal command. This ensures keys like C-a or C-e work seamlessly when pressed during a search.

## C-x Prefix Commands

The C-x prefix creates a two-key command sequence:

| Sequence | Action |
|----------|--------|
| C-x C-s | Save file to disk |
| C-x C-c | Quit (with unsaved-changes warning) |

The quit logic uses `quitWarned`:
1. First C-x C-c on a modified buffer shows a warning message
2. Second C-x C-c confirms and exits
3. `quitWarned` resets on any key that is not C-x

## Normal Mode Key Dispatch

```mermaid
flowchart LR
    subgraph "Movement"
        CF[C-f → MoveForward]
        CB[C-b → MoveBackward]
        CN[C-n → MoveDown]
        CP[C-p → MoveUp]
        CA[C-a → BeginningOfLine]
        CE[C-e → EndOfLine]
        CV[C-v → ScrollDown]
        AR[Arrow keys → Move*]
    end

    subgraph "Editing"
        RN[Rune → InsertChar]
        EN[Enter → InsertNewline]
        BS[Backspace → Backspace]
        CD[C-d → DeleteChar]
    end

    subgraph "Kill / Yank"
        CK[C-k → KillLine]
        CW[C-w → KillRegion]
        CY[C-y → Yank]
    end

    subgraph "Mark / Region"
        CS["C-SPC → SetMark"]
        CG[C-g → DeactivateMark]
    end

    subgraph "Other"
        CU[C-_ → Undo]
        MV["M-v → ScrollUp"]
        MW["M-w → CopyRegion"]
        ML["M-< → BeginningOfBuffer"]
        MG["M-> → EndOfBuffer"]
    end
```

**Important**: All editing operations (InsertChar, InsertNewline, Backspace, DeleteChar, KillLine, KillRegion, Yank) call `buf.SaveUndo()` before the operation. Non-kill keys call `buf.ClearLastKill()` to reset the consecutive-kill tracker.

### Alt Key Handling

Alt+key combinations are handled in two ways:

1. **ModAlt flag** -- The terminal layer detects `ESC + key` within 50ms and delivers a `KeyRune` event with `ModAlt` set. The event loop checks `ev.Modifiers() & term.ModAlt`.

2. **Bare ESC fallback** -- If the terminal delivers a bare `KeyEsc` (timeout expired), the event loop manually calls `PollEvent()` again to read the next key. This handles terminals that send ESC and the key as separate events with a delay.

## Rendering Pipeline

### Screen Layout

```
Row 0 to (height-3):   Text area (buffer content)
Row height-2:           Status line (reverse video)
Row height-1:           Message line
```

`textAreaHeight(screenHeight)` returns `max(screenHeight - 2, 1)`.

### drawBufferWithSearch

The main rendering function iterates through visible lines:

```mermaid
flowchart TD
    A[Clear screen] --> B[Get screen dimensions]
    B --> C[For each visible row]
    C --> D[Get buffer line at row + ScrollOffset]
    D --> E[For each character in line]
    E --> F{Is tab?}
    F -->|yes| G[Expand to spaces<br/>up to next tab stop]
    F -->|no| H[Single character]
    G --> I{In active region?}
    H --> I
    I -->|yes| J[Apply reverse video]
    I -->|no| K{In search match?}
    K -->|yes| J
    K -->|no| L[Default style]
    J --> M[SetContent]
    L --> M
    M --> E
    E -->|done| C
    C -->|done| N[drawStatusLine]
```

### Tab Expansion

Tabs are stored as literal `\t` in the buffer but displayed as spaces aligned to 8-column tab stops.

```go
const tabWidth = 8

func bufColToVisualCol(line []rune, bufCol int) int {
    visualCol := 0
    for i := 0; i < bufCol && i < len(line); i++ {
        if line[i] == '\t' {
            visualCol += tabWidth - (visualCol % tabWidth)
        } else {
            visualCol++
        }
    }
    return visualCol
}
```

This function is used to position the hardware cursor correctly when the line contains tabs.

### drawStatusLine

Renders on the second-to-last row in reverse video:

```
 filename.go [Modified]                    Line 42/100, Col 15
```

- Left side: filename and modification indicator
- Right side: current line / total lines, column number

### drawMessageLine

Renders on the last row in normal style. Shows:
- `"C-x-"` during C-x prefix
- `"I-search: query"` during search
- `"Mark set"`, `"Region copied"`, `"Saved filename"`, etc.
- `"Quit"` after C-g
- Empty string clears the line

### Redraw Closure

The `redraw()` function is defined as a closure inside `main()`:

```
1. AdjustScroll(viewHeight)       — ensure cursor is visible
2. drawBufferWithSearch(...)      — render text with highlighting
3. drawMessageLine(message)       — render message
4. ShowCursor(visualX, screenY)   — position hardware cursor
5. Show()                         — flush to terminal (diff-based)
```

This is called after every key event and resize event.

## Complete Event Processing Flow

```mermaid
sequenceDiagram
    participant Term as Terminal
    participant Loop as Event Loop
    participant Buf as Buffer
    participant Screen as Screen Rendering

    Loop->>Term: PollEvent()
    Term-->>Loop: KeyEvent (e.g., 'x')

    Loop->>Loop: clear message
    Loop->>Loop: ClearLastKill (non-kill key)
    Loop->>Buf: SaveUndo()
    Loop->>Buf: InsertChar('x')
    Buf->>Buf: modify Lines, advance cursor

    Loop->>Buf: AdjustScroll(viewHeight)
    Loop->>Screen: Clear()
    Loop->>Screen: SetContent(...) for each cell
    Loop->>Screen: ShowCursor(visualCol, screenRow)
    Loop->>Screen: Show()
    Screen->>Screen: diff and output ANSI
```
