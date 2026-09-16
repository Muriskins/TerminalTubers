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
	if err.Error() != "avatar: at least one frame required" {
		t.Errorf("NewAvatar() error = %q, want %q", err.Error(), "avatar: at least one frame required")
	}
}

func TestNewAvatar_SingleFrame(t *testing.T) {
	avatar, err := NewAvatar([][]string{{"idle"}})
	if err != nil {
		t.Fatalf("NewAvatar() with single frame returned error: %v", err)
	}
	if got := avatar.Frames(); len(got) != 1 || len(got[0]) != 1 || got[0][0] != "idle" {
		t.Errorf("Frames() = %v, want [[idle]]", got)
	}
}

func TestNewAvatar_EmptyInnerFrame_NoError(t *testing.T) {
	// The implementation validates only the outer slice: a frame that is an
	// empty slice is accepted. This pins the current contract so a future
	// validation change is surfaced by this test.
	avatar, err := NewAvatar([][]string{{}})
	if err != nil {
		t.Fatalf("NewAvatar() with inner empty frame returned error: %v", err)
	}
	if got := avatar.Frames(); len(got) != 1 || len(got[0]) != 0 {
		t.Errorf("Frames() = %v, want one empty frame", got)
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

func TestAvatar_Render_NilTarget(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	avatar, err := NewAvatar([][]string{{"idle"}})
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	avatar.Render(term, nil)
	if avatar.CurrentFrame() != nil {
		t.Errorf("CurrentFrame() = %v, want nil after Render(nil)", avatar.CurrentFrame())
	}

	// Screen must remain blank: RenderFrame(nil) is a no-op.
	grid := getScreenContent(t, sim, 20, 10)
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if grid[y][x] != 0 {
				t.Errorf("Render(nil) wrote to cell(%d,%d): %q", x, y, grid[y][x])
			}
		}
	}
}

func TestAvatar_Render_EmptyTarget(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	avatar, err := NewAvatar([][]string{{"idle"}})
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	avatar.Render(term, []string{})
	if got := avatar.CurrentFrame(); got == nil || len(got) != 0 {
		t.Errorf("CurrentFrame() = %v, want empty non-nil slice after Render([]string{})", got)
	}

	// Screen must remain blank: RenderFrame([]string{}) is a no-op.
	grid := getScreenContent(t, sim, 20, 10)
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if grid[y][x] != 0 {
				t.Errorf("Render([]string{}) wrote to cell(%d,%d): %q", x, y, grid[y][x])
			}
		}
	}
}

func TestAvatar_Render_ArbitraryTarget(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	avatar, err := NewAvatar([][]string{{"idle"}})
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	// Render a frame that was never registered in Frames() — Render accepts
	// any target slice, not just frames from the avatar's own storage.
	external := []string{"Z"}
	avatar.Render(term, external)

	if got := avatar.CurrentFrame(); len(got) != 1 || got[0] != "Z" {
		t.Errorf("CurrentFrame() = %v, want [Z]", got)
	}

	grid := getScreenContent(t, sim, 20, 10)
	if grid[4][9] != 'Z' {
		t.Errorf("cell(9,4) = %q, want 'Z' after Render of external frame", grid[4][9])
	}
}

func TestAvatar_CurrentFrame_ReturnsExactTarget(t *testing.T) {
	term, _ := newTestTerminal(t, 20, 10)
	defer term.Close()

	avatar, err := NewAvatar([][]string{{"idle"}})
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	target := []string{"frame"}
	avatar.Render(term, target)

	// CurrentFrame must return the exact slice passed to Render (identity),
	// not a copy.
	got := avatar.CurrentFrame()
	if got == nil || len(got) != len(target) {
		t.Fatalf("CurrentFrame() = %v, want the exact target slice", got)
	}
	if &got[0] != &target[0] {
		t.Errorf("CurrentFrame() returned a copy, not the exact target slice")
	}
}

func TestAvatar_Frames_ReturnsBackingSlice(t *testing.T) {
	frames := [][]string{{"idle"}, {"tolk0"}}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	got := avatar.Frames()
	if len(got) != 2 {
		t.Fatalf("Frames() returned %d frames, want 2", len(got))
	}
	// Frames() must return the original backing slice, not a copy.
	if &got[0] != &frames[0] {
		t.Errorf("Frames() returned a copy, not the original backing slice")
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
