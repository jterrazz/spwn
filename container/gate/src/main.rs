use clap::Parser;
use std::path::PathBuf;

mod acp;
mod health;
mod recovery;

#[derive(Parser)]
#[command(name = "universe-gate", about = "Container-side Gate for Universe")]
struct Args {
    /// Path to the Unix socket for host-side communication
    #[arg(long, default_value = "/gate/gate.sock")]
    listen: PathBuf,

    /// Agent CLI command to spawn
    #[arg(long, default_value = "claude")]
    agent_cli: String,

    /// Maximum restart attempts before giving up
    #[arg(long, default_value_t = 5)]
    max_restarts: u32,
}

#[tokio::main]
async fn main() {
    let args = Args::parse();

    eprintln!("[gate] universe-gate starting");
    eprintln!("[gate] socket: {}", args.listen.display());
    eprintln!("[gate] agent CLI: {}", args.agent_cli);

    let health = health::Health::new();

    // Run the recovery loop — spawns the agent CLI and restarts on crash
    recovery::run(
        &args.agent_cli,
        args.max_restarts,
        health.clone(),
    )
    .await;
}
