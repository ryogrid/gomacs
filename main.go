package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

func main() {
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

	screen.Clear()
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
		}
	}
}
