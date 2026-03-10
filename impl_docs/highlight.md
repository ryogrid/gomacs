# Syntax Highlighting (highlight.go)

The `Highlighter` in `highlight.go` provides syntax coloring for source code buffers using the [Chroma](https://github.com/alecthomas/chroma) lexer library with the monokai theme and 256-color ANSI output.

## Architecture

```mermaid
graph LR
    subgraph "Chroma Library"
        LM["lexers.Match(filename)"]
        CO["chroma.Coalesce(lexer)"]
        SG["styles.Get('monokai')"]
        TK["lexer.Tokenise()"]
    end

    subgraph "Highlighter"
        NH["NewHighlighter(filename)"]
        HL["Highlight(lines)"]
        SA["StyleAt(row, col)"]
        CS["chromaEntryToStyle()"]
        RC["rgbTo256()"]
    end

    subgraph "Integration"
        BUF["Buffer.Highlight"]
        DWC["drawWindowContent()"]
        SC["Screen.SetContent()"]
    end

    NH --> LM
    NH --> CO
    NH --> SG
    HL --> TK
    HL --> CS
    CS --> RC
    BUF --> HL
    DWC --> SA
    SA --> SC
```

## Highlighter Struct

```mermaid
classDiagram
    class Highlighter {
        -lexer chroma.Lexer
        -style *chroma.Style
        -result [][]term.Style
        +Highlight(lines [][]rune)
        +StyleAt(row, col int) term.Style
    }
```

- `lexer` -- Chroma lexer matched by filename extension (e.g., Go lexer for `.go` files). Wrapped with `chroma.Coalesce()` for token merging.
- `style` -- Chroma theme object (`monokai`). Maps token types to colors.
- `result` -- Cached 2D grid of `term.Style` values, one per character in the buffer. Re-computed on each `Highlight()` call.

## Lifecycle

```mermaid
sequenceDiagram
    participant File as File Open
    participant NH as NewHighlighter
    participant Chroma as Chroma Library
    participant Buf as Buffer

    File->>NH: NewHighlighter("main.go")
    NH->>Chroma: lexers.Match("main.go")
    Chroma-->>NH: Go lexer
    NH->>Chroma: chroma.Coalesce(lexer)
    NH->>Chroma: styles.Get("monokai")
    NH-->>Buf: *Highlighter (attached to Buffer)
    Note over Buf: Highlight is nil for<br/>unknown extensions or<br/>special buffers (*scratch*)
```

## Tokenization Flow

```mermaid
flowchart TD
    A["Highlight(lines [][]rune)"] --> B["Join lines with newline<br/>into single string"]
    B --> C["Initialize result grid<br/>[][]term.Style (same dimensions as lines)"]
    C --> D["lexer.Tokenise(nil, source)"]
    D --> E{Error?}
    E -->|yes| F["Return empty result grid"]
    E -->|no| G["Iterate tokens"]

    G --> H["For each token:"]
    H --> I["Get chroma.StyleEntry<br/>via style.Get(tok.Type)"]
    I --> J["Convert to term.Style<br/>via chromaEntryToStyle()"]
    J --> K["For each rune in token value:"]
    K --> L{Newline?}
    L -->|yes| M["row++, col = 0"]
    L -->|no| N["result[row][col] = style<br/>col++"]
    M --> K
    N --> K
```

## Color Conversion

Chroma provides colors as RGB values. The terminal uses 256-color ANSI palette indices. The conversion is done manually since Chroma v2 lacks a `Nearest256()` method.

### 256-Color Palette Structure

```
Colors   0-7:    Standard colors (black, red, green, etc.)
Colors   8-15:   Bright/high-intensity colors
Colors  16-231:  6x6x6 RGB color cube
Colors 232-255:  Grayscale ramp (24 shades)
```

### RGB to 256-Color Conversion

```mermaid
flowchart TD
    A["rgbTo256(r, g, b)"] --> B{r == g == b?}
    B -->|yes| C[Grayscale path]
    B -->|no| D[Color cube path]

    C --> E{"r < 8?"}
    E -->|yes| F["return 16 (black)"]
    E -->|no| G{"r > 248?"}
    G -->|yes| H["return 231 (white)"]
    G -->|no| I["Map to grayscale ramp<br/>colors 232-255"]

    D --> J["Map r to nearest cube level (0-5)"]
    D --> K["Map g to nearest cube level (0-5)"]
    D --> L["Map b to nearest cube level (0-5)"]
    J --> M["return 16 + 36*ri + 6*gi + bi"]
    K --> M
    L --> M
```

### Color Cube Index Mapping

The 6x6x6 cube uses these levels: `[0, 95, 135, 175, 215, 255]`.

`colorCubeIndex(v)` finds the nearest level by absolute distance:

| Input Range | Nearest Level | Index |
|-------------|---------------|-------|
| 0-47 | 0 | 0 |
| 48-114 | 95 | 1 |
| 115-154 | 135 | 2 |
| 155-194 | 175 | 3 |
| 195-234 | 215 | 4 |
| 235-255 | 255 | 5 |

## Style Conversion

`chromaEntryToStyle()` converts a Chroma `StyleEntry` to a `term.Style`:

| Chroma Property | Mapping | Notes |
|----------------|---------|-------|
| `entry.Colour` | `term.Style.Foreground(Color(rgbTo256(...)))` | Only if `Colour.IsSet()` is true |
| `entry.Bold` | `term.Style.Bold(true)` | Only if `entry.Bold == chroma.Yes` (tristate: Yes/No/Pass) |
| `entry.Background` | Not mapped | Background colors from the theme are intentionally ignored |

## Integration with Buffer and Rendering

```mermaid
flowchart TD
    A["Buffer edited<br/>(InsertChar, Backspace, etc.)"] --> B["HighlightDirty = true"]
    B --> C["Next redraw cycle"]
    C --> D{"HighlightDirty and<br/>Highlight != nil?"}
    D -->|yes| E["Highlight.Highlight(buf.Lines)"]
    E --> F["HighlightDirty = false"]
    D -->|no| G["Use cached result"]
    F --> H["Per-cell rendering"]
    G --> H

    H --> I["base = Highlight.StyleAt(row, col)"]
    I --> J{Region or search match?}
    J -->|yes| K["style = base.Reverse(true)"]
    J -->|no| L["style = base"]
    K --> M["screen.SetContent(x, y, ch, style)"]
    L --> M
```

- Buffers without a matching lexer (`.txt`, unknown extensions) have `Highlight == nil` and render with `StyleDefault` (terminal default color).
- Special buffers (`*scratch*`, `*Buffer List*`) never have a Highlighter.
- Region and search highlighting overlay reverse video on top of syntax colors, preserving the foreground/background.

## Supported Languages

All languages supported by Chroma are available automatically. Language detection is based on filename extension via `lexers.Match()`. Common examples:

| Extension | Language |
|-----------|----------|
| `.go` | Go |
| `.py` | Python |
| `.js`, `.ts` | JavaScript, TypeScript |
| `.rs` | Rust |
| `.c`, `.h` | C |
| `.cpp`, `.cc` | C++ |
| `.java` | Java |
| `.rb` | Ruby |
| `.sh`, `.bash` | Bash |
| `.html` | HTML |
| `.css` | CSS |
| `.json` | JSON |
| `.yaml`, `.yml` | YAML |
| `.md` | Markdown |
| `.sql` | SQL |
