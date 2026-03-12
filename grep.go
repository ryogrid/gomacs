package main

import (
	"strconv"
	"strings"
)

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
