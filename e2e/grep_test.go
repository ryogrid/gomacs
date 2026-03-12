package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupGrepFixtures creates a temp directory with test files for grep tests.
// Returns the directory path.
func setupGrepFixtures(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create nested directories with files containing known content
	writeFixture(t, dir, "hello.go", "package main\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}\n")
	writeFixture(t, dir, "greet.go", "package main\n\nfunc greet() string {\n\treturn \"hello friend\"\n}\n")

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}
	writeFixture(t, dir, "sub/utils.go", "package sub\n\n// hello is a utility function\nfunc hello() {}\n")

	return dir
}

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", name, err)
	}
}

// invokeFindGrep opens M-x, types find-grep, presses Enter, then replaces
// the default command with the given cmd and submits it.
func invokeFindGrep(t *testing.T, h *Harness, cmd string) {
	t.Helper()
	// M-x
	h.SendKeys("Escape")
	time.Sleep(50 * time.Millisecond)
	h.SendKeys("x")
	time.Sleep(200 * time.Millisecond)

	// Type find-grep and submit
	h.SendKeys("find-grep")
	time.Sleep(100 * time.Millisecond)
	h.SendKeys("Enter")
	time.Sleep(200 * time.Millisecond)

	// Clear the default command by sending enough backspaces in one tmux call
	// Default: "find . -type f -exec grep -nH -e '' {} +" (45 chars)
	h.SendKeysRepeat("BSpace", 50)
	time.Sleep(200 * time.Millisecond)

	// Type the custom command
	h.SendKeys(cmd)
	time.Sleep(100 * time.Millisecond)

	// Submit
	h.SendKeys("Enter")
}

// setupGrepFixtures3Files creates a temp directory with 3 files (a.txt, b.txt, c.txt)
// each containing the word "marker" on known lines, for predictable grep result ordering.
func setupGrepFixtures3Files(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "a.txt", "line1\nmarker alpha\nline3\n")
	writeFixture(t, dir, "b.txt", "marker bravo\nline2\nmarker bravo2\n")
	writeFixture(t, dir, "c.txt", "line1\nline2\nmarker charlie\n")
	return dir
}

func TestFindGrep(t *testing.T) {
	t.Run("Invoke", func(t *testing.T) {
		dir := setupGrepFixtures(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH hello .")

		// Wait for *grep* buffer to appear
		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// Verify results contain expected filepath:linenum:text format
		h.AssertStatusBar(t, "*grep*")
		h.AssertScreenContains(t, "hello")
	})

	t.Run("RET_JumpToSource", func(t *testing.T) {
		dir := setupGrepFixtures(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH hello ./hello.go")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// Press Enter on the first result to jump to source
		h.SendKeys("Enter")
		time.Sleep(500 * time.Millisecond)

		// Verify we jumped to the source file
		h.AssertStatusBar(t, "hello.go")
	})

	t.Run("Navigation_NP", func(t *testing.T) {
		dir := setupGrepFixtures(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH hello .")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// Get initial cursor position
		row1, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// Press n to go to next result
		h.SendKeys("n")
		time.Sleep(200 * time.Millisecond)

		row2, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		if row2 <= row1 {
			t.Errorf("n did not advance cursor: was row %d, now row %d", row1, row2)
		}

		// Press p to go back
		h.SendKeys("p")
		time.Sleep(200 * time.Millisecond)

		row3, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		if row3 != row1 {
			t.Errorf("p did not return cursor: expected row %d, got row %d", row1, row3)
		}
	})

	t.Run("Quit", func(t *testing.T) {
		dir := setupGrepFixtures(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH hello .")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// Press q to close the grep buffer
		h.SendKeys("q")
		time.Sleep(300 * time.Millisecond)

		// Should be back to *scratch*
		h.AssertStatusBar(t, "*scratch*")
	})

	t.Run("NoMatches", func(t *testing.T) {
		dir := setupGrepFixtures(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH zzzznonexistent .")

		// Wait for the "No matches found" message
		if err := h.WaitForContent("No matches found", 10*time.Second); err != nil {
			t.Fatalf("no-match message did not appear: %v", err)
		}

		// Should still be on *scratch*
		h.AssertStatusBar(t, "*scratch*")
	})

	t.Run("MalformedCommand", func(t *testing.T) {
		dir := setupGrepFixtures(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "nonexistent_command_xyz123")

		// Wait for an error message (stderr output)
		if err := h.WaitForContent("not found", 10*time.Second); err != nil {
			t.Fatalf("error message did not appear: %v", err)
		}

		// Should still be on *scratch*
		h.AssertStatusBar(t, "*scratch*")
	})

	t.Run("FileNav_MN", func(t *testing.T) {
		dir := setupGrepFixtures3Files(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH marker . | sort")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// Cursor should be on first result (a.txt)
		row1, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		h.AssertLineContains(t, row1, "a.txt")

		// M-n to jump to next file (b.txt)
		h.SendKeys("M-n")
		time.Sleep(300 * time.Millisecond)

		row2, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		h.AssertLineContains(t, row2, "b.txt")

		// M-n again to jump to c.txt
		h.SendKeys("M-n")
		time.Sleep(300 * time.Millisecond)

		row3, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		h.AssertLineContains(t, row3, "c.txt")
	})

	t.Run("FileNav_MP", func(t *testing.T) {
		dir := setupGrepFixtures3Files(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH marker . | sort")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// M-n twice to get to c.txt
		h.SendKeys("M-n")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("M-n")
		time.Sleep(300 * time.Millisecond)

		row1, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		h.AssertLineContains(t, row1, "c.txt")

		// M-p to jump back to b.txt
		h.SendKeys("M-p")
		time.Sleep(300 * time.Millisecond)

		row2, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		h.AssertLineContains(t, row2, "b.txt")

		// M-p to jump back to a.txt
		h.SendKeys("M-p")
		time.Sleep(300 * time.Millisecond)

		row3, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		h.AssertLineContains(t, row3, "a.txt")
	})

	t.Run("FileNav_MN_AtLastFile", func(t *testing.T) {
		dir := setupGrepFixtures3Files(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH marker . | sort")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// M-n twice to get to last file (c.txt)
		h.SendKeys("M-n")
		time.Sleep(200 * time.Millisecond)
		h.SendKeys("M-n")
		time.Sleep(300 * time.Millisecond)

		row1, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// M-n at last file should show "No more files"
		h.SendKeys("M-n")

		if err := h.WaitForContent("No more files", 5*time.Second); err != nil {
			t.Fatalf("No more files message did not appear: %v", err)
		}

		// Cursor should stay
		row2, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		if row1 != row2 {
			t.Errorf("cursor moved after M-n at last file: was row %d, now row %d", row1, row2)
		}
	})

	t.Run("FileNav_MP_AtFirstFile", func(t *testing.T) {
		dir := setupGrepFixtures3Files(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH marker . | sort")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		row1, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// M-p at first file should show "No more files"
		h.SendKeys("M-p")

		if err := h.WaitForContent("No more files", 5*time.Second); err != nil {
			t.Fatalf("No more files message did not appear: %v", err)
		}

		row2, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		if row1 != row2 {
			t.Errorf("cursor moved after M-p at first file: was row %d, now row %d", row1, row2)
		}
	})

	t.Run("Refresh_G", func(t *testing.T) {
		dir := setupGrepFixtures3Files(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH marker . | sort")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// Verify initial result count (should show 4 results: a.txt:2, b.txt:1, b.txt:3, c.txt:3)
		h.AssertStatusBar(t, "Line 1/4")

		// Add a new matching line to a fixture file
		newPath := filepath.Join(dir, "a.txt")
		if err := os.WriteFile(newPath, []byte("line1\nmarker alpha\nline3\nmarker alpha2\n"), 0o644); err != nil {
			t.Fatalf("failed to update fixture: %v", err)
		}

		// Press g to refresh
		h.SendKeys("g")

		// Wait for refreshed results (now 5 lines)
		if err := h.WaitForContent("Line 1/5", 10*time.Second); err != nil {
			t.Fatalf("refresh did not show new results: %v", err)
		}

		h.AssertScreenContains(t, "marker alpha2")
	})

	t.Run("N_AtLastResult", func(t *testing.T) {
		dir := setupGrepFixtures3Files(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH marker . | sort")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		// Navigate to last result: press n 3 times (4 results total, start at first)
		for i := 0; i < 3; i++ {
			h.SendKeys("n")
			time.Sleep(200 * time.Millisecond)
		}

		row1, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// n at last result should show "No more results"
		h.SendKeys("n")
		time.Sleep(200 * time.Millisecond)

		h.AssertMessageLine(t, "No more results")

		row2, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		if row1 != row2 {
			t.Errorf("cursor moved after n at last result: was row %d, now row %d", row1, row2)
		}
	})

	t.Run("P_AtFirstResult", func(t *testing.T) {
		dir := setupGrepFixtures3Files(t)
		h := newHarnessInDir(t, dir)

		if err := h.WaitForContent("*scratch*", 5*time.Second); err != nil {
			t.Fatalf("goomacs did not start: %v", err)
		}

		invokeFindGrep(t, h, "grep -rnH marker . | sort")

		if err := h.WaitForContent("*grep*", 10*time.Second); err != nil {
			t.Fatalf("grep buffer did not appear: %v", err)
		}

		row1, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}

		// p at first result should show "No more results"
		h.SendKeys("p")
		time.Sleep(200 * time.Millisecond)

		h.AssertMessageLine(t, "No more results")

		row2, _, err := h.CursorPosition()
		if err != nil {
			t.Fatalf("failed to get cursor: %v", err)
		}
		if row1 != row2 {
			t.Errorf("cursor moved after p at first result: was row %d, now row %d", row1, row2)
		}
	})
}
