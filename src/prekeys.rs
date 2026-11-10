//! Pre-key management for Signal Protocol.
//!
//! Protocol types are defined in `wacore::iq::prekeys`.

use crate::client::Client;
use anyhow;
use log;
use rand::TryRngCore;
use wacore::iq::prekeys::{PreKeyCountSpec, PreKeyFetchSpec, PreKeyUploadSpec};
use wacore::libsignal::protocol::{KeyPair, PreKeyBundle, PublicKey};
use wacore::libsignal::store::record_helpers::new_pre_key_record;
use wacore_binary::jid::Jid;
use waproto::buffa::Message as _;

const WANTED_PRE_KEY_COUNT: usize = 50;
const MIN_PRE_KEY_COUNT: usize = 5;

/// Whether a count-guarded caller should upload.
///
/// A forced caller (prekey-low, digest 404 repair) skips the server-count IQ
/// entirely: WA Web's `handlePreKeyLow` uploads unconditionally, so a stale or
/// transiently-failing count must never block the replenish.
fn should_upload_pre_keys(force: bool, server_count: usize) -> bool {
    force || server_count < MIN_PRE_KEY_COUNT
}

impl Client {
    pub(crate) async fn fetch_pre_keys(
        &self,
        jids: &[Jid],
        reason: Option<&str>,
    ) -> Result<std::collections::HashMap<Jid, PreKeyBundle>, anyhow::Error> {
        let spec = match reason {
            Some(r) => PreKeyFetchSpec::with_reason(jids.to_vec(), r),
            None => PreKeyFetchSpec::new(jids.to_vec()),
        };

        let bundles = self.execute(spec).await?;

        for jid in bundles.keys() {
            log::debug!("Successfully parsed pre-key bundle for {jid}");
        }

        Ok(bundles)
    }

    /// Query the WhatsApp server for how many pre-keys it currently has for this device.
    pub(crate) async fn get_server_pre_key_count(&self) -> Result<usize, crate::request::IqError> {
        let response = self.execute(PreKeyCountSpec::new()).await?;
        Ok(response.count)
    }

    /// Ensure the server has at least MIN_PRE_KEY_COUNT pre-keys, and upload a batch of
    /// WANTED_PRE_KEY_COUNT pre-keys when it is below the threshold.
    pub(crate) async fn upload_pre_keys(&self) -> Result<(), anyhow::Error> {
        self.upload_pre_keys_inner(false).await
    }

    /// Upload a batch unconditionally, skipping the server-count query.
    ///
    /// Used by the prekey-low notification (WA Web's `handlePreKeyLow` uploads
    /// unconditionally) and by digest-404 repair. The server emits prekey-low
    /// only after crossing its own threshold, so re-querying and skipping when
    /// the local count looks adequate would let the pool drain.
    pub(crate) async fn upload_pre_keys_force(&self) -> Result<(), anyhow::Error> {
        self.upload_pre_keys_inner(true).await
    }

    async fn upload_pre_keys_inner(&self, force: bool) -> Result<(), anyhow::Error> {
        if !force {
            let server_count = match self.get_server_pre_key_count().await {
                Ok(c) => c,
                Err(e) => return Err(anyhow::anyhow!(e)),
            };

            if !should_upload_pre_keys(force, server_count) {
                log::debug!("Server has {} pre-keys, no upload needed.", server_count);
                return Ok(());
            }

            log::debug!("Server has {} pre-keys, uploading more.", server_count);
        } else {
            log::debug!("Forced pre-key upload (skipping server count query).");
        }

        let device_snapshot = self.persistence_manager.get_device_snapshot().await;
        let device_store = self.persistence_manager.get_device_arc().await;

        // Clone the backend Arc and drop the guard early to reduce lock contention.
        // This allows other tasks to access the device while we perform potentially
        // long-running backend operations (loops with many iterations).
        let backend = {
            let device_guard = device_store.read().await;
            device_guard.backend.clone()
        };

        let mut keys_to_upload = Vec::with_capacity(WANTED_PRE_KEY_COUNT);
        let mut key_pairs_to_upload = Vec::with_capacity(WANTED_PRE_KEY_COUNT);

        // New IDs must continue past the highest existing one: consumed one-time
        // pre-keys are removed after use, so the ID space has gaps and reusing an
        // in-gap ID would overwrite a key the server may still hand out.
        let max_existing_id = backend
            .max_prekey_id()
            .await
            .map_err(|e| anyhow::anyhow!("Failed to query max prekey ID: {}", e))?;
        let start_id = max_existing_id.map_or(1, |max| max.saturating_add(1));

        for i in 0..WANTED_PRE_KEY_COUNT {
            let pre_key_id = start_id + i as u32;

            // Ensure we don't exceed the valid range (1 to 0xFFFFFF)
            if pre_key_id > 16777215 {
                log::warn!(
                    "Pre-key ID {} exceeds maximum range, wrapping around",
                    pre_key_id
                );
                break;
            }

            let key_pair = KeyPair::generate(&mut rand::rngs::OsRng.unwrap_err());
            let pre_key_record = new_pre_key_record(pre_key_id, &key_pair);

            keys_to_upload.push((pre_key_id, pre_key_record));
            key_pairs_to_upload.push((pre_key_id, key_pair));
        }

        if keys_to_upload.is_empty() {
            log::warn!("No pre-keys available to upload");
            return Ok(());
        }

        // Step 3: Build upload request using type-safe IqSpec
        let pre_key_pairs: Vec<(u32, PublicKey)> = key_pairs_to_upload
            .iter()
            .map(|(id, key_pair)| (*id, key_pair.public_key))
            .collect();

        let spec = PreKeyUploadSpec::new(
            device_snapshot.registration_id,
            device_snapshot.identity_key.public_key,
            device_snapshot.signed_pre_key_id,
            device_snapshot.signed_pre_key.public_key,
            device_snapshot.signed_pre_key_signature.to_vec(),
            pre_key_pairs,
        );

        // Step 4: Send IQ to upload pre-keys
        self.execute(spec).await?;

        // Step 5: Store the new pre-keys using existing backend interface
        for (id, record) in keys_to_upload {
            // Mark as uploaded since the IQ was successful
            let record_bytes = record.encode_to_vec();
            if let Err(e) = backend.store_prekey(id, &record_bytes, true).await {
                log::warn!("Failed to store prekey id {}: {:?}", id, e);
            }
        }

        log::debug!(
            "Successfully uploaded {} new pre-keys with sequential IDs starting from {}.",
            key_pairs_to_upload.len(),
            start_id
        );

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A prekey-low notification must refill even when the server still reports
    /// a count at or above the low-water mark: the server emits the
    /// notification after crossing its own higher threshold.
    #[test]
    fn forced_prekey_upload_bypasses_the_count_guard() {
        for count in [0, MIN_PRE_KEY_COUNT, MIN_PRE_KEY_COUNT + 1, 50] {
            assert!(
                should_upload_pre_keys(true, count),
                "force must upload at count {count}"
            );
        }
    }

    #[test]
    fn count_guarded_prekey_upload_respects_the_low_water_mark() {
        assert!(should_upload_pre_keys(false, 0));
        assert!(should_upload_pre_keys(false, MIN_PRE_KEY_COUNT - 1));
        assert!(!should_upload_pre_keys(false, MIN_PRE_KEY_COUNT));
        assert!(!should_upload_pre_keys(false, 50));
    }
}
