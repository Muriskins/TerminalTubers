mod config;
mod avatar;
mod audio;
mod keyboard;
mod settings_menu;

use config::Config;
use avatar::AvatarRenderer;
use audio::AudioCapture;
use keyboard::try_key;
use settings_menu::{show_settings, SettingsResult};
use keyboard::KeyAction;
use config::{idle_frame, talk_frames};

use crossterm::{
    terminal::enable_raw_mode,
};
use rand::Rng;
use std::time::Instant;

fn play_animation(
    renderer: &mut AvatarRenderer,
    target: &str,
    config: &Config,
    rng: &mut impl rand::Rng,
) {
    if config.animation_enabled {
        renderer.animate_transition(
            target,
            config.animation_steps,
            config.animation_delay_ms,
            config.animation_style,
            &config.animation_chars,
            config.smart_mode,
            config.smart_wave,
            rng,
        );
    } else {
        renderer.set_frame(target);
        let _ = renderer.render();
    }
}

fn main() {
    // Включаем raw mode один раз
    if let Err(e) = enable_raw_mode() {
        eprintln!("raw mode error: {}", e);
    }

    eprintln!("╔══════════════════════════════════════╗");
    eprintln!("║     Terminal Tuber v0.1.0            ║");
    eprintln!("║     Q=выход  S=настройки             ║");
    eprintln!("╚══════════════════════════════════════╝");

    let mut config = Config::default_config();
    let mut audio = AudioCapture::new(&config.audio_device);
    let mut renderer = match AvatarRenderer::new() {
        Ok(r) => r,
        Err(e) => { eprintln!("renderer error: {}", e); return; }
    };

    let idle_frame_str = idle_frame();
    let talk_frames_list: Vec<String> = talk_frames();

    if talk_frames_list.is_empty() {
        eprintln!("нет talk фреймов!");
        return;
    }

    renderer.set_frame(&idle_frame_str);
    let _ = renderer.render();

    let mut talking = false;
    let mut last_voice = Instant::now();
    let mut rng = rand::thread_rng();

    // Debounce: счётчик подтверждений голоса
    let voice_confirm_count = 3;
    let mut voice_confirm = 0;

    loop {
        // === ВВОД ===
        match try_key() {
            Some(KeyAction::Char('q')) | Some(KeyAction::Char('Q')) => break,
            Some(KeyAction::Char('s')) | Some(KeyAction::Char('S')) => {
                match show_settings(&mut config) {
                    SettingsResult::Done => {}
                    SettingsResult::Quit => break,
                }

                // Перезапуск аудио с новым устройством
                audio = AudioCapture::new(&config.audio_device);
                play_animation(&mut renderer, &idle_frame_str, &config, &mut rng);
                continue;
            }
            _ => {}
        }

        // === АУДИО → АНИМАЦИЯ ===
        let rms = audio.get_rms();
        let voice = rms > config.volume_threshold;

        if voice {
            voice_confirm += 1;
            if voice_confirm >= voice_confirm_count && !talking {
                // Подтверждённый голос — начинаем говорить
                let idx = rng.gen_range(0..talk_frames_list.len());
                play_animation(&mut renderer, &talk_frames_list[idx], &config, &mut rng);
                talking = true;
            }
            last_voice = Instant::now();
        } else {
            voice_confirm = 0;
            if talking && last_voice.elapsed().as_millis() as u64 > config.idle_delay_ms {
                talking = false;
                play_animation(&mut renderer, &idle_frame_str, &config, &mut rng);
            }
        }

        std::thread::sleep(std::time::Duration::from_millis(50));
    }

    audio.stop();
    eprintln!("\nВыход из Terminal Tuber. Пока!");
}
