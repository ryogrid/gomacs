# PRD: Alt+X Command Palette and Comment/Uncomment Region

## Introduction

Add an Emacs-style `M-x` command palette to goomacs, along with `comment-region` and `uncomment-region` as the first two commands available through it. The command palette provides a generic, extensible mechanism for invoking named commands via a minibuffer prompt with tab completion. The comment commands enable toggling comments on selected regions of code, with language detection powered by go-enry.

## Goals

- Provide a discoverable way to invoke editor commands by name via `M-x`
- Support tab completion against a command registry so users can explore available commands
- Implement `comment-region` and `uncomment-region` that correctly handle 30+ programming languages
- Design the command registry to be easily extensible for future commands

## User Stories

### US-001: Command Registry Infrastructure
**Description:** As a developer, I need a command registry so that named commands can be registered and looked up programmatically.

**Acceptance Criteria:**
- [ ] A `Command` struct exists with `Name` (string) and `Fn` (handler function) fields
- [ ] A package-level registry (slice or map) holds all registered commands
- [ ] A registration function allows adding new commands by name and handler
- [ ] A lookup function returns the command matching a given name, or nil if not found
- [ ] A prefix-search function returns all commands whose names start with a given prefix
- [ ] `comment-region` and `uncomment-region` are registered in the registry at startup
- [ ] Typecheck/lint passes (`go vet ./...`)

### US-002: M-x Prompt Activation and Input
**Description:** As a user, I want to press `Alt+X` to open a command input prompt so that I can type a command name to execute.

**Acceptance Criteria:**
- [ ] Pressing `Alt+X` opens a minibuffer prompt displaying `M-x ` at the bottom of the screen
- [ ] The user can type characters into the prompt and see them appear
- [ ] Pressing `Backspace` deletes the last character
- [ ] Pressing `Escape` or `C-g` cancels and closes the prompt without executing anything
- [ ] Pressing `Enter` with a valid command name executes the command
- [ ] Pressing `Enter` with an invalid command name displays an error message (e.g., "Unknown command: foo")
- [ ] Typecheck/lint passes (`go vet ./...`)

### US-003: Tab Completion for M-x Commands
**Description:** As a user, I want tab completion in the M-x prompt so that I can quickly find commands without typing the full name.

**Acceptance Criteria:**
- [ ] Pressing `Tab` with a unique prefix match auto-completes the input to the full command name
- [ ] Pressing `Tab` with multiple matches displays the list of matching candidates in the message area
- [ ] Pressing `Tab` with no matches does nothing (no crash, no change to input)
- [ ] Pressing `Tab` with empty input displays all available commands
- [ ] Typecheck/lint passes (`go vet ./...`)

### US-004: Language Detection for Comment Style
**Description:** As a developer, I need to detect the programming language of the current buffer so that the correct comment delimiters are used.

**Acceptance Criteria:**
- [ ] `go-enry/go-enry/v2` is added as a dependency
- [ ] A function detects the language of a buffer using its filename and content
- [ ] A hardcoded map maps language names (as returned by go-enry) to comment delimiters (line prefix and/or block start/end)
- [ ] At least 30 languages are covered: Go, Python, JavaScript, TypeScript, Rust, C, C++, Java, Kotlin, Swift, Ruby, PHP, Shell (Bash), Perl, Lua, R, SQL, HTML, XML, CSS, SCSS, Haskell, Lisp, Clojure, Scheme, Erlang, Elixir, YAML, TOML, Makefile
- [ ] If the language is unrecognized or not in the map, the function falls back to `#` as the default line comment prefix
- [ ] Typecheck/lint passes (`go vet ./...`)

### US-005: comment-region Command
**Description:** As a user, I want to comment out a selected region of code so that I can quickly disable a block of code.

**Acceptance Criteria:**
- [ ] When invoked with an active region, each line from the start row to the end row (inclusive) is commented
- [ ] For languages with line comments (e.g., Go `//`, Python `#`), the line comment prefix followed by a space is prepended to each line (e.g., `// `)
- [ ] For languages with only block comments (e.g., HTML `<!-- -->`), each line is individually wrapped with block comment delimiters
- [ ] If no region is active, a message is displayed: "No region selected"
- [ ] The operation is recorded as a single undo entry (one `C-_` reverts the entire comment operation)
- [ ] `buf.Modified` and `buf.HighlightDirty` are set to `true` after the operation
- [ ] The mark is deactivated after the operation completes
- [ ] Typecheck/lint passes (`go vet ./...`)

### US-006: uncomment-region Command
**Description:** As a user, I want to uncomment a selected region of code so that I can re-enable previously commented code.

**Acceptance Criteria:**
- [ ] When invoked with an active region, each line from the start row to the end row (inclusive) is uncommented
- [ ] For line comments, the line comment prefix (with optional trailing space) is removed from the beginning of each line (tolerant of leading whitespace before the prefix)
- [ ] For block comments, the block delimiters are removed from each line (tolerant of minor whitespace variations)
- [ ] Lines that are not commented are left unchanged
- [ ] If no region is active, a message is displayed: "No region selected"
- [ ] The operation is recorded as a single undo entry
- [ ] `buf.Modified` and `buf.HighlightDirty` are set to `true` after the operation
- [ ] The mark is deactivated after the operation completes
- [ ] Typecheck/lint passes (`go vet ./...`)

## Functional Requirements

- FR-1: The system must provide a `Command` type with a `Name` string and an `Fn` handler function
- FR-2: The system must maintain a global command registry that can be searched by exact name or by prefix
- FR-3: Pressing `Alt+X` must open a minibuffer prompt with the text `M-x `
- FR-4: Tab key in the M-x prompt must trigger prefix-based completion against the command registry
- FR-5: When a unique prefix match exists, Tab must auto-complete the input to the full command name
- FR-6: When multiple prefix matches exist, Tab must display the candidate names in the message area
- FR-7: Enter in the M-x prompt must look up the typed name in the registry and invoke its handler
- FR-8: Enter with an unrecognized command name must display `Unknown command: <name>`
- FR-9: Escape or C-g in the M-x prompt must cancel without side effects
- FR-10: Language detection must use `go-enry/go-enry/v2` with filename and buffer content as inputs
- FR-11: A comment-style lookup table must map at least 30 language names to their comment delimiters
- FR-12: If the language is not in the lookup table, the system must fall back to `#` as the line comment prefix
- FR-13: `comment-region` must prepend the line comment prefix (+ space) to each line in the selected region
- FR-14: `comment-region` for block-comment-only languages must wrap each line individually with block delimiters
- FR-15: `uncomment-region` must remove the comment prefix/delimiters from each line, tolerating whitespace variations
- FR-16: Both comment commands must save a single undo snapshot before modifying the buffer
- FR-17: Both comment commands must display "No region selected" when no region is active

## Non-Goals

- No commands other than `comment-region` and `uncomment-region` in the initial release
- No command history or repeat-last-command for M-x
- No fuzzy matching or substring matching for tab completion (prefix only)
- No automatic detection of mixed comment styles within a single region
- No nested comment handling (commenting an already-commented region adds another layer)
- No key binding assignment from within the M-x prompt

## Technical Considerations

- **New file:** Create `command.go` for the command registry, language detection, and comment logic
- **Minibuffer reuse:** The M-x prompt reuses the existing minibuffer infrastructure (`minibufferMode`, `minibufferPrompt`, `minibufferInput`, `minibufferCallback`). Tab completion extends the existing Tab handler in the minibuffer event loop (main.go ~line 416)
- **Tab completion disambiguation:** The current Tab handler checks `minibufferPrompt` to decide what to complete (currently only `"Find file: "`). Add a case for `"M-x "` that searches the command registry by prefix
- **Region bounds:** Use `buf.regionBounds()` (buffer.go ~line 398) to get start/end rows for iteration
- **Undo:** Call `buf.SaveUndo()` once before modifying any lines to enable single-step undo
- **New dependency:** `go-enry/go-enry/v2` for language detection (pure Go, no CGo)
- **Handler signature:** Command handlers receive the current editor state needed to operate. Consider `func(buf *Buffer, message *string)` or a similar signature that gives access to the active buffer and a way to set the status message

## Success Metrics

- User can invoke `M-x comment-region` and `M-x uncomment-region` to toggle comments on a selected region
- Tab completion finds the correct command from any unambiguous prefix (e.g., `com` completes to `comment-region`)
- Comment/uncomment works correctly for at least Go, Python, JavaScript, HTML, and CSS files
- All existing tests continue to pass (`go test ./...`)

## Open Questions

- Should the command handler signature include access to the window list and active window, or just the buffer? (Needed if future commands manipulate windows)
- Should `comment-region` toggle behavior (auto-detect commented state and toggle) be a future enhancement?
