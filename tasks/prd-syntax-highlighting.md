# PRD: Syntax Highlighting with Chroma

## Introduction

Add syntax highlighting to goomacs using the Chroma library (github.com/alecthomas/chroma). Currently all buffer text is rendered in a single default color. This feature colorizes source code based on file extension, using Chroma's lexers for language detection and a built-in theme for color mapping. The terminal backend must be extended to support 256-color ANSI output.

## Goals

- Automatically detect language from file extension and apply syntax highlighting
- Support all languages provided by Chroma out of the box
- Use 256-color ANSI escape sequences for rendering colored text
- Use a single built-in dark theme (monokai) for consistent appearance
- Maintain existing functionality: region highlighting, search highlighting, and status line styling must coexist with syntax colors
- Invalidate and re-highlight only when buffer content changes (not on every redraw)

## User Stories

### US-001: Extend Style type to support foreground and background colors
**Description:** As a developer, I need the term.Style type to carry foreground and background color information so that the rendering pipeline can output colored text.

**Acceptance Criteria:**
- [ ] Style type is extended to hold foreground color, background color, and bold flag in addition to the existing reverse flag
- [ ] Color is represented as a type that can express "default/no color" and 256-color palette indices (0-255)
- [ ] Add methods: `Foreground(Color) Style`, `Background(Color) Style`, `Bold(bool) Style`
- [ ] Add accessor methods: `Fg() Color`, `Bg() Color`, `IsBold() bool`
- [ ] `ColorDefault` constant represents "no color set" (terminal default)
- [ ] Existing `Reverse()` and `IsReverse()` methods continue to work
- [ ] `StyleDefault` zero value has no colors set, no reverse, no bold
- [ ] Existing tests pass; new tests for color style methods pass
- [ ] Typecheck passes

### US-002: Update terminal rendering to output 256-color ANSI sequences
**Description:** As a developer, I need the terminal's Show() method to emit ANSI color escape sequences so that styled cells appear in color.

**Acceptance Criteria:**
- [ ] Show() emits `\033[38;5;Nm` for foreground color N (256-color) when a cell has a foreground color set
- [ ] Show() emits `\033[48;5;Nm` for background color N (256-color) when a cell has a background color set
- [ ] Show() emits `\033[1m` for bold when a cell has bold enabled
- [ ] Show() emits `\033[0m` to reset attributes after each styled cell (or uses optimized state tracking)
- [ ] Cells with only reverse video (no colors) continue to work as before
- [ ] Cells with both color and reverse video work correctly
- [ ] Rendering performance is not significantly degraded (buffered writes, minimal escape sequences)
- [ ] Existing terminal tests pass; new tests for color rendering pass
- [ ] Typecheck passes

### US-003: Add Chroma-based syntax highlighting module
**Description:** As a developer, I need a module that tokenizes buffer content using Chroma and produces a per-cell style map so that drawWindowContent can apply syntax colors.

**Acceptance Criteria:**
- [ ] New file `highlight.go` in the main package
- [ ] A `Highlighter` struct that holds the Chroma lexer, Chroma style (monokai), and a cached highlight result
- [ ] `NewHighlighter(filename string)` creates a highlighter, auto-detecting the lexer from filename extension via Chroma's `lexers.Match()`; returns nil if no lexer matches
- [ ] A method `Highlight(lines [][]rune)` that tokenizes the full buffer content, maps Chroma token styles to term.Style values, and stores the result as `[][]term.Style` (per-row, per-column)
- [ ] A method `StyleAt(row, col int) term.Style` that returns the highlight style for a given position, falling back to `term.StyleDefault` if out of range or no highlighter
- [ ] Chroma token color (from monokai theme) is converted to the nearest 256-color ANSI palette index using Chroma's built-in color mapping
- [ ] Bold tokens from the theme are mapped to bold style
- [ ] `go get github.com/alecthomas/chroma/v2` adds Chroma as a dependency
- [ ] Typecheck passes

### US-004: Integrate highlighting into the buffer and rendering pipeline
**Description:** As a user, I want source code files to appear with syntax colors so that I can read code more easily.

**Acceptance Criteria:**
- [ ] Each Buffer has an associated `*Highlighter` (can be nil for unsupported files or special buffers like *scratch*, *Buffer List*)
- [ ] Highlighter is created when a buffer is loaded from file (NewBufferFromFile) or when a buffer's filename changes (C-x C-f)
- [ ] Highlighting is re-computed when the buffer content changes (on any edit operation that modifies Lines)
- [ ] `drawWindowContent()` applies the highlight style from the Highlighter to each cell, merging with region/search highlight styles
- [ ] When region or search highlighting is active on a cell, reverse video is applied on top of the syntax color (syntax color visible through reverse video)
- [ ] Files without a matching Chroma lexer render in default terminal color (no error, no crash)
- [ ] Special buffers (*scratch*, *Buffer List*) have no highlighting
- [ ] Opening a .go file shows Go syntax highlighting (keywords, strings, comments in distinct colors)
- [ ] Existing tests pass (`go test ./...`)
- [ ] Typecheck passes

### US-005: Dirty flag for efficient re-highlighting
**Description:** As a developer, I need to avoid re-running Chroma tokenization on every redraw for performance, only re-highlighting when the buffer content actually changes.

**Acceptance Criteria:**
- [ ] Buffer has a `highlightDirty` flag (or generation counter) that is set to true on any edit operation (InsertChar, InsertNewline, Backspace, DeleteChar, KillLine, KillRegion, Yank, Undo)
- [ ] The rendering pipeline checks the dirty flag before re-running Highlight(); if not dirty, uses the cached result
- [ ] After Highlight() runs, the dirty flag is cleared
- [ ] Switching buffers or opening files does not cause unnecessary re-highlighting of unchanged buffers
- [ ] Typecheck passes

## Functional Requirements

- FR-1: Extend `term.Style` to support 256-color foreground, background, and bold attributes
- FR-2: Update `Terminal.Show()` to emit ANSI 256-color escape sequences (`\033[38;5;Nm`, `\033[48;5;Nm`, `\033[1m`)
- FR-3: Use `github.com/alecthomas/chroma/v2` for lexing and theme support
- FR-4: Auto-detect language from filename extension using `chroma/lexers.Match()`
- FR-5: Use monokai theme from `chroma/styles`
- FR-6: Convert Chroma token colors to 256-color palette indices
- FR-7: Cache highlight results per buffer; re-compute only on content changes
- FR-8: Merge syntax highlight styles with region/search reverse-video in `drawWindowContent()`
- FR-9: Gracefully handle files with no matching lexer (render in default color)

## Non-Goals

- Theme selection or configuration is not included (hardcoded monokai)
- True color (24-bit) terminal support is not included
- Incremental/partial re-highlighting (only changed lines) is not included; full buffer is re-tokenized
- Syntax-aware indentation or code folding is not included
- Background color from theme is not applied to the entire screen (only to highlighted tokens)

## Design Considerations

- The `Style` type in `term/screen.go` is currently `uint8`. It must be changed to a struct to hold foreground color, background color, bold, and reverse flags. This is a breaking change to the `cell` struct in `term/terminal.go` and all `SetContent()` call sites.
- When merging highlight styles with region/search highlighting: apply syntax colors as the base, then overlay reverse video for selection/search. This preserves color information while making selections visible.

## Technical Considerations

- Chroma v2 API: `lexers.Match(filename)` returns a lexer; `chroma.Coalesce(lexer)` ensures token merging; `styles.Get("monokai")` returns the theme; iterate tokens from `lexer.Tokenise()` to build style map
- Chroma's `style.Colour` includes an RGB value; use Chroma's `colour.Nearest256()` or manual mapping to convert to 256-color ANSI index
- The `Highlight()` method must convert `[][]rune` to a single string for Chroma tokenization, then map token positions back to row/col in the `[][]term.Style` grid
- Changing `Style` from `uint8` to a struct changes its comparison behavior in `Show()` diff logic (`cur != t.prev[r][c]` still works since Go structs are comparable if all fields are comparable)
- Adding Chroma as a dependency means goomacs will no longer be "zero dependencies" -- update README accordingly

## Success Metrics

- Opening a `.go` file shows keywords (func, if, for) in one color, strings in another, comments in a third
- Opening a `.py`, `.js`, or `.rs` file shows appropriate language-specific highlighting
- Opening a `.txt` or unknown extension file shows no highlighting (default color, no errors)
- Editing a highlighted file updates colors immediately after each keystroke
- No noticeable lag when typing in files under 10,000 lines

## Open Questions

- Should the monokai theme's background color be applied to the whole terminal, or only to token backgrounds?
- Should we add a command-line flag to disable syntax highlighting for users who prefer plain text?
