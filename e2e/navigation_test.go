package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNavigation(t *testing.T) {
	t.Run("ForwardBackward", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("abc")
		if err := h.WaitForContent("abc", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Cursor should be at col 3 after typing "abc"
		h.AssertCursorAt(t, 0, 3)

		// C-b moves back one
		h.SendKeys("C-b")
		time.Sleep(100 * time.Millisecond)
		h.AssertCursorAt(t, 0, 2)

		// C-f moves forward one
		h.SendKeys("C-f")
		time.Sleep(100 * time.Millisecond)
		h.AssertCursorAt(t, 0, 3)
	})

	t.Run("DownUp", func(t *testing.T) {
		// Create a file with multiple lines
		content := "line one\nline two\nline three\n"
		path := createTestFile(t, "multiline.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("line one", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Cursor starts at row 0
		h.AssertCursorAt(t, 0, 0)

		// C-n moves down
		h.SendKeys("C-n")
		time.Sleep(100 * time.Millisecond)
		h.AssertCursorAt(t, 1, 0)

		// Another C-n
		h.SendKeys("C-n")
		time.Sleep(100 * time.Millisecond)
		h.AssertCursorAt(t, 2, 0)

		// C-p moves back up
		h.SendKeys("C-p")
		time.Sleep(100 * time.Millisecond)
		h.AssertCursorAt(t, 1, 0)
	})

	t.Run("BeginningEndOfLine", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("hello world")
		if err := h.WaitForContent("hello world", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Cursor at end (col 11)
		h.AssertCursorAt(t, 0, 11)

		// C-a goes to beginning
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)
		h.AssertCursorAt(t, 0, 0)

		// C-e goes to end
		h.SendKeys("C-e")
		time.Sleep(100 * time.Millisecond)
		h.AssertCursorAt(t, 0, 11)
	})

	t.Run("PageDownUp", func(t *testing.T) {
		// Create a file with 50+ lines
		var lines []string
		for i := 1; i <= 60; i++ {
			lines = append(lines, fmt.Sprintf("Line %d", i))
		}
		path := createTestFile(t, "longfile.txt", strings.Join(lines, "\n"))
		h := newHarness(t, path)

		if err := h.WaitForContent("Line 1", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Should see Line 1 initially
		h.AssertScreenContains(t, "Line 1")

		// C-v scrolls down a page
		h.SendKeys("C-v")
		time.Sleep(300 * time.Millisecond)

		// After scrolling down, Line 1 should no longer be visible
		// and later lines should be visible
		screen := h.CapturePane()
		if strings.Contains(screen, "Line 1\n") || strings.HasPrefix(strings.TrimSpace(screen), "Line 1") {
			// Check that we actually scrolled — Line 1 shouldn't be on the first content line
			lines := h.Capture()
			line0 := strings.TrimSpace(lines[0])
			if line0 == "Line 1" {
				t.Error("C-v did not scroll down: Line 1 still at top")
			}
		}

		// M-v scrolls back up
		h.SendKeys("Escape")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("v")
		time.Sleep(300 * time.Millisecond)

		// Line 1 should be visible again
		h.AssertScreenContains(t, "Line 1")
	})

	t.Run("BeginningEndOfBuffer", func(t *testing.T) {
		// Create a file with many lines
		var lines []string
		for i := 1; i <= 40; i++ {
			lines = append(lines, fmt.Sprintf("Line %d", i))
		}
		path := createTestFile(t, "buffer_nav.txt", strings.Join(lines, "\n"))
		h := newHarness(t, path)

		if err := h.WaitForContent("Line 1", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// M-> (Escape >) jumps to end of buffer
		h.SendKeys("Escape")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys(">")
		time.Sleep(300 * time.Millisecond)

		// Should see Line 40 on screen and status bar should show last line
		h.AssertScreenContains(t, "Line 40")

		// M-< (Escape <) jumps to beginning
		h.SendKeys("Escape")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("<")
		time.Sleep(300 * time.Millisecond)

		// Should be back at the top
		h.AssertCursorAt(t, 0, 0)
		h.AssertScreenContains(t, "Line 1")
	})

	t.Run("GotoLine", func(t *testing.T) {
		// Create a file with 20+ lines
		var lines []string
		for i := 1; i <= 25; i++ {
			lines = append(lines, fmt.Sprintf("Line %d", i))
		}
		path := createTestFile(t, "gotoline.txt", strings.Join(lines, "\n"))
		h := newHarness(t, path)

		if err := h.WaitForContent("Line 1", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// C-l opens goto-line prompt
		h.SendKeys("C-l")
		time.Sleep(200 * time.Millisecond)

		// Type line number 10 and press Enter
		h.SendKeys("10")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("Enter")
		time.Sleep(300 * time.Millisecond)

		// Status bar should show Line 10
		h.AssertStatusBar(t, "Line 10/")
	})
}
