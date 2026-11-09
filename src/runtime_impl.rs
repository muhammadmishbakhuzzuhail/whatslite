//! Tokio-backed [`Runtime`] used by host-facing helpers and tests.

use std::future::Future;
use std::pin::Pin;
use std::time::Duration;

use wacore::runtime::{AbortHandle, Runtime};

#[derive(Debug, Default, Clone, Copy)]
pub struct TokioRuntime;

impl Runtime for TokioRuntime {
    fn spawn(&self, future: Pin<Box<dyn Future<Output = ()> + Send + 'static>>) -> AbortHandle {
        let handle = tokio::spawn(future);
        AbortHandle::new(move || handle.abort())
    }

    fn sleep(&self, duration: Duration) -> Pin<Box<dyn Future<Output = ()> + Send>> {
        Box::pin(tokio::time::sleep(duration))
    }

    fn spawn_blocking(
        &self,
        f: Box<dyn FnOnce() + Send + 'static>,
    ) -> Pin<Box<dyn Future<Output = ()> + Send>> {
        Box::pin(async move {
            let _ = tokio::task::spawn_blocking(f).await;
        })
    }

    fn yield_now(&self) -> Option<Pin<Box<dyn Future<Output = ()> + Send>>> {
        Some(Box::pin(tokio::task::yield_now()))
    }

    fn yield_frequency(&self) -> u32 {
        1
    }
}
