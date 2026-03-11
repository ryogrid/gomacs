package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// gomacsPath holds the path to the built goomacs binary.
var gomacsPath string

func TestMain(m *testing.M) {
	// Check for tmux in PATH
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Println("skipping e2e tests: tmux not found")
		os.Exit(0)
	}

	// Create temp directory for the binary
	tmpDir, err := os.MkdirTemp("", "goomacs-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Build goomacs binary from repo root (parent of e2e/)
	gomacsPath = filepath.Join(tmpDir, "goomacs")
	cmd := exec.Command("go", "build", "-o", gomacsPath, "..")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build goomacs: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// newHarness creates a Harness with the built binary path.
func newHarness(t *testing.T, args ...string) *Harness {
	t.Helper()
	return Start(t, gomacsPath, args...)
}

// createTestFile writes content to a temp file and returns its path.
func createTestFile(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	return path
}

func TestSmokeTest(t *testing.T) {
	h := newHarness(t)

	// Wait for *scratch* buffer to appear in status bar
	if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
		t.Fatalf("goomacs did not start: %v", err)
	}
	h.AssertStatusBar(t, "*scratch*")

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
}
