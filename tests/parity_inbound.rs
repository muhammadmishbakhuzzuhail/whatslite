//! Phase 2 inbound contract: per-chat ordering, bounded admission, flush
//! draining and secret-buffer deduplication.

use std::sync::Arc;
use std::time::Duration;

use wa_rs::flush_scope::{DrainOutcome, FlushScope};
use wa_rs::inbound_queue::InboundQueue;
use wa_rs::msg_secret_buffer::MsgSecretBuffer;
use wa_rs::runtime_impl::TokioRuntime;

#[test]
fn inbound_queue_preserves_per_chat_order_and_capacity() {
    let mut queue = InboundQueue::new(3);
    assert!(queue.push("a", 1));
    assert!(queue.push("b", 2));
    assert!(queue.push("a", 3));
    assert!(!queue.push("c", 4), "global bound");
    assert_eq!(queue.len(), 3);
    assert_eq!(queue.pop("a"), Some(1));
    assert_eq!(queue.pop("a"), Some(3));
    assert_eq!(queue.pop("a"), None);
    assert_eq!(queue.lane_len("b"), 1);
}

#[tokio::test]
async fn flush_scope_tracks_guards_and_drains() {
    let scope = Arc::new(FlushScope::new());
    let guard = scope.try_track().expect("tracked");
    assert_eq!(scope.pending(), 1);
    scope.close();
    assert!(scope.is_closed());
    assert!(scope.try_track().is_none());
    drop(guard);
    assert_eq!(scope.pending(), 0);
    assert_eq!(
        scope.flush(&TokioRuntime, Duration::from_secs(1)).await,
        DrainOutcome::Completed
    );
}

#[test]
fn secret_buffer_deduplicates_and_drains() {
    let mut buffer = MsgSecretBuffer::new();
    assert!(buffer.insert("chat", "sender", "id", vec![1, 2, 3], 10));
    assert!(!buffer.insert("chat", "sender", "id", vec![4, 5, 6], 20));
    assert_eq!(buffer.len(), 1);
    assert_eq!(
        buffer.get("chat", "sender", "id").unwrap().secret,
        vec![4, 5, 6]
    );

    let report = buffer.drain_into(|secret| secret.timestamp_ms == 20);
    assert_eq!(report.flushed, 1);
    assert_eq!(report.remaining, 0);
    assert!(buffer.is_empty());
}
