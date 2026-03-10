package main

import (
	"fmt"
	"os"
	"strings"

	"goomacs/term"
)

const tabWidth = 8

// bufColToVisualCol converts a buffer column index to a visual (screen) column
// for the given line, accounting for tab expansion.
func bufColToVisualCol(line []rune, bufCol int) int {
	visualCol := 0
	for i := 0; i < bufCol && i < len(line); i++ {
		if line[i] == '\t' {
			visualCol += tabWidth - (visualCol % tabWidth)
		} else {
			visualCol++
		}
	}
	return visualCol
}

func main() {
	// Load buffers from file arguments or create empty *scratch* buffer.
	var buffers []*Buffer
	if len(os.Args) > 1 {
		for _, filename := range os.Args[1:] {
			b, err := NewBufferFromFile(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error opening file: %v\n", err)
				os.Exit(1)
			}
			buffers = append(buffers, b)
		}
	} else {
		scratch := NewBuffer()
		scratch.Filename = "*scratch*"
		buffers = append(buffers, scratch)
	}
	activeBufferIdx := 0
	previousBufferIdx := 0
	buf := buffers[activeBufferIdx]

	screen := term.NewTerminal()
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "error initializing screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	_, screenHeight := screen.Size()
	viewHeight := textAreaHeight(screenHeight)
	buf.AdjustScroll(viewHeight)
	drawBuffer(screen, buf)
	screen.ShowCursor(bufColToVisualCol(buf.Lines[buf.CursorR], buf.CursorC), buf.CursorR-buf.ScrollOffset)
	screen.Show()

	var message string      // message to display in message area
	var prefixCx bool       // true when C-x prefix has been pressed
	var quitWarned bool     // true after warning about unsaved changes on C-x C-c
	var searchMode bool     // true when in incremental search
	var searchForward bool  // true for forward search, false for backward
	var searchQuery []rune  // current search query
	var searchOrigR int     // cursor row before search started
	var searchOrigC int     // cursor col before search started
	var searchMatchR int    // row of current match (for highlight)
	var searchMatchC int    // col of current match (for highlight)
	var searchHasMatch bool // true if current query has a match

	var minibufferMode bool              // true when minibuffer input is active
	var minibufferPrompt string          // prompt shown before input
	var minibufferInput []rune           // current input text
	var minibufferCallback func(string)  // called with input on Enter

	var confirmMode bool           // true when waiting for y/n confirmation
	var confirmCallback func(bool) // called with true for y, false for n

	redraw := func() {
		buf.AdjustScroll(viewHeight)
		if searchMode && searchHasMatch {
			drawBufferWithSearch(screen, buf, searchHighlight{
				active:   true,
				matchR:   searchMatchR,
				matchC:   searchMatchC,
				queryLen: len(searchQuery),
			})
		} else {
			drawBuffer(screen, buf)
		}
		drawMessageLine(screen, message)
		screen.ShowCursor(bufColToVisualCol(buf.Lines[buf.CursorR], buf.CursorC), buf.CursorR-buf.ScrollOffset)
		screen.Show()
	}

	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *term.KeyEvent:
			_, screenHeight = screen.Size()
			viewHeight = textAreaHeight(screenHeight)
			message = "" // clear message on next key

			// Handle search mode
			if searchMode {
				switch ev.Key() {
				case term.KeyCtrlS:
					// Search forward for next match
					searchForward = true
					if len(searchQuery) > 0 {
						startR, startC := buf.CursorR, buf.CursorC+1
						if startC > len(buf.Lines[startR]) {
							startR++
							startC = 0
							if startR >= len(buf.Lines) {
								startR = 0
							}
						}
						r, c, ok := buf.SearchForward(searchQuery, startR, startC)
						if ok {
							buf.CursorR, buf.CursorC = r, c
							searchMatchR, searchMatchC = r, c
							searchHasMatch = true
							message = fmt.Sprintf("I-search: %s", string(searchQuery))
						} else {
							message = fmt.Sprintf("Failing I-search: %s", string(searchQuery))
							searchHasMatch = false
						}
					}
				case term.KeyCtrlR:
					// Search backward for previous match
					searchForward = false
					if len(searchQuery) > 0 {
						startR, startC := buf.CursorR, buf.CursorC
						r, c, ok := buf.SearchBackward(searchQuery, startR, startC)
						if ok {
							buf.CursorR, buf.CursorC = r, c
							searchMatchR, searchMatchC = r, c
							searchHasMatch = true
							message = fmt.Sprintf("I-search backward: %s", string(searchQuery))
						} else {
							message = fmt.Sprintf("Failing I-search backward: %s", string(searchQuery))
							searchHasMatch = false
						}
					}
				case term.KeyCtrlG:
					// Cancel search, restore original position
					buf.CursorR = searchOrigR
					buf.CursorC = searchOrigC
					searchMode = false
					searchHasMatch = false
					message = "Quit"
				case term.KeyEnter:
					// Accept search result, exit search mode
					searchMode = false
					searchHasMatch = false
					message = ""
				case term.KeyBackspace, term.KeyBackspace2:
					// Delete last character from search query
					if len(searchQuery) > 0 {
						searchQuery = searchQuery[:len(searchQuery)-1]
						if len(searchQuery) > 0 {
							// Re-search from original position
							var r, c int
							var ok bool
							if searchForward {
								r, c, ok = buf.SearchForward(searchQuery, searchOrigR, searchOrigC)
							} else {
								r, c, ok = buf.SearchBackward(searchQuery, searchOrigR, searchOrigC)
							}
							if ok {
								buf.CursorR, buf.CursorC = r, c
								searchMatchR, searchMatchC = r, c
								searchHasMatch = true
							} else {
								searchHasMatch = false
							}
							if searchForward {
								message = fmt.Sprintf("I-search: %s", string(searchQuery))
							} else {
								message = fmt.Sprintf("I-search backward: %s", string(searchQuery))
							}
						} else {
							buf.CursorR = searchOrigR
							buf.CursorC = searchOrigC
							searchHasMatch = false
							if searchForward {
								message = "I-search: "
							} else {
								message = "I-search backward: "
							}
						}
					}
				case term.KeyRune:
					// Add character to search query
					searchQuery = append(searchQuery, ev.Rune())
					var r, c int
					var ok bool
					if searchForward {
						r, c, ok = buf.SearchForward(searchQuery, buf.CursorR, buf.CursorC)
					} else {
						r, c, ok = buf.SearchBackward(searchQuery, buf.CursorR, buf.CursorC+1)
					}
					if ok {
						buf.CursorR, buf.CursorC = r, c
						searchMatchR, searchMatchC = r, c
						searchHasMatch = true
						if searchForward {
							message = fmt.Sprintf("I-search: %s", string(searchQuery))
						} else {
							message = fmt.Sprintf("I-search backward: %s", string(searchQuery))
						}
					} else {
						if searchForward {
							message = fmt.Sprintf("Failing I-search: %s", string(searchQuery))
						} else {
							message = fmt.Sprintf("Failing I-search backward: %s", string(searchQuery))
						}
						searchHasMatch = false
					}
				default:
					// Any other key exits search mode and is NOT consumed
					searchMode = false
					searchHasMatch = false
					message = ""
					// Re-post the event so it gets handled normally
					screen.PostEvent(ev)
					redraw()
					continue
				}
				redraw()
				continue
			}

			// Handle minibuffer input mode
			if minibufferMode {
				switch ev.Key() {
				case term.KeyEnter:
					input := string(minibufferInput)
					cb := minibufferCallback
					minibufferMode = false
					minibufferInput = nil
					minibufferCallback = nil
					message = ""
					if cb != nil {
						cb(input)
					}
				case term.KeyCtrlG:
					minibufferMode = false
					minibufferInput = nil
					minibufferCallback = nil
					message = "Quit"
				case term.KeyBackspace, term.KeyBackspace2:
					if len(minibufferInput) > 0 {
						minibufferInput = minibufferInput[:len(minibufferInput)-1]
					}
					message = minibufferPrompt + string(minibufferInput)
				case term.KeyRune:
					minibufferInput = append(minibufferInput, ev.Rune())
					message = minibufferPrompt + string(minibufferInput)
				default:
					// All other keys are ignored in minibuffer mode
				}
				redraw()
				continue
			}

			// Handle y/n confirmation mode
			if confirmMode {
				if ev.Key() == term.KeyRune {
					switch ev.Rune() {
					case 'y':
						confirmMode = false
						cb := confirmCallback
						confirmCallback = nil
						if cb != nil {
							cb(true)
						}
					case 'n':
						confirmMode = false
						confirmCallback = nil
						message = "Cancelled"
					}
				} else if ev.Key() == term.KeyCtrlG {
					confirmMode = false
					confirmCallback = nil
					message = "Quit"
				}
				redraw()
				continue
			}

			// Reset consecutive kill tracking for non-kill keys
			if ev.Key() != term.KeyCtrlK {
				buf.ClearLastKill()
			}

			// Reset quit warning unless we're in a C-x prefix sequence
			if ev.Key() != term.KeyCtrlX && !prefixCx {
				quitWarned = false
			}

			// Handle C-x prefix second key
			if prefixCx {
				prefixCx = false
				switch ev.Key() {
				case term.KeyCtrlS:
					if err := buf.Save(); err != nil {
						if err == errNoFilename {
							message = "No file name"
						} else {
							message = fmt.Sprintf("Error saving: %v", err)
						}
					} else {
						message = fmt.Sprintf("Saved %s", buf.Filename)
					}
				case term.KeyCtrlC:
					anyModified := false
					for _, b := range buffers {
						if b.Modified {
							anyModified = true
							break
						}
					}
					if anyModified && !quitWarned {
						message = "Modified buffers exist; exit anyway? (C-x C-c to confirm)"
						quitWarned = true
					} else {
						return
					}
				case term.KeyCtrlB:
					// Build buffer list content
					var lines []string
					for i, b := range buffers {
						marker := " "
						if i == activeBufferIdx {
							marker = ">"
						}
						modFlag := " "
						if b.Modified {
							modFlag = "*"
						}
						name := b.Filename
						if name == "" {
							name = "[No Name]"
						}
						lines = append(lines, fmt.Sprintf("%s %s %-20s %s", marker, modFlag, name, name))
					}
					content := ""
					for i, l := range lines {
						if i > 0 {
							content += "\n"
						}
						content += l
					}

					// Find existing *Buffer List* or create new one
					var blBuf *Buffer
					blIdx := -1
					for i, b := range buffers {
						if b.Filename == "*Buffer List*" {
							blBuf = b
							blIdx = i
							break
						}
					}
					if blBuf == nil {
						blBuf = NewBuffer()
						blBuf.Filename = "*Buffer List*"
						buffers = append(buffers, blBuf)
						blIdx = len(buffers) - 1
					}

					// Update content
					rawLines := strings.Split(content, "\n")
					blBuf.Lines = make([][]rune, len(rawLines))
					for i, rl := range rawLines {
						blBuf.Lines[i] = []rune(rl)
					}
					blBuf.CursorR = 0
					blBuf.CursorC = 0
					blBuf.ScrollOffset = 0
					blBuf.Modified = false

					// Switch to the buffer list buffer
					previousBufferIdx = activeBufferIdx
					activeBufferIdx = blIdx
					buf = buffers[activeBufferIdx]
				case term.KeyCtrlF:
					minibufferMode = true
					minibufferPrompt = "Find file: "
					minibufferInput = nil
					minibufferCallback = func(input string) {
						if input == "" {
							message = "No file name specified"
							return
						}
						// Check if the file is already open in an existing buffer
						for i, b := range buffers {
							if b.Filename == input {
								previousBufferIdx = activeBufferIdx
								activeBufferIdx = i
								buf = buffers[activeBufferIdx]
								message = fmt.Sprintf("Switch to buffer: %s", input)
								return
							}
						}
						// Try to load the file from disk
						newBuf, err := NewBufferFromFile(input)
						if err != nil {
							if os.IsNotExist(err) {
								// File doesn't exist: create a new empty buffer with that filename
								newBuf = NewBuffer()
								newBuf.Filename = input
								message = fmt.Sprintf("(New file) %s", input)
							} else {
								message = fmt.Sprintf("Error: %v", err)
								return
							}
						}
						buffers = append(buffers, newBuf)
						previousBufferIdx = activeBufferIdx
						activeBufferIdx = len(buffers) - 1
						buf = buffers[activeBufferIdx]
					}
					message = minibufferPrompt
				case term.KeyRune:
					switch ev.Rune() {
					case 'b':
						minibufferMode = true
						minibufferPrompt = "Switch to buffer: "
						minibufferInput = nil
						minibufferCallback = func(input string) {
							if input == "" {
								// Switch to previous buffer
								previousBufferIdx, activeBufferIdx = activeBufferIdx, previousBufferIdx
								buf = buffers[activeBufferIdx]
								return
							}
							// Search for existing buffer by name
							for i, b := range buffers {
								if b.Filename == input {
									previousBufferIdx = activeBufferIdx
									activeBufferIdx = i
									buf = buffers[activeBufferIdx]
									return
								}
							}
							// Create new empty buffer with that name
							newBuf := NewBuffer()
							newBuf.Filename = input
							buffers = append(buffers, newBuf)
							previousBufferIdx = activeBufferIdx
							activeBufferIdx = len(buffers) - 1
							buf = buffers[activeBufferIdx]
						}
						message = minibufferPrompt
					case 'k':
						currentName := buf.Filename
						if currentName == "" {
							currentName = "[No Name]"
						}
						minibufferMode = true
						minibufferPrompt = fmt.Sprintf("Kill buffer: (default %s) ", currentName)
						minibufferInput = nil
						minibufferCallback = func(input string) {
							// Find target buffer
							targetIdx := activeBufferIdx
							if input != "" {
								targetIdx = -1
								for i, b := range buffers {
									if b.Filename == input {
										targetIdx = i
										break
									}
								}
								if targetIdx == -1 {
									message = fmt.Sprintf("No buffer named %s", input)
									return
								}
							}
							
							killBuffer := func() {
								killedName := buffers[targetIdx].Filename
								// Remove buffer from list
								buffers = append(buffers[:targetIdx], buffers[targetIdx+1:]...)
								
								// If no buffers left, create *scratch*
								if len(buffers) == 0 {
									scratch := NewBuffer()
									scratch.Filename = "*scratch*"
									buffers = append(buffers, scratch)
									activeBufferIdx = 0
									previousBufferIdx = 0
									buf = buffers[0]
									message = fmt.Sprintf("Killed buffer %s", killedName)
									return
								}
								
								// Adjust activeBufferIdx
								if targetIdx == activeBufferIdx {
									if activeBufferIdx >= len(buffers) {
										activeBufferIdx = len(buffers) - 1
									}
								} else if targetIdx < activeBufferIdx {
									activeBufferIdx--
								}
								
								// Adjust previousBufferIdx
								if previousBufferIdx == targetIdx {
									previousBufferIdx = activeBufferIdx
								} else if previousBufferIdx > targetIdx {
									previousBufferIdx--
								}
								if previousBufferIdx >= len(buffers) {
									previousBufferIdx = len(buffers) - 1
								}
								
								buf = buffers[activeBufferIdx]
								message = fmt.Sprintf("Killed buffer %s", killedName)
							}
							
							// Check if buffer is modified
							if buffers[targetIdx].Modified {
								message = "Buffer modified; kill anyway? (y/n)"
								confirmMode = true
								confirmCallback = func(yes bool) {
									if yes {
										killBuffer()
									}
								}
								return
							}
							
							killBuffer()
						}
						message = minibufferPrompt
					}
				}
				redraw()
				continue
			}

			switch ev.Key() {
			case term.KeyCtrlX:
				prefixCx = true
				message = "C-x-"
				redraw()
				continue
			case term.KeyCtrlF:
				buf.MoveForward()
			case term.KeyCtrlB:
				buf.MoveBackward()
			case term.KeyCtrlN:
				buf.MoveDown()
			case term.KeyCtrlP:
				buf.MoveUp()
			case term.KeyCtrlA:
				buf.MoveBeginningOfLine()
			case term.KeyCtrlE:
				buf.MoveEndOfLine()
			case term.KeyCtrlS:
				searchMode = true
				searchForward = true
				searchQuery = nil
				searchOrigR = buf.CursorR
				searchOrigC = buf.CursorC
				searchHasMatch = false
				message = "I-search: "
				redraw()
				continue
			case term.KeyCtrlR:
				searchMode = true
				searchForward = false
				searchQuery = nil
				searchOrigR = buf.CursorR
				searchOrigC = buf.CursorC
				searchHasMatch = false
				message = "I-search backward: "
				redraw()
				continue
			case term.KeyCtrlV:
				buf.ScrollDown(viewHeight)
			case term.KeyCtrlSpace, term.KeyNUL:
				buf.SetMark()
				message = "Mark set"
			case term.KeyCtrlG:
				buf.DeactivateMark()
				message = ""
			case term.KeyCtrlW:
				buf.SaveUndo()
				buf.KillRegion()
			case term.KeyCtrlUnderscore:
				if buf.Undo() {
					message = "Undo"
				} else {
					message = "No further undo information"
				}
			case term.KeyCtrlK:
				buf.SaveUndo()
				buf.KillLine()
				redraw()
				continue
			case term.KeyCtrlY:
				buf.SaveUndo()
				buf.Yank()
			case term.KeyCtrlD:
				buf.SaveUndo()
				buf.DeleteChar()
			case term.KeyEnter:
				buf.SaveUndo()
				buf.InsertNewline()
			case term.KeyBackspace, term.KeyBackspace2:
				buf.SaveUndo()
				buf.Backspace()
			case term.KeyRight:
				buf.MoveForward()
			case term.KeyLeft:
				buf.MoveBackward()
			case term.KeyDown:
				buf.MoveDown()
			case term.KeyUp:
				buf.MoveUp()
			case term.KeyRune:
				if ev.Modifiers()&term.ModAlt != 0 {
					switch ev.Rune() {
					case 'v':
						buf.ScrollUp(viewHeight)
					case 'w':
						buf.CopyRegion()
						message = "Region copied"
					case '<':
						buf.MoveBeginningOfBuffer()
					case '>':
						buf.MoveEndOfBuffer()
					}
				} else {
					buf.SaveUndo()
					buf.InsertChar(ev.Rune())
				}
			case term.KeyEsc:
				// Handle Esc-prefixed sequences (for terminals that send Esc then key)
				nextEv := screen.PollEvent()
				if kev, ok := nextEv.(*term.KeyEvent); ok && kev.Key() == term.KeyRune {
					switch kev.Rune() {
					case 'v':
						buf.ScrollUp(viewHeight)
					case 'w':
						buf.CopyRegion()
						message = "Region copied"
					case '<':
						buf.MoveBeginningOfBuffer()
					case '>':
						buf.MoveEndOfBuffer()
					}
				}
			}
			redraw()
		case *term.ResizeEvent:
			screen.Sync()
			_, screenHeight = screen.Size()
			viewHeight = textAreaHeight(screenHeight)
			redraw()
		}
	}
}

// textAreaHeight returns the number of rows available for text (excluding status and message lines).
func textAreaHeight(screenHeight int) int {
	h := screenHeight - 2
	if h < 1 {
		h = 1
	}
	return h
}

// searchHighlight holds the state for highlighting a search match during rendering.
type searchHighlight struct {
	active   bool
	matchR   int
	matchC   int
	queryLen int
}

// drawBuffer renders the buffer content onto the screen, accounting for scroll offset.
func drawBuffer(screen term.Screen, buf *Buffer) {
	drawBufferWithSearch(screen, buf, searchHighlight{})
}

// drawBufferWithSearch renders the buffer with optional search match highlighting.
func drawBufferWithSearch(screen term.Screen, buf *Buffer, sh searchHighlight) {
	screen.Clear()
	width, height := screen.Size()
	textH := textAreaHeight(height)

	for row := 0; row < textH; row++ {
		bufRow := row + buf.ScrollOffset
		if bufRow >= len(buf.Lines) {
			break
		}
		line := buf.Lines[bufRow]
		visualCol := 0
		for bufCol := 0; bufCol < len(line) && visualCol < width; bufCol++ {
			style := term.StyleDefault
			if buf.InRegion(bufRow, bufCol) {
				style = style.Reverse(true)
			}
			if sh.active && bufRow == sh.matchR && bufCol >= sh.matchC && bufCol < sh.matchC+sh.queryLen {
				style = style.Reverse(true)
			}
			if line[bufCol] == '\t' {
				nextStop := visualCol + tabWidth - (visualCol%tabWidth)
				for visualCol < nextStop && visualCol < width {
					screen.SetContent(visualCol, row, ' ', style)
					visualCol++
				}
			} else {
				screen.SetContent(visualCol, row, line[bufCol], style)
				visualCol++
			}
		}
	}

	drawStatusLine(screen, buf)
}

// drawMessageLine renders a message on the last row of the screen.
func drawMessageLine(screen term.Screen, msg string) {
	width, height := screen.Size()
	if height < 1 || msg == "" {
		return
	}
	msgRow := height - 1
	for i, ch := range []rune(msg) {
		if i >= width {
			break
		}
		screen.SetContent(i, msgRow, ch, term.StyleDefault)
	}
}

// drawStatusLine renders the status line on the second-to-last row with reverse video.
func drawStatusLine(screen term.Screen, buf *Buffer) {
	width, height := screen.Size()
	if height < 2 {
		return
	}
	statusRow := height - 2

	name := buf.Filename
	if name == "" {
		name = "[No Name]"
	}
	mod := ""
	if buf.Modified {
		mod = " [Modified]"
	}
	pos := fmt.Sprintf("Line %d/%d, Col %d", buf.CursorR+1, len(buf.Lines), buf.CursorC+1)
	left := fmt.Sprintf(" %s%s", name, mod)
	right := fmt.Sprintf("%s ", pos)

	style := term.StyleDefault.Reverse(true)

	// Fill entire status line with reverse video
	for col := 0; col < width; col++ {
		screen.SetContent(col, statusRow, ' ', style)
	}
	// Draw left-aligned text
	for i, ch := range []rune(left) {
		if i >= width {
			break
		}
		screen.SetContent(i, statusRow, ch, style)
	}
	// Draw right-aligned text
	startCol := width - len([]rune(right))
	if startCol < len([]rune(left)) {
		startCol = len([]rune(left))
	}
	for i, ch := range []rune(right) {
		col := startCol + i
		if col >= width {
			break
		}
		screen.SetContent(col, statusRow, ch, style)
	}
}
