package main

import (
	"math"
	"math/rand"
	"testing"
)

// runeInCharset reports whether r is present in cs.
func runeInCharset(r rune, cs []rune) bool {
	for _, c := range cs {
		if c == r {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// charsetFor
// ---------------------------------------------------------------------------

func TestCharsetFor_Classic(t *testing.T) {
	cs := charsetFor(AnimationCharsetClassic)
	if len(cs) != 10 {
		t.Fatalf("Classic charset has %d runes, want 10", len(cs))
	}
	if cs[9] != ' ' {
		t.Errorf("Classic charset last rune = %q, want ' '", cs[9])
	}
}

func TestCharsetFor_Unicode(t *testing.T) {
	cs := charsetFor(AnimationCharsetUnicode)
	if len(cs) != 11 {
		t.Fatalf("Unicode charset has %d runes, want 11", len(cs))
	}
	if cs[10] != ' ' {
		t.Errorf("Unicode charset last rune = %q, want ' '", cs[10])
	}
}

func TestCharsetFor_All(t *testing.T) {
	cs := charsetFor(AnimationCharsetAll)
	if len(cs) != 95 {
		t.Fatalf("All charset has %d runes, want 95", len(cs))
	}
	if cs[0] != '!' {
		t.Errorf("All charset first rune = %q, want '!'", cs[0])
	}
	if cs[93] != '~' {
		t.Errorf("All charset rune[93] = %q, want '~'", cs[93])
	}
	if cs[94] != ' ' {
		t.Errorf("All charset last rune = %q, want ' '", cs[94])
	}
}

func TestCharsetFor_UnknownFallsBackToClassic(t *testing.T) {
	cs := charsetFor(AnimationCharset("Bogus"))
	classic := charsetFor(AnimationCharsetClassic)
	if len(cs) != len(classic) {
		t.Fatalf("unknown charset length = %d, want %d (Classic)", len(cs), len(classic))
	}
	for i := range cs {
		if cs[i] != classic[i] {
			t.Errorf("unknown charset[%d] = %q, want %q (Classic)", i, cs[i], classic[i])
		}
	}
}

// ---------------------------------------------------------------------------
// toGrid
// ---------------------------------------------------------------------------

func TestToGrid_PadsShortRowsToWidth(t *testing.T) {
	frame := []string{"A", "BB"}
	grid := toGrid(frame, 2, 3)
	want := [][]rune{{'A', ' ', ' '}, {'B', 'B', ' '}}
	if len(grid) != 2 {
		t.Fatalf("grid height = %d, want 2", len(grid))
	}
	for r := 0; r < 2; r++ {
		for c := 0; c < 3; c++ {
			if grid[r][c] != want[r][c] {
				t.Errorf("cell(%d,%d) = %q, want %q", r, c, grid[r][c], want[r][c])
			}
		}
	}
}

func TestToGrid_PadsHeightOnlyWhenShorter(t *testing.T) {
	frame := []string{"AB", "CD"}
	grid := toGrid(frame, 4, 2)
	if len(grid) != 4 {
		t.Fatalf("grid height = %d, want 4 (padded to th)", len(grid))
	}
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			if grid[r][c] != rune(frame[r][c]) {
				t.Errorf("cell(%d,%d) = %q, want %q", r, c, grid[r][c], frame[r][c])
			}
		}
	}
	for r := 2; r < 4; r++ {
		for c := 0; c < 2; c++ {
			if grid[r][c] != ' ' {
				t.Errorf("padded cell(%d,%d) = %q, want ' '", r, c, grid[r][c])
			}
		}
	}
}

func TestToGrid_DoesNotTruncateTallerFrame(t *testing.T) {
	frame := []string{"AAAAA", "BBBBB", "CCCCC", "DDDDD", "EEEEE"}
	grid := toGrid(frame, 3, 5)
	if len(grid) != 5 {
		t.Fatalf("grid height = %d, want 5 (taller frame must not be truncated)", len(grid))
	}
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			if grid[r][c] != rune(frame[r][c]) {
				t.Errorf("cell(%d,%d) = %q, want %q", r, c, grid[r][c], frame[r][c])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// scrambleStep
// ---------------------------------------------------------------------------

func TestScrambleStep_FinalStepAllTarget(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	current := [][]rune{{'A', 'B'}, {'C', 'D'}}
	target := [][]rune{{'X', 'Y'}, {'Z', 'W'}}
	cs := charsetFor(AnimationCharsetClassic)

	grid := scrambleStep(current, target, 5, 5, cs, rng)
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			if grid[r][c] != target[r][c] {
				t.Errorf("cell(%d,%d) = %q, want target %q (fix_chance=1.0)", r, c, grid[r][c], target[r][c])
			}
		}
	}
}

func TestScrambleStep_FirstStepMix(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const size = 10
	current := make([][]rune, size)
	target := make([][]rune, size)
	for r := 0; r < size; r++ {
		current[r] = make([]rune, size)
		target[r] = make([]rune, size)
		for c := 0; c < size; c++ {
			current[r][c] = 'C'
			target[r][c] = 'T'
		}
	}
	cs := charsetFor(AnimationCharsetClassic)

	grid := scrambleStep(current, target, 1, 2, cs, rng)
	fixed := 0
	charsetCells := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if grid[r][c] == target[r][c] {
				fixed++
			} else if runeInCharset(grid[r][c], cs) {
				charsetCells++
			} else {
				t.Errorf("cell(%d,%d) = %q is neither target nor charset", r, c, grid[r][c])
			}
		}
	}
	if fixed == 0 {
		t.Error("step 1 produced no fixed (target) cells")
	}
	if charsetCells == 0 {
		t.Error("step 1 produced no charset cells")
	}
}

// ---------------------------------------------------------------------------
// collapseStep
// ---------------------------------------------------------------------------

func TestCollapseStep_FinalStepOnlyCenterTarget(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	target := make([][]rune, 5)
	for r := range target {
		target[r] = []rune{'T', 'T', 'T', 'T', 'T'}
	}

	grid := collapseStep(target, 5, 5, rng)
	midRow, midCol := 2, 2
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			if r == midRow && c == midCol {
				if grid[r][c] != 'T' {
					t.Errorf("center cell(%d,%d) = %q, want target 'T'", r, c, grid[r][c])
				}
			} else if !runeInCharset(grid[r][c], collapseChars) {
				t.Errorf("cell(%d,%d) = %q, want a collapse char", r, c, grid[r][c])
			}
		}
	}
}

func TestCollapseStep_FirstStepMostlyTarget(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	target := make([][]rune, 5)
	for r := range target {
		target[r] = []rune{'T', 'T', 'T', 'T', 'T'}
	}

	grid := collapseStep(target, 1, 5, rng)
	targetCount := 0
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			if grid[r][c] == 'T' {
				targetCount++
			} else if !runeInCharset(grid[r][c], collapseChars) {
				t.Errorf("cell(%d,%d) = %q, want a collapse char", r, c, grid[r][c])
			}
		}
	}
	// threshold at step 1 = 0.8*sqrt(8) ≈ 2.26 → only the 4 corners collapse.
	if targetCount <= 1 {
		t.Errorf("step 1 target count = %d, want > 1 (large threshold keeps most cells)", targetCount)
	}
}

func TestCollapseChars_ExactSet(t *testing.T) {
	want := []rune{'/', '\\', '*', '#', '@'}
	if len(collapseChars) != len(want) {
		t.Fatalf("collapseChars length = %d, want %d", len(collapseChars), len(want))
	}
	for i := range want {
		if collapseChars[i] != want[i] {
			t.Errorf("collapseChars[%d] = %q, want %q", i, collapseChars[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// revealStep
// ---------------------------------------------------------------------------

func TestRevealStep_FinalStepAllTarget(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const size = 10
	grid := make([][]rune, size)
	target := make([][]rune, size)
	for r := 0; r < size; r++ {
		grid[r] = make([]rune, size)
		target[r] = make([]rune, size)
		for c := 0; c < size; c++ {
			grid[r][c] = 'R'
			target[r][c] = 'T'
		}
	}
	cs := charsetFor(AnimationCharsetClassic)

	out := revealStep(grid, target, 5, 5, cs, rng)
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if out[r][c] != target[r][c] {
				t.Errorf("cell(%d,%d) = %q, want target %q (fix_chance=1.0)", r, c, out[r][c], target[r][c])
			}
		}
	}
}

func TestRevealStep_FirstStepMostlyRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const size = 10
	grid := make([][]rune, size)
	target := make([][]rune, size)
	for r := 0; r < size; r++ {
		grid[r] = make([]rune, size)
		target[r] = make([]rune, size)
		for c := 0; c < size; c++ {
			grid[r][c] = 'R'
			target[r][c] = 'T'
		}
	}
	cs := charsetFor(AnimationCharsetClassic)

	out := revealStep(grid, target, 1, 5, cs, rng)
	targetCount := 0
	charsetCount := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if out[r][c] == target[r][c] {
				targetCount++
			} else if runeInCharset(out[r][c], cs) {
				charsetCount++
			} else {
				t.Errorf("cell(%d,%d) = %q is neither target nor charset", r, c, out[r][c])
			}
		}
	}
	if charsetCount == 0 {
		t.Error("first reveal step produced no charset chars (grid should start all-random)")
	}
	if charsetCount <= targetCount {
		t.Errorf("first reveal step charset=%d target=%d, want mostly random (charset > target)", charsetCount, targetCount)
	}
}

// ---------------------------------------------------------------------------
// AnimateTransition: end state on a simulation screen
// ---------------------------------------------------------------------------

func TestAnimateTransition_EndStateScramble(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleScramble
	cfg.AnimationSteps = 1
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_EndStateCollapse(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleCollapse
	cfg.AnimationSteps = 1
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_EndStateReveal(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleReveal
	cfg.AnimationSteps = 1
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_SmartEndState(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleSmart // task 3.3 — Smart animates, end state is exact target
	cfg.AnimationSteps = 5
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_ClampsZeroValues(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleScramble
	cfg.AnimationSteps = 0 // clamped to 1
	cfg.AnimationDelayMs = 0 // clamped to 1

	// Must not panic and must complete with the exact target rendered.
	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

// ---------------------------------------------------------------------------
// Avatar.PlayAnimation
// ---------------------------------------------------------------------------

func TestPlayAnimation_DisabledIsInstant(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()

	frames := [][]string{{"AAA"}, {"BBB"}}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}
	avatar.Render(term, frames[0])

	cfg := DefaultConfig()
	cfg.AnimationsEnabled = false
	rng := rand.New(rand.NewSource(42))

	avatar.PlayAnimation(term, frames[1], cfg, rng)

	got := avatar.CurrentFrame()
	if got == nil || len(got) != len(frames[1]) || &got[0] != &frames[1][0] {
		t.Errorf("CurrentFrame() = %v, want exact target slice %v", got, frames[1])
	}
	grid := getScreenContent(t, sim, 40, 10)
	if grid[4][19] != 'B' {
		t.Errorf("cell(19,4) = %q, want 'B' after instant switch", grid[4][19])
	}
}

func TestPlayAnimation_EnabledAnimates(t *testing.T) {
	term, _ := newTestTerminal(t, 40, 10)
	defer term.Close()

	frames := [][]string{{"AAA"}, {"BBB"}}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}
	avatar.Render(term, frames[0])

	cfg := DefaultConfig()
	cfg.AnimationsEnabled = true
	cfg.AnimationSteps = 1
	cfg.AnimationDelayMs = 1
	rng := rand.New(rand.NewSource(42))

	avatar.PlayAnimation(term, frames[1], cfg, rng)

	got := avatar.CurrentFrame()
	if got == nil || len(got) != len(frames[1]) || &got[0] != &frames[1][0] {
		t.Errorf("CurrentFrame() = %v, want exact target slice %v after animation", got, frames[1])
	}
}

func TestPlayAnimation_NilCurrentIsInstant(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()

	frames := [][]string{{"AAA"}}
	avatar, err := NewAvatar(frames)
	if err != nil {
		t.Fatalf("NewAvatar() returned error: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AnimationsEnabled = true // still instant: no current frame yet
	rng := rand.New(rand.NewSource(42))

	// Must not panic.
	avatar.PlayAnimation(term, frames[0], cfg, rng)

	got := avatar.CurrentFrame()
	if got == nil || len(got) != len(frames[0]) || &got[0] != &frames[0][0] {
		t.Errorf("CurrentFrame() = %v, want exact target slice %v", got, frames[0])
	}
	grid := getScreenContent(t, sim, 40, 10)
	if grid[4][19] != 'A' {
		t.Errorf("cell(19,4) = %q, want 'A' after instant first render", grid[4][19])
	}
}

// ---------------------------------------------------------------------------
// Targeted tests added by test_engineer (task 3.2 verification pass)
// ---------------------------------------------------------------------------

func TestCharsetFor_Classic_ExactContent(t *testing.T) {
	want := []rune{'@', '#', '%', '*', '=', '-', '+', ':', '.', ' '}
	cs := charsetFor(AnimationCharsetClassic)
	if len(cs) != len(want) {
		t.Fatalf("Classic charset length = %d, want %d", len(cs), len(want))
	}
	for i := range want {
		if cs[i] != want[i] {
			t.Errorf("Classic charset[%d] = %q, want %q", i, cs[i], want[i])
		}
	}
}

func TestCharsetFor_Unicode_ExactContent(t *testing.T) {
	want := []rune{'█', '▓', '▒', '░', '╔', '╗', '╚', '╝', '║', '═', ' '}
	cs := charsetFor(AnimationCharsetUnicode)
	if len(cs) != len(want) {
		t.Fatalf("Unicode charset length = %d, want %d", len(cs), len(want))
	}
	for i := range want {
		if cs[i] != want[i] {
			t.Errorf("Unicode charset[%d] = %q, want %q", i, cs[i], want[i])
		}
	}
}

func TestCharsetFor_All_ExactMembership(t *testing.T) {
	cs := charsetFor(AnimationCharsetAll)
	if len(cs) != 95 {
		t.Fatalf("All charset length = %d, want 95", len(cs))
	}
	for i := 0; i < 94; i++ {
		want := rune('!' + i)
		if cs[i] != want {
			t.Errorf("All charset[%d] = %q, want %q ('!'..'~' in order)", i, cs[i], want)
		}
	}
	if cs[94] != ' ' {
		t.Errorf("All charset[94] = %q, want ' '", cs[94])
	}
}

func TestToGrid_EmptyFrame(t *testing.T) {
	grid := toGrid(nil, 3, 4)
	if len(grid) != 3 {
		t.Fatalf("toGrid(nil, 3, 4) height = %d, want 3", len(grid))
	}
	for r := 0; r < 3; r++ {
		if len(grid[r]) != 4 {
			t.Fatalf("row %d width = %d, want 4", r, len(grid[r]))
		}
		for c := 0; c < 4; c++ {
			if grid[r][c] != ' ' {
				t.Errorf("cell(%d,%d) = %q, want ' '", r, c, grid[r][c])
			}
		}
	}
}

func TestToGrid_UnicodeRunes(t *testing.T) {
	// é is 2 bytes — padding must count runes, not bytes.
	grid := toGrid([]string{"é█"}, 1, 4)
	want := []rune{'é', '█', ' ', ' '}
	if len(grid) != 1 || len(grid[0]) != 4 {
		t.Fatalf("grid dims = %dx%d, want 1x4", len(grid), len(grid[0]))
	}
	for c := 0; c < 4; c++ {
		if grid[0][c] != want[c] {
			t.Errorf("cell(0,%d) = %q, want %q", c, grid[0][c], want[c])
		}
	}
}

func TestToGrid_TruncatesRowLongerThanWidth(t *testing.T) {
	grid := toGrid([]string{"ABCDE"}, 1, 3)
	want := []rune{'A', 'B', 'C'}
	for c := 0; c < 3; c++ {
		if grid[0][c] != want[c] {
			t.Errorf("cell(0,%d) = %q, want %q (row truncated to tw)", c, grid[0][c], want[c])
		}
	}
}

func TestGridToLines_RoundTrip(t *testing.T) {
	frame := []string{"AB", "CDE", "F"}
	grid := toGrid(frame, 5, 4)
	lines := gridToLines(grid)
	if len(lines) != 5 {
		t.Fatalf("gridToLines height = %d, want 5", len(lines))
	}
	want := []string{"AB  ", "CDE ", "F   ", "    ", "    "}
	for r := range want {
		if lines[r] != want[r] {
			t.Errorf("line %d = %q, want %q", r, lines[r], want[r])
		}
	}
}

func TestScrambleStep_TallerCurrentPreserved(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	current := [][]rune{
		{'C', 'C', 'C'},
		{'C', 'C', 'C'},
		{'C', 'C', 'C'},
		{'K', 'K', 'K'},
		{'K', 'K', 'K'},
	}
	target := [][]rune{
		{'T', 'T', 'T'},
		{'T', 'T', 'T'},
		{'T', 'T', 'T'},
	}
	cs := charsetFor(AnimationCharsetClassic)

	grid := scrambleStep(current, target, 1, 2, cs, rng)
	if len(grid) != 5 {
		t.Fatalf("grid height = %d, want 5 (taller current preserved)", len(grid))
	}
	// Rows at or beyond th (3, 4) must be untouched.
	for r := 3; r < 5; r++ {
		for c := 0; c < 3; c++ {
			if grid[r][c] != 'K' {
				t.Errorf("row %d cell %d = %q, want 'K' (row >= th must be preserved)", r, c, grid[r][c])
			}
		}
	}
	// Rows below th are re-evaluated: target or charset.
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if grid[r][c] != 'T' && !runeInCharset(grid[r][c], cs) {
				t.Errorf("cell(%d,%d) = %q, want target or charset", r, c, grid[r][c])
			}
		}
	}
}

func TestScrambleStep_LinearFixChance(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const size = 100
	current := make([][]rune, size)
	target := make([][]rune, size)
	for r := 0; r < size; r++ {
		current[r] = make([]rune, size)
		target[r] = make([]rune, size)
		for c := 0; c < size; c++ {
			current[r][c] = 'C'
			target[r][c] = 'T'
		}
	}
	cs := charsetFor(AnimationCharsetClassic)

	// step=2, steps=4 → fix_chance = 0.5 (linear). ~5000 of 10000 cells fixed.
	grid := scrambleStep(current, target, 2, 4, cs, rng)
	fixed := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if grid[r][c] == 'T' {
				fixed++
			}
		}
	}
	if fixed < 4500 || fixed > 5500 {
		t.Errorf("fixed cells at step 2/4 = %d, want ~5000 (linear fix_chance=0.5)", fixed)
	}
}

func TestScrambleStep_Deterministic(t *testing.T) {
	current := [][]rune{{'A', 'B'}, {'C', 'D'}}
	target := [][]rune{{'X', 'Y'}, {'Z', 'W'}}
	cs := charsetFor(AnimationCharsetClassic)

	g1 := scrambleStep(current, target, 2, 5, cs, rand.New(rand.NewSource(99)))
	g2 := scrambleStep(current, target, 2, 5, cs, rand.New(rand.NewSource(99)))
	for r := range g1 {
		for c := range g1[r] {
			if g1[r][c] != g2[r][c] {
				t.Fatalf("scrambleStep not deterministic: cell(%d,%d) = %q vs %q", r, c, g1[r][c], g2[r][c])
			}
		}
	}
}

func TestCollapseStep_OneByOneGrid(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	target := [][]rune{{'X'}}
	// 1x1: maxDist = 0 → threshold = 0 at every step → the single cell is
	// always the target, never a collapse char.
	for step := 1; step <= 5; step++ {
		grid := collapseStep(target, step, 5, rng)
		if grid[0][0] != 'X' {
			t.Errorf("1x1 collapse step %d cell(0,0) = %q, want target 'X'", step, grid[0][0])
		}
	}
}

func TestCollapseStep_EvenDimensionsFinalStep(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	target := make([][]rune, 4)
	for r := range target {
		target[r] = []rune{'T', 'T', 'T', 'T'}
	}
	// 4x4 → midRow=2, midCol=2. Final step: only (2,2) is target.
	grid := collapseStep(target, 4, 4, rng)
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if r == 2 && c == 2 {
				if grid[r][c] != 'T' {
					t.Errorf("center cell(%d,%d) = %q, want 'T'", r, c, grid[r][c])
				}
			} else if !runeInCharset(grid[r][c], collapseChars) {
				t.Errorf("cell(%d,%d) = %q, want a collapse char", r, c, grid[r][c])
			}
		}
	}
}

func TestCollapseStep_Deterministic(t *testing.T) {
	target := make([][]rune, 5)
	for r := range target {
		target[r] = []rune{'T', 'T', 'T', 'T', 'T'}
	}
	g1 := collapseStep(target, 2, 5, rand.New(rand.NewSource(99)))
	g2 := collapseStep(target, 2, 5, rand.New(rand.NewSource(99)))
	for r := range g1 {
		for c := range g1[r] {
			if g1[r][c] != g2[r][c] {
				t.Fatalf("collapseStep not deterministic: cell(%d,%d) = %q vs %q", r, c, g1[r][c], g2[r][c])
			}
		}
	}
}

func TestRevealStep_QuadraticFixChance(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const size = 100
	grid := make([][]rune, size)
	target := make([][]rune, size)
	for r := 0; r < size; r++ {
		grid[r] = make([]rune, size)
		target[r] = make([]rune, size)
		for c := 0; c < size; c++ {
			grid[r][c] = 'R'
			target[r][c] = 'T'
		}
	}
	cs := charsetFor(AnimationCharsetClassic)

	// step=2, steps=4 → progress=0.5 → fix_chance = 0.25 (quadratic).
	// ~2500 of 10000 cells fixed; linear would give ~5000.
	out := revealStep(grid, target, 2, 4, cs, rng)
	fixed := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if out[r][c] == 'T' {
				fixed++
			}
		}
	}
	if fixed < 2000 || fixed > 3000 {
		t.Errorf("fixed cells at step 2/4 = %d, want ~2500 (quadratic fix_chance=0.25)", fixed)
	}
}

func TestRevealStep_Deterministic(t *testing.T) {
	grid := [][]rune{{'A', 'B'}, {'C', 'D'}}
	target := [][]rune{{'X', 'Y'}, {'Z', 'W'}}
	cs := charsetFor(AnimationCharsetClassic)

	g1 := revealStep(grid, target, 2, 5, cs, rand.New(rand.NewSource(99)))
	g2 := revealStep(grid, target, 2, 5, cs, rand.New(rand.NewSource(99)))
	for r := range g1 {
		for c := range g1[r] {
			if g1[r][c] != g2[r][c] {
				t.Fatalf("revealStep not deterministic: cell(%d,%d) = %q vs %q", r, c, g1[r][c], g2[r][c])
			}
		}
	}
}

func TestAnimateTransition_TallerCurrentEndState(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleScramble
	cfg.AnimationSteps = 1
	cfg.AnimationDelayMs = 1

	// Current frame is TALLER than target — final render must be the exact
	// target, not the taller current frame.
	AnimateTransition(term, []string{"AAAAA", "BBBBB", "CCCCC"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_EmptyTargetNoPanic(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleScramble
	cfg.AnimationSteps = 3
	cfg.AnimationDelayMs = 1

	// Empty target must not panic; final render is a no-op → screen blank.
	AnimateTransition(term, []string{"AA", "BB"}, []string{}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_ClampsStepsUpperBound(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleScramble
	cfg.AnimationSteps = 100 // clamped to 50
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_ClampsDelayUpperBound(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleCollapse
	cfg.AnimationSteps = 1
	cfg.AnimationDelayMs = 1000 // clamped to 500

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_UnknownStyleInstant(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyle("Bogus")
	cfg.AnimationSteps = 5
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

// ---------------------------------------------------------------------------
// Smart animation (task 3.3)
// ---------------------------------------------------------------------------

func TestUnifiedGrids_CurrentTallerWider(t *testing.T) {
	current := []string{"AAAAA", "BBBBB", "CCCCC"}
	target := []string{"XY", "ZW"}

	cur, tgt, gridH, gridW := unifiedGrids(current, target)
	if gridH != 3 {
		t.Errorf("gridH = %d, want 3 (max of 3 and 2)", gridH)
	}
	if gridW != 5 {
		t.Errorf("gridW = %d, want 5 (max of 5 and 2)", gridW)
	}
	if len(cur) != 3 || len(tgt) != 3 {
		t.Fatalf("grid heights = %d/%d, want 3/3", len(cur), len(tgt))
	}
	// Current keeps its content; target is padded with blank rows.
	for r := 0; r < 3; r++ {
		if len(cur[r]) != 5 || len(tgt[r]) != 5 {
			t.Fatalf("row %d widths = %d/%d, want 5/5", r, len(cur[r]), len(tgt[r]))
		}
		for c := 0; c < 5; c++ {
			if cur[r][c] != rune(current[r][c]) {
				t.Errorf("cur(%d,%d) = %q, want %q", r, c, cur[r][c], current[r][c])
			}
		}
	}
	wantTgt := [][]rune{
		{'X', 'Y', ' ', ' ', ' '},
		{'Z', 'W', ' ', ' ', ' '},
		{' ', ' ', ' ', ' ', ' '},
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 5; c++ {
			if tgt[r][c] != wantTgt[r][c] {
				t.Errorf("tgt(%d,%d) = %q, want %q", r, c, tgt[r][c], wantTgt[r][c])
			}
		}
	}
}

func TestUnifiedGrids_TargetTallerWider(t *testing.T) {
	current := []string{"XY", "ZW"}
	target := []string{"AAAAA", "BBBBB", "CCCCC"}

	cur, tgt, gridH, gridW := unifiedGrids(current, target)
	if gridH != 3 {
		t.Errorf("gridH = %d, want 3 (max of 2 and 3)", gridH)
	}
	if gridW != 5 {
		t.Errorf("gridW = %d, want 5 (max of 2 and 5)", gridW)
	}
	if len(cur) != 3 || len(tgt) != 3 {
		t.Fatalf("grid heights = %d/%d, want 3/3", len(cur), len(tgt))
	}
	// Current is padded with blank rows; target keeps its content.
	wantCur := [][]rune{
		{'X', 'Y', ' ', ' ', ' '},
		{'Z', 'W', ' ', ' ', ' '},
		{' ', ' ', ' ', ' ', ' '},
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 5; c++ {
			if cur[r][c] != wantCur[r][c] {
				t.Errorf("cur(%d,%d) = %q, want %q", r, c, cur[r][c], wantCur[r][c])
			}
			if tgt[r][c] != rune(target[r][c]) {
				t.Errorf("tgt(%d,%d) = %q, want %q", r, c, tgt[r][c], target[r][c])
			}
		}
	}
}

func TestDiffPositions_FindsAllDifferingCells(t *testing.T) {
	cur := [][]rune{{'A', 'B', 'C'}, {'D', 'E', 'F'}}
	tgt := [][]rune{{'A', 'X', 'C'}, {'Y', 'E', 'Z'}}

	diffs := diffPositions(cur, tgt)
	if len(diffs) != 3 {
		t.Fatalf("diff count = %d, want 3", len(diffs))
	}
	want := map[gridPos]bool{{0, 1}: true, {1, 0}: true, {1, 2}: true}
	for _, d := range diffs {
		if !want[d] {
			t.Errorf("unexpected diff at %+v", d)
		}
	}
}

func TestDiffPositions_EmptyWhenIdentical(t *testing.T) {
	cur := [][]rune{{'A', 'B'}, {'C', 'D'}}
	tgt := [][]rune{{'A', 'B'}, {'C', 'D'}}

	diffs := diffPositions(cur, tgt)
	if len(diffs) != 0 {
		t.Fatalf("diff count = %d, want 0 for identical frames", len(diffs))
	}
}

func TestSmartBounds_CenterAndMaxDist(t *testing.T) {
	diffs := []gridPos{{0, 0}, {0, 4}, {4, 0}, {4, 4}}
	centerR, centerC, maxDist := smartBounds(diffs)
	if centerR != 2.0 {
		t.Errorf("centerR = %v, want 2.0", centerR)
	}
	if centerC != 2.0 {
		t.Errorf("centerC = %v, want 2.0", centerC)
	}
	wantDist := math.Sqrt(2.0*2.0 + 2.0*2.0) // hypot(2,2)
	if math.Abs(maxDist-wantDist) > 1e-9 {
		t.Errorf("maxDist = %v, want %v", maxDist, wantDist)
	}
}

func TestSmartBounds_SingleCellFloor(t *testing.T) {
	diffs := []gridPos{{2, 3}}
	centerR, centerC, maxDist := smartBounds(diffs)
	if centerR != 2.0 {
		t.Errorf("centerR = %v, want 2.0", centerR)
	}
	if centerC != 3.0 {
		t.Errorf("centerC = %v, want 3.0", centerC)
	}
	if maxDist != 0.01 {
		t.Errorf("maxDist = %v, want 0.01 (floor)", maxDist)
	}
}

func TestSmartFixChance_WaveCenterCell(t *testing.T) {
	// Center cell: normalized dist 0 → chance = 1.0 + progress, clamped to 1.0.
	chance := smartFixChance(SmartWaveWave, 2, 2, 2.0, 2.0, 2.0, 0.5)
	if chance != 1.0 {
		t.Errorf("center cell chance = %v, want 1.0 (clamped)", chance)
	}
}

func TestSmartFixChance_WaveFarCell(t *testing.T) {
	// Far cell at normalized dist 1.0 → chance = 0.0 + progress.
	chance := smartFixChance(SmartWaveWave, 0, 2, 2.0, 2.0, 2.0, 0.5)
	if math.Abs(chance-0.5) > 1e-9 {
		t.Errorf("far cell chance = %v, want 0.5", chance)
	}
	// At progress 0 the far cell has chance 0; the center cell still 1.0.
	if c := smartFixChance(SmartWaveWave, 0, 2, 2.0, 2.0, 2.0, 0.0); c != 0.0 {
		t.Errorf("far cell chance at progress 0 = %v, want 0.0", c)
	}
	if c := smartFixChance(SmartWaveWave, 2, 2, 2.0, 2.0, 2.0, 0.0); c != 1.0 {
		t.Errorf("center cell chance at progress 0 = %v, want 1.0", c)
	}
}

func TestSmartFixChance_WaveMonotonicInDistance(t *testing.T) {
	// At a fixed progress, chance must not increase with distance from centre.
	centerR, centerC := 2.0, 2.0
	maxDist := 2.0
	progress := 0.4
	// Cells ordered by increasing distance from centre (0, 1, 1, 2, 2).
	cells := []gridPos{{2, 2}, {1, 2}, {3, 2}, {0, 2}, {4, 2}}
	prev := 2.0 // chance is clamped to ≤ 1.0
	for _, d := range cells {
		chance := smartFixChance(SmartWaveWave, d.r, d.c, centerR, centerC, maxDist, progress)
		if chance > prev+1e-9 {
			t.Errorf("chance at %+v = %v, greater than previous %v (must be monotonic in distance)", d, chance, prev)
		}
		prev = chance
	}
}

func TestSmartFixChance_RandomAlwaysProgress(t *testing.T) {
	for _, progress := range []float64{0.0, 0.25, 0.5, 1.0} {
		chance := smartFixChance(SmartWaveRandom, 3, 7, 1.0, 1.0, 5.0, progress)
		if chance != progress {
			t.Errorf("Random wave chance = %v, want progress %v", chance, progress)
		}
	}
	// Unknown wave values fall back to uniform progress (Rust behaviour).
	if c := smartFixChance(SmartWave("Bogus"), 0, 0, 0, 0, 1, 0.3); c != 0.3 {
		t.Errorf("unknown wave chance = %v, want 0.3", c)
	}
}

func TestSmartSteps_ModeClamping(t *testing.T) {
	if s := smartSteps(SmartModeInstant, 5); s != 1 {
		t.Errorf("Instant steps = %d, want 1", s)
	}
	if s := smartSteps(SmartModeInstant, 100); s != 1 {
		t.Errorf("Instant steps (100) = %d, want 1", s)
	}
	if s := smartSteps(SmartModeScramble, 5); s != 8 {
		t.Errorf("Scramble steps (5) = %d, want 8 (minimum)", s)
	}
	if s := smartSteps(SmartModeScramble, 8); s != 8 {
		t.Errorf("Scramble steps (8) = %d, want 8", s)
	}
	if s := smartSteps(SmartModeScramble, 20); s != 20 {
		t.Errorf("Scramble steps (20) = %d, want 20", s)
	}
}

func TestSmartDelay_ModeClamping(t *testing.T) {
	if d := smartDelay(SmartModeInstant, 1); d != 50 {
		t.Errorf("Instant delay (1) = %d, want 50 (minimum)", d)
	}
	if d := smartDelay(SmartModeInstant, 50); d != 50 {
		t.Errorf("Instant delay (50) = %d, want 50", d)
	}
	if d := smartDelay(SmartModeInstant, 100); d != 100 {
		t.Errorf("Instant delay (100) = %d, want 100", d)
	}
	if d := smartDelay(SmartModeScramble, 30); d != 30 {
		t.Errorf("Scramble delay (30) = %d, want 30", d)
	}
	if d := smartDelay(SmartModeScramble, 1); d != 1 {
		t.Errorf("Scramble delay (1) = %d, want 1", d)
	}
}

func TestSmartStep_InstantModeFixesAll(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	cur := [][]rune{{'A', 'B'}, {'C', 'D'}}
	tgt := [][]rune{{'X', 'Y'}, {'Z', 'W'}}
	diffs := diffPositions(cur, tgt)
	fixed := make([]bool, len(diffs))
	cs := charsetFor(AnimationCharsetClassic)

	// Instant mode → single step with progress 1.0 → every diff cell locks.
	smartStep(cur, tgt, diffs, fixed, 1, 1, SmartModeInstant, SmartWaveRandom, cs, rng)
	for i, d := range diffs {
		if !fixed[i] {
			t.Errorf("diff %+v not fixed after Instant step", d)
		}
		if cur[d.r][d.c] != tgt[d.r][d.c] {
			t.Errorf("cell(%d,%d) = %q, want target %q", d.r, d.c, cur[d.r][d.c], tgt[d.r][d.c])
		}
	}
}

func TestSmartStep_ScrambleMix(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const size = 10
	cur := make([][]rune, size)
	tgt := make([][]rune, size)
	for r := 0; r < size; r++ {
		cur[r] = make([]rune, size)
		tgt[r] = make([]rune, size)
		for c := 0; c < size; c++ {
			cur[r][c] = 'C'
			tgt[r][c] = 'T'
		}
	}
	diffs := diffPositions(cur, tgt)
	fixed := make([]bool, len(diffs))
	cs := charsetFor(AnimationCharsetClassic)

	// step 2/4 → progress 0.5 → mix of fixed and charset cells.
	smartStep(cur, tgt, diffs, fixed, 2, 4, SmartModeScramble, SmartWaveRandom, cs, rng)
	fixedCount := 0
	charsetCount := 0
	for i, d := range diffs {
		if fixed[i] {
			fixedCount++
			if cur[d.r][d.c] != 'T' {
				t.Errorf("fixed cell(%d,%d) = %q, want target 'T'", d.r, d.c, cur[d.r][d.c])
			}
		} else {
			charsetCount++
			if !runeInCharset(cur[d.r][d.c], cs) {
				t.Errorf("unfixed cell(%d,%d) = %q, want a charset char", d.r, d.c, cur[d.r][d.c])
			}
		}
	}
	if fixedCount == 0 {
		t.Error("step 2/4 produced no fixed cells")
	}
	if charsetCount == 0 {
		t.Error("step 2/4 produced no charset cells")
	}
}

func TestSmartStep_FixedStaysFixed(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const size = 10
	cur := make([][]rune, size)
	tgt := make([][]rune, size)
	for r := 0; r < size; r++ {
		cur[r] = make([]rune, size)
		tgt[r] = make([]rune, size)
		for c := 0; c < size; c++ {
			cur[r][c] = 'C'
			tgt[r][c] = 'T'
		}
	}
	diffs := diffPositions(cur, tgt)
	fixed := make([]bool, len(diffs))
	cs := charsetFor(AnimationCharsetClassic)

	smartStep(cur, tgt, diffs, fixed, 1, 4, SmartModeScramble, SmartWaveRandom, cs, rng)
	// Snapshot which cells fixed in step 1.
	wasFixed := make([]bool, len(fixed))
	copy(wasFixed, fixed)

	smartStep(cur, tgt, diffs, fixed, 2, 4, SmartModeScramble, SmartWaveRandom, cs, rng)
	for i, d := range diffs {
		if wasFixed[i] {
			if !fixed[i] {
				t.Errorf("cell(%d,%d) unfixed after step 2 — fixed[] must never reset", d.r, d.c)
			}
			if cur[d.r][d.c] != tgt[d.r][d.c] {
				t.Errorf("fixed cell(%d,%d) = %q, want target %q (never unfixes)", d.r, d.c, cur[d.r][d.c], tgt[d.r][d.c])
			}
		}
	}
}

func TestSmartStep_Deterministic(t *testing.T) {
	cur := [][]rune{{'A', 'B'}, {'C', 'D'}}
	tgt := [][]rune{{'X', 'Y'}, {'Z', 'W'}}
	diffs := diffPositions(cur, tgt)
	cs := charsetFor(AnimationCharsetClassic)

	run := func() ([][]rune, []bool) {
		g := [][]rune{{'A', 'B'}, {'C', 'D'}}
		f := make([]bool, len(diffs))
		smartStep(g, tgt, diffs, f, 2, 5, SmartModeScramble, SmartWaveRandom, cs, rand.New(rand.NewSource(99)))
		return g, f
	}
	g1, f1 := run()
	g2, f2 := run()
	for i := range f1 {
		if f1[i] != f2[i] {
			t.Fatalf("smartStep not deterministic: fixed[%d] = %v vs %v", i, f1[i], f2[i])
		}
	}
	for r := range g1 {
		for c := range g1[r] {
			if g1[r][c] != g2[r][c] {
				t.Fatalf("smartStep not deterministic: cell(%d,%d) = %q vs %q", r, c, g1[r][c], g2[r][c])
			}
		}
	}
}

func TestAnimateSmart_EndStateExactTarget(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.SmartMode = SmartModeScramble
	cfg.SmartWave = SmartWaveRandom
	cfg.AnimationSteps = 1 // clamped to 8 by Smart mode
	cfg.AnimationDelayMs = 1

	animateSmart(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	// Last rendered step is the fully-fixed padded target == exact target here.
	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateSmart_IdenticalFramesEarlyReturn(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.SmartMode = SmartModeScramble
	cfg.SmartWave = SmartWaveRandom
	cfg.AnimationSteps = 8
	cfg.AnimationDelayMs = 1

	// Render a DIFFERENT frame first; identical-frame animation must return
	// early without rendering, leaving the screen unchanged.
	term.RenderFrame([]string{"XY", "ZW"})
	animateSmart(term, []string{"AA", "BB"}, []string{"AA", "BB"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_SmartTallerCurrentEndState(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleSmart
	cfg.SmartMode = SmartModeScramble
	cfg.SmartWave = SmartWaveRandom
	cfg.AnimationSteps = 1
	cfg.AnimationDelayMs = 1

	// Current frame is TALLER than target — the final render after the switch
	// must be the exact target, not the padded grid.
	AnimateTransition(term, []string{"AAAAA", "BBBBB", "CCCCC"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestSmartWave_Ordering(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	// 1x5 line of diffs: centre at (0,2), max_dist = 2.0.
	cur := [][]rune{{'C', 'C', 'C', 'C', 'C'}}
	tgt := [][]rune{{'T', 'T', 'T', 'T', 'T'}}
	diffs := diffPositions(cur, tgt)
	fixed := make([]bool, len(diffs))
	cs := charsetFor(AnimationCharsetClassic)

	// Wave fix chances (deterministic, seed-independent):
	//   centre (c=2): 1.0 + progress → clamped 1.0 → fixed at step 1
	//   inner  (c=1,3): 0.5 + progress → 1.0 at step 4 → fixed by step 4
	//   outer  (c=0,4): progress → 1.0 at step 8 → fixed by step 8
	const steps = 8
	fixedAt := make([]int, len(diffs)) // step at which each cell first fixed
	for step := 1; step <= steps; step++ {
		smartStep(cur, tgt, diffs, fixed, step, steps, SmartModeScramble, SmartWaveWave, cs, rng)
		for i := range diffs {
			if fixed[i] && fixedAt[i] == 0 {
				fixedAt[i] = step
			}
		}
	}

	// Fixed set must be monotonic (never unfixes) — implied by fixedAt, but
	// verify the final state: everything fixed by the last step.
	for i, d := range diffs {
		if !fixed[i] {
			t.Errorf("cell(%d,%d) not fixed after final step", d.r, d.c)
		}
		if cur[d.r][d.c] != tgt[d.r][d.c] {
			t.Errorf("cell(%d,%d) = %q, want target %q", d.r, d.c, cur[d.r][d.c], tgt[d.r][d.c])
		}
	}

	// Centre fixes at step 1; inner cells by step 4; outer cells by step 8.
	if fixedAt[2] != 1 {
		t.Errorf("centre cell fixed at step %d, want 1", fixedAt[2])
	}
	for _, i := range []int{1, 3} {
		if fixedAt[i] == 0 || fixedAt[i] > 4 {
			t.Errorf("inner cell %d fixed at step %d, want 1..4", i, fixedAt[i])
		}
	}
	for _, i := range []int{0, 4} {
		if fixedAt[i] == 0 || fixedAt[i] > 8 {
			t.Errorf("outer cell %d fixed at step %d, want 1..8", i, fixedAt[i])
		}
	}
	// Ordering: centre before inner before outer (deadlines are strict).
	if fixedAt[2] > fixedAt[1] || fixedAt[2] > fixedAt[3] {
		t.Errorf("centre fixed at %d, later than inner cells (%d,%d)", fixedAt[2], fixedAt[1], fixedAt[3])
	}
}

// ---------------------------------------------------------------------------
// Smart animation — targeted tests added by test_engineer (task 3.3 pass)
// ---------------------------------------------------------------------------

func TestSmartBounds_SingleRowDiff(t *testing.T) {
	// Single row spanning columns 0..4: halfH = 0, halfW = 2 →
	// maxDist = hypot(0, 2) = 2.0 (no floor needed).
	diffs := []gridPos{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}}
	centerR, centerC, maxDist := smartBounds(diffs)
	if centerR != 0.0 {
		t.Errorf("centerR = %v, want 0.0", centerR)
	}
	if centerC != 2.0 {
		t.Errorf("centerC = %v, want 2.0", centerC)
	}
	if math.Abs(maxDist-2.0) > 1e-9 {
		t.Errorf("maxDist = %v, want 2.0 (hypot(0,2))", maxDist)
	}
}

func TestSmartBounds_MinColumnUpdate(t *testing.T) {
	// First diff carries the max row/col; later diffs must shrink minR and
	// minC. Exercises the d.r < minR and d.c < minC branches (not hit by
	// ascending-order inputs).
	diffs := []gridPos{{4, 4}, {0, 0}}
	centerR, centerC, maxDist := smartBounds(diffs)
	if centerR != 2.0 {
		t.Errorf("centerR = %v, want 2.0", centerR)
	}
	if centerC != 2.0 {
		t.Errorf("centerC = %v, want 2.0", centerC)
	}
	wantDist := math.Sqrt(2.0*2.0 + 2.0*2.0) // hypot(2,2)
	if math.Abs(maxDist-wantDist) > 1e-9 {
		t.Errorf("maxDist = %v, want %v", maxDist, wantDist)
	}
}

func TestSmartFixChance_WaveClampsNegative(t *testing.T) {
	// Cell far outside the bounding box: 1.0 - dist/maxDist + progress < 0
	// must clamp to 0.0, not go negative.
	chance := smartFixChance(SmartWaveWave, 10, 10, 2.0, 2.0, 2.0, 0.1)
	if chance != 0.0 {
		t.Errorf("out-of-box cell chance = %v, want 0.0 (clamped)", chance)
	}
}

func TestSmartStep_WaveSingleCellFixesImmediately(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	cur := [][]rune{{'A'}}
	tgt := [][]rune{{'X'}}
	diffs := diffPositions(cur, tgt)
	fixed := make([]bool, len(diffs))
	cs := charsetFor(AnimationCharsetClassic)

	// Single-cell diff: maxDist floored to 0.01, dist = 0 →
	// chance = 1.0 + progress clamped to 1.0 → fixed on step 1 regardless of rng.
	smartStep(cur, tgt, diffs, fixed, 1, 8, SmartModeScramble, SmartWaveWave, cs, rng)
	if !fixed[0] {
		t.Error("single-cell Wave diff not fixed at step 1 (chance clamped to 1.0)")
	}
	if cur[0][0] != 'X' {
		t.Errorf("cell(0,0) = %q, want target 'X'", cur[0][0])
	}
}

func TestSmartStep_NoDiffsNoop(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	cur := [][]rune{{'A', 'B'}, {'C', 'D'}}
	tgt := [][]rune{{'A', 'B'}, {'C', 'D'}}
	diffs := diffPositions(cur, tgt) // empty
	fixed := make([]bool, 0)
	cs := charsetFor(AnimationCharsetClassic)

	smartStep(cur, tgt, diffs, fixed, 1, 8, SmartModeScramble, SmartWaveWave, cs, rng)
	for r := 0; r < 2; r++ {
		for c := 0; c < 2; c++ {
			if cur[r][c] != tgt[r][c] {
				t.Errorf("cell(%d,%d) = %q, want %q (no diffs → no-op)", r, c, cur[r][c], tgt[r][c])
			}
		}
	}
}

func TestSmartStep_UnicodeCharsetCells(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	cur := [][]rune{{'A', 'B'}, {'C', 'D'}}
	tgt := [][]rune{{'X', 'Y'}, {'Z', 'W'}}
	diffs := diffPositions(cur, tgt)
	fixed := make([]bool, len(diffs))
	cs := charsetFor(AnimationCharsetUnicode)

	smartStep(cur, tgt, diffs, fixed, 1, 4, SmartModeScramble, SmartWaveRandom, cs, rng)
	for i, d := range diffs {
		if fixed[i] {
			if cur[d.r][d.c] != tgt[d.r][d.c] {
				t.Errorf("fixed cell(%d,%d) = %q, want target %q", d.r, d.c, cur[d.r][d.c], tgt[d.r][d.c])
			}
		} else if !runeInCharset(cur[d.r][d.c], cs) {
			t.Errorf("unfixed cell(%d,%d) = %q, want a Unicode charset char", d.r, d.c, cur[d.r][d.c])
		}
	}
}

func TestSmartStep_WaveDeterministic(t *testing.T) {
	cur := [][]rune{{'A', 'B', 'C'}, {'D', 'E', 'F'}, {'G', 'H', 'I'}}
	tgt := [][]rune{{'X', 'Y', 'Z'}, {'W', 'V', 'U'}, {'T', 'S', 'R'}}
	diffs := diffPositions(cur, tgt)
	cs := charsetFor(AnimationCharsetClassic)

	run := func() ([][]rune, []bool) {
		g := [][]rune{{'A', 'B', 'C'}, {'D', 'E', 'F'}, {'G', 'H', 'I'}}
		f := make([]bool, len(diffs))
		smartStep(g, tgt, diffs, f, 3, 8, SmartModeScramble, SmartWaveWave, cs, rand.New(rand.NewSource(1234)))
		return g, f
	}
	g1, f1 := run()
	g2, f2 := run()
	for i := range f1 {
		if f1[i] != f2[i] {
			t.Fatalf("Wave smartStep not deterministic: fixed[%d] = %v vs %v", i, f1[i], f2[i])
		}
	}
	for r := range g1 {
		for c := range g1[r] {
			if g1[r][c] != g2[r][c] {
				t.Fatalf("Wave smartStep not deterministic: cell(%d,%d) = %q vs %q", r, c, g1[r][c], g2[r][c])
			}
		}
	}
}

func TestAnimateSmart_WaveEndStateExactTarget(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.SmartMode = SmartModeScramble
	cfg.SmartWave = SmartWaveWave
	cfg.AnimationSteps = 8
	cfg.AnimationDelayMs = 1

	animateSmart(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateSmart_FullGridDiff(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.SmartMode = SmartModeScramble
	cfg.SmartWave = SmartWaveRandom
	cfg.AnimationSteps = 8
	cfg.AnimationDelayMs = 1

	// 5x5 grid where every cell differs — the diff spans the full grid.
	current := []string{"AAAAA", "BBBBB", "CCCCC", "DDDDD", "EEEEE"}
	target := []string{"11111", "22222", "33333", "44444", "55555"}

	animateSmart(term, current, target, cfg, rng)

	// Last animated step is the fully-fixed padded target (5x5), centered on
	// 40x10: originX = (40-5)/2 = 17, originY = (10-5)/2 = 2.
	expected := makeEmptyGrid(40, 10)
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			expected[2+r][17+c] = rune(target[r][c])
		}
	}
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_SmartInstantEndState(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleSmart
	cfg.SmartMode = SmartModeInstant
	cfg.SmartWave = SmartWaveRandom
	cfg.AnimationSteps = 5
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_SmartUnicodeCharset(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleSmart
	cfg.SmartMode = SmartModeScramble
	cfg.SmartWave = SmartWaveRandom
	cfg.AnimationCharset = AnimationCharsetUnicode
	cfg.AnimationSteps = 8
	cfg.AnimationDelayMs = 1

	AnimateTransition(term, []string{"AA", "BB"}, []string{"XY", "ZW"}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	expected[4][19] = 'X'
	expected[4][20] = 'Y'
	expected[5][19] = 'Z'
	expected[5][20] = 'W'
	assertGridExact(t, sim, 40, 10, expected)
}

func TestAnimateTransition_SmartEmptyTarget(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleSmart
	cfg.SmartMode = SmartModeScramble
	cfg.SmartWave = SmartWaveRandom
	cfg.AnimationSteps = 8
	cfg.AnimationDelayMs = 1

	// Empty target: unified grid is all-blank, every current cell is a diff
	// that animates to blank. Must not panic; final render is a no-op.
	AnimateTransition(term, []string{"AA", "BB"}, []string{}, cfg, rng)

	expected := makeEmptyGrid(40, 10)
	assertGridExact(t, sim, 40, 10, expected)
}

func TestUnifiedGrids_EmptyFrames(t *testing.T) {
	cur, tgt, gridH, gridW := unifiedGrids(nil, nil)
	if gridH != 0 || gridW != 0 {
		t.Fatalf("empty frames: gridH=%d gridW=%d, want 0/0", gridH, gridW)
	}
	if len(cur) != 0 || len(tgt) != 0 {
		t.Fatalf("empty frames: grid heights %d/%d, want 0/0", len(cur), len(tgt))
	}

	// Current empty, target non-empty: grid dims follow the target, current is
	// all blank.
	cur2, tgt2, h2, w2 := unifiedGrids(nil, []string{"AB", "CDE"})
	if h2 != 2 || w2 != 3 {
		t.Fatalf("current-empty: gridH=%d gridW=%d, want 2/3", h2, w2)
	}
	if len(cur2) != 2 || len(tgt2) != 2 {
		t.Fatalf("current-empty: heights %d/%d, want 2/2", len(cur2), len(tgt2))
	}
	for r := 0; r < 2; r++ {
		for c := 0; c < 3; c++ {
			if cur2[r][c] != ' ' {
				t.Errorf("cur2(%d,%d) = %q, want ' ' (blank current)", r, c, cur2[r][c])
			}
		}
	}
}

func TestDiffPositions_EmptyGrids(t *testing.T) {
	diffs := diffPositions(nil, nil)
	if len(diffs) != 0 {
		t.Fatalf("empty grids diff count = %d, want 0", len(diffs))
	}
}