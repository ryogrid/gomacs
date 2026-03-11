package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// AssertLineContains checks that the given row contains substr.
func (h *Harness) AssertLineContains(t *testing.T, row int, substr string) {
	t.Helper()
	lines := h.Capture()
	if row < 0 || row >= len(lines) {
		t.Fatalf("line %d: row out of range (screen has %d lines)", row, len(lines))
	}
	if !strings.Contains(lines[row], substr) {
		t.Errorf("line %d: expected to contain %q, got %q", row, substr, lines[row])
	}
}

// AssertLineEquals checks that the given row exactly matches expected (trailing whitespace trimmed).
func (h *Harness) AssertLineEquals(t *testing.T, row int, expected string) {
	t.Helper()
	lines := h.Capture()
	if row < 0 || row >= len(lines) {
		t.Fatalf("line %d: row out of range (screen has %d lines)", row, len(lines))
	}
	got := strings.TrimRight(lines[row], " ")
	if got != expected {
		t.Errorf("line %d: expected %q, got %q", row, expected, got)
	}
}

// AssertStatusBar checks that the status bar (second-to-last row) contains substr.
func (h *Harness) AssertStatusBar(t *testing.T, substr string) {
	t.Helper()
	h.AssertLineContains(t, h.height-2, substr)
}

// AssertMessageLine checks that the message line (last row) contains substr.
func (h *Harness) AssertMessageLine(t *testing.T, substr string) {
	t.Helper()
	h.AssertLineContains(t, h.height-1, substr)
}

// AssertScreenContains checks that substr appears anywhere in the full screen capture.
func (h *Harness) AssertScreenContains(t *testing.T, substr string) {
	t.Helper()
	screen := h.CapturePane()
	if !strings.Contains(screen, substr) {
		t.Errorf("screen: expected to contain %q, got:\n%s", substr, screen)
	}
}

// AssertScreenSnapshot compares the full screen against a golden file.
// If UPDATE_GOLDEN=1 or the golden file doesn't exist, it writes the current screen as the golden file.
func (h *Harness) AssertScreenSnapshot(t *testing.T, name string) {
	t.Helper()
	screen := h.CapturePane()
	goldenPath := filepath.Join("testdata", name+".golden")

	// Write golden file if UPDATE_GOLDEN=1 or file doesn't exist
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("failed to create testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(screen), 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Write golden file if it doesn't exist
			if err := os.MkdirAll("testdata", 0o755); err != nil {
				t.Fatalf("failed to create testdata dir: %v", err)
			}
			if err := os.WriteFile(goldenPath, []byte(screen), 0o644); err != nil {
				t.Fatalf("failed to write golden file: %v", err)
			}
			return
		}
		t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
	}

	if screen != string(golden) {
		t.Errorf("screen snapshot %q does not match golden file.\nGot:\n%s\nExpected:\n%s", name, screen, string(golden))
	}
}

// AssertCursorAt verifies the cursor is at the given row and col using tmux display-message.
func (h *Harness) AssertCursorAt(t *testing.T, row, col int) {
	t.Helper()
	cmd := exec.Command("tmux", "display-message", "-t", h.session, "-p", "#{cursor_x} #{cursor_y}")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get cursor position: %v", err)
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		t.Fatalf("unexpected cursor position output: %q", string(out))
	}
	x, errX := strconv.Atoi(parts[0])
	y, errY := strconv.Atoi(parts[1])
	if errX != nil || errY != nil {
		t.Fatalf("failed to parse cursor position: %q", string(out))
	}
	if x != col || y != row {
		t.Errorf("cursor: expected (%d, %d), got (%d, %d)", row, col, y, x)
	}
}

// CursorPosition returns the current cursor (row, col) from tmux.
func (h *Harness) CursorPosition() (row, col int, err error) {
	cmd := exec.Command("tmux", "display-message", "-t", h.session, "-p", "#{cursor_x} #{cursor_y}")
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get cursor position: %w", err)
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected cursor position output: %q", string(out))
	}
	col, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse cursor_x: %w", err)
	}
	row, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse cursor_y: %w", err)
	}
	return row, col, nil
}
