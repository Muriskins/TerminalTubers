package main

// AnimationStyle selects the transition animation between frames.
type AnimationStyle string

const (
	AnimationStyleScramble AnimationStyle = "Scramble"
	AnimationStyleCollapse AnimationStyle = "Collapse"
	AnimationStyleReveal   AnimationStyle = "Reveal"
	AnimationStyleSmart    AnimationStyle = "Smart"
)

// AnimationCharset selects the character set used by transition animations.
type AnimationCharset string

const (
	AnimationCharsetClassic AnimationCharset = "Classic"
	AnimationCharsetUnicode AnimationCharset = "Unicode"
	AnimationCharsetAll     AnimationCharset = "All"
)

// SmartMode selects how differing characters resolve in Smart animation.
type SmartMode string

const (
	SmartModeScramble SmartMode = "Scramble"
	SmartModeInstant  SmartMode = "Instant"
)

// SmartWave selects how differing characters settle in Smart animation.
type SmartWave string

const (
	SmartWaveRandom SmartWave = "Random"
	SmartWaveWave   SmartWave = "Wave"
)

// Config holds all runtime settings for TerminalTubers.
type Config struct {
	Threshold         float64
	IdleDelayMs       int
	AnimationsEnabled bool
	AnimationStyle    AnimationStyle
	AnimationCharset  AnimationCharset
	AnimationSteps    int
	AnimationDelayMs  int
	SmartMode         SmartMode
	SmartWave         SmartWave
	AudioDevice       string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
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
}