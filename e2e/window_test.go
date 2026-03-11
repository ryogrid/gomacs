package e2e

import (
	"strings"
	"testing"
	"time"
)

func TestWindowSplitting(t *testing.T) {
	t.Run("VerticalSplit", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		// C-x 2 to split vertically (top/bottom)
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("2")
		time.Sleep(300 * time.Millisecond)

		// Verify two status bars visible by scanning for lines filled with dashes.
		// Active window status bar uses reverse video (spaces), inactive uses '-' fill.
		// After split we should see the inactive window's status bar as a line of dashes.
		lines := h.Capture()
		statusBarCount := 0
		for i := 0; i < h.height-1; i++ { // exclude message line
			line := lines[i]
			// A status bar line contains the buffer name and is mostly filled with dashes or spaces.
			if strings.Contains(line, "*scratch*") && (countChar(line, '-') > 10 || countChar(line, ' ') > 40) {
				statusBarCount++
			}
		}
		if statusBarCount < 2 {
			t.Errorf("expected 2 status bars after vertical split, found %d\nScreen:\n%s",
				statusBarCount, h.CapturePane())
		}
	})

	t.Run("HorizontalSplit", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		// C-x 3 to split horizontally (side-by-side)
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("3")
		time.Sleep(300 * time.Millisecond)

		// Verify vertical separator '│' (U+2502) appears in capture
		screen := h.CapturePane()
		if !strings.Contains(screen, "│") {
			t.Errorf("expected vertical separator '│' after horizontal split\nScreen:\n%s", screen)
		}
	})

	t.Run("SwitchWindow", func(t *testing.T) {
		path1 := createTestFile(t, "win1.txt", "window one content\n")
		path2 := createTestFile(t, "win2.txt", "window two content\n")

		h := newHarness(t, path1)

		if err := h.WaitForContent("win1.txt", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		// Open second file
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys(path2)
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("Enter")

		if err := h.WaitForContent("window two content", 5*time.Second); err != nil {
			t.Fatalf("second file did not open: %v", err)
		}

		// Split vertically
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("2")
		time.Sleep(300 * time.Millisecond)

		// Both windows show win2.txt initially. Switch to first file in top window.
		// Get initial screen to identify status bars
		linesBefore := h.Capture()

		// C-x o to switch to next window
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("o")
		time.Sleep(300 * time.Millisecond)

		linesAfter := h.Capture()

		// After switching, the active/inactive markers should swap.
		// Active window status bar is filled with spaces (reverse video),
		// inactive is filled with dashes.
		// Check that the screen changed (status bar rendering changed).
		beforeScreen := strings.Join(linesBefore, "\n")
		afterScreen := strings.Join(linesAfter, "\n")
		if beforeScreen == afterScreen {
			t.Error("screen did not change after C-x o; expected active window indicator to swap")
		}
	})

	t.Run("CloseWindow", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		// Split vertically
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("2")
		time.Sleep(300 * time.Millisecond)

		// Verify we have two status bars
		lines := h.Capture()
		statusBefore := countStatusBars(lines, h.height)
		if statusBefore < 2 {
			t.Fatalf("expected 2 status bars after split, got %d", statusBefore)
		}

		// C-x 0 to close current window
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("0")
		time.Sleep(300 * time.Millisecond)

		// Verify only one status bar remains
		lines = h.Capture()
		statusAfter := countStatusBars(lines, h.height)
		if statusAfter != 1 {
			t.Errorf("expected 1 status bar after C-x 0, got %d\nScreen:\n%s",
				statusAfter, h.CapturePane())
		}
	})

	t.Run("CloseOtherWindows", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		// Split twice to create 3 windows
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("2")
		time.Sleep(300 * time.Millisecond)

		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("2")
		time.Sleep(300 * time.Millisecond)

		// Verify we have multiple status bars
		lines := h.Capture()
		statusBefore := countStatusBars(lines, h.height)
		if statusBefore < 2 {
			t.Fatalf("expected multiple status bars after splits, got %d", statusBefore)
		}

		// C-x 1 to close all other windows
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("1")
		time.Sleep(300 * time.Millisecond)

		// Verify only one status bar remains
		lines = h.Capture()
		statusAfter := countStatusBars(lines, h.height)
		if statusAfter != 1 {
			t.Errorf("expected 1 status bar after C-x 1, got %d\nScreen:\n%s",
				statusAfter, h.CapturePane())
		}
	})
}

// countChar returns the number of occurrences of ch in s.
func countChar(s string, ch byte) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			count++
		}
	}
	return count
}

// countStatusBars counts lines that look like status bars (contain *scratch* or a filename
// and are filled with dashes or spaces).
func countStatusBars(lines []string, height int) int {
	count := 0
	for i := 0; i < height-1; i++ { // exclude message line
		line := lines[i]
		// A status bar contains "Line " (from the position indicator) and
		// is mostly filled with dashes or spaces.
		if strings.Contains(line, "Line ") && (countChar(line, '-') > 10 || countChar(line, ' ') > 40) {
			count++
		}
	}
	return count
}
