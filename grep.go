package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"goomacs/term"
)

// lastGrepCommand stores the most recently executed grep command for re-execution via g.
var lastGrepCommand string

// editorScreen holds a reference to the terminal screen for posting events from goroutines.
var editorScreen term.Screen

// grepResultMsg holds the output from an async grep command execution.
type grepResultMsg struct {
	stdout string
	stderr string
	err    error
}

// grepResultCh is used by the grep goroutine to send results back to the main event loop.
var grepResultCh = make(chan grepResultMsg, 1)

// GrepResult represents a single parsed grep output line.
type GrepResult struct {
	File string
	Line int
	Text string
}

// ParseGrepLine parses a single grep output line in the format "filepath:linenum:text".
// Returns the parsed result and true on success, or a zero value and false if the line
// doesn't match the expected format.
func ParseGrepLine(line string) (GrepResult, bool) {
	// Find the first colon (end of file path)
	firstColon := strings.Index(line, ":")
	if firstColon < 0 {
		return GrepResult{}, false
	}
	rest := line[firstColon+1:]
	// Find the second colon (end of line number)
	secondColon := strings.Index(rest, ":")
	if secondColon < 0 {
		return GrepResult{}, false
	}
	lineNumStr := rest[:secondColon]
	lineNum, err := strconv.Atoi(lineNumStr)
	if err != nil {
		return GrepResult{}, false
	}
	return GrepResult{
		File: line[:firstColon],
		Line: lineNum,
		Text: rest[secondColon+1:],
	}, true
}

// ParseGrepOutput splits grep output by newlines and parses each line,
// returning only successfully parsed results.
func ParseGrepOutput(output string) []GrepResult {
	lines := strings.Split(output, "\n")
	var results []GrepResult
	for _, line := range lines {
		if r, ok := ParseGrepLine(line); ok {
			results = append(results, r)
		}
	}
	return results
}

func init() {
	RegisterCommand("find-grep", findGrepCommand)
	modeHandlers["grep"] = grepModeHandler
}

// grepModeHandler handles keybindings in the *grep* buffer.
func grepModeHandler(ev *term.KeyEvent, buf *Buffer, message *string) bool {
	// Handle Enter key to jump to source
	if ev.Key() == term.KeyEnter {
		if buf.CursorR >= len(buf.Lines) {
			return true
		}
		result, ok := ParseGrepLine(string(buf.Lines[buf.CursorR]))
		if !ok {
			*message = "No grep result on this line"
			return true
		}
		// Remember the grep buffer index so user can return
		grepIdx := activeBufferIdx
		// Check if the file is already open
		targetIdx := -1
		for i, b := range buffers {
			if b.Filename == result.File {
				targetIdx = i
				break
			}
		}
		if targetIdx < 0 {
			// Open the file
			newBuf, err := NewBufferFromFile(result.File)
			if err != nil {
				if os.IsNotExist(err) {
					*message = fmt.Sprintf("File not found: %s", result.File)
					return true
				}
				*message = fmt.Sprintf("Error: %v", err)
				return true
			}
			buffers = append(buffers, newBuf)
			targetIdx = len(buffers) - 1
		}
		previousBufferIdx = grepIdx
		activeBufferIdx = targetIdx
		targetBuf := buffers[targetIdx]
		targetLine := result.Line - 1
		if targetLine < 0 {
			targetLine = 0
		}
		if targetLine >= len(targetBuf.Lines) {
			targetLine = len(targetBuf.Lines) - 1
		}
		targetBuf.CursorR = targetLine
		targetBuf.CursorC = 0
		return true
	}

	if ev.Key() != term.KeyRune {
		return false
	}

	// Handle Alt-modified keys (M-n, M-p)
	if ev.Modifiers()&term.ModAlt != 0 {
		switch ev.Rune() {
		case 'n':
			// M-n: jump to next file's results
			if buf.CursorR >= len(buf.Lines) {
				*message = "No more files"
				return true
			}
			currentResult, ok := ParseGrepLine(string(buf.Lines[buf.CursorR]))
			if !ok {
				// Not on a result line; just try to find the next result
				for i := buf.CursorR + 1; i < len(buf.Lines); i++ {
					if _, ok := ParseGrepLine(string(buf.Lines[i])); ok {
						buf.CursorR = i
						buf.CursorC = 0
						return true
					}
				}
				*message = "No more files"
				return true
			}
			currentFile := currentResult.File
			for i := buf.CursorR + 1; i < len(buf.Lines); i++ {
				if r, ok := ParseGrepLine(string(buf.Lines[i])); ok && r.File != currentFile {
					buf.CursorR = i
					buf.CursorC = 0
					return true
				}
			}
			*message = "No more files"
			return true
		case 'p':
			// M-p: jump to previous file's first result
			if buf.CursorR >= len(buf.Lines) || buf.CursorR <= 0 {
				*message = "No more files"
				return true
			}
			currentResult, ok := ParseGrepLine(string(buf.Lines[buf.CursorR]))
			if !ok {
				// Not on a result line; try to find the previous result
				for i := buf.CursorR - 1; i >= 0; i-- {
					if _, ok := ParseGrepLine(string(buf.Lines[i])); ok {
						buf.CursorR = i
						buf.CursorC = 0
						return true
					}
				}
				*message = "No more files"
				return true
			}
			currentFile := currentResult.File
			// Scan backward to find a line with a different file
			prevFileIdx := -1
			for i := buf.CursorR - 1; i >= 0; i-- {
				if r, ok := ParseGrepLine(string(buf.Lines[i])); ok && r.File != currentFile {
					prevFileIdx = i
					break
				}
			}
			if prevFileIdx < 0 {
				*message = "No more files"
				return true
			}
			// Now find the first result of that file by scanning backward further
			prevFile := ""
			if r, ok := ParseGrepLine(string(buf.Lines[prevFileIdx])); ok {
				prevFile = r.File
			}
			firstOfFile := prevFileIdx
			for i := prevFileIdx - 1; i >= 0; i-- {
				if r, ok := ParseGrepLine(string(buf.Lines[i])); ok && r.File == prevFile {
					firstOfFile = i
				} else if _, ok := ParseGrepLine(string(buf.Lines[i])); ok {
					break
				}
			}
			buf.CursorR = firstOfFile
			buf.CursorC = 0
			return true
		}
		return false
	}

	switch ev.Rune() {
	case 'n':
		// Next grep result
		for i := buf.CursorR + 1; i < len(buf.Lines); i++ {
			if _, ok := ParseGrepLine(string(buf.Lines[i])); ok {
				buf.CursorR = i
				buf.CursorC = 0
				return true
			}
		}
		*message = "No more results"
		return true
	case 'p':
		// Previous grep result
		for i := buf.CursorR - 1; i >= 0; i-- {
			if _, ok := ParseGrepLine(string(buf.Lines[i])); ok {
				buf.CursorR = i
				buf.CursorC = 0
				return true
			}
		}
		*message = "No more results"
		return true
	case 'g':
		// Refresh: re-execute the last grep command
		if lastGrepCommand == "" {
			*message = "No previous grep command"
			return true
		}
		*message = "Searching..."
		executeGrepCommand(lastGrepCommand)
		return true
	case 'q':
		// Close *grep* buffer and switch to previous buffer
		grepIdx := -1
		for i, b := range buffers {
			if b.Filename == "*grep*" {
				grepIdx = i
				break
			}
		}
		if grepIdx < 0 {
			return true
		}
		newActive := previousBufferIdx
		if newActive == grepIdx {
			newActive = 0
		}
		buffers = append(buffers[:grepIdx], buffers[grepIdx+1:]...)
		if newActive > grepIdx {
			newActive--
		}
		if newActive >= len(buffers) {
			newActive = len(buffers) - 1
		}
		if newActive < 0 {
			newActive = 0
		}
		activeBufferIdx = newActive
		previousBufferIdx = newActive
		return true
	}
	return false
}

// findGrepCommand opens a minibuffer prompt for the find-grep command.
func findGrepCommand(buf *Buffer, message *string) {
	defaultCmd := "find . -type f -exec grep -nH -e '' {} +"
	minibufferMode = true
	minibufferPrompt = "Run find-grep: "
	minibufferInput = []rune(defaultCmd)
	minibufferCursorPos = len(minibufferInput)
	minibufferCallback = func(input string) {
		lastGrepCommand = input
		*message = "Searching..."
		executeGrepCommand(input)
	}
	*message = minibufferPrompt + defaultCmd
}

// executeGrepCommand runs a grep command asynchronously and sends the result on grepResultCh.
func executeGrepCommand(cmd string) {
	go func() {
		c := exec.Command("sh", "-c", cmd)
		var stdout, stderr bytes.Buffer
		c.Stdout = &stdout
		c.Stderr = &stderr
		err := c.Run()
		grepResultCh <- grepResultMsg{
			stdout: stdout.String(),
			stderr: stderr.String(),
			err:    err,
		}
		// Wake up the event loop by posting a synthetic event.
		if editorScreen != nil {
			editorScreen.PostEvent(term.NewKeyEvent(term.KeyNUL, 0, term.ModNone))
		}
	}()
}
