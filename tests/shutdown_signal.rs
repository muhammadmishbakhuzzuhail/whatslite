//! Regression guard for graceful shutdown under `docker stop` / k8s / systemd.
//!
//! Those supervisors stop a process with SIGTERM and only escalate to SIGKILL
//! after a grace period. The bundled binaries once watched only SIGINT, so
//! SIGTERM went unhandled and every `docker stop` timed out and hard-killed the
//! container. `wa_rs::shutdown_signal()` must resolve on SIGTERM as well as
//! SIGINT.
//!
//! All scenarios live in one `#[tokio::test]` on purpose: signals are
//! process-global, so running signal-raising cases in parallel would let one
//! case's signal wake another's future.

#![cfg(all(feature = "signal", unix))]

use std::future::Future;
use std::pin::Pin;
use std::task::{Context, Poll};
use std::time::Duration;

use tokio::signal::unix::{SignalKind, signal};

fn assert_pending<F: Future + ?Sized>(mut future: Pin<&mut F>, message: &str) {
    let waker = futures::task::noop_waker();
    let mut context = Context::from_waker(&waker);
    assert!(
        matches!(future.as_mut().poll(&mut context), Poll::Pending),
        "{message}"
    );
}

fn raise(signal: &str) {
    let status = std::process::Command::new("kill")
        .args(["-s", signal, &std::process::id().to_string()])
        .status()
        .expect("spawn kill");
    assert!(status.success(), "kill -s {signal} failed");
}

#[tokio::test]
async fn shutdown_signal_resolves_on_sigint_and_sigterm() {
    // Safety nets: install handlers for both signals up front so a raised
    // signal cannot fall through to the default terminating disposition.
    let _sigint_net = signal(SignalKind::interrupt()).expect("install SIGINT net");
    let mut sigterm_net = signal(SignalKind::terminate()).expect("install SIGTERM net");

    // Reproduction of the original bug: a Ctrl+C-only future ignores SIGTERM.
    {
        let mut ctrl_c = Box::pin(tokio::signal::ctrl_c());
        assert_pending(ctrl_c.as_mut(), "ctrl_c() resolved before any signal");
        raise("TERM");
        sigterm_net.recv().await;
        assert_pending(
            ctrl_c.as_mut(),
            "ctrl_c() resolved on SIGTERM; it must only react to SIGINT",
        );
    }

    // The fix: shutdown_signal() resolves on SIGTERM.
    {
        let mut future = Box::pin(wa_rs::shutdown_signal());
        assert_pending(
            future.as_mut(),
            "shutdown_signal() resolved before any signal",
        );
        raise("TERM");
        tokio::time::timeout(Duration::from_secs(5), future)
            .await
            .expect("shutdown_signal() must resolve on SIGTERM");
    }

    // The fix keeps the original behaviour: it still resolves on SIGINT.
    {
        let mut future = Box::pin(wa_rs::shutdown_signal());
        assert_pending(
            future.as_mut(),
            "shutdown_signal() resolved before any signal",
        );
        raise("INT");
        tokio::time::timeout(Duration::from_secs(5), future)
            .await
            .expect("shutdown_signal() must resolve on SIGINT");
    }
}
