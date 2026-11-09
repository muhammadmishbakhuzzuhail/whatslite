//! Ordered per-chat inbound queue. The production pipeline keeps the full
//! upstream queue budgets; this Phase 2 surface provides the ordering and
//! bounded-admission primitives used by tests and hosts.

use std::collections::{HashMap, VecDeque};

/// One queued inbound item, tagged with its chat lane.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct QueuedInbound<T> {
    pub chat: String,
    pub item: T,
}

/// Per-chat FIFO queues with a global capacity bound.
#[derive(Debug)]
pub struct InboundQueue<T> {
    lanes: HashMap<String, VecDeque<T>>,
    capacity: usize,
    len: usize,
}

impl<T> InboundQueue<T> {
    pub fn new(capacity: usize) -> Self {
        Self {
            lanes: HashMap::new(),
            capacity: capacity.max(1),
            len: 0,
        }
    }

    pub fn len(&self) -> usize {
        self.len
    }

    pub fn is_empty(&self) -> bool {
        self.len == 0
    }

    pub fn capacity(&self) -> usize {
        self.capacity
    }

    /// Enqueue for one chat. Returns false when the global bound is reached.
    pub fn push(&mut self, chat: impl Into<String>, item: T) -> bool {
        if self.len >= self.capacity {
            return false;
        }
        self.lanes.entry(chat.into()).or_default().push_back(item);
        self.len += 1;
        true
    }

    /// Pop the oldest item for a chat.
    pub fn pop(&mut self, chat: &str) -> Option<T> {
        let lane = self.lanes.get_mut(chat)?;
        let item = lane.pop_front();
        if lane.is_empty() {
            self.lanes.remove(chat);
        }
        if item.is_some() {
            self.len -= 1;
        }
        item
    }

    /// Pop the globally oldest item in insertion order across lanes.
    pub fn pop_any(&mut self) -> Option<QueuedInbound<T>> {
        let chat = self.lanes.keys().next()?.clone();
        let item = self.pop(&chat)?;
        Some(QueuedInbound { chat, item })
    }

    pub fn lane_len(&self, chat: &str) -> usize {
        self.lanes.get(chat).map_or(0, VecDeque::len)
    }
}
