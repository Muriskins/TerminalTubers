package main

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed ascii_art/idle.txt ascii_art/tolk0.txt ascii_art/tolk1.txt ascii_art/tolk2.txt
var framesFS embed.FS

// Frame names in load order: idle pose first, then the three talking frames.
var frameNames = []string{
	"ascii_art/idle.txt",
	"ascii_art/tolk0.txt",
	"ascii_art/tolk1.txt",
	"ascii_art/tolk2.txt",
}

// LoadFrames reads the four embedded ASCII art frames and returns them as
// slices of lines. The first frame is the idle pose; the remaining three are
// the talking frames.
func LoadFrames() ([][]string, error) {
	frames := make([][]string, 0, len(frameNames))
	for _, name := range frameNames {
		data, err := framesFS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("load frame %s: %w", name, err)
		}
		frames = append(frames, splitLines(string(data)))
	}
	return frames, nil
}

// splitLines splits embedded frame content into lines, tolerating CRLF line
// endings and a single trailing newline.
func splitLines(content string) []string {
	lines := strings.Split(content, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}