package main

import (
	"sync"

	"github.com/gdamore/tcell/v2"
)

// Terminal wraps a tcell screen and provides rendering primitives for
// displaying ASCII-art frames. It handles alternate-screen setup, cursor
// hiding, and guaranteed terminal state restoration on every exit path.
type Terminal struct {
	screen tcell.Screen
	once   sync.Once
}

// NewTerminal initializes the tcell screen: enters alternate screen, hides
// the cursor, and enables raw mode. The returned Terminal MUST be closed
// via Close() to restore the terminal. Errors are surfaced, not swallowed.
func NewTerminal() (*Terminal, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.Clear()
	screen.HideCursor()
	return &Terminal{screen: screen}, nil
}

// Close restores the terminal to its original state. Safe to call multiple
// times — a sync.Once ensures Fini() runs exactly once. This is guaranteed
// to run on every exit path via defer in main.
func (t *Terminal) Close() {
	t.once.Do(func() {
		t.screen.Fini()
	})
}

// Size returns the terminal dimensions in cells (width, height).
func (t *Terminal) Size() (int, int) {
	return t.screen.Size()
}

// RenderFrame draws a frame centered on screen. If the terminal is smaller
// than the frame, top-left clipping is applied — only cells that fit within
// the visible area are drawn. Returns nil always for now; the error return
// is reserved for future use (e.g., out-of-bounds writes).
func (t *Terminal) RenderFrame(frame []string) {
	screenW, screenH := t.screen.Size()

	frameH := len(frame)
	if frameH == 0 {
		return
	}

	// Find the widest line in the frame.
	frameW := 0
	for _, line := range frame {
		if len(line) > frameW {
			frameW = len(line)
		}
	}

	// Compute top-left origin to center the frame.
	// Negative origin means the frame is wider/taller than the screen —
	// we clip by shifting the origin to 0 and skipping off-screen portions.
	originX := (screenW - frameW) / 2
	originY := (screenH - frameH) / 2

	for row, line := range frame {
		screenY := originY + row
		// Skip rows entirely above the visible area.
		if screenY < 0 {
			continue
		}
		// Stop once we've gone past the bottom edge.
		if screenY >= screenH {
			break
		}
		for col, ch := range line {
			screenX := originX + col
			if screenX < 0 {
				continue
			}
			if screenX >= screenW {
				break
			}
			t.screen.SetContent(screenX, screenY, ch, nil, tcell.StyleDefault)
		}
	}
	t.screen.Show()
}

// PollEvent blocks until the next terminal event (key press, resize, etc.)
// and returns it. The caller inspects the event type to determine action.
func (t *Terminal) PollEvent() tcell.Event {
	return t.screen.PollEvent()
}
