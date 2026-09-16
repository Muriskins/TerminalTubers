package main

import "time"

// State represents the avatar's current speaking state.
type State int

const (
	StateIdle State = iota
	StateTalking
)

// Action is the state machine's response to a single tick.
type Action int

const (
	ActionNone Action = iota
	ActionStartTalking
	ActionReturnToIdle
)

// StateMachine implements the idle/talking voice state machine, ported 1:1
// from the Rust main loop (rust-legacy src/main.rs). It is pure and
// testable: Tick receives the current voice detection and wall-clock time
// and returns the action the caller should perform.
type StateMachine struct {
	state        State
	voiceConfirm int
	lastVoice    time.Time
	idleDelay    time.Duration
	debounce     int
}

// NewStateMachine creates a state machine in the idle state. idleDelay is
// the silence period (measured from the last voiced frame) after which a
// talking avatar returns to idle; debounce is the number of consecutive
// voiced frames required to start talking.
func NewStateMachine(idleDelay time.Duration, debounce int) *StateMachine {
	return &StateMachine{
		state:     StateIdle,
		idleDelay: idleDelay,
		debounce:  debounce,
	}
}

// State returns the current state.
func (sm *StateMachine) State() State {
	return sm.state
}

// Tick advances the state machine by one tick. voice reports whether the
// current RMS exceeds the threshold (strict >). Semantics ported 1:1 from
// Rust:
//
//   - voiceConfirm increments on every voiced frame and resets to 0 on any
//     non-voiced frame
//   - idle→talking fires when voiceConfirm >= debounce and not already
//     talking; the caller picks the random talking frame once per transition
//   - lastVoice refreshes on EVERY voiced frame, so the idle timer measures
//     silence since the last voice, not since talk started
//   - talking→idle fires on a non-voiced frame only when the silence since
//     lastVoice strictly exceeds idleDelay
func (sm *StateMachine) Tick(voice bool, now time.Time) Action {
	if voice {
		sm.voiceConfirm++
		sm.lastVoice = now
		if sm.voiceConfirm >= sm.debounce && sm.state == StateIdle {
			sm.state = StateTalking
			return ActionStartTalking
		}
		return ActionNone
	}

	sm.voiceConfirm = 0
	if sm.state == StateTalking && now.Sub(sm.lastVoice) > sm.idleDelay {
		sm.state = StateIdle
		return ActionReturnToIdle
	}
	return ActionNone
}
