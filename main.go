package main

import (
	"fmt"
	"os"
	"strings"

	"goomacs/term"
)

const tabWidth = 8

// splitMode tracks the current window split orientation.
// It is "vertical" (top/bottom, C-x 2) or "horizontal" (side-by-side, C-x 3).
var splitMode = "vertical"

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

// Window represents a view into a buffer on screen.
type Window struct {
	Buffer       *Buffer
	ScrollOffset int
	StartRow     int // first screen row
	Height       int // total rows including status line
	StartCol     int // first screen column (used in horizontal split)
	Width        int // columns allocated to this window
}

// ViewHeight returns the number of rows available for text (excluding the status line).
func (w *Window) ViewHeight() int {
	h := w.Height - 1
	if h < 1 {
		h = 1
	}
	return h
}

// AdjustScroll ensures the cursor is visible within this window's viewport.
func (w *Window) AdjustScroll() {
	viewH := w.ViewHeight()
	if w.Buffer.CursorR < w.ScrollOffset {
		w.ScrollOffset = w.Buffer.CursorR
	}
	if w.Buffer.CursorR >= w.ScrollOffset+viewH {
		w.ScrollOffset = w.Buffer.CursorR - viewH + 1
	}
}

// ScrollDown scrolls the window down by one page.
func (w *Window) ScrollDown() {
	viewH := w.ViewHeight()
	w.ScrollOffset += viewH
	maxOffset := len(w.Buffer.Lines) - 1
	if w.ScrollOffset > maxOffset {
		w.ScrollOffset = maxOffset
	}
	if w.Buffer.CursorR < w.ScrollOffset {
		w.Buffer.CursorR = w.ScrollOffset
		if w.Buffer.CursorC > len(w.Buffer.Lines[w.Buffer.CursorR]) {
			w.Buffer.CursorC = len(w.Buffer.Lines[w.Buffer.CursorR])
		}
	}
}

// ScrollUp scrolls the window up by one page.
func (w *Window) ScrollUp() {
	viewH := w.ViewHeight()
	w.ScrollOffset -= viewH
	if w.ScrollOffset < 0 {
		w.ScrollOffset = 0
	}
	lastVisible := w.ScrollOffset + viewH - 1
	if lastVisible >= len(w.Buffer.Lines) {
		lastVisible = len(w.Buffer.Lines) - 1
	}
	if w.Buffer.CursorR > lastVisible {
		w.Buffer.CursorR = lastVisible
		if w.Buffer.CursorC > len(w.Buffer.Lines[w.Buffer.CursorR]) {
			w.Buffer.CursorC = len(w.Buffer.Lines[w.Buffer.CursorR])
		}
	}
}

// recalcWindows distributes available screen space evenly among windows.
// The last row is reserved for the message line.
// In vertical mode, windows stack top-to-bottom with full width.
// In horizontal mode, windows are placed side-by-side (handled in later stories).
func recalcWindows(windows []*Window, screenWidth, screenHeight int) {
	n := len(windows)
	if splitMode == "horizontal" && n > 1 {
		// Horizontal (side-by-side) layout: distribute width evenly.
		// Reserve 1 column between each pair of adjacent windows for separators.
		available := screenWidth - (n - 1)
		if available < n {
			available = n
		}
		baseW := available / n
		extra := available % n
		col := 0
		for i, w := range windows {
			w.StartCol = col
			w.Width = baseW
			if i < extra {
				w.Width++
			}
			w.StartRow = 0
			w.Height = screenHeight - 1 // full height minus message line
			col += w.Width + 1          // +1 for separator column
		}
	} else {
		// Vertical (top/bottom) layout: distribute height evenly.
		available := screenHeight - 1 // reserve 1 row for message line
		if available < n {
			available = n
		}
		baseH := available / n
		extra := available % n
		row := 0
		for i, w := range windows {
			w.StartRow = row
			w.Height = baseH
			if i < extra {
				w.Height++
			}
			w.StartCol = 0
			w.Width = screenWidth
			row += w.Height
		}
	}
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

	// Create initial window showing the active buffer.
	windows := []*Window{{Buffer: buf, ScrollOffset: 0}}
	activeWindowIdx := 0

	screen := term.NewTerminal()
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "error initializing screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	screenWidth, screenHeight := screen.Size()
	recalcWindows(windows, screenWidth, screenHeight)

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
		screen.Clear()
		activeWin := windows[activeWindowIdx]
		for i, win := range windows {
			if i == activeWindowIdx {
				win.AdjustScroll()
			}
			isActive := i == activeWindowIdx
			if isActive && searchMode && searchHasMatch {
				drawWindowContent(screen, win, searchHighlight{
					active:   true,
					matchR:   searchMatchR,
					matchC:   searchMatchC,
					queryLen: len(searchQuery),
				})
			} else {
				drawWindowContent(screen, win, searchHighlight{})
			}
			drawWindowStatusLine(screen, win, isActive)
		}
		// Draw vertical separators between horizontal windows.
		if splitMode == "horizontal" && len(windows) > 1 {
			for i := 0; i < len(windows)-1; i++ {
				sepCol := windows[i].StartCol + windows[i].Width
				for row := 0; row < screenHeight-1; row++ {
					screen.SetContent(sepCol, row, '│', term.StyleDefault)
				}
			}
		}
		drawMessageLine(screen, message)
		if minibufferMode {
			cursorX := len([]rune(minibufferPrompt)) + len(minibufferInput)
			screen.ShowCursor(cursorX, screenHeight-1)
		} else {
			screen.ShowCursor(
				activeWin.StartCol+bufColToVisualCol(activeWin.Buffer.Lines[activeWin.Buffer.CursorR], activeWin.Buffer.CursorC),
				activeWin.Buffer.CursorR-activeWin.ScrollOffset+activeWin.StartRow,
			)
		}
		screen.Show()
	}

	redraw()

	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *term.KeyEvent:
			screenWidth, screenHeight = screen.Size()
			recalcWindows(windows, screenWidth, screenHeight)
			message = "" // clear message on next key

			activeWin := windows[activeWindowIdx]

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
				case term.KeyEnter, term.KeyCtrlJ:
					// Accept search result, exit search mode
					searchMode = false
					searchHasMatch = false
					message = ""
				case term.KeyBackspace, term.KeyBackspace2, term.KeyCtrlH:
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
				case term.KeyEnter, term.KeyCtrlJ:
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
				case term.KeyBackspace, term.KeyBackspace2, term.KeyCtrlH:
					if len(minibufferInput) > 0 {
						minibufferInput = minibufferInput[:len(minibufferInput)-1]
					}
					message = minibufferPrompt + string(minibufferInput)
				case term.KeyTab:
					// Tab completion for Find file
					if minibufferPrompt == "Find file: " {
						input := string(minibufferInput)
						dir := "."
						prefix := input
						if idx := strings.LastIndex(input, "/"); idx >= 0 {
							dir = input[:idx]
							if dir == "" {
								dir = "/"
							}
							prefix = input[idx+1:]
						}
						entries, err := os.ReadDir(dir)
						if err == nil {
							var matches []string
							for _, e := range entries {
								name := e.Name()
								if strings.HasPrefix(name, prefix) {
									if e.IsDir() {
										matches = append(matches, name+"/")
									} else {
										matches = append(matches, name)
									}
								}
							}
							if len(matches) == 1 {
								if dir == "." {
									minibufferInput = []rune(matches[0])
								} else {
									minibufferInput = []rune(dir + "/" + matches[0])
								}
							} else if len(matches) > 1 {
								common := longestCommonPrefix(matches)
								if dir == "." {
									minibufferInput = []rune(common)
								} else {
									minibufferInput = []rune(dir + "/" + common)
								}
								message = minibufferPrompt + string(minibufferInput) + " [" + strings.Join(matches, " ") + "]"
								redraw()
								continue
							}
						}
						message = minibufferPrompt + string(minibufferInput)
					}
				case term.KeyRune:
					minibufferInput = append(minibufferInput, ev.Rune())
					message = minibufferPrompt + string(minibufferInput)
				default:
					// All other keys are ignored in minibuffer mode
					message = minibufferPrompt + string(minibufferInput)
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
						lines = append(lines, fmt.Sprintf("%s%s %s", marker, modFlag, name))
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
					activeWin.Buffer = buf
					activeWin.ScrollOffset = 0
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
								activeWin.Buffer = buf
								activeWin.ScrollOffset = 0
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
								newBuf.Highlight = NewHighlighter(input)
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
						activeWin.Buffer = buf
						activeWin.ScrollOffset = 0
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
								activeWin.Buffer = buf
								activeWin.ScrollOffset = 0
								return
							}
							// Search for existing buffer by name
							for i, b := range buffers {
								if b.Filename == input {
									previousBufferIdx = activeBufferIdx
									activeBufferIdx = i
									buf = buffers[activeBufferIdx]
									activeWin.Buffer = buf
									activeWin.ScrollOffset = 0
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
							activeWin.Buffer = buf
							activeWin.ScrollOffset = 0
						}
						message = minibufferPrompt
					case '2':
					// Split current window vertically (top/bottom)
					if splitMode == "horizontal" && len(windows) > 1 {
						message = "Cannot split vertically while in horizontal split mode"
					} else {
						newWin := &Window{
							Buffer:       activeWin.Buffer,
							ScrollOffset: activeWin.ScrollOffset,
						}
						if len(windows) == 1 {
							splitMode = "vertical"
						}
						// Insert new window after the active one
						idx := activeWindowIdx + 1
						windows = append(windows, nil)
						copy(windows[idx+1:], windows[idx:])
						windows[idx] = newWin
						recalcWindows(windows, screenWidth, screenHeight)
					}
				case '3':
					// Split current window horizontally (side-by-side)
					if splitMode == "vertical" && len(windows) > 1 {
						message = "Cannot split horizontally while in vertical split mode"
					} else {
						// Check minimum width: each window needs at least 10 columns,
						// plus 1 separator column between each pair of windows.
						newCount := len(windows) + 1
						availableWidth := screenWidth - (newCount - 1) // subtract separator columns
						if availableWidth/newCount < 10 {
							message = "Window too narrow to split"
						} else {
							newWin := &Window{
								Buffer:       activeWin.Buffer,
								ScrollOffset: activeWin.ScrollOffset,
							}
							if len(windows) == 1 {
								splitMode = "horizontal"
							}
							// Insert new window after the active one
							idx := activeWindowIdx + 1
							windows = append(windows, nil)
							copy(windows[idx+1:], windows[idx:])
							windows[idx] = newWin
							recalcWindows(windows, screenWidth, screenHeight)
						}
					}
				case 'o':
					// Move focus to next window (cycle)
					if len(windows) > 1 {
						activeWindowIdx = (activeWindowIdx + 1) % len(windows)
						activeWin = windows[activeWindowIdx]
						buf = activeWin.Buffer
						for i, b := range buffers {
							if b == activeWin.Buffer {
								previousBufferIdx = activeBufferIdx
								activeBufferIdx = i
								break
							}
						}
					}
				case '0':
					// Close current window (no-op if only one window)
					if len(windows) > 1 {
						windows = append(windows[:activeWindowIdx], windows[activeWindowIdx+1:]...)
						if activeWindowIdx >= len(windows) {
							activeWindowIdx = len(windows) - 1
						}
						if len(windows) == 1 {
							splitMode = "vertical"
						}
						recalcWindows(windows, screenWidth, screenHeight)
						activeWin = windows[activeWindowIdx]
						buf = activeWin.Buffer
						for i, b := range buffers {
							if b == activeWin.Buffer {
								previousBufferIdx = activeBufferIdx
								activeBufferIdx = i
								break
							}
						}
					}
				case '1':
					// Close all windows except current
					if len(windows) > 1 {
						windows = []*Window{activeWin}
						activeWindowIdx = 0
						splitMode = "vertical"
						recalcWindows(windows, screenWidth, screenHeight)
					}
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
								killedBuf := buffers[targetIdx]
								killedName := killedBuf.Filename
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
									// Update all windows displaying the killed buffer
									for _, w := range windows {
										if w.Buffer == killedBuf {
											w.Buffer = buf
											w.ScrollOffset = 0
										}
									}
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
								// Update all windows displaying the killed buffer
								for _, w := range windows {
									if w.Buffer == killedBuf {
										w.Buffer = buf
										w.ScrollOffset = 0
									}
								}
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
				activeWin.ScrollDown()
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
			case term.KeyEnter, term.KeyCtrlJ:
				if buf.Filename == "*Buffer List*" {
					// Line index maps directly to buffers index.
					targetIdx := buf.CursorR
					if targetIdx >= 0 && targetIdx < len(buffers) {
						target := buffers[targetIdx]
						if target.Filename != "*Buffer List*" {
							previousBufferIdx = activeBufferIdx
							activeBufferIdx = targetIdx
							buf = buffers[activeBufferIdx]
							activeWin.Buffer = buf
							activeWin.ScrollOffset = 0
							message = fmt.Sprintf("Switch to buffer: %s", target.Filename)
						}
					}
				} else {
					buf.SaveUndo()
					buf.InsertNewline()
				}
			case term.KeyBackspace, term.KeyBackspace2, term.KeyCtrlH:
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
						activeWin.ScrollUp()
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
						activeWin.ScrollUp()
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
			screenWidth, screenHeight = screen.Size()
			// In horizontal mode, close excess rightmost windows if terminal is too narrow.
			if splitMode == "horizontal" && len(windows) > 1 {
				for len(windows) > 1 {
					n := len(windows)
					available := screenWidth - (n - 1) // subtract separator columns
					if available/n >= 10 {
						break
					}
					// Close the rightmost window
					windows = windows[:n-1]
					if activeWindowIdx >= len(windows) {
						activeWindowIdx = len(windows) - 1
					}
					message = "Window too narrow; closed rightmost window"
				}
				if len(windows) == 1 {
					splitMode = "vertical"
				}
				activeWin := windows[activeWindowIdx]
				buf = activeWin.Buffer
				for i, b := range buffers {
					if b == activeWin.Buffer {
						previousBufferIdx = activeBufferIdx
						activeBufferIdx = i
						break
					}
				}
			}
			recalcWindows(windows, screenWidth, screenHeight)
			redraw()
		}
	}
}

// searchHighlight holds the state for highlighting a search match during rendering.
type searchHighlight struct {
	active   bool
	matchR   int
	matchC   int
	queryLen int
}

// drawWindowContent renders a window's buffer content within its row range.
// Content is drawn starting at win.StartCol and constrained to win.Width columns.
func drawWindowContent(screen term.Screen, win *Window, sh searchHighlight) {
	winWidth := win.Width
	viewH := win.ViewHeight()
	buf := win.Buffer

	// Re-highlight if content changed.
	if buf.HighlightDirty && buf.Highlight != nil {
		buf.Highlight.Highlight(buf.Lines)
		buf.HighlightDirty = false
	}

	for row := 0; row < viewH; row++ {
		screenRow := win.StartRow + row
		bufRow := row + win.ScrollOffset
		if bufRow >= len(buf.Lines) {
			break
		}
		line := buf.Lines[bufRow]
		visualCol := 0
		for bufCol := 0; bufCol < len(line) && visualCol < winWidth; bufCol++ {
			style := term.StyleDefault
			if buf.Highlight != nil {
				style = buf.Highlight.StyleAt(bufRow, bufCol)
			}
			if buf.InRegion(bufRow, bufCol) {
				style = style.Reverse(true)
			}
			if sh.active && bufRow == sh.matchR && bufCol >= sh.matchC && bufCol < sh.matchC+sh.queryLen {
				style = style.Reverse(true)
			}
			if line[bufCol] == '\t' {
				nextStop := visualCol + tabWidth - (visualCol%tabWidth)
				for visualCol < nextStop && visualCol < winWidth {
					screen.SetContent(win.StartCol+visualCol, screenRow, ' ', style)
					visualCol++
				}
			} else {
				screen.SetContent(win.StartCol+visualCol, screenRow, line[bufCol], style)
				visualCol++
			}
		}
	}
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

// drawWindowStatusLine renders a window's status line.
// Active windows use reverse video; inactive windows use dashes with default style.
// The status line is constrained to the window's column range (StartCol to StartCol+Width).
func drawWindowStatusLine(screen term.Screen, win *Window, active bool) {
	winWidth := win.Width
	statusRow := win.StartRow + win.Height - 1

	buf := win.Buffer
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
	fillChar := ' '
	if !active {
		style = term.StyleDefault
		fillChar = '-'
	}

	// Fill entire status line within window's column range
	for col := 0; col < winWidth; col++ {
		screen.SetContent(win.StartCol+col, statusRow, fillChar, style)
	}
	// Draw left-aligned text
	for i, ch := range []rune(left) {
		if i >= winWidth {
			break
		}
		screen.SetContent(win.StartCol+i, statusRow, ch, style)
	}
	// Draw right-aligned text
	startCol := winWidth - len([]rune(right))
	if startCol < len([]rune(left)) {
		startCol = len([]rune(left))
	}
	for i, ch := range []rune(right) {
		col := startCol + i
		if col >= winWidth {
			break
		}
		screen.SetContent(win.StartCol+col, statusRow, ch, style)
	}
}

// longestCommonPrefix returns the longest common prefix of the given strings.
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}
