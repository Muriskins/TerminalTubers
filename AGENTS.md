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
- State-machine tests (`state_machine_test.go`, 21 tests) drive `Tick` with synthetic voice/time sequences — 100% branch coverage of `state_machine.go`.
- The `pickTalkingFrame` helper (main.go:12-18) is covered by `TestPickTalkingFrame_*` tests (3 tests in `state_machine_test.go`).

## Frames (go:embed)

Four ASCII frames are embedded: `ascii_art/idle.txt` + `tolk0/1/2.txt`. Adding a frame requires updating BOTH `frameNames` and the `//go:embed` directive in `assets.go`, then rebuilding — `main.go` asserts exactly 4 frames at startup.

## Gotchas

- README.md documents Phases 1–4 (terminal layer, audio capture via malgo, avatar rendering + transition animations, main loop + idle/talking state machine; CGO required). Keep it in sync with the code as later phases land.
- Current behavior: the avatar reacts to voice — 3 consecutive voiced 50ms ticks (~150ms) switch idle→talking with a random talking frame (chosen once per transition); 500ms of silence returns to idle. The state machine is a pure, testable type in `state_machine.go`; `main.go` only wires it to the ticker, capture RMS, and avatar.
- Main loop: `PollEvent` blocks, so it is drained in a goroutine into a buffered channel (cap 8); the goroutine exits when PollEvent returns nil after terminal close. A 50ms ticker drives the state machine; q/Q/Escape quit, s/S is reserved for the Phase 5 settings menu (recognized but a no-op).
- Animation quirks: `AnimateTransition` clamps steps to [1,50] and delay to [1,500]ms (deliberate deviation from Rust, which does not clamp); Smart mode clamps steps to min 8 and Instant-mode delay to min 50ms; Collapse uses hardcoded `collapseChars`, NOT the configured charset; unknown styles render instantly.
- Build artifacts: `TerminalTubers.exe` is gitignored; `clang-wrap.exe` and `cover` are not (known repo-hygiene issue).
- Commit policy: commit AND push after each completed task (user decision).
- Windows-only: GOOS=windows, CGO_ENABLED=1. Linux is a separate future effort on a separate branch.
- Phase 5 settings menu integration points (verified 2026-09-24):
  - 's'/'S' key case at main.go:101-105 is an empty reserved no-op — the menu hooks here.
  - Esc/q/Q quit unconditionally at main.go:99 — the menu MUST intercept Esc before it reaches the quit case.
  - eventCh (main.go:74-83) hardwires ALL events (keys + resize) from the PollEvent goroutine with no routing hook — the menu must share/restructure this channel.
  - Config is a local value (main.go:67); menu needs its own working copy + apply-on-return; main.go must reassign cfg for immediate effect.
  - IdleDelayMs is baked into StateMachine at construction (main.go:68) — NOT a menu item (spec Assumption 2), so no re-wiring needed for the menu.
  - Config.AudioDevice (config.go:49,64) is declared/defaulted but NEVER read — Phase 5.2 must wire it to capture.SelectDevice (audio.go:222).
  - captureFailureSurfaced (main.go:41) is never reset — device switch in Phase 5 must reset it or FR-026 surfacing breaks after recovery.
  - Terminal.Size() (terminal.go:43-45) has no production caller — ready for menu layout use.
  - Avatar.Frames() (avatar.go:42-44) has no production caller; main.go indexes a local `frames` slice — two sources of truth, pick one when touching frame code.
  - Audio device API is ready: Devices() []DeviceInfo (audio.go:187), SelectDevice(id *malgo.DeviceID) (audio.go:222, nil = automatic), SelectedDeviceID() (audio.go:209). No current-device-NAME accessor exists — menu must match SelectedDeviceID() against Devices() entries.
  - SME menu guidance (CONTEXT.md): hand-roll cursorIdx + editing bool (~100 lines), redraw only menu region.