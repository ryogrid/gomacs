package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Buffer holds the text content as a slice of lines, where each line is a slice of runes.
type Buffer struct {
	Lines        [][]rune
	CursorR      int // cursor row
	CursorC      int // cursor column
	ScrollOffset int // top visible line
	Modified     bool
	Filename     string
}

// NewBuffer creates a new empty buffer with one empty line.
func NewBuffer() *Buffer {
	return &Buffer{
		Lines: [][]rune{{}},
	}
}

// NewBufferFromFile loads a file into a new buffer. Returns an error if the file cannot be read.
func NewBufferFromFile(filename string) (*Buffer, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	content := string(data)
	// Remove trailing newline to avoid an extra empty line at the end.
	content = strings.TrimSuffix(content, "\n")
	rawLines := strings.Split(content, "\n")
	lines := make([][]rune, len(rawLines))
	for i, rl := range rawLines {
		lines[i] = []rune(rl)
	}
	if len(lines) == 0 {
		lines = [][]rune{{}}
	}
	return &Buffer{
		Lines:    lines,
		Filename: filename,
	}, nil
}

// InsertChar inserts a rune at the current cursor position and advances the cursor.
func (b *Buffer) InsertChar(ch rune) {
	line := b.Lines[b.CursorR]
	newLine := make([]rune, len(line)+1)
	copy(newLine, line[:b.CursorC])
	newLine[b.CursorC] = ch
	copy(newLine[b.CursorC+1:], line[b.CursorC:])
	b.Lines[b.CursorR] = newLine
	b.CursorC++
	b.Modified = true
}

// InsertNewline splits the current line at the cursor position.
func (b *Buffer) InsertNewline() {
	line := b.Lines[b.CursorR]
	before := make([]rune, b.CursorC)
	copy(before, line[:b.CursorC])
	after := make([]rune, len(line)-b.CursorC)
	copy(after, line[b.CursorC:])

	b.Lines[b.CursorR] = before

	// Insert the new line after the current one.
	newLines := make([][]rune, len(b.Lines)+1)
	copy(newLines, b.Lines[:b.CursorR+1])
	newLines[b.CursorR+1] = after
	copy(newLines[b.CursorR+2:], b.Lines[b.CursorR+1:])
	b.Lines = newLines

	b.CursorR++
	b.CursorC = 0
	b.Modified = true
}

// Backspace deletes the character before the cursor. If at the beginning of a line,
// it joins the current line with the previous one.
func (b *Buffer) Backspace() {
	if b.CursorC > 0 {
		line := b.Lines[b.CursorR]
		newLine := make([]rune, len(line)-1)
		copy(newLine, line[:b.CursorC-1])
		copy(newLine[b.CursorC-1:], line[b.CursorC:])
		b.Lines[b.CursorR] = newLine
		b.CursorC--
		b.Modified = true
	} else if b.CursorR > 0 {
		// Join with previous line.
		prevLine := b.Lines[b.CursorR-1]
		curLine := b.Lines[b.CursorR]
		newCol := len(prevLine)
		joined := make([]rune, len(prevLine)+len(curLine))
		copy(joined, prevLine)
		copy(joined[len(prevLine):], curLine)
		b.Lines[b.CursorR-1] = joined

		// Remove current line.
		b.Lines = append(b.Lines[:b.CursorR], b.Lines[b.CursorR+1:]...)

		b.CursorR--
		b.CursorC = newCol
		b.Modified = true
	}
}

// MoveForward moves the cursor forward one character.
func (b *Buffer) MoveForward() {
	if b.CursorC < len(b.Lines[b.CursorR]) {
		b.CursorC++
	} else if b.CursorR < len(b.Lines)-1 {
		b.CursorR++
		b.CursorC = 0
	}
}

// MoveBackward moves the cursor backward one character.
func (b *Buffer) MoveBackward() {
	if b.CursorC > 0 {
		b.CursorC--
	} else if b.CursorR > 0 {
		b.CursorR--
		b.CursorC = len(b.Lines[b.CursorR])
	}
}

// MoveUp moves the cursor up one line.
func (b *Buffer) MoveUp() {
	if b.CursorR > 0 {
		b.CursorR--
		if b.CursorC > len(b.Lines[b.CursorR]) {
			b.CursorC = len(b.Lines[b.CursorR])
		}
	}
}

// MoveDown moves the cursor down one line.
func (b *Buffer) MoveDown() {
	if b.CursorR < len(b.Lines)-1 {
		b.CursorR++
		if b.CursorC > len(b.Lines[b.CursorR]) {
			b.CursorC = len(b.Lines[b.CursorR])
		}
	}
}

// MoveBeginningOfLine moves the cursor to the beginning of the current line.
func (b *Buffer) MoveBeginningOfLine() {
	b.CursorC = 0
}

// MoveEndOfLine moves the cursor to the end of the current line.
func (b *Buffer) MoveEndOfLine() {
	b.CursorC = len(b.Lines[b.CursorR])
}

// ScrollDown scrolls the view down by one page.
func (b *Buffer) ScrollDown(viewHeight int) {
	b.ScrollOffset += viewHeight
	maxOffset := len(b.Lines) - 1
	if b.ScrollOffset > maxOffset {
		b.ScrollOffset = maxOffset
	}
	// Move cursor to top of new view if it's above the viewport
	if b.CursorR < b.ScrollOffset {
		b.CursorR = b.ScrollOffset
		if b.CursorC > len(b.Lines[b.CursorR]) {
			b.CursorC = len(b.Lines[b.CursorR])
		}
	}
}

// ScrollUp scrolls the view up by one page.
func (b *Buffer) ScrollUp(viewHeight int) {
	b.ScrollOffset -= viewHeight
	if b.ScrollOffset < 0 {
		b.ScrollOffset = 0
	}
	// Move cursor to bottom of new view if it's below the viewport
	lastVisible := b.ScrollOffset + viewHeight - 1
	if lastVisible >= len(b.Lines) {
		lastVisible = len(b.Lines) - 1
	}
	if b.CursorR > lastVisible {
		b.CursorR = lastVisible
		if b.CursorC > len(b.Lines[b.CursorR]) {
			b.CursorC = len(b.Lines[b.CursorR])
		}
	}
}

// MoveBeginningOfBuffer moves cursor to the beginning of the buffer.
func (b *Buffer) MoveBeginningOfBuffer() {
	b.CursorR = 0
	b.CursorC = 0
}

// MoveEndOfBuffer moves cursor to the end of the buffer.
func (b *Buffer) MoveEndOfBuffer() {
	b.CursorR = len(b.Lines) - 1
	b.CursorC = len(b.Lines[b.CursorR])
}

// AdjustScroll ensures the cursor is visible within the viewport of the given height.
func (b *Buffer) AdjustScroll(viewHeight int) {
	if b.CursorR < b.ScrollOffset {
		b.ScrollOffset = b.CursorR
	}
	if b.CursorR >= b.ScrollOffset+viewHeight {
		b.ScrollOffset = b.CursorR - viewHeight + 1
	}
}

// Save writes the buffer contents to disk. It writes to a temp file first, then renames
// for safety. Returns an error if the buffer has no filename or the write fails.
func (b *Buffer) Save() error {
	if b.Filename == "" {
		return errNoFilename
	}
	var sb strings.Builder
	for i, line := range b.Lines {
		sb.WriteString(string(line))
		if i < len(b.Lines)-1 {
			sb.WriteByte('\n')
		}
	}
	sb.WriteByte('\n') // trailing newline

	dir := filepath.Dir(b.Filename)
	tmp, err := os.CreateTemp(dir, ".gomacs-save-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, err = tmp.WriteString(sb.String())
	if err2 := tmp.Close(); err == nil {
		err = err2
	}
	if err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, b.Filename); err != nil {
		os.Remove(tmpName)
		return err
	}
	b.Modified = false
	return nil
}

var errNoFilename = fmt.Errorf("no filename")

// DeleteChar deletes the character at the cursor (forward delete). If at the end of a line,
// it joins the next line to the current one. This will be used by C-d in US-006.
func (b *Buffer) DeleteChar() {
	line := b.Lines[b.CursorR]
	if b.CursorC < len(line) {
		newLine := make([]rune, len(line)-1)
		copy(newLine, line[:b.CursorC])
		copy(newLine[b.CursorC:], line[b.CursorC+1:])
		b.Lines[b.CursorR] = newLine
		b.Modified = true
	} else if b.CursorR < len(b.Lines)-1 {
		// Join next line to current line.
		nextLine := b.Lines[b.CursorR+1]
		joined := make([]rune, len(line)+len(nextLine))
		copy(joined, line)
		copy(joined[len(line):], nextLine)
		b.Lines[b.CursorR] = joined

		// Remove next line.
		b.Lines = append(b.Lines[:b.CursorR+1], b.Lines[b.CursorR+2:]...)
		b.Modified = true
	}
}
