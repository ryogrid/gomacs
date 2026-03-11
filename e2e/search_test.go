package e2e

import (
	"strings"
	"testing"
	"time"
)

func TestSearch(t *testing.T) {
	t.Run("ForwardSearch", func(t *testing.T) {
		content := "apple banana cherry\napple grape orange\napple mango peach\n"
		path := createTestFile(t, "search.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("apple banana", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Start forward search with C-s
		h.SendKeys("C-s")
		time.Sleep(200 * time.Millisecond)

		// Type query incrementally
		h.SendKeys("banana")
		if err := h.WaitForContent("I-search: banana", 3*time.Second); err != nil {
			t.Fatalf("search prompt not shown: %v", err)
		}
		h.AssertMessageLine(t, "I-search: banana")

		// Cursor should have moved to the match
		row, col, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		if row != 0 || col != 6 {
			t.Errorf("cursor: expected (0, 6), got (%d, %d)", row, col)
		}
	})

	t.Run("SearchNext", func(t *testing.T) {
		content := "apple banana cherry\napple grape orange\napple mango peach\n"
		path := createTestFile(t, "searchnext.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("apple banana", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// C-s and type "apple"
		h.SendKeys("C-s")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("apple")
		if err := h.WaitForContent("I-search: apple", 3*time.Second); err != nil {
			t.Fatalf("search prompt not shown: %v", err)
		}

		// First match at (0, 0)
		row1, col1, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// C-s again to go to next match
		h.SendKeys("C-s")
		time.Sleep(300 * time.Millisecond)

		row2, col2, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// Cursor should have moved to a different position
		if row1 == row2 && col1 == col2 {
			t.Errorf("C-s C-s did not advance to next match: still at (%d, %d)", row1, col1)
		}
		// Second match should be on row 1
		if row2 != 1 || col2 != 0 {
			t.Errorf("second match: expected (1, 0), got (%d, %d)", row2, col2)
		}
	})

	t.Run("BackwardSearch", func(t *testing.T) {
		content := "apple banana cherry\napple grape orange\napple mango peach\n"
		path := createTestFile(t, "searchback.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("apple banana", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Move to end of buffer first
		h.SendKeys("Escape")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys(">")
		time.Sleep(300 * time.Millisecond)

		// Start backward search with C-r
		h.SendKeys("C-r")
		time.Sleep(200 * time.Millisecond)

		h.SendKeys("grape")
		if err := h.WaitForContent("I-search backward: grape", 3*time.Second); err != nil {
			t.Fatalf("backward search prompt not shown: %v", err)
		}
		h.AssertMessageLine(t, "I-search backward: grape")
	})

	t.Run("SearchAccept", func(t *testing.T) {
		content := "apple banana cherry\napple grape orange\napple mango peach\n"
		path := createTestFile(t, "searchaccept.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("apple banana", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Search for "grape"
		h.SendKeys("C-s")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("grape")
		if err := h.WaitForContent("I-search: grape", 3*time.Second); err != nil {
			t.Fatalf("search prompt not shown: %v", err)
		}

		// Press Enter to accept
		h.SendKeys("Enter")
		time.Sleep(200 * time.Millisecond)

		// Cursor should stay at the match position
		row, col, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		if row != 1 || col != 6 {
			t.Errorf("cursor after accept: expected (1, 6), got (%d, %d)", row, col)
		}

		// Message line should be clear (no I-search prompt)
		lines := h.Capture()
		msgLine := strings.TrimSpace(lines[h.height-1])
		if strings.Contains(msgLine, "I-search") {
			t.Errorf("message line should be clear after accept, got %q", msgLine)
		}
	})

	t.Run("SearchCancel", func(t *testing.T) {
		content := "apple banana cherry\napple grape orange\napple mango peach\n"
		path := createTestFile(t, "searchcancel.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("apple banana", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Record original cursor position
		origRow, origCol, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// Search for "grape" (moves cursor to row 1)
		h.SendKeys("C-s")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("grape")
		if err := h.WaitForContent("I-search: grape", 3*time.Second); err != nil {
			t.Fatalf("search prompt not shown: %v", err)
		}

		// Cancel with C-g
		h.SendKeys("C-g")
		time.Sleep(200 * time.Millisecond)

		// Cursor should return to original position
		row, col, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		if row != origRow || col != origCol {
			t.Errorf("cursor after cancel: expected (%d, %d), got (%d, %d)", origRow, origCol, row, col)
		}
	})

	t.Run("SearchBackspace", func(t *testing.T) {
		content := "apple banana cherry\napple grape orange\napple mango peach\n"
		path := createTestFile(t, "searchbs.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("apple banana", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Search for "ban"
		h.SendKeys("C-s")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("ban")
		if err := h.WaitForContent("I-search: ban", 3*time.Second); err != nil {
			t.Fatalf("search prompt not shown: %v", err)
		}

		// Backspace removes last char
		h.SendKeys("BSpace")
		time.Sleep(200 * time.Millisecond)

		// Message line should show shorter query
		h.AssertMessageLine(t, "I-search: ba")
	})
}
