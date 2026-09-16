package main

import (
	"fmt"
	"math/rand"
)

// Avatar manages frame storage and instant (non-animated) frame switching.
// Animation transitions (Scramble, Collapse, Reveal, Smart) will be added
// in tasks 3.2/3.3 on top of this base.
type Avatar struct {
	frames  [][]string // all loaded frames [idle, tolk0, tolk1, tolk2]
	current []string   // currently displayed frame
}

// NewAvatar creates an Avatar from the loaded frames. The first frame is
// considered the idle frame. Frames must be non-empty.
func NewAvatar(frames [][]string) (*Avatar, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("avatar: at least one frame required")
	}
	return &Avatar{frames: frames}, nil
}

// Render performs an instant frame switch: sets the target as the current
// frame and renders it via the Terminal. This is the animation-disabled path
// (FR-013): frame switches are instant. Animation hooks will be added in
// tasks 3.2/3.3.
func (a *Avatar) Render(term *Terminal, target []string) {
	a.current = target
	term.RenderFrame(target)
}

// CurrentFrame returns the currently displayed frame. Returns nil if no frame
// has been rendered yet.
func (a *Avatar) CurrentFrame() []string {
	return a.current
}

// Frames returns all loaded frames for the state machine to index into.
// Convention: frames[0] = idle, frames[1:] = talking frames.
func (a *Avatar) Frames() [][]string {
	return a.frames
}

// PlayAnimation switches to target, animating the transition when animations
// are enabled and a current frame exists. When animations are disabled or no
// frame has been rendered yet, the switch is instant (FR-013). After either
// path the target becomes the current frame.
func (a *Avatar) PlayAnimation(term *Terminal, target []string, cfg Config, rng *rand.Rand) {
	if cfg.AnimationsEnabled && a.current != nil {
		AnimateTransition(term, a.current, target, cfg, rng)
	} else {
		term.RenderFrame(target)
	}
	a.current = target
}
