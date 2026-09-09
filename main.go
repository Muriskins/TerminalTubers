package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

func main() {
	frames, err := LoadFrames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TerminalTubers: failed to load frames: %v\n", err)
		os.Exit(1)
	}
	if len(frames) != 4 {
		fmt.Fprintf(os.Stderr, "TerminalTubers: expected 4 frames, got %d\n", len(frames))
		os.Exit(1)
	}

	term, err := NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TerminalTubers: failed to initialize terminal: %v\n", err)
		os.Exit(1)
	}
	defer term.Close()

	// Render the idle frame once to start.
	term.RenderFrame(frames[0])

	// Event loop: wait for q/Q to quit, or resize to re-render.
	for {
		ev := term.PollEvent()
		switch e := ev.(type) {
		case *tcell.EventKey:
			switch {
			case e.Key() == tcell.KeyEscape, e.Rune() == 'q' || e.Rune() == 'Q':
				return
			}
		case *tcell.EventResize:
			term.screen.Sync()
			term.RenderFrame(frames[0])
		}
	}
}
