package main

import "strings"

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
