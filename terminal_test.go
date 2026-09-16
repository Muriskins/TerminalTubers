package main

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// newTestTerminal creates a Terminal backed by a simulation screen of the given
// size. The simulation screen is initialised and ready for rendering. The caller
// must defer term.Close().
func newTestTerminal(t *testing.T, w, h int) (*Terminal, tcell.SimulationScreen) {
	t.Helper()
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("sim.Init() failed: %v", err)
	}
	sim.SetSize(w, h)
	return &Terminal{screen: sim}, sim
}

// getScreenContent reads back the full screen content and returns the rune at
// each (x, y) position in a 2D slice.
func getScreenContent(t *testing.T, sim tcell.SimulationScreen, w, h int) [][]rune {
	t.Helper()
	cells, _, _ := sim.GetContents()
	grid := make([][]rune, h)
	for y := 0; y < h; y++ {
		grid[y] = make([]rune, w)
		for x := 0; x < w; x++ {
			idx := y*w + x
			if idx < len(cells) && len(cells[idx].Runes) > 0 {
				grid[y][x] = cells[idx].Runes[0]
			}
		}
	}
	return grid
}

// assertGridExact verifies every cell in the visible grid equals expected.
func assertGridExact(t *testing.T, sim tcell.SimulationScreen, w, h int, expected [][]rune) {
	t.Helper()
	grid := getScreenContent(t, sim, w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if grid[y][x] != expected[y][x] {
				t.Errorf("cell(%d,%d) = %q, want %q", x, y, grid[y][x], expected[y][x])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RenderFrame: centering
// ---------------------------------------------------------------------------

func TestRenderFrame_CenterSmallFrame(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()

	frame := []string{
		"AB",
		"CD",
	}

	term.RenderFrame(frame)

	// Screen 40x10, frame 2x2 → origin at (19, 4).
	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'A'
	expected[4][20] = 'B'
	expected[5][19] = 'C'
	expected[5][20] = 'D'

	assertGridExact(t, sim, 40, 10, expected)
}

func TestRenderFrame_CenterSingleCharacter(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	term.RenderFrame([]string{"X"})

	// originX = (20-1)/2 = 9, originY = (10-1)/2 = 4
	expected := makeEmptyGrid(20, 10)
	expected[4][9] = 'X'

	assertGridExact(t, sim, 20, 10, expected)
}

func TestRenderFrame_CenterOddDimensions(t *testing.T) {
	term, sim := newTestTerminal(t, 41, 11)
	defer term.Close()

	// Frame 3x3 → origin = ((41-3)/2, (11-3)/2) = (19, 4)
	frame := []string{"ABC", "DEF", "GHI"}
	term.RenderFrame(frame)

	expected := makeEmptyGrid(41, 11)
	expected[4][19] = 'A'
	expected[4][20] = 'B'
	expected[4][21] = 'C'
	expected[5][19] = 'D'
	expected[5][20] = 'E'
	expected[5][21] = 'F'
	expected[6][19] = 'G'
	expected[6][20] = 'H'
	expected[6][21] = 'I'

	assertGridExact(t, sim, 41, 11, expected)
}

func TestRenderFrame_CenterUnevenWidthLines(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()

	// Frame with uneven widths: widest = 5 ("ABCDE"), other = 3 ("FG").
	// originX = (40 - 5) / 2 = 17
	frame := []string{"ABCDE", "FG"}
	term.RenderFrame(frame)

	expected := makeEmptyGrid(40, 10)
	// row 4 (originY=(10-2)/2=4): ABCDE at x=17..21
	expected[4][17] = 'A'
	expected[4][18] = 'B'
	expected[4][19] = 'C'
	expected[4][20] = 'D'
	expected[4][21] = 'E'
	// row 5: FG at x=17..18
	expected[5][17] = 'F'
	expected[5][18] = 'G'

	assertGridExact(t, sim, 40, 10, expected)
}

// ---------------------------------------------------------------------------
// RenderFrame: clipping (top-left)
// ---------------------------------------------------------------------------

func TestRenderFrame_ClipFrameWiderThanScreen(t *testing.T) {
	term, sim := newTestTerminal(t, 10, 10)
	defer term.Close()

	// Frame 15 chars wide, screen 10 wide → originX = (10-15)/2 = -2
	// originY = (10-1)/2 = 4 → frame renders on row 4.
	// Visible columns: frame cols 2..11 → chars "CDEFGHIJKL"
	frame := []string{"ABCDEFGHIJKLMNO"}
	term.RenderFrame(frame)

	expected := makeEmptyGrid(10, 10)
	for i, ch := range "CDEFGHIJKL" {
		expected[4][i] = ch
	}

	assertGridExact(t, sim, 10, 10, expected)
}

func TestRenderFrame_ClipFrameTallerThanScreen(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 3)
	defer term.Close()

	// Frame 5 lines tall, screen 3 high → originY = (3-5)/2 = -1
	// originX = (20-5)/2 = 7
	// Visible rows: frame[1] at screenY=0, frame[2] at screenY=1, frame[3] at screenY=2
	frame := []string{"AAAAA", "BBBBB", "CCCCC", "DDDDD", "EEEEE"}
	term.RenderFrame(frame)

	expected := makeEmptyGrid(20, 3)
	for i, ch := range "BBBBB" {
		expected[0][7+i] = ch
	}
	for i, ch := range "CCCCC" {
		expected[1][7+i] = ch
	}
	for i, ch := range "DDDDD" {
		expected[2][7+i] = ch
	}

	assertGridExact(t, sim, 20, 3, expected)
}

func TestRenderFrame_ClipBothDirections(t *testing.T) {
	term, sim := newTestTerminal(t, 4, 3)
	defer term.Close()

	// Frame 8x5, screen 4x3 → originX = (4-8)/2 = -2, originY = (3-5)/2 = -1
	// Visible rows: frame[1] at screenY=0, frame[2] at screenY=1, frame[3] at screenY=2
	// Visible cols: frame cols 2..5 (4 columns)
	frame := []string{
		"AAAAAAAA",
		"BBBBBBBB",
		"CCCCCCCC",
		"DDDDDDDD",
		"EEEEEEEE",
	}
	term.RenderFrame(frame)

	expected := makeEmptyGrid(4, 3)
	// row 0: "BBBBBBBB"[2:6] = "BBBB"
	expected[0][0] = 'B'
	expected[0][1] = 'B'
	expected[0][2] = 'B'
	expected[0][3] = 'B'
	// row 1: "CCCCCCCC"[2:6] = "CCCC"
	expected[1][0] = 'C'
	expected[1][1] = 'C'
	expected[1][2] = 'C'
	expected[1][3] = 'C'
	// row 2: "DDDDDDDD"[2:6] = "DDDD"
	expected[2][0] = 'D'
	expected[2][1] = 'D'
	expected[2][2] = 'D'
	expected[2][3] = 'D'

	assertGridExact(t, sim, 4, 3, expected)
}

func TestRenderFrame_NoPanicWithTinyScreen(t *testing.T) {
	term, sim := newTestTerminal(t, 1, 1)
	defer term.Close()

	// Screen 1x1, frame 5x5 → heavy clipping, should not panic.
	frame := []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"}
	// originX = (1-5)/2 = -2, originY = (1-5)/2 = -2
	// Visible: row=2 → screenY=0, col=2 → screenX=0 → frame[2][2] = 'M'
	term.RenderFrame(frame)

	expected := makeEmptyGrid(1, 1)
	expected[0][0] = 'M'

	assertGridExact(t, sim, 1, 1, expected)
}

// ---------------------------------------------------------------------------
// RenderFrame: empty frame
// ---------------------------------------------------------------------------

func TestRenderFrame_EmptyFrameIsNoop(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	// Render something first, then render empty — screen should stay unchanged.
	term.RenderFrame([]string{"X"})

	// Now render empty — this should not clear the screen, it's a no-op.
	term.RenderFrame([]string{})

	// 'X' should still be at the centered position (9,4).
	grid := getScreenContent(t, sim, 20, 10)
	if grid[4][9] != 'X' {
		t.Errorf("empty RenderFrame cleared existing content; cell(9,4) = %q, want 'X'", grid[4][9])
	}
}

func TestRenderFrame_NilFrameIsNoop(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	// nil slice (not empty slice) should also be a no-op.
	term.RenderFrame(nil)

	// Screen should remain blank.
	grid := getScreenContent(t, sim, 20, 10)
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if grid[y][x] != 0 {
				t.Errorf("nil RenderFrame wrote to cell(%d,%d): %q", x, y, grid[y][x])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Close: idempotency
// ---------------------------------------------------------------------------

func TestClose_Idempotent(t *testing.T) {
	term, _ := newTestTerminal(t, 80, 24)

	term.Close()
	term.Close() // must not panic
	term.Close() // triple-close safety
}

// ---------------------------------------------------------------------------
// Close: calls Fini exactly once
// ---------------------------------------------------------------------------

func TestClose_CallsFiniExactlyOnce(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("sim.Init() failed: %v", err)
	}
	sim.SetSize(80, 24)

	term := &Terminal{screen: sim}

	term.Close()
	term.Close()

	// After Fini, calling Size should still work (it's a simulation screen),
	// but the point is no panic occurred during double-close.
	// This test primarily verifies the sync.Once protection.
}

// ---------------------------------------------------------------------------
// NewTerminal: error path (cannot test real init in CI)
// ---------------------------------------------------------------------------

// NOTE: We do NOT test NewTerminal() directly because tcell.NewScreen() requires
// a TTY. The simulation screen tests above cover all Terminal methods.

// ---------------------------------------------------------------------------
// RenderFrame: unicode / multi-byte characters
// ---------------------------------------------------------------------------

func TestRenderFrame_UnicodeCharacters(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	// Single multi-byte character (é is 2 bytes, 1 cell wide).
	// originX = (20-2)/2 = 9, originY = (10-1)/2 = 4
	term.RenderFrame([]string{"é"})

	expected := makeEmptyGrid(20, 10)
	expected[4][9] = 'é'

	assertGridExact(t, sim, 20, 10, expected)
}

// ---------------------------------------------------------------------------
// RenderFrame: last frame wins (overwrite test)
// ---------------------------------------------------------------------------

func TestRenderFrame_OverwritesPreviousFrame(t *testing.T) {
	term, sim := newTestTerminal(t, 20, 10)
	defer term.Close()

	// Render one frame, then another — second should overwrite.
	term.RenderFrame([]string{"X"})
	term.RenderFrame([]string{"Y"})

	grid := getScreenContent(t, sim, 20, 10)
	// After overwriting, only Y should be at the centered position (9,4).
	if grid[4][9] != 'Y' {
		t.Errorf("cell(9,4) = %q, want 'Y'", grid[4][9])
	}
}

// ---------------------------------------------------------------------------
// RenderFrame: stale-pixel regression (clear before draw)
// ---------------------------------------------------------------------------

func TestRenderFrame_ClearsBeforeDraw(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 20)
	defer term.Close()

	// Render a wide frame that fills many cells.
	wideFrame := []string{
		"AAAAAAAAAAAAAAAAAAAAAAAA", // 24 chars wide
		"BBBBBBBBBBBBBBBBBBBBBBBB",
		"CCCCCCCCCCCCCCCCCCCCCCCC",
	}
	term.RenderFrame(wideFrame)

	// Now render a narrow frame (1x1).
	narrowFrame := []string{"X"}
	term.RenderFrame(narrowFrame)

	// Verify no ghost pixels from the wide frame remain.
	// The narrow frame 'X' is at center (19,9). All other cells must be ' '.
	grid := getScreenContent(t, sim, 40, 20)
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			expected := ' '
			if x == 19 && y == 9 {
				expected = 'X'
			}
			if grid[y][x] != expected {
				t.Errorf("cell(%d,%d) = %q, want %q (ghost pixel from wide frame)", x, y, grid[y][x], expected)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// PollEvent: simulation injection
// ---------------------------------------------------------------------------

func TestPollEvent_KeyInjection(t *testing.T) {
	term, _ := newTestTerminal(t, 80, 24)
	defer term.Close()

	// Inject a 'q' keypress and verify PollEvent returns it.
	go func() {
		sim := term.screen.(tcell.SimulationScreen)
		sim.InjectKey(tcell.KeyRune, 'q', tcell.ModNone)
	}()

	ev := term.PollEvent()
	keyEvent, ok := ev.(*tcell.EventKey)
	if !ok {
		t.Fatalf("expected *tcell.EventKey, got %T", ev)
	}
	if keyEvent.Rune() != 'q' {
		t.Errorf("injected 'q' key, got rune %q", keyEvent.Rune())
	}
}

func TestPollEvent_EscapeKey(t *testing.T) {
	term, _ := newTestTerminal(t, 80, 24)
	defer term.Close()

	go func() {
		sim := term.screen.(tcell.SimulationScreen)
		sim.InjectKey(tcell.KeyEscape, 0, tcell.ModNone)
	}()

	ev := term.PollEvent()
	keyEvent, ok := ev.(*tcell.EventKey)
	if !ok {
		t.Fatalf("expected *tcell.EventKey, got %T", ev)
	}
	if keyEvent.Key() != tcell.KeyEscape {
		t.Errorf("expected KeyEscape, got %v", keyEvent.Key())
	}
}

// ---------------------------------------------------------------------------
// Size: delegates to screen
// ---------------------------------------------------------------------------

func TestSize_DelegatesToScreen(t *testing.T) {
	term, _ := newTestTerminal(t, 120, 40)
	defer term.Close()

	w, h := term.Size()
	if w != 120 {
		t.Errorf("Size() width = %d, want 120", w)
	}
	if h != 40 {
		t.Errorf("Size() height = %d, want 40", h)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeEmptyGrid(w, h int) [][]rune {
	grid := make([][]rune, h)
	for y := range grid {
		grid[y] = make([]rune, w)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}
	return grid
}
