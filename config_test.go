package main

import (
	"reflect"
	"testing"
)

// TestDefaultConfig verifies every field of DefaultConfig matches the spec
// (Assumption 4) exactly.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	want := Config{
		Threshold:         0.055,
		IdleDelayMs:       500,
		AnimationsEnabled: true,
		AnimationStyle:    AnimationStyleScramble,
		AnimationCharset:  AnimationCharsetClassic,
		AnimationSteps:    5,
		AnimationDelayMs:  30,
		SmartMode:         SmartModeScramble,
		SmartWave:         SmartWaveRandom,
		AudioDevice:       "",
	}

	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("DefaultConfig() = %+v, want %+v", cfg, want)
	}

	// Individual field checks for precise failure diagnostics.
	if cfg.Threshold != 0.055 {
		t.Errorf("Threshold = %v, want 0.055", cfg.Threshold)
	}
	if cfg.IdleDelayMs != 500 {
		t.Errorf("IdleDelayMs = %d, want 500", cfg.IdleDelayMs)
	}
	if cfg.AnimationsEnabled != true {
		t.Errorf("AnimationsEnabled = %v, want true", cfg.AnimationsEnabled)
	}
	if cfg.AnimationStyle != AnimationStyleScramble {
		t.Errorf("AnimationStyle = %q, want %q", cfg.AnimationStyle, AnimationStyleScramble)
	}
	if cfg.AnimationCharset != AnimationCharsetClassic {
		t.Errorf("AnimationCharset = %q, want %q", cfg.AnimationCharset, AnimationCharsetClassic)
	}
	if cfg.AnimationSteps != 5 {
		t.Errorf("AnimationSteps = %d, want 5", cfg.AnimationSteps)
	}
	if cfg.AnimationDelayMs != 30 {
		t.Errorf("AnimationDelayMs = %d, want 30", cfg.AnimationDelayMs)
	}
	if cfg.SmartMode != SmartModeScramble {
		t.Errorf("SmartMode = %q, want %q", cfg.SmartMode, SmartModeScramble)
	}
	if cfg.SmartWave != SmartWaveRandom {
		t.Errorf("SmartWave = %q, want %q", cfg.SmartWave, SmartWaveRandom)
	}
	if cfg.AudioDevice != "" {
		t.Errorf("AudioDevice = %q, want empty string", cfg.AudioDevice)
	}
}

// TestEnumConstants verifies the exact string values of every enum constant
// used by Config.
func TestEnumConstants(t *testing.T) {
	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"AnimationStyleScramble", AnimationStyleScramble, AnimationStyle("Scramble")},
		{"AnimationStyleCollapse", AnimationStyleCollapse, AnimationStyle("Collapse")},
		{"AnimationStyleReveal", AnimationStyleReveal, AnimationStyle("Reveal")},
		{"AnimationStyleSmart", AnimationStyleSmart, AnimationStyle("Smart")},
		{"AnimationCharsetClassic", AnimationCharsetClassic, AnimationCharset("Classic")},
		{"AnimationCharsetUnicode", AnimationCharsetUnicode, AnimationCharset("Unicode")},
		{"AnimationCharsetAll", AnimationCharsetAll, AnimationCharset("All")},
		{"SmartModeScramble", SmartModeScramble, SmartMode("Scramble")},
		{"SmartModeInstant", SmartModeInstant, SmartMode("Instant")},
		{"SmartWaveRandom", SmartWaveRandom, SmartWave("Random")},
		{"SmartWaveWave", SmartWaveWave, SmartWave("Wave")},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}