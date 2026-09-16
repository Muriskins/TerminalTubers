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

	// Start microphone capture for the audio-reactive avatar. When no capture
	// device is available, degrade gracefully: keep the idle avatar and print
	// a user-visible notice (FR-025).
	captureFailureSurfaced := false
	capture, err := NewCapture()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TerminalTubers: audio capture unavailable: %v\n", err)
	} else if capture.Available() {
		if err := capture.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "TerminalTubers: failed to start audio capture: %v\n", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "TerminalTubers: no microphone found - running with idle avatar")
	}
	if capture != nil {
		defer capture.Close()
	}

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

		// Surface mid-session capture failures (FR-026): return to the idle
		// avatar and print a notice. The idle frame is already on screen, so
		// no re-render is needed here - the state machine (task 4.1) will
		// handle transitions.
		if capture != nil && !captureFailureSurfaced {
			if err := capture.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "TerminalTubers: audio capture failed: %v\n", err)
				captureFailureSurfaced = true
			}
		}
	}
}
