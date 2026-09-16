package main

import (
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

func TestAnimateTransition_SmartStyleIsInstant(t *testing.T) {
	term, sim := newTestTerminal(t, 40, 10)
	defer term.Close()
	rng := rand.New(rand.NewSource(42))

	cfg := DefaultConfig()
	cfg.AnimationStyle = AnimationStyleSmart // task 3.3 — must fall back to instant
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