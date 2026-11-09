//! In-memory buffer for message secrets awaiting a durable store write.

use std::collections::HashMap;

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
struct Key {
    chat: String,
    sender: String,
    id: String,
}

/// One buffered message secret.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BufferedSecret {
    pub secret: Vec<u8>,
    pub timestamp_ms: i64,
}

/// Outcome of a flush attempt.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SecretFlushReport {
    pub flushed: usize,
    pub failed: usize,
    pub remaining: usize,
}

/// A bounded, deduplicating buffer keyed by `(chat, sender, id)`.
#[derive(Debug, Default)]
pub struct MsgSecretBuffer {
    entries: HashMap<Key, BufferedSecret>,
}

impl MsgSecretBuffer {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn len(&self) -> usize {
        self.entries.len()
    }

    pub fn is_empty(&self) -> bool {
        self.entries.is_empty()
    }

    pub fn insert(
        &mut self,
        chat: impl Into<String>,
        sender: impl Into<String>,
        id: impl Into<String>,
        secret: Vec<u8>,
        timestamp_ms: i64,
    ) -> bool {
        let key = Key {
            chat: chat.into(),
            sender: sender.into(),
            id: id.into(),
        };
        self.entries
            .insert(
                key,
                BufferedSecret {
                    secret,
                    timestamp_ms,
                },
            )
            .is_none()
    }

    pub fn get(&self, chat: &str, sender: &str, id: &str) -> Option<&BufferedSecret> {
        self.entries.get(&Key {
            chat: chat.to_string(),
            sender: sender.to_string(),
            id: id.to_string(),
        })
    }

    pub fn take(&mut self, chat: &str, sender: &str, id: &str) -> Option<BufferedSecret> {
        self.entries.remove(&Key {
            chat: chat.to_string(),
            sender: sender.to_string(),
            id: id.to_string(),
        })
    }

    pub fn drain_into(
        &mut self,
        mut sink: impl FnMut(BufferedSecret) -> bool,
    ) -> SecretFlushReport {
        let total = self.entries.len();
        let mut flushed = 0;
        let mut failed = 0;
        self.entries.retain(|_, value| {
            if sink(value.clone()) {
                flushed += 1;
                false
            } else {
                failed += 1;
                true
            }
        });
        SecretFlushReport {
            flushed,
            failed,
            remaining: total - flushed,
        }
    }
}
