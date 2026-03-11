package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestBasicEditing(t *testing.T) {
	t.Run("ScratchBuffer", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}
		h.AssertStatusBar(t, "*scratch*")
	})

	t.Run("TextInsertion", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("Hello, World!")
		if err := h.WaitForContent("Hello, World!", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}
		h.AssertLineContains(t, 0, "Hello, World!")
	})

	t.Run("Backspace", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("abc")
		if err := h.WaitForContent("abc", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		h.SendKeys("BSpace")
		time.Sleep(200 * time.Millisecond)

		// Verify 'c' was deleted — line should contain "ab" but not "abc"
		lines := h.Capture()
		line0 := strings.TrimRight(lines[0], " ")
		if !strings.Contains(line0, "ab") {
			t.Errorf("line 0: expected to contain %q, got %q", "ab", line0)
		}
		if strings.Contains(line0, "abc") {
			t.Errorf("line 0: expected 'abc' to be gone after backspace, got %q", line0)
		}
	})

	t.Run("DeleteForward", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("abc")
		if err := h.WaitForContent("abc", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Move to beginning, delete first char
		h.SendKeys("C-a")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("C-d")
		time.Sleep(200 * time.Millisecond)

		if err := h.WaitForContent("bc", 3*time.Second); err != nil {
			t.Fatalf("C-d did not delete: %v", err)
		}
		h.AssertLineContains(t, 0, "bc")
	})

	t.Run("Newline", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		h.SendKeys("line1")
		if err := h.WaitForContent("line1", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		h.SendKeys("Enter")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("line2")
		if err := h.WaitForContent("line2", 3*time.Second); err != nil {
			t.Fatalf("line2 not inserted: %v", err)
		}

		h.AssertLineContains(t, 0, "line1")
		h.AssertLineContains(t, 1, "line2")
	})

	t.Run("Save", func(t *testing.T) {
		tmpFile := createTestFile(t, "saveme.txt", "")
		h := newHarness(t, tmpFile)

		if err := h.WaitForContent("saveme.txt", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}

		h.SendKeys("saved content")
		if err := h.WaitForContent("saved content", 3*time.Second); err != nil {
			t.Fatalf("text not inserted: %v", err)
		}

		// Save with C-x C-s
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-s")
		time.Sleep(500 * time.Millisecond)

		// Verify file content on disk
		data, err := os.ReadFile(tmpFile)
		if err != nil {
			t.Fatalf("failed to read saved file: %v", err)
		}
		if !strings.Contains(string(data), "saved content") {
			t.Errorf("file content: expected to contain %q, got %q", "saved content", string(data))
		}
	})

	t.Run("Quit", func(t *testing.T) {
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		// Quit with C-x C-c
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-c")
		time.Sleep(500 * time.Millisecond)

		// Verify session is gone
		check := exec.Command("tmux", "has-session", "-t", h.session)
		if err := check.Run(); err == nil {
			t.Error("expected tmux session to be gone after C-x C-c, but it still exists")
		}
	})
}
