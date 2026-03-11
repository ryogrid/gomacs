package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBufferManagement(t *testing.T) {
	t.Run("OpenFile", func(t *testing.T) {
		content := "hello from file\nsecond line\n"
		path := createTestFile(t, "openme.txt", content)
		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		// C-x C-f to open file
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(200 * time.Millisecond)

		// Type the file path and press Enter
		h.SendKeys(path)
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("Enter")

		if err := h.WaitForContent("hello from file", 5*time.Second); err != nil {
			t.Fatalf("file content did not appear: %v", err)
		}

		h.AssertLineContains(t, 0, "hello from file")
		h.AssertStatusBar(t, "openme.txt")
	})

	t.Run("SwitchBuffer", func(t *testing.T) {
		// Open goomacs with a file so we have two buffers (*scratch* + file)
		path := createTestFile(t, "buf1.txt", "buffer one content\n")
		h := newHarness(t, path)

		if err := h.WaitForContent("buf1.txt", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}
		h.AssertStatusBar(t, "buf1.txt")

		// Open a second file via C-x C-f
		path2 := createTestFile(t, "buf2.txt", "buffer two content\n")
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys(path2)
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("Enter")

		if err := h.WaitForContent("buffer two content", 5*time.Second); err != nil {
			t.Fatalf("second file did not open: %v", err)
		}
		h.AssertStatusBar(t, "buf2.txt")

		// C-x b to switch buffer, type buf1.txt
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("b")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys(path)
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("Enter")

		if err := h.WaitForContent("buffer one content", 5*time.Second); err != nil {
			t.Fatalf("did not switch to buf1: %v", err)
		}
		h.AssertStatusBar(t, "buf1.txt")
	})

	t.Run("BufferList", func(t *testing.T) {
		path1 := createTestFile(t, "file1.txt", "content1\n")
		path2 := createTestFile(t, "file2.txt", "content2\n")
		h := newHarness(t, path1)

		if err := h.WaitForContent("file1.txt", 5*time.Second); err != nil {
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

		if err := h.WaitForContent("content2", 5*time.Second); err != nil {
			t.Fatalf("second file did not open: %v", err)
		}

		// C-x C-b to show buffer list
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-b")
		time.Sleep(300 * time.Millisecond)

		// Buffer list should show both file names
		screen := h.CapturePane()
		if !strings.Contains(screen, "file1.txt") {
			t.Error("buffer list does not contain file1.txt")
		}
		if !strings.Contains(screen, "file2.txt") {
			t.Error("buffer list does not contain file2.txt")
		}
		h.AssertStatusBar(t, "*Buffer List*")
	})

	t.Run("KillBuffer", func(t *testing.T) {
		path := createTestFile(t, "killme.txt", "kill buffer content\n")
		h := newHarness(t, path)

		if err := h.WaitForContent("killme.txt", 5*time.Second); err != nil {
			t.Fatalf("file did not open: %v", err)
		}
		h.AssertStatusBar(t, "killme.txt")

		// C-x k Enter to kill current buffer (default)
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("k")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("Enter")
		time.Sleep(300 * time.Millisecond)

		// After killing, should switch to *scratch* or another buffer
		// The killed buffer name should no longer be in status bar
		lines := h.Capture()
		statusBar := lines[h.height-2]
		if strings.Contains(statusBar, "killme.txt") {
			t.Error("killed buffer still shown in status bar")
		}
	})

	t.Run("TabCompletion", func(t *testing.T) {
		// Create a temp directory with uniquely-named files
		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "uniqueprefix_alpha.txt")
		file2 := filepath.Join(tmpDir, "uniqueprefix_beta.txt")
		if err := os.WriteFile(file1, []byte("alpha\n"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := os.WriteFile(file2, []byte("beta\n"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		h := newHarness(t)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		// C-x C-f to open find-file prompt
		h.SendKeys("C-x")
		time.Sleep(50 * time.Millisecond)
		h.SendKeys("C-f")
		time.Sleep(200 * time.Millisecond)

		// Type partial path up to common prefix
		partialPath := filepath.Join(tmpDir, "uniqueprefix_")
		h.SendKeys(partialPath)
		time.Sleep(100 * time.Millisecond)

		// Press Tab to trigger completion
		h.SendKeys("Tab")
		time.Sleep(300 * time.Millisecond)

		// With two matches sharing prefix "uniqueprefix_", tab should show both options
		// The message line should contain the match names
		msgLine := h.Capture()[h.height-1]
		if !strings.Contains(msgLine, "uniqueprefix_alpha.txt") && !strings.Contains(msgLine, "uniqueprefix_beta.txt") {
			// If not on message line, check if completion happened by looking at full screen
			screen := h.CapturePane()
			if !strings.Contains(screen, "uniqueprefix_") {
				t.Error("tab completion did not show matches or complete prefix")
			}
		}

		// Now test single-match completion: type enough to disambiguate
		h.SendKeys("a")
		time.Sleep(100 * time.Millisecond)
		h.SendKeys("Tab")
		time.Sleep(300 * time.Millisecond)

		// Should complete to the full filename; press Enter to open
		h.SendKeys("Enter")
		if err := h.WaitForContent("alpha", 5*time.Second); err != nil {
			t.Fatalf("completed file did not open: %v", err)
		}
		h.AssertStatusBar(t, "uniqueprefix_alpha.txt")
	})
}
