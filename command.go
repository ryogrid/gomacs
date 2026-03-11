package main

import (
	"strings"

	enry "github.com/go-enry/go-enry/v2"
)

// CommentStyle holds the comment delimiters for a language.
type CommentStyle struct {
	LinePrefix string
	BlockStart string
	BlockEnd   string
}

// commentStyles maps language names (as returned by go-enry) to their comment delimiters.
var commentStyles = map[string]CommentStyle{
	"Go":            {LinePrefix: "//"},
	"Python":        {LinePrefix: "#"},
	"JavaScript":    {LinePrefix: "//"},
	"TypeScript":    {LinePrefix: "//"},
	"Rust":          {LinePrefix: "//"},
	"C":             {LinePrefix: "//"},
	"C++":           {LinePrefix: "//"},
	"Java":          {LinePrefix: "//"},
	"Kotlin":        {LinePrefix: "//"},
	"Swift":         {LinePrefix: "//"},
	"Ruby":          {LinePrefix: "#"},
	"PHP":           {LinePrefix: "//"},
	"Shell":         {LinePrefix: "#"},
	"Perl":          {LinePrefix: "#"},
	"Lua":           {LinePrefix: "--"},
	"R":             {LinePrefix: "#"},
	"SQL":           {LinePrefix: "--"},
	"HTML":          {BlockStart: "<!--", BlockEnd: "-->"},
	"XML":           {BlockStart: "<!--", BlockEnd: "-->"},
	"CSS":           {BlockStart: "/*", BlockEnd: "*/"},
	"SCSS":          {LinePrefix: "//"},
	"Haskell":       {LinePrefix: "--"},
	"Common Lisp":   {LinePrefix: ";"},
	"Clojure":       {LinePrefix: ";"},
	"Scheme":        {LinePrefix: ";"},
	"Erlang":        {LinePrefix: "%"},
	"Elixir":        {LinePrefix: "#"},
	"YAML":          {LinePrefix: "#"},
	"TOML":          {LinePrefix: "#"},
	"Makefile":      {LinePrefix: "#"},
}

// detectCommentStyle detects the language of the given file and returns its comment style.
func detectCommentStyle(filename string, content string) CommentStyle {
	lang := enry.GetLanguage(filename, []byte(content))
	if style, ok := commentStyles[lang]; ok {
		return style
	}
	return CommentStyle{LinePrefix: "#"}
}

// Command represents a named editor command.
type Command struct {
	Name string
	Fn   func(*Buffer, *string)
}

// commands holds all registered commands.
var commands []Command

// RegisterCommand appends a new command to the registry.
func RegisterCommand(name string, fn func(*Buffer, *string)) {
	commands = append(commands, Command{Name: name, Fn: fn})
}

// FindCommand returns a pointer to the command with the exact given name, or nil.
func FindCommand(name string) *Command {
	for i := range commands {
		if commands[i].Name == name {
			return &commands[i]
		}
	}
	return nil
}

func init() {
	RegisterCommand("comment-region", commentRegion)
	RegisterCommand("uncomment-region", uncommentRegion)
}

// commentRegion comments out the selected region of code.
func commentRegion(buf *Buffer, message *string) {
	if !buf.MarkActive {
		*message = "No region selected"
		return
	}

	buf.SaveUndo()

	startR, _, endR, _ := buf.regionBounds()

	// Build buffer content string for language detection
	lines := make([]string, len(buf.Lines))
	for i, line := range buf.Lines {
		lines[i] = string(line)
	}
	content := strings.Join(lines, "\n")

	style := detectCommentStyle(buf.Filename, content)

	for row := startR; row <= endR; row++ {
		line := buf.Lines[row]
		if style.LinePrefix != "" {
			prefix := []rune(style.LinePrefix + " ")
			buf.Lines[row] = append(prefix, line...)
		} else {
			wrapped := []rune(style.BlockStart + " " + string(line) + " " + style.BlockEnd)
			buf.Lines[row] = wrapped
		}
	}

	buf.Modified = true
	buf.HighlightDirty = true
	buf.DeactivateMark()
	*message = "Region commented"
}

// uncommentRegion removes comment markers from the selected region of code.
func uncommentRegion(buf *Buffer, message *string) {
	if !buf.MarkActive {
		*message = "No region selected"
		return
	}

	buf.SaveUndo()

	startR, _, endR, _ := buf.regionBounds()

	// Build buffer content string for language detection
	lineStrs := make([]string, len(buf.Lines))
	for i, line := range buf.Lines {
		lineStrs[i] = string(line)
	}
	content := strings.Join(lineStrs, "\n")

	style := detectCommentStyle(buf.Filename, content)

	for row := startR; row <= endR; row++ {
		line := string(buf.Lines[row])
		if style.LinePrefix != "" {
			// Find the line prefix, tolerating leading whitespace
			trimmed := strings.TrimLeft(line, " \t")
			idx := strings.Index(line, trimmed)
			indent := line[:idx]
			if strings.HasPrefix(trimmed, style.LinePrefix+" ") {
				// Remove prefix with trailing space
				buf.Lines[row] = []rune(indent + trimmed[len(style.LinePrefix)+1:])
			} else if strings.HasPrefix(trimmed, style.LinePrefix) {
				// Remove prefix without trailing space
				buf.Lines[row] = []rune(indent + trimmed[len(style.LinePrefix):])
			}
		} else {
			// Block comments: remove BlockStart (+ optional space) from beginning and BlockEnd (+ optional space) from end
			modified := line
			trimmedLeft := strings.TrimLeft(modified, " \t")
			idxL := strings.Index(modified, trimmedLeft)
			indentL := modified[:idxL]
			if strings.HasPrefix(trimmedLeft, style.BlockStart+" ") {
				modified = indentL + trimmedLeft[len(style.BlockStart)+1:]
			} else if strings.HasPrefix(trimmedLeft, style.BlockStart) {
				modified = indentL + trimmedLeft[len(style.BlockStart):]
			}
			if strings.HasSuffix(modified, " "+style.BlockEnd) {
				modified = modified[:len(modified)-len(style.BlockEnd)-1]
			} else if strings.HasSuffix(modified, style.BlockEnd) {
				modified = modified[:len(modified)-len(style.BlockEnd)]
			}
			buf.Lines[row] = []rune(modified)
		}
	}

	buf.Modified = true
	buf.HighlightDirty = true
	buf.DeactivateMark()
	*message = "Region uncommented"
}

// FindCommandsByPrefix returns all commands whose names start with the given prefix.
func FindCommandsByPrefix(prefix string) []Command {
	var result []Command
	for _, cmd := range commands {
		if strings.HasPrefix(cmd.Name, prefix) {
			result = append(result, cmd)
		}
	}
	return result
}
