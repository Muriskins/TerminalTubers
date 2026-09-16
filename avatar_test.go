package main

import "testing"

func TestNewAvatar_Valid(t *testing.T) {
	frames := [][]string{
		{"idle"},
		{"tolk0"},
		{"tolk1"},
		{"tolk2"},
	}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}
	if avatar == nil {
		t.Fatal("NewAvatar() returned nil avatar")
	}
}

func TestNewAvatar_EmptyFrames(t *testing.T) {
	_, err := NewAvatar([][]string{})
	if err == nil {
		t.Fatal("NewAvatar() with empty frames should return error")
	}
}

func TestAvatar_CurrentFrame_InitiallyNil(t *testing.T) {
	avatar, err := NewAvatar([][]string{{"idle"}})
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}
	if avatar.CurrentFrame() != nil {
		t.Errorf("CurrentFrame() = %v, want nil before first Render", avatar.CurrentFrame())
	}
}

func TestAvatar_Render_SwitchesFrame(t *testing.T) {
	term, _ := newTestTerminal(t, 40, 10)
	defer term.Close()

	frames := [][]string{
		{"AAA", "BBB"},
		{"CCC", "DDD"},
	}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	avatar.Render(term, frames[0])
	if got := avatar.CurrentFrame(); len(got) != 2 || got[0] != "AAA" {
		t.Errorf("after first Render, CurrentFrame = %v, want [AAA BBB]", got)
	}

	avatar.Render(term, frames[1])
	if got := avatar.CurrentFrame(); len(got) != 2 || got[0] != "CCC" {
		t.Errorf("after second Render, CurrentFrame = %v, want [CCC DDD]", got)
	}
}

func TestAvatar_Render_Integration(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()

	frames := [][]string{
		{"X"},
		{"Y"},
	}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	avatar.Render(term, frames[0])

	// Verify the frame was drawn on screen: 'X' at center (19, 4).
	grid := getScreenContent(t, sim, 40, 10)
	if grid[4][19] != 'X' {
		t.Errorf("cell(19,4) = %q, want 'X' after Avatar.Render", grid[4][19])
	}

	// Switch to second frame.
	avatar.Render(term, frames[1])

	// After Clear + draw, 'Y' should be at center and everything else ' '.
	grid = getScreenContent(t, sim, 40, 10)
	for y := 0; y < 10; y++ {
		for x := 0; x < 40; x++ {
			expected := ' '
			if x == 19 && y == 4 {
				expected = 'Y'
			}
			if grid[y][x] != expected {
				t.Errorf("cell(%d,%d) = %q, want %q after frame switch", x, y, grid[y][x], expected)
			}
		}
	}
}

func TestAvatar_Frames_ReturnsAllFrames(t *testing.T) {
	frames := [][]string{
		{"idle"},
		{"tolk0"},
		{"tolk1"},
		{"tolk2"},
	}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	got := avatar.Frames()
	if len(got) != 4 {
		t.Fatalf("Frames() returned %d frames, want 4", len(got))
	}
	for i, frame := range frames {
		if len(got[i]) != 1 || got[i][0] != frame[0] {
			t.Errorf("Frames()[%d] = %v, want %v", i, got[i], frame)
		}
	}
}
