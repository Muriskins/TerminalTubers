package main

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// Capture constants. The RMS window is 100ms at the capture sample rate.
const (
	captureSampleRate   = 48000
	captureChannels     = 1
	capturePeriodMs     = 10
	captureWindowFrames = 4800 // 100ms @ 48kHz mono
	captureStallTimeout = 3 * time.Second
)

// Capture wraps a malgo microphone capture device and exposes the current
// input RMS level to the main loop. The RMS value is shared lock-free via an
// atomic uint32 holding the float32 bit pattern (math.Float32bits).
//
// A Capture may be "unavailable" (no microphone present): the struct is still
// returned so the caller can proceed with an idle avatar, and Close() releases
// the audio context. Mid-session failures (device stopped, stream stalled) are
// recorded and surfaced via Err().
type Capture struct {
	ctx       *malgo.AllocatedContext
	device    *malgo.Device
	available bool

	// RMS shared with the main loop: atomic uint32 holding float32 bits.
	rms atomic.Uint32

	// Accumulators touched only from the malgo Data callback goroutine.
	sumSquares float64
	frameCount uint32

	// Timestamp of the last Data callback (unix nanoseconds).
	lastData atomic.Int64

	// Failure state, written once by the audio thread or the watchdog.
	failed   atomic.Bool
	errMsg   atomic.Value // string
	stopping atomic.Bool
	started  atomic.Bool

	stopWatch chan struct{}
	watchDone chan struct{}
	closeOnce sync.Once
}

// NewCapture initializes the audio context and opens the default capture
// device. When no capture device is available, a Capture with
// Available() == false is returned with a nil error so the caller can degrade
// gracefully (idle avatar + user notice). A non-nil error is returned only for
// real failures (e.g. the audio backend cannot be initialized at all).
func NewCapture() (*Capture, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("initialize audio context: %w", err)
	}

	c := &Capture{
		ctx:       ctx,
		stopWatch: make(chan struct{}),
		watchDone: make(chan struct{}),
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatF32
	cfg.Capture.Channels = captureChannels
	cfg.SampleRate = captureSampleRate
	cfg.PeriodSizeInMilliseconds = capturePeriodMs
	// cfg.Capture.DeviceID stays nil: use the default capture device.

	device, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{
		Data: c.dataCallback,
		Stop: c.stopCallback,
	})
	if err != nil {
		// No usable capture device — degrade gracefully. The context is
		// retained so Close() can release it.
		return c, nil
	}

	c.device = device
	c.available = true
	go c.watch()
	return c, nil
}

// Available reports whether a capture device was opened successfully.
func (c *Capture) Available() bool {
	return c.available
}

// Start begins continuous capture. Returns an error if no device is available
// or the device fails to start; the caller surfaces the error.
func (c *Capture) Start() error {
	if !c.available || c.device == nil {
		return errors.New("no capture device available")
	}
	c.lastData.Store(time.Now().UnixNano())
	c.started.Store(true)
	if err := c.device.Start(); err != nil {
		c.started.Store(false)
		return fmt.Errorf("start audio capture: %w", err)
	}
	return nil
}

// Stop pauses capture. The device can be restarted with Start().
func (c *Capture) Stop() error {
	if c.device == nil {
		return nil
	}
	c.stopping.Store(true)
	c.started.Store(false)
	err := c.device.Stop()
	c.stopping.Store(false)
	return err
}

// Close releases the capture device and audio context. Safe to call multiple
// times; the first call performs the cleanup.
func (c *Capture) Close() {
	c.closeOnce.Do(func() {
		c.stopping.Store(true)
		c.started.Store(false)
		if c.available {
			close(c.stopWatch)
			<-c.watchDone
		}
		if c.device != nil {
			c.device.Uninit()
			c.device = nil
		}
		if c.ctx != nil {
			_ = c.ctx.Uninit()
			c.ctx.Free()
			c.ctx = nil
		}
	})
}

// RMS returns the current input level as a root-mean-square amplitude in
// [0, 1], computed over the most recent 100ms window. Returns 0 when no
// capture is active.
func (c *Capture) RMS() float32 {
	return math.Float32frombits(c.rms.Load())
}

// Err returns the mid-session failure reason, or nil if capture is healthy
// (or merely unavailable). The main loop polls this to surface failures.
func (c *Capture) Err() error {
	if !c.failed.Load() {
		return nil
	}
	msg, _ := c.errMsg.Load().(string)
	if msg == "" {
		msg = "audio capture failed"
	}
	return errors.New(msg)
}

// dataCallback runs on the malgo audio thread. It accumulates squared samples
// and emits an RMS value every captureWindowFrames frames (100ms). It must
// stay allocation-free and lock-free.
func (c *Capture) dataCallback(pOut, pIn []byte, frameCount uint32) {
	if len(pIn) == 0 {
		return
	}
	samples := unsafe.Slice((*float32)(unsafe.Pointer(&pIn[0])), len(pIn)/4)
	for _, s := range samples {
		c.sumSquares += float64(s) * float64(s)
	}
	c.frameCount += frameCount
	if c.frameCount >= captureWindowFrames {
		rms := math.Sqrt(c.sumSquares / float64(c.frameCount))
		c.rms.Store(math.Float32bits(float32(rms)))
		c.sumSquares = 0
		c.frameCount = 0
	}
	c.lastData.Store(time.Now().UnixNano())
}

// stopCallback runs when the device stops. If the stop was not initiated by
// Stop()/Close(), it is a mid-session failure and is recorded for Err().
func (c *Capture) stopCallback() {
	if c.stopping.Load() {
		return
	}
	c.fail("audio capture device stopped unexpectedly")
}

// fail records the first mid-session failure. Subsequent failures are ignored
// so the original cause is preserved.
func (c *Capture) fail(msg string) {
	if c.failed.CompareAndSwap(false, true) {
		c.errMsg.Store(msg)
	}
}

// watch monitors the capture stream for silent stalls: while started, if no
// Data callback has fired for captureStallTimeout, the stream is dead and a
// failure is recorded.
func (c *Capture) watch() {
	defer close(c.watchDone)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopWatch:
			return
		case <-ticker.C:
			if !c.started.Load() {
				continue
			}
			last := time.Unix(0, c.lastData.Load())
			if time.Since(last) > captureStallTimeout {
				c.fail("audio capture stopped delivering data")
				return
			}
		}
	}
}
