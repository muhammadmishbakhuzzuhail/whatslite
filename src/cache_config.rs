//! Cache sizing/retention configuration and optional shared stores.

use std::sync::Arc;
use std::time::Duration;

/// Per-cache entry timeout and capacity.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct CacheEntryConfig {
    pub timeout: Option<Duration>,
    pub capacity: u64,
}

impl CacheEntryConfig {
    pub fn new(timeout: Option<Duration>, capacity: u64) -> Self {
        Self { timeout, capacity }
    }
}

/// Optional shared cache backends. `None` keeps the in-process default.
#[derive(Clone, Default)]
pub struct CacheStores {
    pub group_cache: Option<Arc<dyn CacheStore>>,
    pub device_registry_cache: Option<Arc<dyn CacheStore>>,
    pub lid_pn_cache: Option<Arc<dyn CacheStore>>,
}

impl CacheStores {
    pub fn with_group_cache(mut self, store: Arc<dyn CacheStore>) -> Self {
        self.group_cache = Some(store);
        self
    }

    pub fn with_device_registry_cache(mut self, store: Arc<dyn CacheStore>) -> Self {
        self.device_registry_cache = Some(store);
        self
    }

    pub fn with_lid_pn_cache(mut self, store: Arc<dyn CacheStore>) -> Self {
        self.lid_pn_cache = Some(store);
        self
    }

    pub fn all(store: Arc<dyn CacheStore>) -> Self {
        Self {
            group_cache: Some(store.clone()),
            device_registry_cache: Some(store.clone()),
            lid_pn_cache: Some(store),
        }
    }
}

impl std::fmt::Debug for CacheStores {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("CacheStores")
            .field("group_cache", &self.group_cache.is_some())
            .field(
                "device_registry_cache",
                &self.device_registry_cache.is_some(),
            )
            .field("lid_pn_cache", &self.lid_pn_cache.is_some())
            .finish()
    }
}

/// Portable cache backend seam.
pub trait CacheStore: Send + Sync + 'static {
    fn get(&self, key: &str) -> Option<Vec<u8>>;
    fn put(&self, key: &str, value: Vec<u8>);
    fn remove(&self, key: &str);
}

/// Cache freshness of an original-message lookup.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Freshness {
    Fresh,
    Stale,
    Missing,
}

/// Resolves an original message for edits/replies.
pub trait OriginalMessageResolver: Send + Sync + 'static {
    fn resolve(&self, chat: &str, message_id: &str) -> Freshness;
}

/// Aggregate cache configuration.
#[derive(Debug, Clone, Default)]
pub struct CacheConfig {
    pub group: CacheEntryConfig,
    pub device_registry: CacheEntryConfig,
    pub lid_pn: CacheEntryConfig,
    pub recent_messages: CacheEntryConfig,
    pub stores: CacheStores,
}

impl CacheConfig {
    pub fn with_group_cache(mut self, config: CacheEntryConfig) -> Self {
        self.group = config;
        self
    }
    pub fn with_device_registry_cache(mut self, config: CacheEntryConfig) -> Self {
        self.device_registry = config;
        self
    }
    pub fn with_lid_pn_cache(mut self, config: CacheEntryConfig) -> Self {
        self.lid_pn = config;
        self
    }
    pub fn with_recent_messages(mut self, config: CacheEntryConfig) -> Self {
        self.recent_messages = config;
        self
    }
    pub fn with_cache_stores(mut self, stores: CacheStores) -> Self {
        self.stores = stores;
        self
    }
}
