use crossterm::event::{self, Event, KeyCode, KeyEventKind};
use std::time::Duration;

pub enum KeyAction {
    Char(char),
    Up,
    Down,
    Escape,
    Enter,
    Backspace,
}

pub fn poll_key(timeout_ms: u64) -> Option<KeyAction> {
    if event::poll(Duration::from_millis(timeout_ms)).unwrap_or(false) {
        if let Event::Key(key_event) = event::read().ok()? {
            if key_event.kind != KeyEventKind::Press {
                return None;
            }
            match key_event.code {
                KeyCode::Char(c) => return Some(KeyAction::Char(c)),
                KeyCode::Up => return Some(KeyAction::Up),
                KeyCode::Down => return Some(KeyAction::Down),
                KeyCode::Esc => return Some(KeyAction::Escape),
                KeyCode::Enter => return Some(KeyAction::Enter),
                KeyCode::Backspace => return Some(KeyAction::Backspace),
                _ => {}
            }
        }
    }
    None
}

pub fn try_key() -> Option<KeyAction> {
    poll_key(0)
}
