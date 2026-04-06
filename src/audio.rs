use std::sync::atomic::{AtomicBool, AtomicU32, Ordering};
use std::sync::Arc;
use std::thread;
use std::process::{Child, Command, Stdio};
use std::io::Read;

pub struct AudioCapture {
    rms: Arc<AtomicU32>,
    running: Arc<AtomicBool>,
}

impl AudioCapture {
    pub fn new(device: &str) -> Self {
        let rms = Arc::new(AtomicU32::new(0));
        let running = Arc::new(AtomicBool::new(true));

        let method = detect_method();
        let (cmd, args) = build_args(method, device);

        let child = spawn_process(&cmd, &args);
        let rms_clone = rms.clone();
        let running_clone = running.clone();

        thread::spawn(move || {
            read_rms(child, rms_clone, running_clone);
        });

        // Тест — ждём пока данные пойдут
        thread::sleep(std::time::Duration::from_millis(300));
        let test = f32::from_bits(rms.load(Ordering::Relaxed));
        eprintln!("Аудио: {} {} (RMS тест: {:.6})", cmd, args.join(" "), test);

        Self { rms, running }
    }

    pub fn get_rms(&self) -> f32 {
        f32::from_bits(self.rms.load(Ordering::Relaxed))
    }

    pub fn stop(&mut self) {
        self.running.store(false, Ordering::Relaxed);
    }
}

fn detect_method() -> &'static str {
    if command_exists("pw-record") { "pw" }
    else if command_exists("arecord") { "alsa" }
    else { "none" }
}

fn build_args(method: &str, device: &str) -> (String, Vec<String>) {
    match method {
        "pw" => {
            let mut args = vec!["--format=f32".into(), "--rate=48000".into(), "--channels=1".into(), "-".into()];
            if !device.is_empty() && !["default", "pipewire", "pulse"].contains(&device) {
                args[0] = format!("--target={}", device);
            }
            ("pw-record".into(), args)
        }
        "alsa" => {
            let mut args = vec!["-f".into(), "FLOAT_LE".into(), "-c".into(), "1".into(),
                                "-r".into(), "48000".into(), "-t".into(), "raw".into(), "-q".into()];
            if !device.is_empty() && !["default", "pipewire", "pulse"].contains(&device) {
                args.push("-D".into());
                args.push(device.into());
            }
            ("arecord".into(), args)
        }
        _ => ("echo".into(), vec!["нет аудио".into()]),
    }
}

fn command_exists(cmd: &str) -> bool {
    Command::new("which").arg(cmd).output().map(|o| o.status.success()).unwrap_or(false)
}

fn spawn_process(cmd: &str, args: &[String]) -> Child {
    Command::new(cmd)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()
        .expect("Не удалось запустить аудио процесс")
}

fn read_rms(mut child: Child, rms: Arc<AtomicU32>, running: Arc<AtomicBool>) {
    let Some(mut stdout) = child.stdout.take() else { return };
    let mut buf = [0u8; 4096];
    let mut sum: f64 = 0.0;
    let mut count: usize = 0;

    while running.load(Ordering::Relaxed) {
        match stdout.read(&mut buf) {
            Ok(0) => break,
            Ok(n) => {
                for i in 0..(n / 4) {
                    let b = &buf[i*4..(i+1)*4];
                    let s = f32::from_le_bytes([b[0], b[1], b[2], b[3]]);
                    sum += (s * s) as f64;
                    count += 1;
                }
                if count >= 4800 {
                    let v = (sum / count as f64).sqrt() as f32;
                    rms.store(v.to_bits(), Ordering::Relaxed);
                    sum = 0.0;
                    count = 0;
                }
            }
            Err(_) => break,
        }
    }
    let _ = child.kill();
}

pub fn list_audio_devices() -> Vec<String> {
    let mut devs = Vec::new();

    if command_exists("pactl") {
        if let Ok(out) = Command::new("pactl").args(["list", "sources", "short"]).output() {
            for line in String::from_utf8_lossy(&out.stdout).lines() {
                if let Some(name) = line.split_whitespace().nth(1) {
                    devs.push(name.to_string());
                }
            }
        }
    }

    for s in &["pipewire", "pulse", "default"] {
        if !devs.iter().any(|d| d == s) {
            devs.push(s.to_string());
        }
    }

    devs
}
