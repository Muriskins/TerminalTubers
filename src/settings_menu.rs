use crossterm::{
    cursor::MoveTo,
    execute,
    style::{Color, Print, SetForegroundColor},
    terminal::{Clear, ClearType},
};
use std::io::{stdout, Write};

use crate::audio::list_audio_devices;
use crate::config::Config;
use crate::keyboard::{poll_key, KeyAction};

pub enum SettingsResult {
    Done,
    Quit,
}

pub fn show_settings(config: &mut Config) -> SettingsResult {
    let devices = list_audio_devices();
    let mut cursor: usize = 0; // 0=устройство, 1=порог, 2=анимация, 3=стиль, 4=символы, 5=шаги, 6=задержка, 7=smart_mode, 8=smart_wave
    let max_cursor = 8;
    let mut editing = false;
    let mut buf = String::new();
    let mut error_msg: Option<String> = None;

    loop {
        // Очистка и рендер
        let _ = execute!(stdout(), Clear(ClearType::All));

        let mut y = 0u16;

        // Заголовок
        draw_centered(&mut stdout(), y, "═══ НАСТРОЙКИ TERMINAL TUBER ═══", Color::Cyan);
        y += 2;

        // [0] Строка: устройство
        let sel = cursor == 0 && !editing;
        draw_centered(&mut stdout(), y,
            &format!("Устройство: {}", if config.audio_device.is_empty() { "авто (pipewire)" } else { &config.audio_device }),
            if sel { Color::Yellow } else { Color::White });
        y += 1;

        // Список устройств (всегда виден)
        y += 1;
        for (i, dev) in devices.iter().enumerate() {
            let active = config.audio_device == *dev;
            let c = if active { Color::Green } else { Color::White };
            let marker = if active { "► " } else { "  " };
            draw_left(&mut stdout(), y,
                &format!("[{}] {}{}", i + 1, marker, dev), c);
            y += 1;
        }
        y += 1;

        // [1] Строка: порог
        let sel1 = cursor == 1 && !editing;
        let thresh_text = if editing && cursor == 1 {
            format!("Порог: {}█", buf)
        } else {
            format!("Порог: {:.4}", config.volume_threshold)
        };
        draw_centered(&mut stdout(), y, &thresh_text, if sel1 { Color::Yellow } else { Color::White });
        y += 2;

        // [2] Анимация: ВКЛ/ВЫКЛ
        let sel2 = cursor == 2;
        draw_centered(&mut stdout(), y,
            &format!("Анимация: {}", if config.animation_enabled { "ВКЛ" } else { "ВЫКЛ" }),
            if sel2 { Color::Yellow } else { Color::White });
        y += 1;

        // [3] Стиль анимации
        let sel3 = cursor == 3;
        draw_centered(&mut stdout(), y,
            &format!("Стиль: {}", config.animation_style.name()),
            if sel3 { Color::Yellow } else { Color::White });
        y += 1;

        // [4] Символы анимации
        let sel4 = cursor == 4;
        draw_centered(&mut stdout(), y,
            &format!("Символы: {}", config.animation_chars.name()),
            if sel4 { Color::Yellow } else { Color::White });
        y += 1;

        // [5] Шаги анимации
        let sel5 = cursor == 5 && !editing;
        let steps_text = if editing && cursor == 5 {
            format!("{}█", buf)
        } else {
            format!("{}", config.animation_steps)
        };
        draw_centered(&mut stdout(), y,
            &format!("Шаги: {}", steps_text),
            if sel5 { Color::Yellow } else { Color::White });
        y += 1;

        // [6] Задержка шагов
        let sel6 = cursor == 6 && !editing;
        let delay_text = if editing && cursor == 6 {
            format!("{}█", buf)
        } else {
            format!("{}", config.animation_delay_ms)
        };
        draw_centered(&mut stdout(), y,
            &format!("Задержка: {} мс", delay_text),
            if sel6 { Color::Yellow } else { Color::White });
        y += 1;

        // [7] Smart-режим
        let sel7 = cursor == 7;
        draw_centered(&mut stdout(), y,
            &format!("Smart-режим: {}", config.smart_mode.name()),
            if sel7 { Color::Yellow } else { Color::White });
        y += 1;

        // [8] Smart-волна
        let sel8 = cursor == 8;
        draw_centered(&mut stdout(), y,
            &format!("Smart-волна: {}", config.smart_wave.name()),
            if sel8 { Color::Yellow } else { Color::White });
        y += 1;

        y += 1;

        // Подсказки
        if editing {
            draw_centered(&mut stdout(), y, "цифры/точка — ввод", Color::DarkGrey); y += 1;
            draw_centered(&mut stdout(), y, "Enter — принять | Esc — отмена | Bksp — стереть", Color::DarkGrey); y += 1;
            if let Some(ref msg) = error_msg {
                draw_centered(&mut stdout(), y, msg, Color::Red);
            }
        } else {
            draw_centered(&mut stdout(), y, "↑/↓ — курсор | Enter — изменить | 1-9 — устройство", Color::DarkGrey); y += 1;
            draw_centered(&mut stdout(), y, "s/Esc — выйти | q — выход из программы", Color::DarkGrey);
        }

        let _ = stdout().flush();

        // Ввод
        let key = poll_key(50);
        let Some(key) = key else { continue };

        if editing {
            match key {
                KeyAction::Enter => {
                    if cursor == 1 {
                        // Порог
                        if let Ok(v) = buf.parse::<f32>() {
                            if v > 0.0 { config.volume_threshold = v; }
                            editing = false;
                            buf.clear();
                            error_msg = None;
                        } else {
                            error_msg = Some("Ошибка: введите число (например, 0.01)".into());
                            buf.clear();
                        }
                    } else if cursor == 5 {
                        // Шаги
                        if let Ok(v) = buf.parse::<u32>() {
                            if v > 0 && v <= 50 { config.animation_steps = v; }
                            editing = false;
                            buf.clear();
                            error_msg = None;
                        } else {
                            error_msg = Some("Ошибка: введите число (1-50)".into());
                            buf.clear();
                        }
                    } else if cursor == 6 {
                        // Задержка
                        if let Ok(v) = buf.parse::<u64>() {
                            if v > 0 && v <= 500 { config.animation_delay_ms = v; }
                            editing = false;
                            buf.clear();
                            error_msg = None;
                        } else {
                            error_msg = Some("Ошибка: введите число (1-500)".into());
                            buf.clear();
                        }
                    } else {
                        editing = false;
                        buf.clear();
                    }
                }
                KeyAction::Escape => { editing = false; buf.clear(); error_msg = None; }
                KeyAction::Backspace => { buf.pop(); error_msg = None; }
                KeyAction::Char(c) if c.is_ascii_digit() || c == '.' => { buf.push(c); error_msg = None; }
                KeyAction::Char('q') | KeyAction::Char('Q') => return SettingsResult::Quit,
                _ => {}
            }
        } else {
            match key {
                KeyAction::Escape | KeyAction::Char('s') | KeyAction::Char('S') => return SettingsResult::Done,
                KeyAction::Char('q') | KeyAction::Char('Q') => return SettingsResult::Quit,
                KeyAction::Down => {
                    if cursor < max_cursor { cursor += 1; }
                }
                KeyAction::Up => {
                    if cursor > 0 { cursor -= 1; }
                }
                KeyAction::Enter => {
                    if cursor == 1 {
                        // Редактирование порога
                        editing = true;
                        buf = format!("{:.4}", config.volume_threshold);
                    } else if cursor == 2 {
                        // Переключение анимации
                        config.animation_enabled = !config.animation_enabled;
                    } else if cursor == 3 {
                        // Переключение стиля
                        config.animation_style = config.animation_style.next();
                    } else if cursor == 4 {
                        // Переключение символов
                        config.animation_chars = config.animation_chars.next();
                    } else if cursor == 5 {
                        // Редактирование шагов
                        editing = true;
                        buf = format!("{}", config.animation_steps);
                    } else if cursor == 6 {
                        // Редактирование задержки
                        editing = true;
                        buf = format!("{}", config.animation_delay_ms);
                    } else if cursor == 7 {
                        // Переключение smart-режима
                        config.smart_mode = config.smart_mode.next();
                    } else if cursor == 8 {
                        // Переключение smart-волны
                        config.smart_wave = config.smart_wave.next();
                    }
                }
                KeyAction::Char(c) if ('1'..='9').contains(&c) => {
                    let idx = c.to_digit(10).unwrap() as usize - 1;
                    if idx < devices.len() {
                        config.audio_device = devices[idx].clone();
                        return SettingsResult::Done;
                    }
                }
                _ => {}
            }
        }
    }
}

fn draw_centered<W: Write>(w: &mut W, y: u16, text: &str, color: Color) {
    let (width, _) = crossterm::terminal::size().unwrap_or((80, 24));
    let x = (width as usize).saturating_sub(text.chars().count()) / 2;
    let _ = execute!(*w, MoveTo(x as u16, y), SetForegroundColor(color), Print(text));
}

fn draw_left<W: Write>(w: &mut W, y: u16, text: &str, color: Color) {
    let _ = execute!(*w, MoveTo(4, y), SetForegroundColor(color), Print(text));
}
