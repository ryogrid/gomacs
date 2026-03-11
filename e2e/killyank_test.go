package e2e

import (
	"strings"
	"testing"
	"time"
)

func TestKillYankUndo(t *testing.T) {
	t.Run("KillLine", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("hello world")
		if err := h.WaitForContent("hello world", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Move to beginning, kill entire line
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-k")
		time.Sleep(200 * time.Millisecond)

		// Line should be empty
		lines := h.Capture()
		line0 := strings.TrimRight(lines[0], " ")
		if line0 != "" {
			t.Errorf("line 0 should be empty after C-k, got %q", line0)
		}
	})

	t.Run("Yank", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("hello world")
		if err := h.WaitForContent("hello world", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Kill the line
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-k")
		time.Sleep(200 * time.Millisecond)

		// Move to a new line and yank
		h.SendKeys("Enter")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-y")
		time.Sleep(200 * time.Millisecond)

		// Verify killed text was yanked
		if err := h.WaitForContent("hello world", 3*time.Second); err != nil {
			t.Fatalf("yanked text not visible: %v", err)
		}
		h.AssertLineContains(t, 1, "hello world")
	})

	t.Run("KillRegion", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("abcdef")
		if err := h.WaitForContent("abcdef", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Go to beginning, set mark, move forward 3 chars, kill region
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-Space")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-w")
		time.Sleep(200 * time.Millisecond)

		// First 3 chars should be removed
		if err := h.WaitForContent("def", 3*time.Second); err != nil {
			t.Fatalf("kill region did not work: %v", err)
		}
		lines := h.Capture()
		line0 := strings.TrimRight(lines[0], " ")
		if strings.Contains(line0, "abc") {
			t.Errorf("line 0 should not contain 'abc' after C-w, got %q", line0)
		}
		if !strings.Contains(line0, "def") {
			t.Errorf("line 0 should contain 'def' after C-w, got %q", line0)
		}
	})

	t.Run("CopyRegion", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("abcdef")
		if err := h.WaitForContent("abcdef", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Go to beginning, set mark, move forward 3 chars, copy region (M-w)
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-Space")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("Escape")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("w")
		time.Sleep(200 * time.Millisecond)

		// Original text should still be intact
		lines := h.Capture()
		line0 := strings.TrimRight(lines[0], " ")
		if !strings.Contains(line0, "abcdef") {
			t.Errorf("M-w should not delete text, got %q", line0)
		}

		// Move to end, add newline, yank — should paste "abc"
		h.SendKeys("C-e")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("Enter")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-y")
		time.Sleep(200 * time.Millisecond)

		if err := h.WaitForContent("abc", 3*time.Second); err != nil {
			t.Fatalf("yanked text not visible: %v", err)
		}
		// Line 1 should have the copied "abc"
		h.AssertLineContains(t, 1, "abc")
		// Line 0 should still have the original text
		h.AssertLineContains(t, 0, "abcdef")
	})

	t.Run("Undo", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("abc")
		if err := h.WaitForContent("abc", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Undo with C-_
		h.SendKeys("C-_")
		time.Sleep(200 * time.Millisecond)

		if err := h.WaitForContent("Undo", 3*time.Second); err != nil {
			t.Fatalf("undo message not shown: %v", err)
		}

		// Verify last edit was reversed — check that "abc" is no longer on line 0
		lines := h.Capture()
		line0 := strings.TrimRight(lines[0], " ")
		if strings.Contains(line0, "abc") {
			t.Errorf("line 0 should not contain 'abc' after undo, got %q", line0)
		}
	})

	t.Run("ConsecutiveKillAccumulation", func(t *testing.T) {
		content := "first line\nsecond line\nthird line\n"
		path := createTestFile(t, "killaccum.txt", content)
		h := newHarness(t, path)

		if err := h.WaitForContent("first line", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Move to beginning of first line
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)

		// C-k kills "first line" text, C-k again kills the newline (joining lines)
		h.SendKeys("C-k")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("C-k")
		time.Sleep(200 * time.Millisecond)

		// Now cursor should be at beginning of what was "second line"
		// Move down to third line
		h.SendKeys("C-n")
		time.Sleep(100 * time.Millisecond)

		// Yank should paste both the killed text and the newline
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-y")
		time.Sleep(300 * time.Millisecond)

		// The yanked content should include "first line" followed by a newline
		if err := h.WaitForContent("first line", 3*time.Second); err != nil {
			t.Fatalf("yanked text not visible: %v", err)
		}

		// Verify the screen shows the accumulated kill was yanked
		screen := h.CapturePane()
		if !strings.Contains(screen, "first line") {
			t.Errorf("expected yanked text to contain 'first line', screen:\n%s", screen)
		}
	})
}
