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
	default:
		// Unknown style (including Smart, which is task 3.3): instant render.
		term.RenderFrame(target)
		return
	}

	// Final exact render — essential for Collapse, whose last step is NOT
	// entirely target (only the centre cell is target at threshold=0).
	term.RenderFrame(target)
}