//! Admission and cancellation for group-notification effects.

use super::Client;
use std::future::{Future, poll_fn};
use std::pin::pin;
use std::sync::atomic::Ordering;
use std::task::Poll;
use wacore::runtime::{ShutdownSignal, wait_for_shutdown};

/// One connection's permission to produce a notification effect.
///
/// Effects derived from a `<notification>` are only worth producing while the
/// connection that delivered it is still the current one: a reconnect retires
/// the generation, and a shutdown retires everything. Both are checked here
/// rather than at each effect, so the rule is stated once.
#[allow(
    dead_code,
    reason = "the notification path that wraps effects in a scope is wired in a follow-up"
)]
#[derive(Clone, Copy)]
pub(crate) struct NotificationScope<'a> {
    generation: u64,
    shutdown: &'a ShutdownSignal,
}

#[allow(
    dead_code,
    reason = "the notification path that wraps effects in a scope is wired in a follow-up"
)]
impl<'a> NotificationScope<'a> {
    pub(crate) fn new(generation: u64, shutdown: &'a ShutdownSignal) -> Self {
        Self {
            generation,
            shutdown,
        }
    }

    pub(crate) fn shutdown(self) -> ShutdownSignal {
        self.shutdown.clone()
    }

    pub(crate) fn is_current(self, client: &Client) -> bool {
        !self.shutdown.is_fired()
            && client.connection_generation.load(Ordering::Acquire) == self.generation
    }

    /// Cancel admission waits, reads and host capture on connection retirement.
    /// Already-started mutations must finish their storage/cache commit without
    /// this wrapper; dropping them can leave routing inconsistent with storage.
    pub(crate) async fn run<F: Future>(self, client: &Client, future: F) -> Option<F::Output> {
        let mut future = pin!(future);
        let mut shutdown = pin!(wait_for_shutdown(self.shutdown));
        poll_fn(|cx| {
            if !self.is_current(client) || shutdown.as_mut().poll(cx).is_ready() {
                Poll::Ready(None)
            } else {
                future.as_mut().poll(cx).map(Some)
            }
        })
        .await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use wacore::runtime::ShutdownNotifier;

    fn generation(client: &Client) -> u64 {
        client.connection_generation.load(Ordering::Acquire)
    }

    #[tokio::test]
    async fn a_scope_is_current_while_its_generation_matches() {
        let client = crate::test_utils::create_test_client().await;
        let signal = ShutdownSignal::never();
        let scope = NotificationScope::new(generation(&client), &signal);

        assert!(scope.is_current(&client));
    }

    #[tokio::test]
    async fn a_scope_stops_being_current_when_the_generation_advances() {
        let client = crate::test_utils::create_test_client().await;
        let signal = ShutdownSignal::never();
        let scope = NotificationScope::new(generation(&client), &signal);

        client.connection_generation.fetch_add(1, Ordering::SeqCst);

        assert!(!scope.is_current(&client));
    }

    #[tokio::test]
    async fn a_scope_stops_being_current_when_shutdown_fires() {
        let client = crate::test_utils::create_test_client().await;
        let notifier = ShutdownNotifier::new();
        let signal = notifier.subscribe();
        let scope = NotificationScope::new(generation(&client), &signal);

        notifier.notify();

        assert!(!scope.is_current(&client));
    }

    #[tokio::test]
    async fn run_returns_the_output_while_the_connection_is_current() {
        let client = crate::test_utils::create_test_client().await;
        let signal = ShutdownSignal::never();
        let scope = NotificationScope::new(generation(&client), &signal);

        assert_eq!(scope.run(&client, async { 7 }).await, Some(7));
    }

    #[tokio::test]
    async fn run_gives_up_when_the_connection_is_retired() {
        let client = crate::test_utils::create_test_client().await;
        let signal = ShutdownSignal::never();
        let scope = NotificationScope::new(generation(&client), &signal);

        client.connection_generation.fetch_add(1, Ordering::SeqCst);

        assert_eq!(scope.run(&client, async { 7 }).await, None);
    }

    #[tokio::test]
    async fn run_gives_up_when_shutdown_fires_mid_flight() {
        let client = crate::test_utils::create_test_client().await;
        let notifier = Arc::new(ShutdownNotifier::new());
        let signal = notifier.subscribe();
        let scope = NotificationScope::new(generation(&client), &signal);

        let run = scope.run(&client, std::future::pending::<()>());
        let fire = async move {
            tokio::task::yield_now().await;
            notifier.notify();
        };
        let (outcome, ()) = tokio::join!(run, fire);

        assert_eq!(outcome, None);
    }

    #[tokio::test]
    async fn a_scope_hands_out_the_signal_it_was_given() {
        let notifier = ShutdownNotifier::new();
        let signal = notifier.subscribe();
        let scope = NotificationScope::new(0, &signal);

        let handed_out = scope.shutdown();
        notifier.notify();

        assert!(handed_out.is_fired());
    }
}
