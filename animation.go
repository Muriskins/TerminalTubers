package main

import (
	"math"
	"math/rand"
	"time"
)

// collapseChars are hardcoded for the Collapse animation — they do NOT use the
// configured charset. Kept as a package-level variable for testability.
var collapseChars = []rune{'/', '\\', '*', '#', '@'}

// charsetFor returns the rune slice for the given AnimationCharset. Unknown
// values fall back to Classic (matching Rust behaviour).
func charsetFor(cs AnimationCharset) []rune {
	switch cs {
	case AnimationCharsetClassic:
		return []rune{'@', '#', '%', '*', '=', '-', '+', ':', '.', ' '}
	case AnimationCharsetUnicode:
		return []rune{'█', '▓', '▒', '░', '╔', '╗', '╚', '╝', '║', '═', ' '}
	case AnimationCharsetAll:
		chars := make([]rune, 0, 95)
		for r := '!'; r <= '~'; r++ {
			chars = append(chars, r)
		}
		chars = append(chars, ' ')
		return chars
	default:
		return charsetFor(AnimationCharsetClassic)
	}
}

// toGrid converts a frame (slice of strings) into a rune grid. Every row is
// padded with spaces to tw. If the frame has fewer than th rows, blank rows are
// appended. If the frame has MORE than th rows, the extra rows are kept — the
// grid is never truncated (1:1 Rust fidelity: a taller current frame keeps its
// extra rows untouched during Scramble).
func toGrid(frame []string, th, tw int) [][]rune {
	rows := len(frame)
	if rows < th {
		rows = th
	}
	grid := make([][]rune, rows)
	for r := 0; r < rows; r++ {
		row := make([]rune, tw)
		if r < len(frame) {
			runes := []rune(frame[r])
			for c := 0; c < tw; c++ {
				if c < len(runes) {
					row[c] = runes[c]
				} else {
					row[c] = ' '
				}
			}
		} else {
			for c := 0; c < tw; c++ {
				row[c] = ' '
			}
		}
		grid[r] = row
	}
	return grid
}

// gridToLines converts a rune grid back to a string slice suitable for
// RenderFrame. Each row is converted to a string.
func gridToLines(grid [][]rune) []string {
	lines := make([]string, len(grid))
	for r, row := range grid {
		lines[r] = string(row)
	}
	return lines
}

// scrambleStep returns the next grid for the Scramble animation. Rows 0..th-1
// are re-evaluated: fix_chance = step/steps (linear); if rng < fix_chance the
// cell is fixed to the target, otherwise a random charset character is placed.
// Rows at or beyond th (present only when the current frame is taller than the
// target) are preserved untouched.
func scrambleStep(current, target [][]rune, step, steps int, charset []rune, rng *rand.Rand) [][]rune {
	th := len(target)
	tw := 0
	if th > 0 {
		tw = len(target[0])
	}
	fixChance := float64(step) / float64(steps)
	grid := make([][]rune, len(current))
	for r := range current {
		row := make([]rune, len(current[r]))
		copy(row, current[r])
		if r < th {
			for c := 0; c < tw && c < len(row); c++ {
				if rng.Float64() < fixChance {
					row[c] = target[r][c]
				} else {
					row[c] = charset[rng.Intn(len(charset))]
				}
			}
		}
		grid[r] = row
	}
	return grid
}

// collapseStep returns a new grid for the given step of the Collapse animation.
// The centre of the grid reveals the target first; as progress increases the
// threshold shrinks until only the centre cell is target at the final step.
// Uses hardcoded collapseChars regardless of the configured charset.
func collapseStep(target [][]rune, step, steps int, rng *rand.Rand) [][]rune {
	th := len(target)
	tw := 0
	if th > 0 {
		tw = len(target[0])
	}
	midRow := th / 2
	midCol := tw / 2
	maxDist := float64(midRow*midRow + midCol*midCol)
	progress := float64(step) / float64(steps)
	threshold := (1.0 - progress) * math.Sqrt(maxDist)

	grid := make([][]rune, th)
	for r := 0; r < th; r++ {
		row := make([]rune, tw)
		for c := 0; c < tw; c++ {
			dr := r - midRow
			dc := c - midCol
			dist := math.Sqrt(float64(dr*dr + dc*dc))
			if dist <= threshold {
				row[c] = target[r][c]
			} else {
				row[c] = collapseChars[rng.Intn(len(collapseChars))]
			}
		}
		grid[r] = row
	}
	return grid
}

// revealStep returns a new grid for the given step of the Reveal animation.
// fix_chance = progress² (quadratic). Every cell is re-evaluated every step —
// there is no fixed[] array (that is Smart mode, task 3.3). The grid argument
// is accepted for signature symmetry but is not consulted.
func revealStep(grid, target [][]rune, step, steps int, charset []rune, rng *rand.Rand) [][]rune {
	th := len(target)
	tw := 0
	if th > 0 {
		tw = len(target[0])
	}
	progress := float64(step) / float64(steps)
	fixChance := progress * progress

	newGrid := make([][]rune, th)
	for r := 0; r < th; r++ {
		row := make([]rune, tw)
		for c := 0; c < tw; c++ {
			if rng.Float64() < fixChance {
				row[c] = target[r][c]
			} else {
				row[c] = charset[rng.Intn(len(charset))]
			}
		}
		newGrid[r] = row
	}
	return newGrid
}

// ---------------------------------------------------------------------------
// Smart animation (diff-only) — task 3.3
// ---------------------------------------------------------------------------

// gridPos holds a row/column position in a rune grid.
type gridPos struct{ r, c int }

// unifiedGrids pads both frames to the unified grid dimensions — the max
// height and max width across the two frames. Rows beyond a frame's own height
// are blank; characters beyond the unified width are dropped. Uses toGrid,
// which never truncates a taller/wider frame.
func unifiedGrids(current, target []string) (cur, tgt [][]rune, gridH, gridW int) {
	gridH = len(target)
	gridW = 0
	for _, line := range target {
		if len(line) > gridW {
			gridW = len(line)
		}
	}
	if ch := len(current); ch > gridH {
		gridH = ch
	}
	for _, line := range current {
		if len(line) > gridW {
			gridW = len(line)
		}
	}
	return toGrid(current, gridH, gridW), toGrid(target, gridH, gridW), gridH, gridW
}

// diffPositions returns every (r,c) where cur and tgt differ. Both grids must
// have the same dimensions.
func diffPositions(cur, tgt [][]rune) []gridPos {
	var diffs []gridPos
	for r := range cur {
		for c := range cur[r] {
			if cur[r][c] != tgt[r][c] {
				diffs = append(diffs, gridPos{r, c})
			}
		}
	}
	return diffs
}

// smartBounds computes the bounding box centre and max_dist (half-diagonal)
// over diff positions. max_dist is floored to 0.01 to avoid division by zero
// for single-cell or single-row/column diffs.
func smartBounds(diffs []gridPos) (centerR, centerC, maxDist float64) {
	minR, maxR := diffs[0].r, diffs[0].r
	minC, maxC := diffs[0].c, diffs[0].c
	for _, d := range diffs[1:] {
		if d.r < minR {
			minR = d.r
		}
		if d.r > maxR {
			maxR = d.r
		}
		if d.c < minC {
			minC = d.c
		}
		if d.c > maxC {
			maxC = d.c
		}
	}
	centerR = float64(minR+maxR) / 2.0
	centerC = float64(minC+maxC) / 2.0
	halfH := float64(maxR-minR) / 2.0
	halfW := float64(maxC-minC) / 2.0
	maxDist = math.Max(math.Sqrt(halfH*halfH+halfW*halfW), 0.01)
	return centerR, centerC, maxDist
}

// smartFixChance returns the probability that diff cell (r,c) locks to the
// target in the current step. Wave mode biases cells closer to the bounding
// box centre; Random mode uses uniform progress.
func smartFixChance(wave SmartWave, r, c int, centerR, centerC, maxDist, progress float64) float64 {
	if wave == SmartWaveWave {
		dr := float64(r) - centerR
		dc := float64(c) - centerC
		dist := math.Sqrt(dr*dr + dc*dc)
		normalized := dist / maxDist
		chance := 1.0 - normalized + progress
		if chance < 0.0 {
			return 0.0
		}
		if chance > 1.0 {
			return 1.0
		}
		return chance
	}
	// SmartWaveRandom (and any other value): uniform linear progress.
	return progress
}

// smartSteps returns the effective step count after mode clamping.
func smartSteps(mode SmartMode, steps int) int {
	if mode == SmartModeInstant {
		return 1
	}
	// SmartModeScramble: minimum 8 steps.
	if steps < 8 {
		return 8
	}
	return steps
}

// smartDelay returns the effective delay in milliseconds after mode clamping.
func smartDelay(mode SmartMode, delayMs int) int {
	if mode == SmartModeInstant {
		if delayMs < 50 {
			return 50
		}
		return delayMs
	}
	return delayMs
}

// smartStep applies one step of the Smart animation: for each unfixed diff
// cell, compute the fix chance and either lock it to the target or fill it
// with a random charset character. Cells that are already fixed are never
// modified (the fixed array is never reset). The mode parameter is carried for
// signature symmetry — its effect is already encoded in steps (Instant → 1).
func smartStep(cur, tgt [][]rune, diffs []gridPos, fixed []bool, step, steps int,
	mode SmartMode, wave SmartWave, charset []rune, rng *rand.Rand,
) {
	progress := float64(step) / float64(steps)

	// Pre-compute bounds only needed for Wave mode (constant across cells in
	// a step, but computed once per call for clarity).
	var centerR, centerC, maxDist float64
	if wave == SmartWaveWave && len(diffs) > 0 {
		centerR, centerC, maxDist = smartBounds(diffs)
	}

	for i, d := range diffs {
		if fixed[i] {
			continue
		}
		var chance float64
		if wave == SmartWaveWave {
			chance = smartFixChance(wave, d.r, d.c, centerR, centerC, maxDist, progress)
		} else {
			chance = progress
		}
		if rng.Float64() < chance {
			cur[d.r][d.c] = tgt[d.r][d.c]
			fixed[i] = true
		} else {
			cur[d.r][d.c] = charset[rng.Intn(len(charset))]
		}
	}
}

// animateSmart orchestrates the Smart diff-only animation. It builds unified
// grids (padded to max dims of both frames), finds differing positions, and
// progressively locks them to the target over multiple steps.
//
// Deviation from Rust: the Rust prints a debug eprintln on every animation —
// deliberately dropped here (stderr debug noise, not user-facing).
//
// Deviation from Rust: Rust sets self.frame to the PADDED full_target after
// animation; Go keeps a.current = target (original, unpadded) via
// PlayAnimation. Visually equivalent (trailing spaces/blank rows are invisible)
// and behaviorally equivalent (both sides get padded to the unified grid each
// time).
func animateSmart(term *Terminal, current, target []string, cfg Config, rng *rand.Rand) {
	// Unified grids: both frames padded to max dims.
	curGrid, tgtGrid, _, _ := unifiedGrids(current, target)

	diffs := diffPositions(curGrid, tgtGrid)
	if len(diffs) == 0 {
		// Frames are identical — no animation needed. The existing final
		// RenderFrame(target) in AnimateTransition handles this.
		return
	}

	// Mode clamping (applied on top of the already-clamped values from
	// AnimateTransition).
	actualSteps := smartSteps(cfg.SmartMode, cfg.AnimationSteps)
	actualDelay := smartDelay(cfg.SmartMode, cfg.AnimationDelayMs)

	fixed := make([]bool, len(diffs))

	for step := 1; step <= actualSteps; step++ {
		smartStep(curGrid, tgtGrid, diffs, fixed, step, actualSteps,
			cfg.SmartMode, cfg.SmartWave, charsetFor(cfg.AnimationCharset), rng)
		term.RenderFrame(gridToLines(curGrid))
		time.Sleep(time.Duration(actualDelay) * time.Millisecond)
	}
	// Final exact render is handled by AnimateTransition's existing
	// term.RenderFrame(target) after the switch.
}

// AnimateTransition drives a visual transition from current to target using the
// configured animation style. Steps and delay are clamped to safe ranges —
// a deliberate defensive deviation from Rust, which does not clamp. This is
// invisible for all valid configs (defaults 5 steps / 30ms are in range).
func AnimateTransition(term *Terminal, current []string, target []string, cfg Config, rng *rand.Rand) {
	steps := cfg.AnimationSteps
	delayMs := cfg.AnimationDelayMs

	// Clamp to safe ranges (Rust does not clamp; defaults are always in range).
	if steps < 1 {
		steps = 1
	} else if steps > 50 {
		steps = 50
	}
	if delayMs < 1 {
		delayMs = 1
	} else if delayMs > 500 {
		delayMs = 500
	}

	// Target dimensions: th rows, tw = widest line.
	th := len(target)
	tw := 0
	for _, line := range target {
		if len(line) > tw {
			tw = len(line)
		}
	}

	targetGrid := toGrid(target, th, tw)
	currentGrid := toGrid(current, th, tw)
	cs := charsetFor(cfg.AnimationCharset)

	switch cfg.AnimationStyle {
	case AnimationStyleScramble:
		for step := 1; step <= steps; step++ {
			currentGrid = scrambleStep(currentGrid, targetGrid, step, steps, cs, rng)
			term.RenderFrame(gridToLines(currentGrid))
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	case AnimationStyleCollapse:
		for step := 1; step <= steps; step++ {
			g := collapseStep(targetGrid, step, steps, rng)
			term.RenderFrame(gridToLines(g))
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	case AnimationStyleReveal:
		// Initial state: grid filled entirely with random charset characters.
		randomGrid := make([][]rune, th)
		for r := 0; r < th; r++ {
			row := make([]rune, tw)
			for c := 0; c < tw; c++ {
				row[c] = cs[rng.Intn(len(cs))]
			}
			randomGrid[r] = row
		}
		for step := 1; step <= steps; step++ {
			randomGrid = revealStep(randomGrid, targetGrid, step, steps, cs, rng)
			term.RenderFrame(gridToLines(randomGrid))
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	case AnimationStyleSmart:
		animateSmart(term, current, target, cfg, rng)
	default:
		// Unknown style: instant render.
		term.RenderFrame(target)
		return
	}

	// Final exact render — essential for Collapse, whose last step is NOT
	// entirely target (only the centre cell is target at threshold=0).
	term.RenderFrame(target)
}