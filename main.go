package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

func main() {
	// Load buffer from file argument or create empty buffer.
	var buf *Buffer
	if len(os.Args) > 1 {
		var err error
		buf, err = NewBufferFromFile(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening file: %v\n", err)
			os.Exit(1)
		}
	} else {
		buf = NewBuffer()
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating screen: %v\n", err)
		os.Exit(1)
	}

	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "error initializing screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	_, screenHeight := screen.Size()
	viewHeight := textAreaHeight(screenHeight)
	buf.AdjustScroll(viewHeight)
	drawBuffer(screen, buf)
	screen.ShowCursor(buf.CursorC, buf.CursorR-buf.ScrollOffset)
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
		screen.ShowCursor(buf.CursorC, buf.CursorR-buf.ScrollOffset)
		screen.Show()
	}

	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			_, screenHeight = screen.Size()
			viewHeight = textAreaHeight(screenHeight)
			message = "" // clear message on next key

			// Handle search mode
			if searchMode {
				switch ev.Key() {
				case tcell.KeyCtrlS:
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
				case tcell.KeyCtrlR:
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
				case tcell.KeyCtrlG:
					// Cancel search, restore original position
					buf.CursorR = searchOrigR
					buf.CursorC = searchOrigC
					searchMode = false
					searchHasMatch = false
					message = "Quit"
				case tcell.KeyEnter:
					// Accept search result, exit search mode
					searchMode = false
					searchHasMatch = false
					message = ""
				case tcell.KeyBackspace, tcell.KeyBackspace2:
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
				case tcell.KeyRune:
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

			// Reset consecutive kill tracking for non-kill keys
			if ev.Key() != tcell.KeyCtrlK {
				buf.ClearLastKill()
			}

			// Reset quit warning unless we're in a C-x prefix sequence
			if ev.Key() != tcell.KeyCtrlX && !prefixCx {
				quitWarned = false
			}

			// Handle C-x prefix second key
			if prefixCx {
				prefixCx = false
				switch ev.Key() {
				case tcell.KeyCtrlS:
					if err := buf.Save(); err != nil {
						if err == errNoFilename {
							message = "No file name"
						} else {
							message = fmt.Sprintf("Error saving: %v", err)
						}
					} else {
						message = fmt.Sprintf("Saved %s", buf.Filename)
					}
				case tcell.KeyCtrlC:
					if buf.Modified && !quitWarned {
						message = "Modified buffers exist; exit anyway? (C-x C-c to confirm)"
						quitWarned = true
					} else {
						return
					}
				}
				redraw()
				continue
			}

			switch ev.Key() {
			case tcell.KeyCtrlX:
				prefixCx = true
				message = "C-x-"
				redraw()
				continue
			case tcell.KeyCtrlF:
				buf.MoveForward()
			case tcell.KeyCtrlB:
				buf.MoveBackward()
			case tcell.KeyCtrlN:
				buf.MoveDown()
			case tcell.KeyCtrlP:
				buf.MoveUp()
			case tcell.KeyCtrlA:
				buf.MoveBeginningOfLine()
			case tcell.KeyCtrlE:
				buf.MoveEndOfLine()
			case tcell.KeyCtrlS:
				searchMode = true
				searchForward = true
				searchQuery = nil
				searchOrigR = buf.CursorR
				searchOrigC = buf.CursorC
				searchHasMatch = false
				message = "I-search: "
				redraw()
				continue
			case tcell.KeyCtrlR:
				searchMode = true
				searchForward = false
				searchQuery = nil
				searchOrigR = buf.CursorR
				searchOrigC = buf.CursorC
				searchHasMatch = false
				message = "I-search backward: "
				redraw()
				continue
			case tcell.KeyCtrlV:
				buf.ScrollDown(viewHeight)
			case tcell.KeyCtrlSpace:
				buf.SetMark()
				message = "Mark set"
			case tcell.KeyCtrlG:
				buf.DeactivateMark()
				message = ""
			case tcell.KeyCtrlW:
				buf.SaveUndo()
				buf.KillRegion()
			case tcell.KeyCtrlUnderscore:
				if buf.Undo() {
					message = "Undo"
				} else {
					message = "No further undo information"
				}
			case tcell.KeyCtrlK:
				buf.SaveUndo()
				buf.KillLine()
				redraw()
				continue
			case tcell.KeyCtrlY:
				buf.SaveUndo()
				buf.Yank()
			case tcell.KeyCtrlD:
				buf.SaveUndo()
				buf.DeleteChar()
			case tcell.KeyEnter:
				buf.SaveUndo()
				buf.InsertNewline()
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				buf.SaveUndo()
				buf.Backspace()
			case tcell.KeyRight:
				buf.MoveForward()
			case tcell.KeyLeft:
				buf.MoveBackward()
			case tcell.KeyDown:
				buf.MoveDown()
			case tcell.KeyUp:
				buf.MoveUp()
			case tcell.KeyRune:
				if ev.Modifiers()&tcell.ModAlt != 0 {
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
			case tcell.KeyEsc:
				// Handle Esc-prefixed sequences (for terminals that send Esc then key)
				nextEv := screen.PollEvent()
				if kev, ok := nextEv.(*tcell.EventKey); ok && kev.Key() == tcell.KeyRune {
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
		case *tcell.EventResize:
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
func drawBuffer(screen tcell.Screen, buf *Buffer) {
	drawBufferWithSearch(screen, buf, searchHighlight{})
}

// drawBufferWithSearch renders the buffer with optional search match highlighting.
func drawBufferWithSearch(screen tcell.Screen, buf *Buffer, sh searchHighlight) {
	screen.Clear()
	width, height := screen.Size()
	textH := textAreaHeight(height)

	for row := 0; row < textH; row++ {
		bufRow := row + buf.ScrollOffset
		if bufRow >= len(buf.Lines) {
			break
		}
		line := buf.Lines[bufRow]
		for col := 0; col < width && col < len(line); col++ {
			style := tcell.StyleDefault
			if buf.InRegion(bufRow, col) {
				style = style.Reverse(true)
			}
			if sh.active && bufRow == sh.matchR && col >= sh.matchC && col < sh.matchC+sh.queryLen {
				style = style.Reverse(true)
			}
			screen.SetContent(col, row, line[col], nil, style)
		}
	}

	drawStatusLine(screen, buf)
}

// drawMessageLine renders a message on the last row of the screen.
func drawMessageLine(screen tcell.Screen, msg string) {
	width, height := screen.Size()
	if height < 1 || msg == "" {
		return
	}
	msgRow := height - 1
	for i, ch := range []rune(msg) {
		if i >= width {
			break
		}
		screen.SetContent(i, msgRow, ch, nil, tcell.StyleDefault)
	}
}

// drawStatusLine renders the status line on the second-to-last row with reverse video.
func drawStatusLine(screen tcell.Screen, buf *Buffer) {
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
	pos := fmt.Sprintf("Line %d, Col %d", buf.CursorR+1, buf.CursorC+1)
	left := fmt.Sprintf(" %s%s", name, mod)
	right := fmt.Sprintf("%s ", pos)

	style := tcell.StyleDefault.Reverse(true)

	// Fill entire status line with reverse video
	for col := 0; col < width; col++ {
		screen.SetContent(col, statusRow, ' ', nil, style)
	}
	// Draw left-aligned text
	for i, ch := range []rune(left) {
		if i >= width {
			break
		}
		screen.SetContent(i, statusRow, ch, nil, style)
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
		screen.SetContent(col, statusRow, ch, nil, style)
	}
}
