package main

import (
	"strings"

	"goomacs/term"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter tokenizes buffer content using Chroma and produces per-cell styles.
type Highlighter struct {
	lexer  chroma.Lexer
	style  *chroma.Style
	result [][]term.Style
}

// NewHighlighter creates a highlighter for the given filename.
// Returns nil if no lexer matches the file extension.
func NewHighlighter(filename string) *Highlighter {
	lexer := lexers.Match(filename)
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)
	style := styles.Get("monokai")
	return &Highlighter{
		lexer: lexer,
		style: style,
	}
}

// Highlight tokenizes the given lines and caches per-cell styles.
func (h *Highlighter) Highlight(lines [][]rune) {
	// Join lines into a single string for tokenization.
	parts := make([]string, len(lines))
	for i, line := range lines {
		parts[i] = string(line)
	}
	source := strings.Join(parts, "\n")

	// Initialize result grid.
	result := make([][]term.Style, len(lines))
	for i, line := range lines {
		result[i] = make([]term.Style, len(line))
	}

	// Tokenize.
	iter, err := h.lexer.Tokenise(nil, source)
	if err != nil {
		h.result = result
		return
	}

	row, col := 0, 0
	for _, tok := range iter.Tokens() {
		entry := h.style.Get(tok.Type)
		s := chromaEntryToStyle(entry)

		for _, ch := range tok.Value {
			if ch == '\n' {
				row++
				col = 0
				continue
			}
			if row < len(result) && col < len(result[row]) {
				result[row][col] = s
			}
			col++
		}
	}

	h.result = result
}

// StyleAt returns the cached highlight style at the given position,
// or term.StyleDefault if out of range.
func (h *Highlighter) StyleAt(row, col int) term.Style {
	if row < 0 || row >= len(h.result) {
		return term.StyleDefault
	}
	if col < 0 || col >= len(h.result[row]) {
		return term.StyleDefault
	}
	return h.result[row][col]
}

// chromaEntryToStyle converts a chroma StyleEntry to a term.Style.
func chromaEntryToStyle(entry chroma.StyleEntry) term.Style {
	s := term.StyleDefault
	if entry.Colour.IsSet() {
		s = s.Foreground(term.Color(rgbTo256(entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue())))
	}
	if entry.Bold == chroma.Yes {
		s = s.Bold(true)
	}
	return s
}

// rgbTo256 converts an RGB color to the nearest 256-color palette index.
func rgbTo256(r, g, b uint8) int {
	// Check if it's close to a grayscale value (colors 232-255).
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return int((r-8)/247.0*24.0) + 232
	}

	// Map to the 6x6x6 color cube (colors 16-231).
	ri := colorCubeIndex(r)
	gi := colorCubeIndex(g)
	bi := colorCubeIndex(b)
	return 16 + 36*ri + 6*gi + bi
}

// colorCubeIndex maps a 0-255 value to the nearest 6-level cube index (0-5).
func colorCubeIndex(v uint8) int {
	// The 6 cube levels are: 0, 95, 135, 175, 215, 255.
	levels := [6]uint8{0, 0x5f, 0x87, 0xaf, 0xd7, 0xff}
	best := 0
	bestDist := int(255)
	for i, lv := range levels {
		d := int(v) - int(lv)
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best
}
