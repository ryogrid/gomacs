package e2e

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Harness manages a tmux session for E2E testing of goomacs.
type Harness struct {
	session string
	binary  string
	width   int
	height  int
}

// Start creates a detached tmux session running the given binary with args.
func Start(t *testing.T, binary string, args ...string) *Harness {
	t.Helper()

	session := fmt.Sprintf("%s-%d", sanitizeSessionName(t.Name()), rand.Intn(100000))

	h := &Harness{
		session: session,
		binary:  binary,
		width:   80,
		height:  24,
	}

	// Build tmux command: tmux new-session -d -s <session> -x 80 -y 24 <binary> <args>
	tmuxArgs := []string{
		"new-session", "-d",
		"-s", session,
		"-x", fmt.Sprintf("%d", h.width),
		"-y", fmt.Sprintf("%d", h.height),
		binary,
	}
	tmuxArgs = append(tmuxArgs, args...)

	cmd := exec.Command("tmux", tmuxArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to start tmux session: %v\n%s", err, out)
	}

	t.Cleanup(func() {
		h.Close()
	})

	// Let goomacs initialize
	time.Sleep(200 * time.Millisecond)

	return h
}

// SendKeys sends keys to the tmux session.
func (h *Harness) SendKeys(keys string) {
	cmd := exec.Command("tmux", "send-keys", "-t", h.session, keys)
	cmd.CombinedOutput() //nolint: errcheck
}

// Capture captures the tmux pane and returns each row as a string slice.
func (h *Harness) Capture() []string {
	cmd := exec.Command("tmux", "capture-pane", "-t", h.session, "-p")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(string(out), "\n")
	// tmux capture-pane -p outputs exactly height lines (plus possible trailing newline)
	// Trim to height
	if len(lines) > h.height {
		lines = lines[:h.height]
	}
	return lines
}

// CaptureWithEscapes captures the tmux pane preserving ANSI escape sequences.
func (h *Harness) CaptureWithEscapes() []string {
	cmd := exec.Command("tmux", "capture-pane", "-e", "-t", h.session, "-p")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > h.height {
		lines = lines[:h.height]
	}
	return lines
}

// CapturePane returns the full screen as a single string (rows joined by newlines).
func (h *Harness) CapturePane() string {
	lines := h.Capture()
	return strings.Join(lines, "\n")
}

// Close kills the tmux session.
func (h *Harness) Close() {
	exec.Command("tmux", "kill-session", "-t", h.session).Run() //nolint: errcheck
}

// WaitForContent polls Capture() every 100ms until substr appears or timeout expires.
func (h *Harness) WaitForContent(substr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		screen := h.CapturePane()
		if strings.Contains(screen, substr) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %q after %v", substr, timeout)
}

// sanitizeSessionName replaces characters not allowed in tmux session names.
func sanitizeSessionName(name string) string {
	replacer := strings.NewReplacer("/", "-", ".", "-", ":", "-")
	return replacer.Replace(name)
}
