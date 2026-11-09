//! Graceful-shutdown signal for the bundled binaries.
//!
//! Watches SIGINT *and* SIGTERM (Ctrl+C only on non-Unix): supervisors like
//! `docker stop` send SIGTERM, and as PID 1 in a `scratch` image the kernel
//! silently drops any signal without an installed handler.

/// Resolves on the first stop signal: SIGINT or SIGTERM on Unix, Ctrl+C
/// elsewhere. Both handlers are armed before the future first suspends.
#[cfg(unix)]
pub async fn shutdown_signal() {
    use tokio::signal::unix::{Signal, SignalKind, signal};

    fn install(kind: SignalKind, name: &str) -> Option<Signal> {
        signal(kind)
            .inspect_err(|error| log::error!("shutdown_signal: cannot watch {name}: {error}"))
            .ok()
    }

    let mut sigint = install(SignalKind::interrupt(), "SIGINT");
    let mut sigterm = install(SignalKind::terminate(), "SIGTERM");

    if sigint.is_none() && sigterm.is_none() {
        log::error!(
            "shutdown_signal: no stop signal could be installed; graceful shutdown disabled"
        );
    }

    async fn wait(signal: &mut Option<Signal>) {
        match signal {
            Some(signal) => {
                signal.recv().await;
            }
            None => std::future::pending::<()>().await,
        }
    }

    tokio::select! {
        _ = wait(&mut sigint) => {}
        _ = wait(&mut sigterm) => {}
    }
}

/// Non-Unix fallback: Ctrl+C is the only portable stop signal Tokio exposes.
#[cfg(not(unix))]
pub async fn shutdown_signal() {
    let _ = tokio::signal::ctrl_c().await;
}
