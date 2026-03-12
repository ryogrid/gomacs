package main

import (
	"bytes"
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
}

// findGrepCommand opens a minibuffer prompt for the find-grep command.
func findGrepCommand(buf *Buffer, message *string) {
	defaultCmd := "find . -type f -exec grep -nH -e '' {} +"
	minibufferMode = true
	minibufferPrompt = "Run find-grep: "
	minibufferInput = []rune(defaultCmd)
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
