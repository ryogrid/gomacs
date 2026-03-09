package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// undoEntry stores a snapshot of buffer state for undo.
type undoEntry struct {
	Lines   [][]rune
	CursorR int
	CursorC int
}

const maxUndoEntries = 100

// Buffer holds the text content as a slice of lines, where each line is a slice of runes.
type Buffer struct {
	Lines        [][]rune
	CursorR      int // cursor row
	CursorC      int // cursor column
	ScrollOffset int // top visible line
	Modified     bool
	Filename     string
	KillRing     [][]rune // kill ring entries (each entry is a sequence of runes, newlines included)
	lastKill     bool     // true if the last operation was a kill (for appending consecutive kills)
	MarkR        int      // mark row
	MarkC        int      // mark column
	MarkActive   bool     // true when mark is set and region is active
	UndoStack    []undoEntry
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

// SaveUndo saves the current buffer state onto the undo stack.
// Call this before any editing operation.
func (b *Buffer) SaveUndo() {
	snapshot := undoEntry{
		Lines:   make([][]rune, len(b.Lines)),
		CursorR: b.CursorR,
		CursorC: b.CursorC,
	}
	for i, line := range b.Lines {
		cp := make([]rune, len(line))
		copy(cp, line)
		snapshot.Lines[i] = cp
	}
	b.UndoStack = append(b.UndoStack, snapshot)
	if len(b.UndoStack) > maxUndoEntries {
		b.UndoStack = b.UndoStack[len(b.UndoStack)-maxUndoEntries:]
	}
}

// Undo restores the buffer to the most recent undo snapshot.
// Returns true if undo was performed, false if nothing to undo.
func (b *Buffer) Undo() bool {
	if len(b.UndoStack) == 0 {
		return false
	}
	entry := b.UndoStack[len(b.UndoStack)-1]
	b.UndoStack = b.UndoStack[:len(b.UndoStack)-1]
	b.Lines = entry.Lines
	b.CursorR = entry.CursorR
	b.CursorC = entry.CursorC
	b.Modified = true
	return true
}

// KillLine kills text from cursor to end of line (C-k). If cursor is at end of line,
// kills the newline (joins with next line). Consecutive kills append to the kill ring entry.
func (b *Buffer) KillLine() {
	line := b.Lines[b.CursorR]
	if b.CursorC < len(line) {
		// Kill from cursor to end of line
		killed := make([]rune, len(line)-b.CursorC)
		copy(killed, line[b.CursorC:])
		b.Lines[b.CursorR] = line[:b.CursorC]
		b.appendKill(killed)
	} else if b.CursorR < len(b.Lines)-1 {
		// At end of line: kill the newline (join with next line)
		b.appendKill([]rune{'\n'})
		nextLine := b.Lines[b.CursorR+1]
		joined := make([]rune, len(line)+len(nextLine))
		copy(joined, line)
		copy(joined[len(line):], nextLine)
		b.Lines[b.CursorR] = joined
		b.Lines = append(b.Lines[:b.CursorR+1], b.Lines[b.CursorR+2:]...)
	} else {
		// At end of last line: nothing to kill, but mark as kill for consecutive tracking
		b.lastKill = true
		return
	}
	b.Modified = true
	b.lastKill = true
}

// appendKill adds killed text to the kill ring. If the last operation was also a kill,
// it appends to the current kill ring entry instead of creating a new one.
func (b *Buffer) appendKill(text []rune) {
	if b.lastKill && len(b.KillRing) > 0 {
		// Append to existing entry
		last := b.KillRing[len(b.KillRing)-1]
		combined := make([]rune, len(last)+len(text))
		copy(combined, last)
		copy(combined[len(last):], text)
		b.KillRing[len(b.KillRing)-1] = combined
	} else {
		b.KillRing = append(b.KillRing, text)
	}
}

// ClearLastKill resets the consecutive kill tracking. Call this on any non-kill operation.
func (b *Buffer) ClearLastKill() {
	b.lastKill = false
}

// Yank inserts the last killed text at the cursor position.
func (b *Buffer) Yank() {
	if len(b.KillRing) == 0 {
		return
	}
	text := b.KillRing[len(b.KillRing)-1]
	for _, ch := range text {
		if ch == '\n' {
			b.InsertNewline()
		} else {
			b.InsertChar(ch)
		}
	}
}

// SetMark sets the mark at the current cursor position and activates the region.
func (b *Buffer) SetMark() {
	b.MarkR = b.CursorR
	b.MarkC = b.CursorC
	b.MarkActive = true
}

// DeactivateMark deactivates the mark (cancels selection).
func (b *Buffer) DeactivateMark() {
	b.MarkActive = false
}

// regionBounds returns the start and end positions of the region (mark to point),
// ordered so that start <= end. Returns (startR, startC, endR, endC).
func (b *Buffer) regionBounds() (int, int, int, int) {
	r1, c1 := b.MarkR, b.MarkC
	r2, c2 := b.CursorR, b.CursorC
	if r1 > r2 || (r1 == r2 && c1 > c2) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}
	return r1, c1, r2, c2
}

// RegionText returns the text in the region between mark and point as a slice of runes.
func (b *Buffer) RegionText() []rune {
	if !b.MarkActive {
		return nil
	}
	startR, startC, endR, endC := b.regionBounds()
	var result []rune
	for r := startR; r <= endR; r++ {
		line := b.Lines[r]
		cStart := 0
		cEnd := len(line)
		if r == startR {
			cStart = startC
		}
		if r == endR {
			cEnd = endC
		}
		result = append(result, line[cStart:cEnd]...)
		if r < endR {
			result = append(result, '\n')
		}
	}
	return result
}

// KillRegion kills (cuts) the region between mark and point, storing it in the kill ring.
func (b *Buffer) KillRegion() {
	if !b.MarkActive {
		return
	}
	text := b.RegionText()
	if len(text) == 0 {
		b.MarkActive = false
		return
	}
	b.KillRing = append(b.KillRing, text)
	b.deleteRegion()
	b.MarkActive = false
}

// CopyRegion copies the region to the kill ring without deleting it.
func (b *Buffer) CopyRegion() {
	if !b.MarkActive {
		return
	}
	text := b.RegionText()
	if len(text) > 0 {
		b.KillRing = append(b.KillRing, text)
	}
	b.MarkActive = false
}

// deleteRegion removes the text between mark and point, placing cursor at the start.
func (b *Buffer) deleteRegion() {
	startR, startC, endR, endC := b.regionBounds()

	if startR == endR {
		// Same line: remove characters between startC and endC
		line := b.Lines[startR]
		newLine := make([]rune, startC+len(line)-endC)
		copy(newLine, line[:startC])
		copy(newLine[startC:], line[endC:])
		b.Lines[startR] = newLine
	} else {
		// Multi-line: join start of first line with end of last line, remove lines in between
		startLine := b.Lines[startR][:startC]
		endLine := b.Lines[endR][endC:]
		joined := make([]rune, len(startLine)+len(endLine))
		copy(joined, startLine)
		copy(joined[len(startLine):], endLine)
		b.Lines[startR] = joined
		b.Lines = append(b.Lines[:startR+1], b.Lines[endR+1:]...)
	}

	b.CursorR = startR
	b.CursorC = startC
	b.Modified = true
}

// InRegion returns true if the given buffer position (row, col) is within the active region.
func (b *Buffer) InRegion(row, col int) bool {
	if !b.MarkActive {
		return false
	}
	startR, startC, endR, endC := b.regionBounds()
	if row < startR || row > endR {
		return false
	}
	if row == startR && col < startC {
		return false
	}
	if row == endR && col >= endC {
		return false
	}
	return true
}

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
