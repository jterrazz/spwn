use std::process::Stdio;
use std::time::Duration;
use tokio::process::Command;

use crate::health::{Health, Status};

const BASE_DELAY: Duration = Duration::from_secs(2);
const MAX_BACKOFF: Duration = Duration::from_secs(30);
const BACKOFF_FACTOR: f64 = 2.0;

/// Run the agent CLI with crash recovery and exponential backoff.
pub async fn run(agent_cli: &str, max_restarts: u32, health: Health) {
    let mut consecutive_failures: u32 = 0;

    loop {
        health.set_status(Status::Running);
        eprintln!("[gate] spawning agent: {}", agent_cli);

        let result = Command::new(agent_cli)
            .arg("--dangerously-skip-permissions")
            .stdin(Stdio::inherit())
            .stdout(Stdio::inherit())
            .stderr(Stdio::inherit())
            .status()
            .await;

        match result {
            Ok(status) => {
                let code = status.code().unwrap_or(-1);
                eprintln!("[gate] agent exited with code {}", code);

                if status.success() {
                    // Clean exit — stop the loop
                    health.set_status(Status::Stopped);
                    eprintln!("[gate] agent exited cleanly, shutting down");
                    return;
                }

                // Non-zero exit — crash
                health.record_restart(code);
                consecutive_failures += 1;
            }
            Err(e) => {
                eprintln!("[gate] failed to spawn agent: {}", e);
                health.record_restart(-1);
                consecutive_failures += 1;
            }
        }

        // Check max restarts
        if max_restarts > 0 && consecutive_failures >= max_restarts {
            health.set_status(Status::Crashed);
            eprintln!(
                "[gate] max restarts ({}) exceeded, giving up",
                max_restarts
            );
            return;
        }

        // Exponential backoff
        let delay = backoff_delay(consecutive_failures);
        eprintln!(
            "[gate] restarting in {:?} (attempt {}/{})",
            delay, consecutive_failures, max_restarts
        );
        tokio::time::sleep(delay).await;
    }
}

fn backoff_delay(failures: u32) -> Duration {
    let delay_secs = BASE_DELAY.as_secs_f64() * BACKOFF_FACTOR.powi(failures.saturating_sub(1) as i32);
    let capped = Duration::from_secs_f64(delay_secs.min(MAX_BACKOFF.as_secs_f64()));
    capped
}
