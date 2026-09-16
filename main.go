package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

// pickTalkingFrame returns a random talking frame from frames[1:].
// frames[0] is the idle frame; talking frames are indices 1..len(frames)-1.
// Requires len(frames) >= 2.
func pickTalkingFrame(frames [][]string, rng *rand.Rand) []string {
	idx := rng.Intn(len(frames)-1) + 1
	return frames[idx]
}

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

	// Create the avatar from the loaded frames. The first frame is the idle
	// pose; the remaining three are the talking frames.
	avatar, err := NewAvatar(frames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TerminalTubers: %v\n", err)
		os.Exit(1)
	}

	// Render the idle frame once to start.
	avatar.Render(term, frames[0])

	cfg := DefaultConfig()
	sm := NewStateMachine(time.Duration(cfg.IdleDelayMs)*time.Millisecond, 3)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// PollEvent blocks, so drain it in a goroutine into a buffered channel.
	// The goroutine exits when PollEvent returns nil, which happens once the
	// terminal is closed (deferred Close on return).
	eventCh := make(chan tcell.Event, 8)
	go func() {
		for {
			ev := term.PollEvent()
			if ev == nil {
				return
			}
			eventCh <- ev
		}
	}()

	// 50ms state-machine tick (FR-028) — the loop evaluates voice and
	// state at 20Hz. Display updates are transition-driven (render-on-
	// transition, matching the Rust original); SC-009's literal
	// >=15 display-updates/sec floor is not met during sustained
	// silence and is tracked separately.
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-eventCh:
			switch e := ev.(type) {
			case *tcell.EventKey:
				switch {
				case e.Key() == tcell.KeyEscape, e.Rune() == 'q' || e.Rune() == 'Q':
					return
				case e.Rune() == 's' || e.Rune() == 'S':
					// Settings menu is owned by Phase 5; the key binding is
					// reserved here so it is recognized but intentionally a
					// no-op until the menu lands.
				}
			case *tcell.EventResize:
				term.Sync()
				term.RenderFrame(avatar.CurrentFrame())
			}
		case <-ticker.C:
			// Surface mid-session capture failures once (FR-026). A failed
			// capture reports RMS 0.0, which the state machine treats as "no
			// voice" — the avatar naturally returns to idle, so no extra
			// failure logic is needed here.
			if capture != nil && !captureFailureSurfaced {
				if err := capture.Err(); err != nil {
					fmt.Fprintf(os.Stderr, "TerminalTubers: audio capture failed: %v\n", err)
					captureFailureSurfaced = true
				}
			}

			// RMS() returns float32 while Threshold is float64 — the explicit
			// cast is required for the strict > comparison.
			voice := capture != nil && capture.RMS() > float32(cfg.Threshold)
			switch sm.Tick(voice, time.Now()) {
			case ActionStartTalking:
				// Random talking frame chosen once per transition (FR-011);
				// frames[0] is idle, frames[1:] are the talking frames.
				avatar.PlayAnimation(term, pickTalkingFrame(frames, rng), cfg, rng)
			case ActionReturnToIdle:
				avatar.PlayAnimation(term, frames[0], cfg, rng)
			}
		}
	}
}
