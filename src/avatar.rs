use crossterm::{
    cursor::{Hide, MoveTo, Show},
    execute,
    style::Print,
    terminal::{Clear, ClearType, EnterAlternateScreen, LeaveAlternateScreen},
};
use rand::Rng;
use std::io::{stdout, Result, Stdout, Write};

use crate::config::{AnimationChars, AnimationStyle, SmartMode, SmartWave};

pub struct AvatarRenderer {
    stdout: Stdout,
    frame: String,
    pub h: usize,
    pub w: usize,
}

impl AvatarRenderer {
    pub fn new() -> Result<Self> {
        let mut stdout = stdout();
        execute!(stdout, EnterAlternateScreen, Hide)?;
        Ok(Self { stdout, frame: String::new(), h: 0, w: 0 })
    }

    pub fn set_frame(&mut self, s: &str) {
        self.frame = s.to_string();
        self.h = s.lines().count();
        self.w = s.lines().map(|l| l.len()).max().unwrap_or(0);
    }

    pub fn render(&mut self) -> Result<()> {
        let (tw, th) = crossterm::terminal::size()?;
        let tw = tw as usize;
        let th = th as usize;
        let cx = tw.saturating_sub(self.w) / 2;
        let cy = th.saturating_sub(self.h) / 2;

        execute!(self.stdout, Clear(ClearType::All))?;

        for (i, line) in self.frame.lines().enumerate() {
            execute!(
                self.stdout,
                MoveTo(cx as u16, (cy + i) as u16),
                Print(line)
            )?;
        }
        self.stdout.flush()?;
        Ok(())
    }

    /// Анимированный переход к целевому фрейму
    pub fn animate_transition(
        &mut self,
        target: &str,
        steps: u32,
        delay_ms: u64,
        style: AnimationStyle,
        chars: &AnimationChars,
        smart_mode: SmartMode,
        smart_wave: SmartWave,
        rng: &mut impl Rng,
    ) {
        let target_lines: Vec<&str> = target.lines().collect();
        let th = target_lines.len();
        let tw = target_lines.iter().map(|l| l.len()).max().unwrap_or(0);

        // Расширяем target до равномерной сетки
        let mut target_grid: Vec<Vec<char>> = target_lines
            .iter()
            .map(|l| {
                let mut row: Vec<char> = l.chars().collect();
                while row.len() < tw { row.push(' '); }
                row
            })
            .collect();
        // Пустые строки для выравнивания
        while target_grid.len() < th {
            target_grid.push(vec![' '; tw]);
        }

        let (term_w, _) = crossterm::terminal::size().unwrap_or((80, 24));
        let cx = (term_w as usize).saturating_sub(tw) / 2;

        let charset = chars.chars();

        match style {
            AnimationStyle::Scramble => {
                // Текущий grid = предыдущий фрейм или случайный
                let mut current_grid: Vec<Vec<char>> = self.frame.lines()
                    .map(|l| {
                        let mut row: Vec<char> = l.chars().collect();
                        while row.len() < tw { row.push(' '); }
                        row
                    })
                    .collect();
                while current_grid.len() < th {
                    current_grid.push(vec![' '; tw]);
                }

                for step in 1..=steps {
                    for r in 0..th {
                        for c in 0..tw {
                            // Вероятность «застывания» растёт с каждым шагом
                            let fix_chance = step as f32 / steps as f32;
                            if rng.gen::<f32>() < fix_chance {
                                current_grid[r][c] = target_grid[r][c];
                            } else {
                                current_grid[r][c] = charset[rng.gen_range(0..charset.len())];
                            }
                        }
                    }
                    self.render_grid(&current_grid, cx, th);
                    std::thread::sleep(std::time::Duration::from_millis(delay_ms));
                }
                // Финальный рендер — точно целевой
                self.set_frame(target);
                let _ = self.render();
            }
            AnimationStyle::Collapse => {
                // Схлопывание от краёв к центру
                let mid_row = th / 2;
                let mid_col = tw / 2;
                let max_dist = (mid_row * mid_row + mid_col * mid_col) as f32;
                let collapse_chars = &['/', '\\', '*', '#', '@'];

                for step in 1..=steps {
                    let progress = step as f32 / steps as f32;
                    let threshold = (1.0 - progress) * (max_dist.sqrt());

                    let mut grid: Vec<Vec<char>> = vec![vec![' '; tw]; th];
                    for r in 0..th {
                        for c in 0..tw {
                            let dist = ((r as f32 - mid_row as f32).powi(2)
                                + (c as f32 - mid_col as f32).powi(2)).sqrt();
                            if dist <= threshold {
                                // Уже «дошёл» — рисуем целевой
                                grid[r][c] = target_grid[r][c];
                            } else {
                                // Ещё в процессе — случайные collapse-символы
                                grid[r][c] = collapse_chars[rng.gen_range(0..collapse_chars.len())];
                            }
                        }
                    }
                    self.render_grid(&grid, cx, th);
                    std::thread::sleep(std::time::Duration::from_millis(delay_ms));
                }
                self.set_frame(target);
                let _ = self.render();
            }
            AnimationStyle::Reveal => {
                // Полный рандом → постепенное проявление
                let mut grid: Vec<Vec<char>> = vec![vec![' '; tw]; th];
                // Шаг 0 — полностью случайный
                for r in 0..th {
                    for c in 0..tw {
                        grid[r][c] = charset[rng.gen_range(0..charset.len())];
                    }
                }

                for step in 1..=steps {
                    let progress = step as f32 / steps as f32;
                    for r in 0..th {
                        for c in 0..tw {
                            let fix_chance = progress * progress; // квадратичное ускорение
                            if rng.gen::<f32>() < fix_chance {
                                grid[r][c] = target_grid[r][c];
                            } else {
                                grid[r][c] = charset[rng.gen_range(0..charset.len())];
                            }
                        }
                    }
                    self.render_grid(&grid, cx, th);
                    std::thread::sleep(std::time::Duration::from_millis(delay_ms));
                }
                self.set_frame(target);
                let _ = self.render();
            }
            AnimationStyle::Smart => {
                self.animate_smart(&target_grid, steps, delay_ms, smart_mode, smart_wave, charset, rng, cx, th);
            }
        }
    }

    fn render_grid(&mut self, grid: &[Vec<char>], cx: usize, height: usize) {
        let (_, th) = crossterm::terminal::size().unwrap_or((80, 24));
        let cy = (th as usize).saturating_sub(height) / 2;
        let _ = execute!(self.stdout, Clear(ClearType::All));
        for (r, row) in grid.iter().enumerate() {
            let line: String = row.iter().collect();
            let _ = execute!(
                self.stdout,
                MoveTo(cx as u16, (cy + r) as u16),
                Print(line)
            );
        }
        let _ = self.stdout.flush();
    }

    /// Smart-анимация: меняются только differing символы
    fn animate_smart(
        &mut self,
        target_grid: &[Vec<char>],
        steps: u32,
        delay_ms: u64,
        smart_mode: SmartMode,
        smart_wave: SmartWave,
        charset: &'static [char],
        rng: &mut impl Rng,
        cx: usize,
        _target_h: usize,
    ) {
        let th_target = target_grid.len();
        let tw_target = target_grid.first().map(|r| r.len()).unwrap_or(0);

        // Текущий grid из self.frame
        let current_lines: Vec<&str> = self.frame.lines().collect();
        let th_current = current_lines.len();
        let tw_current = current_lines.iter().map(|l| l.len()).max().unwrap_or(0);

        // Общая сетка — максимум обоих
        let grid_h = th_current.max(th_target);
        let grid_w = tw_current.max(tw_target);

        // Строим current_grid до общего размера
        let mut current_grid: Vec<Vec<char>> = Vec::with_capacity(grid_h);
        for r in 0..grid_h {
            let mut row = vec![' '; grid_w];
            if r < th_current {
                let line_chars: Vec<char> = current_lines[r].chars().collect();
                for (c, &ch) in line_chars.iter().enumerate() {
                    if c < grid_w { row[c] = ch; }
                }
            }
            current_grid.push(row);
        }

        // Строим target_grid до общего размера
        let mut full_target: Vec<Vec<char>> = Vec::with_capacity(grid_h);
        for r in 0..grid_h {
            let mut row = vec![' '; grid_w];
            if r < th_target {
                for (c, &ch) in target_grid[r].iter().enumerate() {
                    if c < grid_w { row[c] = ch; }
                }
            }
            full_target.push(row);
        }

        // Найти differing позиции
        let mut diff_positions: Vec<(usize, usize)> = Vec::new();
        for r in 0..grid_h {
            for c in 0..grid_w {
                if current_grid[r][c] != full_target[r][c] {
                    diff_positions.push((r, c));
                }
            }
        }

        eprintln!("Smart: {} differing из {}×{}", diff_positions.len(), grid_w, grid_h);

        if diff_positions.is_empty() {
            return; // Ничего не меняется
        }

        // Instant режим — быстрая минимальная анимация (1 шаг)
        let actual_steps = if matches!(smart_mode, SmartMode::Instant) {
            1
        } else {
            steps.max(8) // Минимум 8 шагов для видимости
        };
        let actual_delay = if matches!(smart_mode, SmartMode::Instant) {
            delay_ms.max(50)
        } else {
            delay_ms
        };

        // Вычислить bounding box differing-зоны для Wave
        let min_r = diff_positions.iter().map(|(r, _)| *r).min().unwrap();
        let max_r = diff_positions.iter().map(|(r, _)| *r).max().unwrap();
        let min_c = diff_positions.iter().map(|(_, c)| *c).min().unwrap();
        let max_c = diff_positions.iter().map(|(_, c)| *c).max().unwrap();
        let center_r = (min_r + max_r) as f32 / 2.0;
        let center_c = (min_c + max_c) as f32 / 2.0;
        let max_dist = ((max_r - min_r) as f32 / 2.0).hypot((max_c - min_c) as f32 / 2.0).max(0.01);

        let mut fixed: Vec<bool> = vec![false; diff_positions.len()];

        for step in 1..=actual_steps {
            let progress = step as f32 / actual_steps as f32;

            for (i, &(r, c)) in diff_positions.iter().enumerate() {
                if fixed[i] { continue; }

                let fix_chance = if matches!(smart_wave, SmartWave::Wave) {
                    let dist = ((r as f32 - center_r).powi(2) + (c as f32 - center_c).powi(2)).sqrt();
                    let normalized_dist = dist / max_dist;
                    (1.0 - normalized_dist + progress).clamp(0.0, 1.0)
                } else {
                    progress
                };

                if rng.gen::<f32>() < fix_chance {
                    current_grid[r][c] = full_target[r][c];
                    fixed[i] = true;
                } else {
                    current_grid[r][c] = charset[rng.gen_range(0..charset.len())];
                }
            }

            self.render_grid(&current_grid, cx, grid_h);
            std::thread::sleep(std::time::Duration::from_millis(actual_delay));
        }

        // Финальный рендер
        self.set_frame(&full_target.iter().map(|r| r.iter().collect::<String>()).collect::<Vec<_>>().join("\n"));
        let _ = self.render();
    }
}

impl Drop for AvatarRenderer {
    fn drop(&mut self) {
        let _ = execute!(self.stdout, Show, LeaveAlternateScreen);
    }
}
