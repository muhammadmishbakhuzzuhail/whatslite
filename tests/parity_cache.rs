//! Phase 2 cache contract.

use std::sync::Arc;
use std::time::Duration;

use wa_rs::cache::{CacheConfig, CacheEntryConfig, CacheStore, CacheStores, Freshness};
use wa_rs::portable_cache::PortableCache;

#[test]
fn cache_entry_config_and_builder_apply_overrides() {
    let entry = CacheEntryConfig::new(Some(Duration::from_secs(60)), 42);
    assert_eq!(entry.timeout, Some(Duration::from_secs(60)));
    assert_eq!(entry.capacity, 42);

    let config = CacheConfig::default()
        .with_group_cache(entry)
        .with_recent_messages(CacheEntryConfig::new(None, 7));
    assert_eq!(config.group.capacity, 42);
    assert_eq!(config.recent_messages.timeout, None);
    assert_eq!(config.recent_messages.capacity, 7);
}

#[test]
fn portable_cache_roundtrips_and_removes() {
    let cache = PortableCache::new();
    cache.put("k", vec![1, 2, 3]);
    assert_eq!(cache.get("k"), Some(vec![1, 2, 3]));
    cache.remove("k");
    assert_eq!(cache.get("k"), None);

    let shared: Arc<dyn CacheStore> = Arc::new(PortableCache::new());
    shared.put("a", vec![9]);
    assert_eq!(shared.get("a"), Some(vec![9]));
}

#[test]
fn cache_stores_debug_and_freshness_are_stable() {
    let stores = CacheStores::default();
    assert!(format!("{stores:?}").contains("CacheStores"));
    assert_eq!(Freshness::Fresh, Freshness::Fresh);
    assert_ne!(Freshness::Fresh, Freshness::Stale);
    assert_ne!(Freshness::Stale, Freshness::Missing);
}
