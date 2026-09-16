# AGENTS.md

TerminalTubers — Windows-only Go port of a terminal ASCII VTuber (avatar reacts to microphone input). Single `package main` at repo root; no subpackages. Ported from Rust (original preserved on the `rust-legacy` branch). Active development is swarm-driven: `.swarm/plan.md` (6-phase port plan), `.swarm/spec.md` (FR/SC requirements), and `.swarm/CONTEXT.md` (decisions, SME cache, toolchain resolution) are authoritative.

## Commands

- Run: `go run .` — requires a real TTY; fails in sandboxes/CI
- Build: `go build -o TerminalTubers .`
- Test: `go test ./...` — white-box tests, `package main`
- Vet: `go vet ./...` — passes; only harmless lld-link DWARF section-name warnings
- Deps: `go mod tidy`

## cgo toolchain — REQUIRED for any build (malgo is cgo)

Go appends `-mthreads` to the compiler line for windows cgo builds; clang (MSVC target) rejects it. The fix is a native Go CC wrapper:

- `clang-wrap.go` (`//go:build ignore`) compiles to `clang-wrap.exe` in the repo root, registered via `go env -w CC=C:\Users\Muriskins\Documents\TerminalTuber\TerminalTubers\clang-wrap.exe`.
- If `clang-wrap.exe` is missing, rebuild it: `go build -o clang-wrap.exe clang-wrap.go`. Do NOT delete it — builds depend on it.
- NEVER replace it with a `.cmd`/`.bat` wrapper: cmd.exe splits batch args on `=`, `,`, `;` and corrupts trailing-backslash paths. The CC path must be absolute and space-free (Go's `quoted.Split` breaks at spaces).
- The `go env -w` setting is user-global, not repo-local — a fresh machine/CI must re-run it before building.

## malgo v0.11.26 (audio) — verified API quirks

- No `SliceToFloat32` helper: convert via `unsafe.Slice((*float32)(unsafe.Pointer(&pIn[0])), len(pIn)/4)`.
- No Notification callback (no hot-plug events): poll `ctx.Devices()` and diff IDs.
- Go `sync/atomic` has no Float32 type: share RMS via `atomic.Uint32` + `math.Float32bits`/`Float32frombits`.
- The malgo Data callback runs on the audio thread: it must stay allocation-free and lock-free.
- Pin the malgo version — the API churns.

## Testing

- Terminal-layer tests use `tcell.NewSimulationScreen` (no TTY needed): RenderFrame centering/clipping, PollEvent key injection. `NewTerminal()` requires a real TTY and is not tested directly.
- Animation tests (`animation_test.go`, 76 tests) drive `AnimateTransition` and the Smart helpers on `tcell.NewSimulationScreen` — 100% coverage of `animation.go`.
- `TestNewCapture_GracefulDegradation` exercises real malgo init and may hang in sandboxes without an audio backend — run it separately with a short `-timeout` if it hangs.
- Audio tests drive the production `dataCallback` with synthetic little-endian float32 buffers.

## Frames (go:embed)

Four ASCII frames are embedded: `ascii_art/idle.txt` + `tolk0/1/2.txt`. Adding a frame requires updating BOTH `frameNames` and the `//go:embed` directive in `assets.go`, then rebuilding — `main.go` asserts exactly 4 frames at startup.

## Gotchas

- README.md documents Phases 1–3 (terminal layer, audio capture via malgo, avatar rendering + transition animations; CGO required). Keep it in sync with the code as later phases land.
- Current behavior: idle avatar + background audio capture; `PlayAnimation`/`AnimateTransition` implement the full animation system (Scramble/Collapse/Reveal/Smart), but RMS is not yet wired to frame switching — the state machine is Phase 4.
- Animation quirks: `AnimateTransition` clamps steps to [1,50] and delay to [1,500]ms (deliberate deviation from Rust, which does not clamp); Smart mode clamps steps to min 8 and Instant-mode delay to min 50ms; Collapse uses hardcoded `collapseChars`, NOT the configured charset; unknown styles render instantly.
- Build artifacts: `TerminalTubers.exe` is gitignored; `clang-wrap.exe` and `cover` are not (known repo-hygiene issue).
- Commit policy: commit AND push after each completed task (user decision).
- Windows-only: GOOS=windows, CGO_ENABLED=1. Linux is a separate future effort on a separate branch.