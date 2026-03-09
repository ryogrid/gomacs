package main

import (
	"reflect"
	"testing"
)

func lines(strs ...string) [][]rune {
	result := make([][]rune, len(strs))
	for i, s := range strs {
		result[i] = []rune(s)
	}
	return result
}

func bufLines(b *Buffer) []string {
	result := make([]string, len(b.Lines))
	for i, l := range b.Lines {
		result[i] = string(l)
	}
	return result
}

func TestNewBuffer(t *testing.T) {
	b := NewBuffer()
	if len(b.Lines) != 1 || len(b.Lines[0]) != 0 {
		t.Fatal("NewBuffer should have one empty line")
	}
	if b.CursorR != 0 || b.CursorC != 0 {
		t.Fatal("Cursor should be at 0,0")
	}
}

func TestInsertChar(t *testing.T) {
	b := NewBuffer()
	b.InsertChar('H')
	b.InsertChar('i')

	got := bufLines(b)
	want := []string{"Hi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorC != 2 {
		t.Fatalf("cursor col = %d, want 2", b.CursorC)
	}
	if !b.Modified {
		t.Fatal("buffer should be modified")
	}
}

func TestInsertCharMiddle(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("ac")
	b.CursorC = 1
	b.InsertChar('b')

	got := bufLines(b)
	want := []string{"abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorC != 2 {
		t.Fatalf("cursor col = %d, want 2", b.CursorC)
	}
}

func TestInsertNewline(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 5

	b.InsertNewline()

	got := bufLines(b)
	want := []string{"Hello", " World"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorR != 1 || b.CursorC != 0 {
		t.Fatalf("cursor = (%d,%d), want (1,0)", b.CursorR, b.CursorC)
	}
}

func TestInsertNewlineAtEnd(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 3

	b.InsertNewline()

	got := bufLines(b)
	want := []string{"abc", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestInsertNewlineMultipleLines(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("aaa", "bbb", "ccc")
	b.CursorR = 1
	b.CursorC = 1

	b.InsertNewline()

	got := bufLines(b)
	want := []string{"aaa", "b", "bb", "ccc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBackspaceMiddle(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 2

	b.Backspace()

	got := bufLines(b)
	want := []string{"ac"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorC != 1 {
		t.Fatalf("cursor col = %d, want 1", b.CursorC)
	}
}

func TestBackspaceBeginningOfLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello", " World")
	b.CursorR = 1
	b.CursorC = 0

	b.Backspace()

	got := bufLines(b)
	want := []string{"Hello World"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorR != 0 || b.CursorC != 5 {
		t.Fatalf("cursor = (%d,%d), want (0,5)", b.CursorR, b.CursorC)
	}
}

func TestBackspaceAtOrigin(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorR = 0
	b.CursorC = 0

	b.Backspace()

	got := bufLines(b)
	want := []string{"abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDeleteCharMiddle(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 1

	b.DeleteChar()

	got := bufLines(b)
	want := []string{"ac"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorC != 1 {
		t.Fatalf("cursor col = %d, want 1", b.CursorC)
	}
}

func TestDeleteCharEndOfLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello", " World")
	b.CursorR = 0
	b.CursorC = 5

	b.DeleteChar()

	got := bufLines(b)
	want := []string{"Hello World"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorC != 5 {
		t.Fatalf("cursor col = %d, want 5", b.CursorC)
	}
}

// --- Cursor Movement Tests ---

func TestMoveForward(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 0

	b.MoveForward()
	if b.CursorC != 1 {
		t.Fatalf("cursor col = %d, want 1", b.CursorC)
	}

	// Move to end of line
	b.CursorC = 3
	b.MoveForward() // should not move (single line)
	if b.CursorR != 0 || b.CursorC != 3 {
		t.Fatalf("cursor = (%d,%d), want (0,3)", b.CursorR, b.CursorC)
	}
}

func TestMoveForwardWraps(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("ab", "cd")
	b.CursorC = 2

	b.MoveForward() // should wrap to next line
	if b.CursorR != 1 || b.CursorC != 0 {
		t.Fatalf("cursor = (%d,%d), want (1,0)", b.CursorR, b.CursorC)
	}
}

func TestMoveBackward(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 2

	b.MoveBackward()
	if b.CursorC != 1 {
		t.Fatalf("cursor col = %d, want 1", b.CursorC)
	}

	// At beginning, should not move
	b.CursorC = 0
	b.MoveBackward()
	if b.CursorR != 0 || b.CursorC != 0 {
		t.Fatalf("cursor = (%d,%d), want (0,0)", b.CursorR, b.CursorC)
	}
}

func TestMoveBackwardWraps(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("ab", "cd")
	b.CursorR = 1
	b.CursorC = 0

	b.MoveBackward() // should wrap to end of previous line
	if b.CursorR != 0 || b.CursorC != 2 {
		t.Fatalf("cursor = (%d,%d), want (0,2)", b.CursorR, b.CursorC)
	}
}

func TestMoveUp(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc", "de")
	b.CursorR = 1
	b.CursorC = 1

	b.MoveUp()
	if b.CursorR != 0 || b.CursorC != 1 {
		t.Fatalf("cursor = (%d,%d), want (0,1)", b.CursorR, b.CursorC)
	}

	// At top, should not move
	b.MoveUp()
	if b.CursorR != 0 {
		t.Fatalf("cursor row = %d, want 0", b.CursorR)
	}
}

func TestMoveUpClampsCursor(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("ab", "cdef")
	b.CursorR = 1
	b.CursorC = 4

	b.MoveUp() // shorter line, should clamp
	if b.CursorR != 0 || b.CursorC != 2 {
		t.Fatalf("cursor = (%d,%d), want (0,2)", b.CursorR, b.CursorC)
	}
}

func TestMoveDown(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc", "de")
	b.CursorC = 1

	b.MoveDown()
	if b.CursorR != 1 || b.CursorC != 1 {
		t.Fatalf("cursor = (%d,%d), want (1,1)", b.CursorR, b.CursorC)
	}

	// At bottom, should not move
	b.MoveDown()
	if b.CursorR != 1 {
		t.Fatalf("cursor row = %d, want 1", b.CursorR)
	}
}

func TestMoveDownClampsCursor(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abcd", "ef")
	b.CursorC = 4

	b.MoveDown() // shorter line, should clamp
	if b.CursorR != 1 || b.CursorC != 2 {
		t.Fatalf("cursor = (%d,%d), want (1,2)", b.CursorR, b.CursorC)
	}
}

func TestMoveBeginningOfLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 2

	b.MoveBeginningOfLine()
	if b.CursorC != 0 {
		t.Fatalf("cursor col = %d, want 0", b.CursorC)
	}
}

func TestMoveEndOfLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 0

	b.MoveEndOfLine()
	if b.CursorC != 3 {
		t.Fatalf("cursor col = %d, want 3", b.CursorC)
	}
}

// --- Scroll and Viewport Tests ---

func TestAdjustScrollDown(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("0", "1", "2", "3", "4", "5", "6", "7", "8", "9")
	b.CursorR = 5
	b.AdjustScroll(3) // viewport shows 3 lines
	if b.ScrollOffset != 3 {
		t.Fatalf("ScrollOffset = %d, want 3", b.ScrollOffset)
	}
}

func TestAdjustScrollUp(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("0", "1", "2", "3", "4")
	b.ScrollOffset = 3
	b.CursorR = 1
	b.AdjustScroll(3)
	if b.ScrollOffset != 1 {
		t.Fatalf("ScrollOffset = %d, want 1", b.ScrollOffset)
	}
}

func TestAdjustScrollNoChange(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("0", "1", "2", "3", "4")
	b.ScrollOffset = 1
	b.CursorR = 2
	b.AdjustScroll(5)
	if b.ScrollOffset != 1 {
		t.Fatalf("ScrollOffset = %d, want 1", b.ScrollOffset)
	}
}

func TestScrollDown(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("0", "1", "2", "3", "4", "5", "6", "7", "8", "9")
	b.CursorR = 0
	b.ScrollDown(5)
	if b.ScrollOffset != 5 {
		t.Fatalf("ScrollOffset = %d, want 5", b.ScrollOffset)
	}
	if b.CursorR != 5 {
		t.Fatalf("CursorR = %d, want 5", b.CursorR)
	}
}

func TestScrollDownClamps(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("0", "1", "2")
	b.ScrollDown(10)
	if b.ScrollOffset != 2 {
		t.Fatalf("ScrollOffset = %d, want 2", b.ScrollOffset)
	}
}

func TestScrollUp(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("0", "1", "2", "3", "4", "5", "6", "7", "8", "9")
	b.ScrollOffset = 5
	b.CursorR = 8
	b.ScrollUp(5)
	if b.ScrollOffset != 0 {
		t.Fatalf("ScrollOffset = %d, want 0", b.ScrollOffset)
	}
	if b.CursorR != 4 {
		t.Fatalf("CursorR = %d, want 4", b.CursorR)
	}
}

func TestScrollUpClamps(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("0", "1", "2")
	b.ScrollOffset = 1
	b.CursorR = 2
	b.ScrollUp(10)
	if b.ScrollOffset != 0 {
		t.Fatalf("ScrollOffset = %d, want 0", b.ScrollOffset)
	}
}

func TestMoveBeginningOfBuffer(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc", "def", "ghi")
	b.CursorR = 2
	b.CursorC = 3
	b.MoveBeginningOfBuffer()
	if b.CursorR != 0 || b.CursorC != 0 {
		t.Fatalf("cursor = (%d,%d), want (0,0)", b.CursorR, b.CursorC)
	}
}

func TestMoveEndOfBuffer(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc", "def", "ghi")
	b.CursorR = 0
	b.CursorC = 0
	b.MoveEndOfBuffer()
	if b.CursorR != 2 || b.CursorC != 3 {
		t.Fatalf("cursor = (%d,%d), want (2,3)", b.CursorR, b.CursorC)
	}
}

func TestDeleteCharEndOfBuffer(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 3

	b.DeleteChar()

	got := bufLines(b)
	want := []string{"abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// --- Kill and Yank Tests ---

func TestKillLineMiddle(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 5

	b.KillLine()

	got := bufLines(b)
	want := []string{"Hello"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if len(b.KillRing) != 1 || string(b.KillRing[0]) != " World" {
		t.Fatalf("kill ring = %v, want [\" World\"]", b.KillRing)
	}
}

func TestKillLineAtEnd(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello", "World")
	b.CursorC = 5 // end of "Hello"

	b.KillLine()

	got := bufLines(b)
	want := []string{"HelloWorld"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if len(b.KillRing) != 1 || string(b.KillRing[0]) != "\n" {
		t.Fatalf("kill ring = %v, want [\"\\n\"]", b.KillRing)
	}
}

func TestKillLineEmptyLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("", "World")
	b.CursorC = 0

	b.KillLine()

	got := bufLines(b)
	want := []string{"World"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestConsecutiveKillsAppend(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("aaa", "bbb", "ccc")
	b.CursorC = 0

	b.KillLine() // kills "aaa"
	b.KillLine() // kills newline (joins with bbb)
	b.KillLine() // kills "bbb"

	if len(b.KillRing) != 1 {
		t.Fatalf("expected 1 kill ring entry, got %d", len(b.KillRing))
	}
	if string(b.KillRing[0]) != "aaa\nbbb" {
		t.Fatalf("kill ring = %q, want %q", string(b.KillRing[0]), "aaa\nbbb")
	}
}

func TestNonConsecutiveKillsNewEntry(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("aaa", "bbb")
	b.CursorC = 0

	b.KillLine()      // kills "aaa"
	b.ClearLastKill() // simulate non-kill key
	b.KillLine()      // kills newline — new entry

	if len(b.KillRing) != 2 {
		t.Fatalf("expected 2 kill ring entries, got %d", len(b.KillRing))
	}
}

func TestYankSimple(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 5

	b.KillLine() // kills " World"

	// Move to beginning and yank
	b.CursorC = 0
	b.ClearLastKill()
	b.Yank()

	got := bufLines(b)
	want := []string{" WorldHello"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestYankMultiLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("aaa", "bbb", "ccc")
	b.CursorC = 0

	b.KillLine() // kills "aaa"
	b.KillLine() // kills newline
	b.KillLine() // kills "bbb"

	// Now at line "ccc", yank the killed text
	b.ClearLastKill()
	b.CursorC = 0
	b.Yank()

	got := bufLines(b)
	// After kills: buffer is ["", "ccc"]. Yank "aaa\nbbb" at (0,0) → "aaa", "bbb", "ccc"
	want := []string{"aaa", "bbb", "ccc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestYankEmptyKillRing(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 0

	b.Yank() // should do nothing

	got := bufLines(b)
	want := []string{"abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// --- Mark and Region Tests ---

func TestSetMark(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc", "def")
	b.CursorR = 1
	b.CursorC = 2
	b.SetMark()
	if !b.MarkActive {
		t.Fatal("mark should be active")
	}
	if b.MarkR != 1 || b.MarkC != 2 {
		t.Fatalf("mark = (%d,%d), want (1,2)", b.MarkR, b.MarkC)
	}
}

func TestDeactivateMark(t *testing.T) {
	b := NewBuffer()
	b.SetMark()
	b.DeactivateMark()
	if b.MarkActive {
		t.Fatal("mark should be inactive")
	}
}

func TestRegionTextSameLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 0
	b.SetMark()
	b.CursorC = 5
	text := b.RegionText()
	if string(text) != "Hello" {
		t.Fatalf("region text = %q, want %q", string(text), "Hello")
	}
}

func TestRegionTextMultiLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("aaa", "bbb", "ccc")
	b.CursorR = 0
	b.CursorC = 1
	b.SetMark()
	b.CursorR = 2
	b.CursorC = 2
	text := b.RegionText()
	if string(text) != "aa\nbbb\ncc" {
		t.Fatalf("region text = %q, want %q", string(text), "aa\nbbb\ncc")
	}
}

func TestRegionTextReversed(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 5
	b.SetMark()
	b.CursorC = 0 // point before mark
	text := b.RegionText()
	if string(text) != "Hello" {
		t.Fatalf("region text = %q, want %q", string(text), "Hello")
	}
}

func TestKillRegionSameLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 0
	b.SetMark()
	b.CursorC = 5
	b.KillRegion()

	got := bufLines(b)
	want := []string{" World"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorC != 0 {
		t.Fatalf("cursor col = %d, want 0", b.CursorC)
	}
	if b.MarkActive {
		t.Fatal("mark should be deactivated after kill")
	}
	if len(b.KillRing) != 1 || string(b.KillRing[0]) != "Hello" {
		t.Fatalf("kill ring = %v, want [\"Hello\"]", b.KillRing)
	}
}

func TestKillRegionMultiLine(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("aaa", "bbb", "ccc")
	b.CursorR = 0
	b.CursorC = 1
	b.SetMark()
	b.CursorR = 2
	b.CursorC = 2
	b.KillRegion()

	got := bufLines(b)
	want := []string{"ac"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.CursorR != 0 || b.CursorC != 1 {
		t.Fatalf("cursor = (%d,%d), want (0,1)", b.CursorR, b.CursorC)
	}
}

func TestCopyRegion(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 0
	b.SetMark()
	b.CursorC = 5
	b.CopyRegion()

	// Buffer content should be unchanged
	got := bufLines(b)
	want := []string{"Hello World"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if b.MarkActive {
		t.Fatal("mark should be deactivated after copy")
	}
	if len(b.KillRing) != 1 || string(b.KillRing[0]) != "Hello" {
		t.Fatalf("kill ring = %v, want [\"Hello\"]", b.KillRing)
	}
}

func TestCopyRegionThenYank(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("Hello World")
	b.CursorC = 0
	b.SetMark()
	b.CursorC = 5
	b.CopyRegion()
	b.MoveEndOfLine()
	b.Yank()

	got := bufLines(b)
	want := []string{"Hello WorldHello"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestInRegion(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("aaa", "bbb", "ccc")
	b.CursorR = 0
	b.CursorC = 1
	b.SetMark()
	b.CursorR = 2
	b.CursorC = 2

	// Before region
	if b.InRegion(0, 0) {
		t.Fatal("(0,0) should not be in region")
	}
	// In region
	if !b.InRegion(0, 1) {
		t.Fatal("(0,1) should be in region")
	}
	if !b.InRegion(1, 0) {
		t.Fatal("(1,0) should be in region")
	}
	if !b.InRegion(2, 1) {
		t.Fatal("(2,1) should be in region")
	}
	// At/after end
	if b.InRegion(2, 2) {
		t.Fatal("(2,2) should not be in region (end is exclusive)")
	}
	if b.InRegion(2, 3) {
		t.Fatal("(2,3) should not be in region")
	}
}

func TestInRegionInactive(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	if b.InRegion(0, 0) {
		t.Fatal("should not be in region when mark is inactive")
	}
}

func TestKillRegionInactive(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.KillRegion() // should do nothing
	got := bufLines(b)
	want := []string{"abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestKillLineEndOfBuffer(t *testing.T) {
	b := NewBuffer()
	b.Lines = lines("abc")
	b.CursorC = 3

	b.KillLine() // at end of last line, nothing to kill

	got := bufLines(b)
	want := []string{"abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if len(b.KillRing) != 0 {
		t.Fatalf("kill ring should be empty, got %v", b.KillRing)
	}
}
