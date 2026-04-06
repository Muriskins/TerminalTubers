
/// Стиль анимации перехода между кадрами
#[derive(Clone, Copy, PartialEq)]
pub enum AnimationStyle {
    Scramble,  // символы случайно меняются, постепенно «застывая»
    Collapse,  // схлопывание от краёв к центру
    Reveal,    // полный рандом → проявление
    Smart,     // ТОЛЬКО differing символы
}

impl AnimationStyle {
    pub fn name(&self) -> &'static str {
        match self {
            AnimationStyle::Scramble => "Scramble",
            AnimationStyle::Collapse => "Collapse",
            AnimationStyle::Reveal => "Reveal",
            AnimationStyle::Smart => "Smart",
        }
    }

    pub fn next(&self) -> Self {
        match self {
            AnimationStyle::Scramble => AnimationStyle::Collapse,
            AnimationStyle::Collapse => AnimationStyle::Reveal,
            AnimationStyle::Reveal => AnimationStyle::Smart,
            AnimationStyle::Smart => AnimationStyle::Scramble,
        }
    }
}

/// Режим Smart-анимации
#[derive(Clone, Copy, PartialEq)]
pub enum SmartMode {
    Scramble, // differing символы крутятся через случайные
    Instant,  // мгновенная замена differing
}

impl SmartMode {
    pub fn name(&self) -> &'static str {
        match self {
            SmartMode::Scramble => "Scramble",
            SmartMode::Instant => "Instant",
        }
    }

    pub fn next(&self) -> Self {
        match self {
            SmartMode::Scramble => SmartMode::Instant,
            SmartMode::Instant => SmartMode::Scramble,
        }
    }
}

/// Волна в Smart-режиме
#[derive(Clone, Copy, PartialEq)]
pub enum SmartWave {
    Random, // случайная фиксация differing
    Wave,   // волна от центра differing-зоны
}

impl SmartWave {
    pub fn name(&self) -> &'static str {
        match self {
            SmartWave::Random => "Random",
            SmartWave::Wave => "Wave",
        }
    }

    pub fn next(&self) -> Self {
        match self {
            SmartWave::Random => SmartWave::Wave,
            SmartWave::Wave => SmartWave::Random,
        }
    }
}

/// Набор символов для случайных символов в анимации
#[derive(Clone, Copy, PartialEq)]
pub enum AnimationChars {
    Classic,   // @#%*=-+:.
    Unicode,   // █▓▒░╔╗╚╝║═
    AllPrint,  // все печатные ASCII
}

impl AnimationChars {
    pub fn name(&self) -> &'static str {
        match self {
            AnimationChars::Classic => "Classic (@#%*)",
            AnimationChars::Unicode => "Unicode (█▓▒░)",
            AnimationChars::AllPrint => "All printable",
        }
    }

    pub fn next(&self) -> Self {
        match self {
            AnimationChars::Classic => AnimationChars::Unicode,
            AnimationChars::Unicode => AnimationChars::AllPrint,
            AnimationChars::AllPrint => AnimationChars::Classic,
        }
    }

    pub fn chars(&self) -> &'static [char] {
        match self {
            AnimationChars::Classic => &['@', '#', '%', '*', '=', '-', '+', ':', '.', ' '],
            AnimationChars::Unicode => &['█', '▓', '▒', '░', '╔', '╗', '╚', '╝', '║', '═', ' '],
            AnimationChars::AllPrint => &[
                '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/',
                '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
                ':', ';', '<', '=', '>', '?', '@',
                'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
                'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
                '[', '\\', ']', '^', '_', '`',
                'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
                'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
                '{', '|', '}', '~', ' ',
            ],
        }
    }
}

pub struct Config {
    pub audio_device: String,
    pub volume_threshold: f32,
    pub idle_delay_ms: u64,
    // Анимация
    pub animation_enabled: bool,
    pub animation_style: AnimationStyle,
    pub animation_chars: AnimationChars,
    pub animation_steps: u32,
    pub animation_delay_ms: u64,
    // Smart-режим
    pub smart_mode: SmartMode,
    pub smart_wave: SmartWave,
}

/// Встроенные ASCII-фреймы (компилируются в бинарник)
pub fn idle_frame() -> String {
    include_str!("../ascii_art/idle.txt").to_string()
}

pub fn talk_frames() -> Vec<String> {
    vec![
        include_str!("../ascii_art/tolk0.txt").to_string(),
        include_str!("../ascii_art/tolk1.txt").to_string(),
        include_str!("../ascii_art/tolk2.txt").to_string(),
    ]
}

impl Config {
    pub fn default_config() -> Self {
        Self {
            audio_device: String::new(),
            volume_threshold: 0.055,
            idle_delay_ms: 500,
            animation_enabled: true,
            animation_style: AnimationStyle::Scramble,
            animation_chars: AnimationChars::Classic,
            animation_steps: 5,
            animation_delay_ms: 30,
            smart_mode: SmartMode::Scramble,
            smart_wave: SmartWave::Random,
        }
    }
}
