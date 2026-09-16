package main

import (
	"testing"
	"time"
)

// fixedNow returns a deterministic base time for tick sequences.
func fixedNow() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestStateMachine_InitialState(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	if got := sm.State(); got != StateIdle {
		t.Errorf("State() = %v, want StateIdle", got)
	}
}

func TestStateMachine_StartTalkingOnThirdConsecutiveVoice(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	// First two voiced ticks: still idle, no action.
	for i := 0; i < 2; i++ {
		if got := sm.Tick(true, now); got != ActionNone {
			t.Fatalf("Tick(%d) = %v, want ActionNone before debounce", i+1, got)
		}
		if got := sm.State(); got != StateIdle {
			t.Fatalf("State() after Tick(%d) = %v, want StateIdle", i+1, got)
		}
	}

	// Third consecutive voiced tick: the transition fires.
	if got := sm.Tick(true, now); got != ActionStartTalking {
		t.Errorf("Tick(3) = %v, want ActionStartTalking", got)
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() after transition = %v, want StateTalking", got)
	}
}

func TestStateMachine_VoiceConfirmResetsOnNonVoice(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	// Two voiced ticks, then one non-voiced tick resets the counter.
	sm.Tick(true, now)
	sm.Tick(true, now)
	sm.Tick(false, now)

	// Two more voiced ticks must NOT fire the transition (counter was reset).
	sm.Tick(true, now)
	if got := sm.Tick(true, now); got != ActionNone {
		t.Errorf("Tick after reset = %v, want ActionNone (debounce counter reset)", got)
	}
	if got := sm.State(); got != StateIdle {
		t.Errorf("State() = %v, want StateIdle", got)
	}
}

func TestStateMachine_NoStartTalkingWhileTalking(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	// Reach the talking state.
	sm.Tick(true, now)
	sm.Tick(true, now)
	if got := sm.Tick(true, now); got != ActionStartTalking {
		t.Fatalf("Tick(3) = %v, want ActionStartTalking", got)
	}

	// Continued voice while talking must not re-fire the transition (FR-011).
	for i := 0; i < 5; i++ {
		if got := sm.Tick(true, now); got != ActionNone {
			t.Fatalf("Tick while talking = %v, want ActionNone (FR-011)", got)
		}
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() = %v, want StateTalking", got)
	}
}

func TestStateMachine_ReturnToIdleStrictGreater(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	start := fixedNow()

	// Reach the talking state; lastVoice = start.
	sm.Tick(true, start)
	sm.Tick(true, start)
	if got := sm.Tick(true, start); got != ActionStartTalking {
		t.Fatalf("Tick(3) = %v, want ActionStartTalking", got)
	}

	// Silence for exactly idleDelay: still talking (strict >).
	if got := sm.Tick(false, start.Add(500*time.Millisecond)); got != ActionNone {
		t.Errorf("Tick at exactly idleDelay = %v, want ActionNone (strict >)", got)
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() at exactly idleDelay = %v, want StateTalking", got)
	}

	// One more millisecond of silence: return to idle.
	if got := sm.Tick(false, start.Add(501*time.Millisecond)); got != ActionReturnToIdle {
		t.Errorf("Tick at idleDelay+1ms = %v, want ActionReturnToIdle", got)
	}
	if got := sm.State(); got != StateIdle {
		t.Errorf("State() after return = %v, want StateIdle", got)
	}
}

func TestStateMachine_LastVoiceRefreshesPerVoicedFrame(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	start := fixedNow()

	// Reach the talking state.
	sm.Tick(true, start)
	sm.Tick(true, start)
	if got := sm.Tick(true, start); got != ActionStartTalking {
		t.Fatalf("Tick(3) = %v, want ActionStartTalking", got)
	}

	// A voiced frame 400ms after talk start refreshes lastVoice.
	sm.Tick(true, start.Add(400*time.Millisecond))

	// Silence is measured from the refreshed lastVoice, not talk start:
	// 400ms after the refresh (800ms after talk start) is still within
	// idleDelay, so the avatar must keep talking.
	if got := sm.Tick(false, start.Add(800*time.Millisecond)); got != ActionNone {
		t.Errorf("Tick 400ms after refreshed lastVoice = %v, want ActionNone", got)
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() = %v, want StateTalking (silence measured from last voice)", got)
	}

	// 501ms after the refreshed lastVoice: return to idle.
	if got := sm.Tick(false, start.Add(901*time.Millisecond)); got != ActionReturnToIdle {
		t.Errorf("Tick 501ms after refreshed lastVoice = %v, want ActionReturnToIdle", got)
	}
}

// --- Additional edge-case tests ---

func TestStateMachine_Debounce1_TransitionsImmediately(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 1)
	now := fixedNow()

	// debounce=1: a single voiced tick should start talking.
	if got := sm.Tick(true, now); got != ActionStartTalking {
		t.Errorf("Tick(debounce=1) = %v, want ActionStartTalking", got)
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() = %v, want StateTalking", got)
	}
}

func TestStateMachine_Debounce5_RequiresFiveConsecutive(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 5)
	now := fixedNow()

	// First 4 voiced ticks: still idle.
	for i := 1; i <= 4; i++ {
		if got := sm.Tick(true, now); got != ActionNone {
			t.Fatalf("Tick(%d) = %v, want ActionNone", i, got)
		}
		if got := sm.State(); got != StateIdle {
			t.Fatalf("State() after Tick(%d) = %v, want StateIdle", i, got)
		}
	}

	// 5th tick: transition fires.
	if got := sm.Tick(true, now); got != ActionStartTalking {
		t.Errorf("Tick(5) = %v, want ActionStartTalking", got)
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() = %v, want StateTalking", got)
	}
}

func TestStateMachine_ZeroIdleDelay_ReturnsToIdleImmediately(t *testing.T) {
	sm := NewStateMachine(0, 3)
	now := fixedNow()

	// Reach talking state.
	sm.Tick(true, now)
	sm.Tick(true, now)
	if got := sm.Tick(true, now); got != ActionStartTalking {
		t.Fatalf("Tick(3) = %v, want ActionStartTalking", got)
	}

	// A single non-voiced tick: now.Sub(lastVoice) > 0 is true, so returns to idle.
	if got := sm.Tick(false, now.Add(time.Nanosecond)); got != ActionReturnToIdle {
		t.Errorf("Tick(zero idleDelay) = %v, want ActionReturnToIdle", got)
	}
	if got := sm.State(); got != StateIdle {
		t.Errorf("State() = %v, want StateIdle", got)
	}
}

func TestStateMachine_VoiceResumeJustBeforeDeadline_ExtendsTalk(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	start := fixedNow()

	// Reach talking state.
	sm.Tick(true, start)
	sm.Tick(true, start)
	if got := sm.Tick(true, start); got != ActionStartTalking {
		t.Fatalf("Tick(3) = %v, want ActionStartTalking", got)
	}

	// Silence for 499ms (just before deadline), then voice resumes.
	sm.Tick(false, start.Add(499*time.Millisecond))

	// Voice at 500ms mark — lastVoice is still `start` since no voice
	// during the silence, but we need to send a voice tick to refresh it.
	// Actually: voiceConfirm was reset to 0 by the non-voice tick. Sending
	// voice now refreshes lastVoice.
	sm.Tick(true, start.Add(500*time.Millisecond))

	// Now silence for 499ms from the refreshed lastVoice — still talking.
	if got := sm.Tick(false, start.Add(999*time.Millisecond)); got != ActionNone {
		t.Errorf("Tick 499ms after refreshed lastVoice = %v, want ActionNone", got)
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() = %v, want StateTalking", got)
	}

	// 501ms after refreshed lastVoice: return to idle.
	if got := sm.Tick(false, start.Add(1001*time.Millisecond)); got != ActionReturnToIdle {
		t.Errorf("Tick 501ms after refreshed lastVoice = %v, want ActionReturnToIdle", got)
	}
}

func TestStateMachine_VoiceConfirmOverflow_NoSideEffects(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	// Send 1000 consecutive voiced ticks while idle (debounce=3, fires on 3rd).
	sm.Tick(true, now) // voiceConfirm=1
	sm.Tick(true, now) // voiceConfirm=2
	sm.Tick(true, now) // voiceConfirm=3 → ActionStartTalking
	for i := 0; i < 997; i++ {
		sm.Tick(true, now) // voiceConfirm keeps growing, but already talking
	}

	if got := sm.State(); got != StateTalking {
		t.Errorf("State() after 1000 voiced ticks = %v, want StateTalking", got)
	}

	// Return to idle and verify the machine still works.
	if got := sm.Tick(false, now.Add(501*time.Millisecond)); got != ActionReturnToIdle {
		t.Errorf("Tick after overflow = %v, want ActionReturnToIdle", got)
	}
	if got := sm.State(); got != StateIdle {
		t.Errorf("State() = %v, want StateIdle", got)
	}

	// Verify re-entry: debounce still works.
	sm.Tick(true, now.Add(600*time.Millisecond))
	sm.Tick(true, now.Add(600*time.Millisecond))
	if got := sm.Tick(true, now.Add(600*time.Millisecond)); got != ActionStartTalking {
		t.Errorf("Re-entry after overflow = %v, want ActionStartTalking", got)
	}
}

func TestStateMachine_MultipleTalkIdleCycles(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	for cycle := 0; cycle < 5; cycle++ {
		base := now.Add(time.Duration(cycle) * 2 * time.Second)

		// Build up debounce.
		sm.Tick(true, base)
		sm.Tick(true, base)
		if got := sm.Tick(true, base); got != ActionStartTalking {
			t.Fatalf("Cycle %d: Tick(3) = %v, want ActionStartTalking", cycle, got)
		}

		// Return to idle.
		if got := sm.Tick(false, base.Add(501*time.Millisecond)); got != ActionReturnToIdle {
			t.Fatalf("Cycle %d: Tick(return) = %v, want ActionReturnToIdle", cycle, got)
		}
		if got := sm.State(); got != StateIdle {
			t.Fatalf("Cycle %d: State() = %v, want StateIdle", cycle, got)
		}
	}
}

func TestStateMachine_AlternatingVoiceNeverReachesDebounce(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	// Alternate voice on/off: voiceConfirm never exceeds 1.
	for i := 0; i < 20; i++ {
		sm.Tick(true, now)
		sm.Tick(false, now)
	}

	if got := sm.State(); got != StateIdle {
		t.Errorf("State() after alternating = %v, want StateIdle", got)
	}
}

func TestStateMachine_VoiceDuringIdleDoesNotSetLastVoiceBeforeTalking(t *testing.T) {
	// Verify that during the debounce build-up in idle, voice ticks set
	// lastVoice so that when talking starts, lastVoice is the time of
	// the last voiced frame (the one that triggered the transition).
	sm := NewStateMachine(500*time.Millisecond, 3)
	t0 := fixedNow()

	// Debounce with time gaps to verify lastVoice tracks each voiced tick.
	sm.Tick(true, t0)
	sm.Tick(true, t0.Add(100*time.Millisecond))
	// This tick triggers the transition; lastVoice should be t0+200ms.
	if got := sm.Tick(true, t0.Add(200*time.Millisecond)); got != ActionStartTalking {
		t.Fatalf("Tick(3) = %v, want ActionStartTalking", got)
	}

	// Silence for 500ms from transition (lastVoice=t0+200ms):
	// t0+700ms - (t0+200ms) = 500ms — exactly idleDelay, not strictly greater.
	if got := sm.Tick(false, t0.Add(700*time.Millisecond)); got != ActionNone {
		t.Errorf("Tick at exactly idleDelay from lastVoice = %v, want ActionNone", got)
	}

	// 501ms from lastVoice: return to idle.
	if got := sm.Tick(false, t0.Add(701*time.Millisecond)); got != ActionReturnToIdle {
		t.Errorf("Tick at idleDelay+1ms from lastVoice = %v, want ActionReturnToIdle", got)
	}
}

func TestStateMachine_NonVoiceWhileIdleIsNoOp(t *testing.T) {
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	// Non-voiced ticks while idle should be harmless.
	for i := 0; i < 10; i++ {
		if got := sm.Tick(false, now); got != ActionNone {
			t.Fatalf("Tick(false) #%d = %v, want ActionNone", i+1, got)
		}
		if got := sm.State(); got != StateIdle {
			t.Fatalf("State() after Tick(false) #%d = %v, want StateIdle", i+1, got)
		}
	}
}

func TestStateMachine_TransitionActionMatchesStateChange(t *testing.T) {
	// Every ActionStartTalking must be accompanied by state becoming Talking,
	// and every ActionReturnToIdle must be accompanied by state becoming Idle.
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	// Start talking.
	sm.Tick(true, now)
	sm.Tick(true, now)
	action := sm.Tick(true, now)
	if action != ActionStartTalking {
		t.Fatalf("Expected ActionStartTalking, got %v", action)
	}
	if sm.State() != StateTalking {
		t.Fatalf("State should be Talking after ActionStartTalking, got %v", sm.State())
	}

	// Return to idle.
	action = sm.Tick(false, now.Add(501*time.Millisecond))
	if action != ActionReturnToIdle {
		t.Fatalf("Expected ActionReturnToIdle, got %v", action)
	}
	if sm.State() != StateIdle {
		t.Fatalf("State should be Idle after ActionReturnToIdle, got %v", sm.State())
	}
}

func TestStateMachine_TalkingStateRejectsVoiceAction(t *testing.T) {
	// When already talking, continued voice must NOT produce ActionStartTalking.
	sm := NewStateMachine(500*time.Millisecond, 3)
	now := fixedNow()

	sm.Tick(true, now)
	sm.Tick(true, now)
	if got := sm.Tick(true, now); got != ActionStartTalking {
		t.Fatalf("initial transition = %v, want ActionStartTalking", got)
	}

	// 100 more voiced ticks — all must return ActionNone.
	for i := 0; i < 100; i++ {
		if got := sm.Tick(true, now); got != ActionNone {
			t.Fatalf("Tick while talking #%d = %v, want ActionNone", i+1, got)
		}
	}
}

func TestStateMachine_IdleDelayExactBoundary_MultipleExtensions(t *testing.T) {
	// Repeated voice resumes extend the deadline each time; only a sustained
	// silence exceeds idleDelay.
	sm := NewStateMachine(500*time.Millisecond, 3)
	start := fixedNow()

	sm.Tick(true, start)
	sm.Tick(true, start)
	sm.Tick(true, start) // ActionStartTalking, lastVoice=start

	// Extend 3 times: voice at 200ms, 400ms, 600ms after start.
	sm.Tick(true, start.Add(200*time.Millisecond)) // lastVoice=start+200ms
	sm.Tick(true, start.Add(400*time.Millisecond)) // lastVoice=start+400ms
	sm.Tick(true, start.Add(600*time.Millisecond)) // lastVoice=start+600ms

	// Silence at start+1000ms: only 400ms since lastVoice — still talking.
	if got := sm.Tick(false, start.Add(1000*time.Millisecond)); got != ActionNone {
		t.Errorf("Tick 400ms after last extend = %v, want ActionNone", got)
	}
	if got := sm.State(); got != StateTalking {
		t.Errorf("State() = %v, want StateTalking", got)
	}

	// Silence at start+1101ms: 501ms since lastVoice — return to idle.
	if got := sm.Tick(false, start.Add(1101*time.Millisecond)); got != ActionReturnToIdle {
		t.Errorf("Tick 501ms after last extend = %v, want ActionReturnToIdle", got)
	}
}
