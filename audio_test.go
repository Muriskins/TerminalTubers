package main

import (
	"encoding/binary"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// ---------------------------------------------------------------------------
// Helpers: build synthetic float32 sample buffers (little-endian, matching the
// platform byte layout that dataCallback reads via unsafe).
// ---------------------------------------------------------------------------

func f32Bytes(samples ...float32) []byte {
	b := make([]byte, len(samples)*4)
	for i, s := range samples {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(s))
	}
	return b
}

func constSamplesF32(n int, v float32) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func mixedSamples(nA int, a float32, nB int, b float32) []float32 {
	s := make([]float32, 0, nA+nB)
	for i := 0; i < nA; i++ {
		s = append(s, a)
	}
	for i := 0; i < nB; i++ {
		s = append(s, b)
	}
	return s
}

func alternatingSamples(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		if i%2 == 0 {
			s[i] = 1.0
		} else {
			s[i] = -1.0
		}
	}
	return s
}

// ---------------------------------------------------------------------------
// dataCallback: RMS math over 4800-frame (100ms) windows
// ---------------------------------------------------------------------------

// TestAudioDataCallback_RMSMath drives the real production dataCallback with
// synthetic buffers and asserts the exact RMS value emitted per window.
// RMS of N identical samples equals |sample|, so constant-amplitude cases have
// exact expected values; the mixed case (1200×1.0 + 3600×0.0) is also exact
// (sqrt(1200/4800) = 0.5).
func TestAudioDataCallback_RMSMath(t *testing.T) {
	tests := []struct {
		name    string
		samples []float32
		want    float32
		tol     float32
	}{
		{"constant 0.5", constSamplesF32(4800, 0.5), 0.5, 0},
		{"constant 1.0", constSamplesF32(4800, 1.0), 1.0, 0},
		{"silence", constSamplesF32(4800, 0.0), 0.0, 0},
		{"constant 0.75", constSamplesF32(4800, 0.75), 0.75, 0},
		{"constant negative -0.5", constSamplesF32(4800, -0.5), 0.5, 0},
		{"constant 0.3 (float32 precision)", constSamplesF32(4800, 0.3), 0.3, 1e-7},
		{"mixed 1200 ones + 3600 zeros", mixedSamples(1200, 1.0, 3600, 0.0), 0.5, 0},
		{"alternating +1/-1", alternatingSamples(4800), 1.0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Capture{}
			c.dataCallback(nil, f32Bytes(tt.samples...), uint32(len(tt.samples)))
			got := c.RMS()
			if math.Abs(float64(got-tt.want)) > float64(tt.tol) {
				t.Errorf("RMS() = %v, want %v (tol %v)", got, tt.want, tt.tol)
			}
		})
	}
}

// TestAudioDataCallback_PartialWindowNoEmit verifies no RMS is published until
// the window reaches captureWindowFrames (4800).
func TestAudioDataCallback_PartialWindowNoEmit(t *testing.T) {
	c := &Capture{}
	c.dataCallback(nil, f32Bytes(constSamplesF32(4799, 0.5)...), 4799)
	if got := c.RMS(); got != 0 {
		t.Errorf("RMS() = %v, want 0 before window completes", got)
	}
	if c.frameCount != 4799 {
		t.Errorf("frameCount = %d, want 4799", c.frameCount)
	}
}

// TestAudioDataCallback_WindowBoundaryCarry verifies frames accumulate across
// callbacks and the window emits exactly at 4800, then resets.
func TestAudioDataCallback_WindowBoundaryCarry(t *testing.T) {
	c := &Capture{}
	// 2400 frames of 0.5 → no emit yet.
	c.dataCallback(nil, f32Bytes(constSamplesF32(2400, 0.5)...), 2400)
	if got := c.RMS(); got != 0 {
		t.Errorf("RMS() = %v, want 0 after 2400 frames", got)
	}
	// 2400 more → window completes at 4800 → RMS 0.5.
	c.dataCallback(nil, f32Bytes(constSamplesF32(2400, 0.5)...), 2400)
	if got := c.RMS(); got != 0.5 {
		t.Errorf("RMS() = %v, want 0.5 after window completes", got)
	}
	// Accumulators must reset: a fresh silence window must emit 0, not blend
	// with the previous window's sumSquares.
	c.dataCallback(nil, f32Bytes(constSamplesF32(4800, 0.0)...), 4800)
	if got := c.RMS(); got != 0 {
		t.Errorf("RMS() = %v, want 0 after reset + silence window", got)
	}
}

// TestAudioDataCallback_AccumulatorReset verifies two consecutive full windows
// each emit their own correct value (no leakage between windows).
func TestAudioDataCallback_AccumulatorReset(t *testing.T) {
	c := &Capture{}
	c.dataCallback(nil, f32Bytes(constSamplesF32(4800, 1.0)...), 4800)
	if got := c.RMS(); got != 1.0 {
		t.Errorf("first window RMS() = %v, want 1.0", got)
	}
	c.dataCallback(nil, f32Bytes(constSamplesF32(4800, 0.5)...), 4800)
	if got := c.RMS(); got != 0.5 {
		t.Errorf("second window RMS() = %v, want 0.5 (accumulators leaked)", got)
	}
}

// TestAudioDataCallback_EmptyInput verifies the early return: no panic, no
// state mutation.
func TestAudioDataCallback_EmptyInput(t *testing.T) {
	c := &Capture{}
	c.dataCallback(nil, nil, 0)
	c.dataCallback(nil, []byte{}, 0)
	if got := c.RMS(); got != 0 {
		t.Errorf("RMS() = %v, want 0", got)
	}
	if c.frameCount != 0 {
		t.Errorf("frameCount = %d, want 0", c.frameCount)
	}
	if c.sumSquares != 0 {
		t.Errorf("sumSquares = %v, want 0", c.sumSquares)
	}
}

// ---------------------------------------------------------------------------
// RMS: default state and atomic float32 round-trip
// ---------------------------------------------------------------------------

// TestAudioRMS_DefaultZero verifies a fresh Capture reports level 0.
func TestAudioRMS_DefaultZero(t *testing.T) {
	c := &Capture{}
	if got := c.RMS(); got != 0 {
		t.Errorf("RMS() = %v, want 0", got)
	}
}

// TestAudioAtomicFloat32RoundTrip is a property test of the lock-free sharing
// mechanism: atomic.Uint32 + math.Float32bits/Float32frombits must preserve
// every float32 bit pattern exactly, including edge values (NaN, ±Inf, ±0,
// denormals, MaxFloat32).
func TestAudioAtomicFloat32RoundTrip(t *testing.T) {
	values := []float32{
		0,
		float32(math.Copysign(0, -1)), // -0
		1, -1,
		0.5, -0.5,
		0.1, 0.3, 123.456,
		math.MaxFloat32,
		math.SmallestNonzeroFloat32,
		1.17549435e-38, // smallest normal float32
		1e-30,          // denormal
		float32(math.Inf(1)),
		float32(math.Inf(-1)),
		float32(math.NaN()),
	}
	c := &Capture{}
	for _, v := range values {
		bits := math.Float32bits(v)
		c.rms.Store(bits)
		got := c.RMS()
		if math.Float32bits(got) != bits {
			t.Errorf("round-trip(%v): stored bits %#x, got %v (bits %#x)", v, bits, got, math.Float32bits(got))
		}
	}
}

// ---------------------------------------------------------------------------
// Close: idempotency (no device / degraded paths — no hardware required)
// ---------------------------------------------------------------------------

// TestAudioClose_Idempotent verifies Close() is safe to call repeatedly on a
// zero-value Capture and on a degraded Capture (channels created, no device).
func TestAudioClose_Idempotent(t *testing.T) {
	c := &Capture{}
	c.Close()
	c.Close()
	c.Close()

	d := &Capture{
		stopWatch: make(chan struct{}),
		watchDone: make(chan struct{}),
	}
	d.Close()
	d.Close()
}

// ---------------------------------------------------------------------------
// Available / Start / Stop guards (no hardware required)
// ---------------------------------------------------------------------------

func TestAudioAvailable_FalseWhenNoDevice(t *testing.T) {
	c := &Capture{}
	if c.Available() {
		t.Error("Available() = true on Capture with no device, want false")
	}
}

func TestAudioStart_UnavailableReturnsError(t *testing.T) {
	c := &Capture{}
	err := c.Start()
	if err == nil {
		t.Fatal("Start() on unavailable capture returned nil error")
	}
	if err.Error() != "no capture device available" {
		t.Errorf("Start() error = %q, want %q", err, "no capture device available")
	}
}

func TestAudioStop_NilDeviceNoop(t *testing.T) {
	c := &Capture{}
	if err := c.Stop(); err != nil {
		t.Errorf("Stop() with nil device returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Err / fail: first-failure-wins semantics
// ---------------------------------------------------------------------------

func TestAudioErr_DefaultNil(t *testing.T) {
	c := &Capture{}
	if err := c.Err(); err != nil {
		t.Errorf("Err() on healthy capture = %v, want nil", err)
	}
}

func TestAudioFail_FirstWins(t *testing.T) {
	c := &Capture{}
	c.fail("first failure")
	c.fail("second failure")
	err := c.Err()
	if err == nil || err.Error() != "first failure" {
		t.Errorf("Err() = %v, want %q (first failure must win)", err, "first failure")
	}
}

func TestAudioErr_DefaultMessage(t *testing.T) {
	c := &Capture{}
	c.failed.Store(true)
	c.errMsg.Store("")
	err := c.Err()
	if err == nil || err.Error() != "audio capture failed" {
		t.Errorf("Err() = %v, want %q", err, "audio capture failed")
	}
}

// ---------------------------------------------------------------------------
// stopCallback: intentional stop vs unexpected stop
// ---------------------------------------------------------------------------

func TestAudioStopCallback_StoppingSuppressesFailure(t *testing.T) {
	c := &Capture{}
	c.stopping.Store(true)
	c.stopCallback()
	if c.failed.Load() {
		t.Error("stopCallback recorded failure during intentional stop")
	}
	if err := c.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestAudioStopCallback_UnexpectedStopRecordsFailure(t *testing.T) {
	c := &Capture{}
	c.stopCallback()
	if !c.failed.Load() {
		t.Error("stopCallback did not record failure for unexpected stop")
	}
	err := c.Err()
	if err == nil || err.Error() != "audio capture device stopped unexpectedly" {
		t.Errorf("Err() = %v, want %q", err, "audio capture device stopped unexpectedly")
	}
}

// ---------------------------------------------------------------------------
// watch: 3s-stall watchdog and clean exit
// ---------------------------------------------------------------------------

// TestAudioWatch_StallDetection verifies the watchdog records a failure when
// started but no Data callback has fired for captureStallTimeout. The failure
// is polled (the watchdog keeps running after recording it — fail() is
// first-wins), then the watchdog is terminated cleanly by closing stopWatch,
// mirroring the production Close() path.
func TestAudioWatch_StallDetection(t *testing.T) {
	c := &Capture{
		stopWatch: make(chan struct{}),
		watchDone: make(chan struct{}),
	}
	c.started.Store(true)
	c.lastData.Store(time.Now().Add(-10 * time.Second).UnixNano())

	go c.watch()

	// The watchdog ticker fires every second, so the stale lastData timestamp
	// trips the stall check on the first tick. Poll failed instead of blocking
	// on watchDone, which closes only when the watchdog exits.
	deadline := time.Now().Add(5 * time.Second)
	for !c.failed.Load() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !c.failed.Load() {
		t.Fatal("watch() did not record failure for stalled stream")
	}
	err := c.Err()
	if err == nil || err.Error() != "audio capture stopped delivering data" {
		t.Errorf("Err() = %v, want %q", err, "audio capture stopped delivering data")
	}

	// Terminate the watchdog goroutine cleanly, mirroring Close().
	close(c.stopWatch)
	select {
	case <-c.watchDone:
	case <-time.After(5 * time.Second):
		t.Fatal("watch() did not exit after stopWatch closed")
	}
}

// TestAudioWatch_CleanExit verifies closing stopWatch terminates the watchdog
// without recording a failure.
func TestAudioWatch_CleanExit(t *testing.T) {
	c := &Capture{
		stopWatch: make(chan struct{}),
		watchDone: make(chan struct{}),
	}
	c.started.Store(true)
	c.lastData.Store(time.Now().UnixNano())

	go c.watch()
	close(c.stopWatch)

	select {
	case <-c.watchDone:
	case <-time.After(5 * time.Second):
		t.Fatal("watch() did not exit after stopWatch closed")
	}

	if c.failed.Load() {
		t.Error("watch() recorded failure on clean stop")
	}
}

// ---------------------------------------------------------------------------
// NewCapture: graceful degradation (hardware-dependent — run separately)
// ---------------------------------------------------------------------------

// TestNewCapture_GracefulDegradation exercises the real malgo init path. It
// does NOT depend on a microphone: every outcome is asserted against the
// documented contract — (a) backend init failure → nil capture + non-nil error,
// (b) no capture device → degraded Capture with Available()==false and nil
// error, (c) device present → Available()==true and Start/Stop work. Close()
// must always be safe. In a sandbox with no audio backend this test may hang;
// it is run separately with a short -timeout so a hang is reported as SKIPPED
// rather than blocking the pure-logic tests.
func TestNewCapture_GracefulDegradation(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		if c != nil {
			t.Fatalf("NewCapture() returned non-nil capture with error %v", err)
		}
		t.Logf("NewCapture() could not init audio backend: %v (acceptable in sandbox)", err)
		return
	}
	if c == nil {
		t.Fatal("NewCapture() returned nil capture with nil error")
	}
	if c.Available() {
		if err := c.Start(); err != nil {
			t.Errorf("Start() on available capture failed: %v", err)
		}
		if err := c.Stop(); err != nil {
			t.Errorf("Stop() on available capture failed: %v", err)
		}
	} else {
		t.Log("no capture device available — degraded capture returned (graceful path)")
	}
	c.Close()
	c.Close() // idempotent
}

// ---------------------------------------------------------------------------
// FR-020: device enumeration / selection (Devices, SelectedDeviceID,
// SelectDevice)
// ---------------------------------------------------------------------------

// waitForData polls lastData until it advances past `after` or the timeout
// elapses. Returns true when Data callbacks were observed (i.e. the capture
// stream is delivering).
func waitForData(t *testing.T, c *Capture, after int64, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.lastData.Load() > after {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestDevices_AfterCloseReturnsEmptyNonNil verifies Devices() on a Capture
// whose context is nil (zero value — the exact post-Close state) returns an
// empty NON-NIL slice without panicking.
func TestDevices_AfterCloseReturnsEmptyNonNil(t *testing.T) {
	c := &Capture{}
	c.Close() // idempotent no-op on a zero-value capture
	devices := c.Devices()
	if devices == nil {
		t.Fatal("Devices() returned nil slice, want empty non-nil")
	}
	if len(devices) != 0 {
		t.Errorf("Devices() returned %d devices, want 0", len(devices))
	}
}

// TestSelectedDeviceID_DefaultNil verifies a fresh Capture reports automatic
// (nil) selection.
func TestSelectedDeviceID_DefaultNil(t *testing.T) {
	c := &Capture{}
	if got := c.SelectedDeviceID(); got != nil {
		t.Errorf("SelectedDeviceID() = %v, want nil (automatic)", got)
	}
}

// TestSelectDevice_AfterCloseReturnsError verifies SelectDevice on a closed
// capture returns "capture closed" for both nil and non-nil device IDs.
func TestSelectDevice_AfterCloseReturnsError(t *testing.T) {
	c := &Capture{} // ctx == nil, same state as after Close()
	err := c.SelectDevice(nil)
	if err == nil || err.Error() != "capture closed" {
		t.Errorf("SelectDevice(nil) error = %v, want %q", err, "capture closed")
	}
	id := malgo.DeviceID{1, 2, 3}
	err = c.SelectDevice(&id)
	if err == nil || err.Error() != "capture closed" {
		t.Errorf("SelectDevice(id) error = %v, want %q", err, "capture closed")
	}
}

// TestDevices_LiveCaptureEnumeratesDevices exercises the real malgo
// enumeration path: an available capture must enumerate at least one named
// device and exactly one default device. After Close() the slice must be empty
// and non-nil. Hardware-dependent — skipped when no audio backend or capture
// device is present.
func TestDevices_LiveCaptureEnumeratesDevices(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}
	devices := c.Devices()
	if devices == nil {
		t.Fatal("Devices() returned nil slice, want non-nil")
	}
	if len(devices) == 0 {
		t.Fatal("Devices() returned 0 devices on an available capture")
	}
	defaults := 0
	for i, d := range devices {
		if d.Name == "" {
			t.Errorf("Devices()[%d].Name is empty", i)
		}
		if d.IsDefault {
			defaults++
		}
	}
	if defaults == 0 {
		t.Error("no device flagged IsDefault, want exactly one default device")
	}
	if defaults > 1 {
		t.Errorf("%d devices flagged IsDefault, want exactly one", defaults)
	}
	// After Close() the context is gone: enumeration must degrade to an empty
	// non-nil slice.
	c.Close()
	devices = c.Devices()
	if devices == nil {
		t.Fatal("Devices() after Close() returned nil slice, want empty non-nil")
	}
	if len(devices) != 0 {
		t.Errorf("Devices() after Close() returned %d devices, want 0", len(devices))
	}
}

// TestSelectDevice_NilKeepsAutomaticAndDeliversRMS verifies SelectDevice(nil)
// on a running capture succeeds, leaves the selection automatic (nil), and the
// capture keeps delivering data (RMS resumes) on the restarted device.
func TestSelectDevice_NilKeepsAutomaticAndDeliversRMS(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}
	if err := c.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	start := c.lastData.Load()
	if !waitForData(t, c, start, 3*time.Second) {
		t.Fatal("no data callbacks after Start()")
	}
	if got := c.SelectedDeviceID(); got != nil {
		t.Fatalf("SelectedDeviceID() = %v before any selection, want nil", got)
	}
	if err := c.SelectDevice(nil); err != nil {
		t.Fatalf("SelectDevice(nil) failed: %v", err)
	}
	if got := c.SelectedDeviceID(); got != nil {
		t.Errorf("SelectedDeviceID() = %v after SelectDevice(nil), want nil", got)
	}
	if !c.started.Load() {
		t.Error("started = false after SelectDevice(nil) on running capture, want true")
	}
	before := c.lastData.Load()
	if !waitForData(t, c, before, 3*time.Second) {
		t.Fatal("no data callbacks after SelectDevice(nil) — capture did not restart")
	}
	rms := c.RMS()
	if math.IsNaN(float64(rms)) || rms < 0 {
		t.Errorf("RMS() = %v after switch, want valid non-negative level", rms)
	}
	t.Logf("RMS after SelectDevice(nil): %v", rms)
}

// TestSelectDevice_WithDeviceIDSwitchesAndRestarts verifies selecting a real
// enumerated device: the selection is stored as a COPY equal to the requested
// ID, the capture restarts and keeps delivering data, and switching back to
// nil clears the selection.
func TestSelectDevice_WithDeviceIDSwitchesAndRestarts(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}
	devices := c.Devices()
	if len(devices) == 0 {
		t.Skip("no capture devices enumerated")
	}
	target := devices[0]
	if err := c.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	start := c.lastData.Load()
	if !waitForData(t, c, start, 3*time.Second) {
		t.Fatal("no data callbacks after Start()")
	}
	if err := c.SelectDevice(&target.ID); err != nil {
		t.Fatalf("SelectDevice(%v) failed: %v", target.ID, err)
	}
	got := c.SelectedDeviceID()
	if got == nil {
		t.Fatal("SelectedDeviceID() = nil after selecting a device, want non-nil")
	}
	if *got != target.ID {
		t.Errorf("SelectedDeviceID() = %v, want %v (byte-for-byte)", *got, target.ID)
	}
	if got == &target.ID {
		t.Error("SelectedDeviceID() returned the caller's pointer, want a stored copy")
	}
	if !c.started.Load() {
		t.Error("started = false after SelectDevice on running capture, want true")
	}
	before := c.lastData.Load()
	if !waitForData(t, c, before, 3*time.Second) {
		t.Fatal("no data callbacks after device switch — capture did not restart")
	}
	// Switching back to automatic must clear the stored selection.
	if err := c.SelectDevice(nil); err != nil {
		t.Fatalf("SelectDevice(nil) after device switch failed: %v", err)
	}
	if got := c.SelectedDeviceID(); got != nil {
		t.Errorf("SelectedDeviceID() = %v after SelectDevice(nil), want nil", got)
	}
}

// TestSelectDevice_PreservesStartedState verifies the started/stopped state is
// preserved across a device switch: a stopped capture stays stopped (no data
// flows), a running capture restarts and delivers data.
func TestSelectDevice_PreservesStartedState(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}

	// Stopped capture: switch must leave it stopped with no data flowing.
	if err := c.SelectDevice(nil); err != nil {
		t.Fatalf("SelectDevice(nil) on stopped capture failed: %v", err)
	}
	if c.started.Load() {
		t.Error("started = true after SelectDevice on stopped capture, want false")
	}
	if got := c.lastData.Load(); got != 0 {
		t.Errorf("lastData = %d after switch on stopped capture, want 0 (no data)", got)
	}

	// Running capture: switch must restart it and keep delivering data.
	if err := c.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	start := c.lastData.Load()
	if !waitForData(t, c, start, 3*time.Second) {
		t.Fatal("no data callbacks after Start()")
	}
	if err := c.SelectDevice(nil); err != nil {
		t.Fatalf("SelectDevice(nil) on running capture failed: %v", err)
	}
	if !c.started.Load() {
		t.Error("started = false after SelectDevice on running capture, want true (restarted)")
	}
	before := c.lastData.Load()
	if !waitForData(t, c, before, 3*time.Second) {
		t.Fatal("no data callbacks after restart")
	}
}

// TestSelectDevice_RecoversFromWatchdogStall verifies the watchdog keeps
// running after recording a stall failure (fail() is first-wins) and that a
// subsequent SelectDevice(nil) clears the failure and restarts capture. The
// stall is simulated by marking the capture started with a stale lastData
// timestamp while the device is not actually running, so the real watchdog
// goroutine (started by NewCapture) records the failure. Hardware-dependent;
// takes ~1-3s.
func TestSelectDevice_RecoversFromWatchdogStall(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}

	// Simulate a stalled stream: started with no data for > captureStallTimeout.
	c.started.Store(true)
	c.lastData.Store(time.Now().Add(-10 * time.Second).UnixNano())

	deadline := time.Now().Add(5 * time.Second)
	for !c.failed.Load() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !c.failed.Load() {
		t.Fatal("watchdog did not record the stall failure")
	}
	if err := c.Err(); err == nil || err.Error() != "audio capture stopped delivering data" {
		t.Fatalf("Err() = %v, want %q", err, "audio capture stopped delivering data")
	}

	// SelectDevice(nil) must clear the failure and restart capture (wasStarted
	// is true, so Start() runs inside SelectDevice).
	if err := c.SelectDevice(nil); err != nil {
		t.Fatalf("SelectDevice(nil) after stall failed: %v", err)
	}
	if err := c.Err(); err != nil {
		t.Errorf("Err() = %v after SelectDevice, want nil (failure cleared)", err)
	}
	if !c.started.Load() {
		t.Error("started = false after SelectDevice, want true (restarted)")
	}
	before := c.lastData.Load()
	if !waitForData(t, c, before, 3*time.Second) {
		t.Fatal("no data callbacks after recovery — capture did not resume")
	}
	rms := c.RMS()
	if math.IsNaN(float64(rms)) || rms < 0 {
		t.Errorf("RMS() = %v after recovery, want valid non-negative level", rms)
	}
	t.Logf("RMS after recovery: %v", rms)
}

// ---------------------------------------------------------------------------
// SelectDevice fallback: graceful degradation when the requested device is
// gone (FR-025/026). The fallback path (audio.go lines 264-268) retries
// InitDevice once with the default device when the requested ID fails, and
// keeps reporting the requested ID via SelectedDeviceID() — the fallback is
// transparent to the caller.
// ---------------------------------------------------------------------------

// bogusDeviceID returns a well-formed but nonexistent capture device ID: the
// WASAPI endpoint string is a single non-character (U+FFFF) followed by a null
// terminator. Unlike a raw 0xFF-filled buffer, this is null-terminated, so
// IMMDeviceEnumerator::GetDevice fails cleanly with E_NOTFOUND instead of
// scanning past the buffer for a terminator. No real device can match it, so
// malgo.InitDevice must fail for it and SelectDevice must take the fallback.
// (On non-WASAPI Windows backends the bytes are likewise a fixed-size ID —
// GUID or UINT — that matches no device.)
func bogusDeviceID() malgo.DeviceID {
	var id malgo.DeviceID
	id[0] = 0xFF
	id[1] = 0xFF
	id[2] = 0x00
	id[3] = 0x00
	return id
}

// TestSelectDevice_FallbackToDefaultWhenDeviceGone exercises the graceful
// degradation path of SelectDevice: when the requested device ID cannot be
// opened (stale/invalid — here a nonexistent ID), SelectDevice retries once
// with the default device. The fallback is transparent: on success the capture
// is available, no failure is recorded, and SelectedDeviceID() still reports
// the requested ID. If the default device also cannot be opened, SelectDevice
// must return the wrapped InitDevice error and record the failure via Err().
// Any panic in the fallback path fails the test. Hardware-dependent —
// self-skips when no capture device is present.
func TestSelectDevice_FallbackToDefaultWhenDeviceGone(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}
	if len(c.Devices()) == 0 {
		t.Skip("no capture devices enumerated")
	}

	bogus := bogusDeviceID()
	err = c.SelectDevice(&bogus)
	if err != nil {
		// Only acceptable when the default device cannot be opened either:
		// the error must be the wrapped InitDevice error and the failure must
		// be recorded for Err().
		if !strings.Contains(err.Error(), "initialize audio capture device") {
			t.Errorf("SelectDevice(bogus) error = %v, want wrapped %q", err, "initialize audio capture device")
		}
		if c.Available() {
			t.Error("Available() = true after total SelectDevice failure, want false")
		}
		if c.Err() == nil {
			t.Error("Err() = nil after total SelectDevice failure, want recorded failure")
		}
		return
	}

	// Fallback succeeded on the default device: the capture must be available,
	// healthy, and still report the requested (bogus) ID.
	if !c.Available() {
		t.Error("Available() = false after fallback, want true")
	}
	if got := c.Err(); got != nil {
		t.Errorf("Err() = %v after fallback, want nil (failure state reset)", got)
	}
	got := c.SelectedDeviceID()
	if got == nil {
		t.Fatal("SelectedDeviceID() = nil after fallback, want the requested ID (transparent fallback)")
	}
	if *got != bogus {
		t.Errorf("SelectedDeviceID() = %v, want requested bogus ID %v", *got, bogus)
	}
	if got == &bogus {
		t.Error("SelectedDeviceID() returned the caller's pointer, want a stored copy")
	}
}

// TestSelectDevice_FallbackPreservesAutomaticSelection documents the
// transparent-fallback contract on a RUNNING capture: when the requested
// device is gone, SelectDevice falls back to the default device but the
// reported selection stays the requested ID (never nil) — the caller asked for
// that device, so SelectedDeviceID() returns it. The capture must restart
// (wasStarted) and keep delivering data on the fallback device, proving the
// fallback opened a real device. Hardware-dependent — self-skips when no
// capture device is present.
func TestSelectDevice_FallbackPreservesAutomaticSelection(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}
	if len(c.Devices()) == 0 {
		t.Skip("no capture devices enumerated")
	}
	if err := c.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	start := c.lastData.Load()
	if !waitForData(t, c, start, 3*time.Second) {
		t.Fatal("no data callbacks after Start()")
	}

	bogus := bogusDeviceID()
	if err := c.SelectDevice(&bogus); err != nil {
		t.Fatalf("SelectDevice(bogus) fallback failed: %v", err)
	}

	// The fallback is transparent: the selection reports the requested ID,
	// never nil, even though the physical device is the default.
	got := c.SelectedDeviceID()
	if got == nil {
		t.Fatal("SelectedDeviceID() = nil after fallback, want the requested ID (transparent fallback)")
	}
	if *got != bogus {
		t.Errorf("SelectedDeviceID() = %v, want requested bogus ID %v", *got, bogus)
	}
	if got == &bogus {
		t.Error("SelectedDeviceID() returned the caller's pointer, want a stored copy")
	}
	// The running capture must have restarted on the fallback device.
	if !c.started.Load() {
		t.Error("started = false after fallback on running capture, want true (restarted)")
	}
	before := c.lastData.Load()
	if !waitForData(t, c, before, 3*time.Second) {
		t.Fatal("no data callbacks after fallback — capture did not restart on the default device")
	}
	if err := c.Err(); err != nil {
		t.Errorf("Err() = %v after fallback, want nil (failure state reset)", err)
	}
}

// TestSelectDevice_InitDeviceFailureWithNonexistentID verifies the mechanism
// behind the fallback: malgo.InitDevice must FAIL for a nonexistent DeviceID,
// and SelectDevice must then succeed by retrying with the default device. The
// premise is checked directly with a probe InitDevice call (replicating the
// pinned-copy pattern from SelectDevice); if a backend ever accepted the bogus
// ID, the test self-skips because the fallback cannot be exercised. Data must
// flow after the switch, proving the fallback opened the real default device.
// Hardware-dependent — self-skips when no capture device is present.
func TestSelectDevice_InitDeviceFailureWithNonexistentID(t *testing.T) {
	c, err := NewCapture()
	if err != nil {
		t.Skipf("audio backend unavailable: %v", err)
	}
	defer c.Close()
	if !c.Available() {
		t.Skip("no capture device available")
	}
	if len(c.Devices()) == 0 {
		t.Skip("no capture devices enumerated")
	}

	bogus := bogusDeviceID()

	// Premise: InitDevice with the bogus ID must fail. Replicate the pinned
	// copy pattern from SelectDevice so the cgo pointer checker accepts it.
	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatF32
	cfg.Capture.Channels = captureChannels
	cfg.SampleRate = captureSampleRate
	cfg.PeriodSizeInMilliseconds = capturePeriodMs
	idCopy := bogus
	var pinner runtime.Pinner
	pinner.Pin(&idCopy)
	defer pinner.Unpin()
	cfg.Capture.DeviceID = unsafe.Pointer(&idCopy)
	probe, probeErr := malgo.InitDevice(c.ctx.Context, cfg, malgo.DeviceCallbacks{})
	if probeErr == nil {
		probe.Uninit()
		t.Skip("backend accepted bogus device ID — fallback path cannot be exercised")
	}

	// With the premise confirmed, SelectDevice must succeed via the fallback.
	if err := c.SelectDevice(&bogus); err != nil {
		t.Fatalf("SelectDevice(bogus) fallback failed: %v", err)
	}
	if !c.Available() {
		t.Error("Available() = false after fallback, want true")
	}
	got := c.SelectedDeviceID()
	if got == nil || *got != bogus {
		t.Errorf("SelectedDeviceID() = %v, want requested bogus ID %v", got, bogus)
	}
	// Data must flow: the fallback opened the real default device (the bogus
	// device cannot deliver audio).
	if err := c.Start(); err != nil {
		t.Fatalf("Start() after fallback failed: %v", err)
	}
	start := c.lastData.Load()
	if !waitForData(t, c, start, 3*time.Second) {
		t.Fatal("no data callbacks after fallback — capture did not open the default device")
	}
}
