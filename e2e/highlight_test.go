package e2e

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiColorRe = regexp.MustCompile(`\x1b\[38;5;(\d+)m`)

func TestSyntaxHighlighting(t *testing.T) {
	t.Run("GoFileHighlighting", func(t *testing.T) {
		goContent := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
		path := createTestFile(t, "test.go", goContent)
		h := newHarness(t, path)

		if err := h.WaitForContent("test.go", 5*time.Second); err != nil {
			t.Fatalf("failed to open .go file: %v", err)
		}

		lines := h.CaptureWithEscapes()
		if lines == nil {
			t.Fatal("CaptureWithEscapes returned nil")
		}

		// Check content rows (0 to height-3) for ANSI escape sequences
		contentRows := lines[:h.height-2]
		content := strings.Join(contentRows, "\n")

		if !strings.Contains(content, "\x1b[") {
			t.Fatal("expected ANSI escape sequences in .go file content rows, but found none")
		}

		// Syntax-highlighted Go code should have multiple distinct foreground colors
		// (keywords, strings, functions get different colors from monokai theme)
		matches := ansiColorRe.FindAllStringSubmatch(content, -1)
		colors := make(map[string]bool)
		for _, m := range matches {
			colors[m[1]] = true
		}
		if len(colors) < 2 {
			t.Errorf("expected multiple distinct colors for syntax highlighting, got %d: %v", len(colors), colors)
		}
	})

	t.Run("TxtFileNoHighlighting", func(t *testing.T) {
		txtContent := "This is plain text.\nNo syntax highlighting here.\n"
		path := createTestFile(t, "test.txt", txtContent)
		h := newHarness(t, path)

		if err := h.WaitForContent("test.txt", 5*time.Second); err != nil {
			t.Fatalf("failed to open .txt file: %v", err)
		}

		lines := h.CaptureWithEscapes()
		if lines == nil {
			t.Fatal("CaptureWithEscapes returned nil")
		}

		// Check content rows (0 to height-3)
		// A .txt file should have at most 1 distinct foreground color (the default)
		// since there's no syntax highlighting
		contentRows := lines[:h.height-2]
		content := strings.Join(contentRows, "\n")

		matches := ansiColorRe.FindAllStringSubmatch(content, -1)
		colors := make(map[string]bool)
		for _, m := range matches {
			colors[m[1]] = true
		}
		if len(colors) > 1 {
			t.Errorf("expected at most 1 color for .txt file (no syntax highlighting), got %d: %v", len(colors), colors)
		}
	})
}
