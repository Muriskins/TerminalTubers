package main

import (
	"fmt"
	"os"
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
	fmt.Println("TerminalTubers: 4 frames loaded")
}