//! Plugin-defined event routing.
//!
//! Publication authority is bound to a plugin id; consumers subscribe to
//! exact `(plugin_id, topic)` routes through a [`PluginEventRouter`].

use std::collections::{HashMap, HashSet};
use std::fmt;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::{Arc, Mutex};

use async_channel::{Receiver, Sender, TryRecvError, TrySendError};
use bytes::Bytes;

use super::{PluginResourceError, PluginState, valid_plugin_id};

const MAX_ENDPOINT_CAPACITY: usize = 65_536;
const MAX_ENDPOINT_SELECTORS: usize = 1_024;

/// Encoding of a custom plugin event payload.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginEventPayloadEncoding {
    Json,
    Binary,
}

impl PluginEventPayloadEncoding {
    pub const fn identifier(self) -> &'static str {
        match self {
            Self::Json => "json",
            Self::Binary => "binary",
        }
    }
}

/// Validated second-level topic within one plugin namespace.
#[derive(Clone, PartialEq, Eq, Hash)]
pub struct PluginEventTopic(Arc<str>);

impl PluginEventTopic {
    pub fn new(topic: impl Into<String>) -> Result<Self, PluginEventRouteError> {
        let topic = topic.into();
        if !valid_plugin_id(&topic) {
            return Err(PluginEventRouteError::InvalidTopic { topic });
        }
        Ok(Self(Arc::from(topic)))
    }

    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl fmt::Debug for PluginEventTopic {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_tuple("PluginEventTopic")
            .field(&self.0)
            .finish()
    }
}

impl fmt::Display for PluginEventTopic {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(&self.0)
    }
}

/// Exact `(plugin_id, topic)` route selected by one endpoint.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct PluginEventSelector {
    route: RouteKey,
}

impl PluginEventSelector {
    pub fn new(
        plugin_id: impl Into<String>,
        topic: PluginEventTopic,
    ) -> Result<Self, PluginEventRouteError> {
        let plugin_id = plugin_id.into();
        if !valid_plugin_id(&plugin_id) {
            return Err(PluginEventRouteError::InvalidPluginId { plugin_id });
        }
        Ok(Self {
            route: RouteKey {
                plugin_id: Arc::from(plugin_id),
                topic,
            },
        })
    }

    pub fn plugin_id(&self) -> &str {
        &self.route.plugin_id
    }

    pub fn topic(&self) -> &PluginEventTopic {
        &self.route.topic
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
struct RouteKey {
    plugin_id: Arc<str>,
    topic: PluginEventTopic,
}

/// Routed event shared by every matching endpoint without copying its payload.
#[derive(Debug, Clone)]
#[non_exhaustive]
pub struct PluginEventEnvelope {
    pub plugin_id: Arc<str>,
    pub topic: PluginEventTopic,
    pub schema_version: u32,
    pub payload_encoding: PluginEventPayloadEncoding,
    pub payload: Bytes,
    pub connection_generation: u64,
    /// Monotonic sequence for this route while it has at least one subscriber.
    pub sequence: u64,
}

/// Behavior when one endpoint cannot keep up with publishers.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginEventOverflow {
    DropNewest,
    DropOldest,
}

/// Required queue policy for one independent consumer endpoint.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PluginEventEndpointConfig {
    capacity: usize,
    overflow: PluginEventOverflow,
}

impl PluginEventEndpointConfig {
    pub const fn new(capacity: usize, overflow: PluginEventOverflow) -> Self {
        Self { capacity, overflow }
    }

    pub const fn capacity(self) -> usize {
        self.capacity
    }

    pub const fn overflow(self) -> PluginEventOverflow {
        self.overflow
    }
}

/// Syntactic route validation failure.
#[derive(Debug, thiserror::Error, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginEventRouteError {
    #[error("invalid plugin id `{plugin_id}`")]
    InvalidPluginId { plugin_id: String },
    #[error("invalid plugin event topic `{topic}`")]
    InvalidTopic { topic: String },
}

/// Endpoint registration failure.
#[derive(Debug, thiserror::Error, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginEventSubscribeError {
    #[error("{0}")]
    Resource(#[from] PluginResourceError),
    #[error("at least one plugin event selector is required")]
    EmptySelectors,
    #[error("plugin event endpoint selector count exceeds the maximum of {max}")]
    TooManySelectors { max: usize },
    #[error("plugin event endpoint capacity {capacity} is outside 1..={max}")]
    InvalidCapacity { capacity: usize, max: usize },
    #[error("plugin `{plugin_id}` is not registered as a custom-event publisher")]
    UnknownPublisher { plugin_id: String },
    #[error("plugin event endpoint identifiers are exhausted")]
    EndpointIdsExhausted,
    #[error("the plugin event router is closed")]
    Closed,
}

/// Custom event publication failure.
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginEventPublishError {
    #[error("{0}")]
    Resource(#[from] PluginResourceError),
    #[error("plugin event schema version must be greater than zero")]
    InvalidSchemaVersion,
    #[error("the plugin event router is closed")]
    Closed,
    #[error("the plugin event sequence is exhausted")]
    SequenceExhausted,
}

/// Result of one non-blocking fan-out attempt.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct PluginEventPublishReport {
    pub matched: u64,
    pub enqueued: u64,
    pub dropped: u64,
}

/// Receive failure for one endpoint.
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginEventReceiveError {
    #[error("the plugin event endpoint is closed")]
    Closed,
}

/// Non-blocking receive failure for one endpoint.
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginEventTryReceiveError {
    #[error("no plugin event is queued")]
    Empty,
    #[error("the plugin event endpoint is closed")]
    Closed,
}

/// Cumulative counters for one registered publisher.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
#[non_exhaustive]
pub struct PluginEventPublisherStats {
    pub state: PluginState,
    pub published: u64,
    pub publish_failures: u64,
    pub matched: u64,
    pub enqueued: u64,
    pub delivered: u64,
    pub dropped: u64,
    pub closed: u64,
}

/// Aggregate router counters and current queue occupancy.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
#[non_exhaustive]
pub struct PluginEventRouterStats {
    pub registered_publishers: u64,
    pub active_routes: u64,
    pub active_endpoints: u64,
    pub endpoint_capacity: u64,
    pub queued_events: u64,
    pub queued_payload_bytes: u64,
    pub published: u64,
    pub publish_failures: u64,
    pub matched: u64,
    pub enqueued: u64,
    pub delivered: u64,
    pub dropped: u64,
    pub closed: u64,
}

#[derive(Default)]
struct RouterCounters {
    published: AtomicU64,
    publish_failures: AtomicU64,
    matched: AtomicU64,
    enqueued: AtomicU64,
    delivered: AtomicU64,
    dropped: AtomicU64,
    closed: AtomicU64,
    queued_events: AtomicU64,
    queued_payload_bytes: AtomicU64,
}

impl RouterCounters {
    fn snapshot(&self) -> PluginEventRouterStats {
        PluginEventRouterStats {
            queued_events: self.queued_events.load(Ordering::Relaxed),
            queued_payload_bytes: self.queued_payload_bytes.load(Ordering::Relaxed),
            published: self.published.load(Ordering::Relaxed),
            publish_failures: self.publish_failures.load(Ordering::Relaxed),
            matched: self.matched.load(Ordering::Relaxed),
            enqueued: self.enqueued.load(Ordering::Relaxed),
            delivered: self.delivered.load(Ordering::Relaxed),
            dropped: self.dropped.load(Ordering::Relaxed),
            closed: self.closed.load(Ordering::Relaxed),
            ..PluginEventRouterStats::default()
        }
    }
}

struct Endpoint {
    selectors: Vec<RouteKey>,
    sender: Sender<Arc<PluginEventEnvelope>>,
    receiver: Receiver<Arc<PluginEventEnvelope>>,
    capacity: usize,
    overflow: PluginEventOverflow,
}

struct RouterInner {
    publishers: Mutex<HashSet<Arc<str>>>,
    routes: Mutex<HashMap<RouteKey, Vec<u64>>>,
    endpoints: Mutex<HashMap<u64, Endpoint>>,
    sequences: Mutex<HashMap<RouteKey, u64>>,
    next_endpoint_id: AtomicU64,
    counters: RouterCounters,
    closed: AtomicBool,
}

/// Read-only subscription boundary for native consumers.
#[derive(Clone)]
pub struct PluginEventRouter {
    inner: Arc<RouterInner>,
}

impl Default for PluginEventRouter {
    fn default() -> Self {
        Self::new(std::iter::empty())
    }
}

impl PluginEventRouter {
    pub fn new(plugin_ids: impl IntoIterator<Item = String>) -> Self {
        Self {
            inner: Arc::new(RouterInner {
                publishers: Mutex::new(plugin_ids.into_iter().map(Arc::from).collect()),
                routes: Mutex::new(HashMap::new()),
                endpoints: Mutex::new(HashMap::new()),
                sequences: Mutex::new(HashMap::new()),
                next_endpoint_id: AtomicU64::new(1),
                counters: RouterCounters::default(),
                closed: AtomicBool::new(false),
            }),
        }
    }

    pub fn has_subscribers(&self, selector: &PluginEventSelector) -> bool {
        if self.inner.closed.load(Ordering::Acquire) {
            return false;
        }
        self.inner
            .routes
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner())
            .get(&selector.route)
            .is_some_and(|endpoints| !endpoints.is_empty())
    }

    /// Register a plugin id as a custom-event publisher.
    pub fn register_publisher(&self, plugin_id: impl Into<String>) -> bool {
        if self.inner.closed.load(Ordering::Acquire) {
            return false;
        }
        self.inner
            .publishers
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner())
            .insert(Arc::from(plugin_id.into()))
    }

    /// Cumulative counters for one registered publisher.
    pub fn publisher_stats(&self, plugin_id: &str) -> Option<PluginEventPublisherStats> {
        let publishers = self
            .inner
            .publishers
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        if !publishers.contains(plugin_id) {
            return None;
        }
        drop(publishers);
        let stats = self.inner.counters.snapshot();
        Some(PluginEventPublisherStats {
            state: PluginState::Active,
            published: stats.published,
            publish_failures: stats.publish_failures,
            matched: stats.matched,
            enqueued: stats.enqueued,
            delivered: stats.delivered,
            dropped: stats.dropped,
            closed: stats.closed,
        })
    }

    /// Aggregate counters and current queue occupancy.
    pub fn stats(&self) -> PluginEventRouterStats {
        let mut snapshot = self.inner.counters.snapshot();
        let endpoints = self
            .inner
            .endpoints
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        let routes = self
            .inner
            .routes
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        snapshot.registered_publishers = self
            .inner
            .publishers
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner())
            .len() as u64;
        snapshot.active_routes = routes.len() as u64;
        snapshot.active_endpoints = endpoints.len() as u64;
        snapshot.endpoint_capacity = endpoints.values().fold(0u64, |total, endpoint| {
            total.saturating_add(endpoint.capacity as u64)
        });
        snapshot
    }

    pub fn subscribe(
        &self,
        selectors: impl IntoIterator<Item = PluginEventSelector>,
        config: PluginEventEndpointConfig,
    ) -> Result<PluginEventSubscription, PluginEventSubscribeError> {
        if config.capacity == 0 || config.capacity > MAX_ENDPOINT_CAPACITY {
            return Err(PluginEventSubscribeError::InvalidCapacity {
                capacity: config.capacity,
                max: MAX_ENDPOINT_CAPACITY,
            });
        }
        if self.inner.closed.load(Ordering::Acquire) {
            return Err(PluginEventSubscribeError::Closed);
        }

        let mut seen = HashSet::new();
        let mut unique_selectors = Vec::new();
        for selector in selectors {
            if !seen.insert(selector.route.clone()) {
                continue;
            }
            if unique_selectors.len() == MAX_ENDPOINT_SELECTORS {
                return Err(PluginEventSubscribeError::TooManySelectors {
                    max: MAX_ENDPOINT_SELECTORS,
                });
            }
            let publishers = self
                .inner
                .publishers
                .lock()
                .unwrap_or_else(|poisoned| poisoned.into_inner());
            if !publishers.contains(&selector.route.plugin_id) {
                return Err(PluginEventSubscribeError::UnknownPublisher {
                    plugin_id: selector.route.plugin_id.to_string(),
                });
            }
            unique_selectors.push(selector);
        }
        if unique_selectors.is_empty() {
            return Err(PluginEventSubscribeError::EmptySelectors);
        }

        let id = self.inner.next_endpoint_id.fetch_add(1, Ordering::Relaxed);
        if id == u64::MAX {
            return Err(PluginEventSubscribeError::EndpointIdsExhausted);
        }
        let (sender, receiver) = async_channel::bounded(config.capacity);
        {
            let mut endpoints = self
                .inner
                .endpoints
                .lock()
                .unwrap_or_else(|poisoned| poisoned.into_inner());
            endpoints.insert(
                id,
                Endpoint {
                    selectors: unique_selectors.iter().map(|s| s.route.clone()).collect(),
                    sender,
                    receiver: receiver.clone(),
                    capacity: config.capacity,
                    overflow: config.overflow,
                },
            );
        }
        {
            let mut routes = self
                .inner
                .routes
                .lock()
                .unwrap_or_else(|poisoned| poisoned.into_inner());
            for selector in &unique_selectors {
                routes.entry(selector.route.clone()).or_default().push(id);
            }
        }

        Ok(PluginEventSubscription {
            id,
            selectors: unique_selectors,
            receiver,
            router: self.clone(),
        })
    }

    /// Stop accepting publications and close every endpoint.
    pub fn close(&self) {
        self.inner.closed.store(true, Ordering::Release);
        let endpoints = self
            .inner
            .endpoints
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        for endpoint in endpoints.values() {
            endpoint.sender.close();
        }
    }

    fn publish(
        &self,
        plugin_id: &str,
        topic: &PluginEventTopic,
        schema_version: u32,
        payload_encoding: PluginEventPayloadEncoding,
        payload: Bytes,
        connection_generation: u64,
    ) -> Result<PluginEventPublishReport, PluginEventPublishError> {
        if schema_version == 0 {
            self.inner
                .counters
                .publish_failures
                .fetch_add(1, Ordering::Relaxed);
            return Err(PluginEventPublishError::InvalidSchemaVersion);
        }
        if self.inner.closed.load(Ordering::Acquire) {
            self.inner
                .counters
                .publish_failures
                .fetch_add(1, Ordering::Relaxed);
            return Err(PluginEventPublishError::Closed);
        }

        let route = RouteKey {
            plugin_id: Arc::from(plugin_id),
            topic: topic.clone(),
        };
        let sequence = {
            let mut sequences = self
                .inner
                .sequences
                .lock()
                .unwrap_or_else(|poisoned| poisoned.into_inner());
            let entry = sequences.entry(route.clone()).or_insert(0);
            *entry = entry
                .checked_add(1)
                .ok_or(PluginEventPublishError::SequenceExhausted)?;
            *entry
        };
        let envelope = Arc::new(PluginEventEnvelope {
            plugin_id: route.plugin_id.clone(),
            topic: topic.clone(),
            schema_version,
            payload_encoding,
            payload,
            connection_generation,
            sequence,
        });

        let endpoint_ids = self
            .inner
            .routes
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner())
            .get(&route)
            .cloned()
            .unwrap_or_default();
        let mut report = PluginEventPublishReport {
            matched: endpoint_ids.len() as u64,
            ..PluginEventPublishReport::default()
        };
        self.inner
            .counters
            .published
            .fetch_add(1, Ordering::Relaxed);
        self.inner
            .counters
            .matched
            .fetch_add(report.matched, Ordering::Relaxed);

        let endpoints = self
            .inner
            .endpoints
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        for endpoint_id in endpoint_ids {
            let Some(endpoint) = endpoints.get(&endpoint_id) else {
                continue;
            };
            if !endpoint.selectors.contains(&route) {
                continue;
            }
            match endpoint.sender.try_send(envelope.clone()) {
                Ok(()) => {
                    report.enqueued += 1;
                    self.inner.counters.enqueued.fetch_add(1, Ordering::Relaxed);
                    self.inner
                        .counters
                        .queued_events
                        .fetch_add(1, Ordering::Relaxed);
                    self.inner
                        .counters
                        .queued_payload_bytes
                        .fetch_add(envelope.payload.len() as u64, Ordering::Relaxed);
                }
                Err(TrySendError::Full(_)) => {
                    if endpoint.overflow == PluginEventOverflow::DropOldest {
                        if let Ok(dropped) = endpoint.receiver.try_recv() {
                            report.dropped += 1;
                            self.inner.counters.dropped.fetch_add(1, Ordering::Relaxed);
                            saturating_sub(&self.inner.counters.queued_events, 1);
                            saturating_sub(
                                &self.inner.counters.queued_payload_bytes,
                                dropped.payload.len() as u64,
                            );
                        }
                        if endpoint.sender.try_send(envelope.clone()).is_ok() {
                            report.enqueued += 1;
                            self.inner.counters.enqueued.fetch_add(1, Ordering::Relaxed);
                            self.inner
                                .counters
                                .queued_events
                                .fetch_add(1, Ordering::Relaxed);
                            self.inner
                                .counters
                                .queued_payload_bytes
                                .fetch_add(envelope.payload.len() as u64, Ordering::Relaxed);
                        } else {
                            report.dropped += 1;
                            self.inner.counters.dropped.fetch_add(1, Ordering::Relaxed);
                        }
                    } else {
                        report.dropped += 1;
                        self.inner.counters.dropped.fetch_add(1, Ordering::Relaxed);
                    }
                }
                Err(TrySendError::Closed(_)) => {
                    self.inner.counters.closed.fetch_add(1, Ordering::Relaxed);
                }
            }
        }
        Ok(report)
    }
}

fn saturating_sub(counter: &AtomicU64, amount: u64) {
    let _ = counter.try_update(Ordering::Relaxed, Ordering::Relaxed, |value| {
        Some(value.saturating_sub(amount))
    });
}

/// One subscribed endpoint.
pub struct PluginEventSubscription {
    id: u64,
    selectors: Vec<PluginEventSelector>,
    receiver: Receiver<Arc<PluginEventEnvelope>>,
    router: PluginEventRouter,
}

impl PluginEventSubscription {
    pub fn id(&self) -> u64 {
        self.id
    }

    pub fn selectors(&self) -> &[PluginEventSelector] {
        &self.selectors
    }

    pub async fn recv(&self) -> Result<Arc<PluginEventEnvelope>, PluginEventReceiveError> {
        let envelope = self
            .receiver
            .recv()
            .await
            .map_err(|_| PluginEventReceiveError::Closed)?;
        self.record_delivery(&envelope);
        Ok(envelope)
    }

    pub fn try_recv(&self) -> Result<Arc<PluginEventEnvelope>, PluginEventTryReceiveError> {
        let envelope = self.receiver.try_recv().map_err(|error| match error {
            TryRecvError::Empty => PluginEventTryReceiveError::Empty,
            TryRecvError::Closed => PluginEventTryReceiveError::Closed,
        })?;
        self.record_delivery(&envelope);
        Ok(envelope)
    }

    fn record_delivery(&self, envelope: &PluginEventEnvelope) {
        self.router
            .inner
            .counters
            .delivered
            .fetch_add(1, Ordering::Relaxed);
        saturating_sub(&self.router.inner.counters.queued_events, 1);
        saturating_sub(
            &self.router.inner.counters.queued_payload_bytes,
            envelope.payload.len() as u64,
        );
    }

    /// Number of currently queued envelopes for this endpoint.
    pub fn queued(&self) -> usize {
        self.receiver.len()
    }
}

impl Drop for PluginEventSubscription {
    fn drop(&mut self) {
        let mut endpoints = self
            .router
            .inner
            .endpoints
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        endpoints.remove(&self.id);
        drop(endpoints);
        let mut routes = self
            .router
            .inner
            .routes
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        for selector in &self.selectors {
            if let Some(ids) = routes.get_mut(&selector.route) {
                ids.retain(|id| *id != self.id);
                if ids.is_empty() {
                    routes.remove(&selector.route);
                }
            }
        }
    }
}

/// Context-bound custom event capability. A plugin can publish only under its
/// own ID; consumers subscribe through [`PluginEventRouter`].
#[derive(Clone)]
pub struct PluginEvents {
    plugin_id: Arc<str>,
    router: PluginEventRouter,
}

impl PluginEvents {
    pub fn new(plugin_id: impl Into<String>, router: PluginEventRouter) -> Self {
        Self {
            plugin_id: Arc::from(plugin_id.into()),
            router,
        }
    }

    pub fn selector(&self, topic: &PluginEventTopic) -> PluginEventSelector {
        PluginEventSelector {
            route: RouteKey {
                plugin_id: self.plugin_id.clone(),
                topic: topic.clone(),
            },
        }
    }

    pub fn has_subscribers(&self, topic: &PluginEventTopic) -> bool {
        self.router.has_subscribers(&self.selector(topic))
    }

    pub fn stats(&self) -> PluginEventPublisherStats {
        self.router
            .publisher_stats(&self.plugin_id)
            .unwrap_or_default()
    }

    pub fn publish(
        &self,
        topic: &PluginEventTopic,
        schema_version: u32,
        payload_encoding: PluginEventPayloadEncoding,
        payload: impl Into<Bytes>,
    ) -> Result<PluginEventPublishReport, PluginEventPublishError> {
        self.router.publish(
            &self.plugin_id,
            topic,
            schema_version,
            payload_encoding,
            payload.into(),
            0,
        )
    }
}
