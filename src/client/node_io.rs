//! Node filters and one-shot waiters for protocol stanzas.

use std::future::Future;
use std::pin::Pin;
use std::sync::atomic::{AtomicU64, AtomicUsize, Ordering};
use std::sync::{Arc, Mutex, Weak};
use std::task::{Context, Poll};

use futures::channel::oneshot;
use wacore_binary::{Jid, NodeRef};

#[derive(Debug, Clone)]
pub struct NodeFilter {
    tag: String,
    attrs: Vec<(String, Option<String>)>,
}

impl NodeFilter {
    pub fn tag(tag: impl Into<String>) -> Self {
        Self {
            tag: tag.into(),
            attrs: Vec::new(),
        }
    }

    pub fn attr(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {
        self.attrs.push((key.into(), Some(value.into())));
        self
    }

    pub fn without_attr(mut self, key: impl Into<String>) -> Self {
        self.attrs.push((key.into(), None));
        self
    }

    pub fn from_jid(self, jid: &Jid) -> Self {
        self.attr("from", jid.to_string())
    }

    /// Whether this filter matches one decoded node.
    pub fn matches(&self, node: &NodeRef<'_>) -> bool {
        node.tag == self.tag.as_str()
            && self.attrs.iter().all(|(key, value)| match value {
                Some(value) => node
                    .get_attr(key.as_str())
                    .is_some_and(|attr| attr.to_string() == *value),
                None => node.get_attr(key.as_str()).is_none(),
            })
    }
}

struct NodeWaiterEntry<T> {
    id: u64,
    filter: NodeFilter,
    tx: oneshot::Sender<Arc<T>>,
}

type NodeWaiterEntries<T> = Mutex<Vec<NodeWaiterEntry<T>>>;

#[must_use = "dropping the waiter cancels its subscription"]
pub struct NodeWaiter<T> {
    rx: oneshot::Receiver<Arc<T>>,
    entries: Weak<NodeWaiterEntries<T>>,
    count: Arc<AtomicUsize>,
    id: u64,
}

impl<T> NodeWaiter<T> {
    pub fn try_recv(&mut self) -> Result<Option<Arc<T>>, oneshot::Canceled> {
        self.rx.try_recv()
    }
}

impl<T> Future for NodeWaiter<T> {
    type Output = Result<Arc<T>, oneshot::Canceled>;

    fn poll(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
        Pin::new(&mut self.rx).poll(cx)
    }
}

impl<T> Drop for NodeWaiter<T> {
    fn drop(&mut self) {
        if let Some(entries) = self.entries.upgrade() {
            let mut entries = entries.lock().unwrap_or_else(|p| p.into_inner());
            if let Some(index) = entries.iter().position(|entry| entry.id == self.id) {
                entries.swap_remove(index);
                self.count.fetch_sub(1, Ordering::Release);
            }
        }
    }
}

/// Registry backing `NodeWaiter`s. Kept independent of `Client` so filters and
/// cancellation semantics are testable without a live connection.
pub struct NodeWaiterRegistry<T> {
    entries: Arc<NodeWaiterEntries<T>>,
    count: Arc<AtomicUsize>,
}

impl<T> Default for NodeWaiterRegistry<T> {
    fn default() -> Self {
        Self::new()
    }
}

impl<T> NodeWaiterRegistry<T> {
    pub fn new() -> Self {
        Self {
            entries: Arc::new(Mutex::new(Vec::new())),
            count: Arc::new(AtomicUsize::new(0)),
        }
    }

    pub fn active(&self) -> usize {
        self.count.load(Ordering::Acquire)
    }

    pub fn wait(&self, filter: NodeFilter) -> NodeWaiter<T> {
        static NEXT_ID: AtomicU64 = AtomicU64::new(0);
        let id = NEXT_ID.fetch_add(1, Ordering::Relaxed);
        let (tx, rx) = oneshot::channel();
        let mut entries = self.entries.lock().unwrap_or_else(|p| p.into_inner());
        entries.push(NodeWaiterEntry { id, filter, tx });
        self.count.fetch_add(1, Ordering::Release);
        NodeWaiter {
            rx,
            entries: Arc::downgrade(&self.entries),
            count: self.count.clone(),
            id,
        }
    }

    /// Resolve every waiter whose filter matches `node`.
    pub fn resolve(&self, node: &Arc<T>, nr: &NodeRef<'_>) {
        let mut waiters = self.entries.lock().unwrap_or_else(|p| p.into_inner());
        let mut ready = Vec::new();
        let mut i = 0;
        while i < waiters.len() {
            if waiters[i].filter.matches(nr) {
                let waiter = waiters.swap_remove(i);
                self.count.fetch_sub(1, Ordering::Release);
                ready.push(waiter.tx);
            } else {
                i += 1;
            }
        }
        drop(waiters);
        for tx in ready {
            let _ = tx.send(Arc::clone(node));
        }
    }
}
