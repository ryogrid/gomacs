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
