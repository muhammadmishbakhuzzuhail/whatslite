//! Minimal portable in-memory cache used by hosts without a shared store.

use std::collections::HashMap;
use std::sync::Mutex;

use crate::cache_config::CacheStore;

#[derive(Default)]
pub struct PortableCache {
    entries: Mutex<HashMap<String, Vec<u8>>>,
}

impl PortableCache {
    pub fn new() -> Self {
        Self::default()
    }
}

impl CacheStore for PortableCache {
    fn get(&self, key: &str) -> Option<Vec<u8>> {
        self.entries.lock().ok()?.get(key).cloned()
    }
    fn put(&self, key: &str, value: Vec<u8>) {
        if let Ok(mut entries) = self.entries.lock() {
            entries.insert(key.to_string(), value);
        }
    }
    fn remove(&self, key: &str) {
        if let Ok(mut entries) = self.entries.lock() {
            entries.remove(key);
        }
    }
}
