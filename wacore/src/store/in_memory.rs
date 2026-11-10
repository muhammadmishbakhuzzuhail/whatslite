//! In-memory message-secret storage.
//!
//! This is deliberately a msg-secret-only partial backend: it implements
//! [`MsgSecretStore`] and nothing else, so it does not satisfy [`Backend`]
//! (which still requires `SignalStore`/`AppSyncStore`/`ProtocolStore`/
//! `DeviceStore`). The full in-memory backend — including the 8192-row bounded
//! eviction upstream uses to cap wasm32 linear memory — is a later task; only
//! deadline-based retention via `delete_expired_msg_secrets` is ported here.

use std::collections::HashMap;
use std::collections::hash_map::Entry;
use std::sync::Arc;

use async_lock::Mutex;
use async_trait::async_trait;

use crate::store::error::Result;
use crate::store::traits::{
    MessageSecret, MessageSecretBytes, MsgSecretEntry, MsgSecretStore, StoredMessageSecret,
    merge_msg_secret_expiry, merge_msg_secret_message_ts,
};

/// Stored msg-secret value: `(secret_bytes, expires_at_secs, message_ts_secs)`.
type MsgSecretRow = (MessageSecretBytes, i64, i64);

/// `chat -> sender -> msg_id -> row`. Nesting lets lookups borrow `&str` keys
/// without allocating an owned tuple; a flat map of `Arc<str>` keys cannot.
type MsgSecretMap = HashMap<Arc<str>, HashMap<Arc<str>, HashMap<Arc<str>, MsgSecretRow>>>;

#[derive(Default)]
struct InMemoryState {
    /// `expires_at = 0` means never expire; `message_ts = 0` means the parent
    /// event time is unknown.
    msg_secrets: MsgSecretMap,
}

/// In-memory [`MsgSecretStore`] backend.
///
/// Intended for tests and as a reference for the msg-secret subset. All data
/// lives in RAM behind a single [`async_lock::Mutex`] and is lost on drop.
/// See the module docs for what is intentionally not implemented yet.
#[derive(Default)]
pub struct InMemoryBackend {
    state: Mutex<InMemoryState>,
}

impl InMemoryBackend {
    pub fn new() -> Self {
        Self::default()
    }
}

#[async_trait]
impl MsgSecretStore for InMemoryBackend {
    async fn put_msg_secrets(&self, entries: Vec<MsgSecretEntry>) -> Result<usize> {
        let stored = entries.len();
        let mut state = self.state.lock().await;
        for entry in entries {
            let by_sender = state.msg_secrets.entry(entry.chat).or_default();
            let by_id = by_sender.entry(entry.sender).or_default();
            match by_id.entry(entry.msg_id) {
                Entry::Occupied(mut occupied) => {
                    let (secret, expires_at, message_ts) = occupied.get_mut();
                    *secret = entry.secret;
                    *expires_at = merge_msg_secret_expiry(*expires_at, entry.expires_at);
                    *message_ts = merge_msg_secret_message_ts(*message_ts, entry.message_ts);
                }
                Entry::Vacant(vacant) => {
                    vacant.insert((entry.secret, entry.expires_at, entry.message_ts));
                }
            }
        }
        Ok(stored)
    }

    async fn get_stored_msg_secret(
        &self,
        chat: &str,
        sender: &str,
        msg_id: &str,
    ) -> Result<Option<StoredMessageSecret>> {
        Ok(self
            .state
            .lock()
            .await
            .msg_secrets
            .get(chat)
            .and_then(|by_sender| by_sender.get(sender))
            .and_then(|by_id| by_id.get(msg_id))
            .map(|(secret, _, message_ts)| {
                StoredMessageSecret::new(MessageSecret::from_bytes(*secret), Some(*message_ts))
            }))
    }

    async fn delete_expired_msg_secrets(&self, cutoff_timestamp: i64) -> Result<u32> {
        let mut state = self.state.lock().await;
        let mut removed = 0u32;
        // Keep rows with no deadline (0 = never) or a deadline still in the future.
        state.msg_secrets.retain(|_, by_sender| {
            by_sender.retain(|_, by_id| {
                by_id.retain(|_, (_, expires_at, _)| {
                    let keep = *expires_at == 0 || *expires_at > cutoff_timestamp;
                    if !keep {
                        removed += 1;
                    }
                    keep
                });
                !by_id.is_empty()
            });
            !by_sender.is_empty()
        });
        Ok(removed)
    }
}
