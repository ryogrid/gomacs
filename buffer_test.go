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
