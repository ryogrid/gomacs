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

	drawBuffer(screen, buf)
	screen.Show()

	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlC {
				return
			}
		case *tcell.EventResize:
			screen.Sync()
			drawBuffer(screen, buf)
			screen.Show()
		}
	}
}

// drawBuffer renders the buffer content onto the screen.
func drawBuffer(screen tcell.Screen, buf *Buffer) {
	screen.Clear()
	width, height := screen.Size()

	for row := 0; row < height && row < len(buf.Lines); row++ {
		line := buf.Lines[row]
		for col := 0; col < width && col < len(line); col++ {
			screen.SetContent(col, row, line[col], nil, tcell.StyleDefault)
		}
	}
}
