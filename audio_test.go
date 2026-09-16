package main

import (
	"encoding/binary"
	"math"
	"testing"
	"time"
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
// started but no Data callback has fired for captureStallTimeout.
func TestAudioWatch_StallDetection(t *testing.T) {
	c := &Capture{
		stopWatch: make(chan struct{}),
		watchDone: make(chan struct{}),
	}
	c.started.Store(true)
	c.lastData.Store(time.Now().Add(-10 * time.Second).UnixNano())

	go c.watch()

	select {
	case <-c.watchDone:
	case <-time.After(5 * time.Second):
		t.Fatal("watch() did not detect stall within 5s")
	}

	if !c.failed.Load() {
		t.Error("watch() did not record failure for stalled stream")
	}
	err := c.Err()
	if err == nil || err.Error() != "audio capture stopped delivering data" {
		t.Errorf("Err() = %v, want %q", err, "audio capture stopped delivering data")
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