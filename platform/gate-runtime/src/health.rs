use serde::Serialize;
use std::sync::{Arc, Mutex};
use std::time::Instant;

#[derive(Debug, Clone, Serialize, PartialEq)]
pub enum Status {
    #[serde(rename = "starting")]
    Starting,
    #[serde(rename = "running")]
    Running,
    #[serde(rename = "stopped")]
    Stopped,
    #[serde(rename = "crashed")]
    Crashed,
}

#[derive(Debug, Serialize)]
pub struct HealthSnapshot {
    pub status: Status,
    pub restart_count: u32,
    pub last_exit_code: Option<i32>,
    pub uptime_secs: f64,
}

struct Inner {
    status: Status,
    restart_count: u32,
    last_exit_code: Option<i32>,
    started_at: Instant,
}

#[derive(Clone)]
pub struct Health {
    inner: Arc<Mutex<Inner>>,
}

impl Health {
    pub fn new() -> Self {
        Self {
            inner: Arc::new(Mutex::new(Inner {
                status: Status::Starting,
                restart_count: 0,
                last_exit_code: None,
                started_at: Instant::now(),
            })),
        }
    }

    pub fn set_status(&self, status: Status) {
        self.inner.lock().unwrap().status = status;
    }

    pub fn record_restart(&self, exit_code: i32) {
        let mut inner = self.inner.lock().unwrap();
        inner.restart_count += 1;
        inner.last_exit_code = Some(exit_code);
    }

    pub fn snapshot(&self) -> HealthSnapshot {
        let inner = self.inner.lock().unwrap();
        HealthSnapshot {
            status: inner.status.clone(),
            restart_count: inner.restart_count,
            last_exit_code: inner.last_exit_code,
            uptime_secs: inner.started_at.elapsed().as_secs_f64(),
        }
    }
}
