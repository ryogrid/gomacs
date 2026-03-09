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

	_, viewHeight := screen.Size()
	buf.AdjustScroll(viewHeight)
	drawBuffer(screen, buf)
	screen.ShowCursor(buf.CursorC, buf.CursorR-buf.ScrollOffset)
	screen.Show()

	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			_, viewHeight = screen.Size()
			switch ev.Key() {
			case tcell.KeyCtrlC:
				return
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
			case tcell.KeyCtrlV:
				buf.ScrollDown(viewHeight)
			case tcell.KeyCtrlD:
				buf.DeleteChar()
			case tcell.KeyEnter:
				buf.InsertNewline()
			case tcell.KeyBackspace, tcell.KeyBackspace2:
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
					case '<':
						buf.MoveBeginningOfBuffer()
					case '>':
						buf.MoveEndOfBuffer()
					}
				} else {
					buf.InsertChar(ev.Rune())
				}
			case tcell.KeyEsc:
				// Handle Esc-prefixed sequences (for terminals that send Esc then key)
				nextEv := screen.PollEvent()
				if kev, ok := nextEv.(*tcell.EventKey); ok && kev.Key() == tcell.KeyRune {
					switch kev.Rune() {
					case 'v':
						buf.ScrollUp(viewHeight)
					case '<':
						buf.MoveBeginningOfBuffer()
					case '>':
						buf.MoveEndOfBuffer()
					}
				}
			}
			buf.AdjustScroll(viewHeight)
			drawBuffer(screen, buf)
			screen.ShowCursor(buf.CursorC, buf.CursorR-buf.ScrollOffset)
			screen.Show()
		case *tcell.EventResize:
			screen.Sync()
			_, viewHeight = screen.Size()
			buf.AdjustScroll(viewHeight)
			drawBuffer(screen, buf)
			screen.ShowCursor(buf.CursorC, buf.CursorR-buf.ScrollOffset)
			screen.Show()
		}
	}
}

// drawBuffer renders the buffer content onto the screen, accounting for scroll offset.
func drawBuffer(screen tcell.Screen, buf *Buffer) {
	screen.Clear()
	width, height := screen.Size()

	for row := 0; row < height; row++ {
		bufRow := row + buf.ScrollOffset
		if bufRow >= len(buf.Lines) {
			break
		}
		line := buf.Lines[bufRow]
		for col := 0; col < width && col < len(line); col++ {
			screen.SetContent(col, row, line[col], nil, tcell.StyleDefault)
		}
	}
}
