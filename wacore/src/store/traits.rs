//! Storage traits for the WhatsApp client.
//!
//! This module defines 4 domain-grouped traits that together form the `Backend` trait:
//!
//! - [`SignalStore`]: Signal protocol cryptographic operations (identity, sessions, keys)
//! - [`AppSyncStore`]: WhatsApp app state synchronization
//! - [`ProtocolStore`]: WhatsApp Web protocol alignment (SKDM, LID mapping, device registry)
//! - [`DeviceStore`]: Device persistence operations

use crate::appstate::hash::HashState;
use crate::store::error::Result;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use wacore_appstate::processor::AppStateMutationMAC;
use wacore_binary::jid::{Jid, JidExt as _};

/// Inline protocol-sized message secret. The array makes invalid lengths
/// unrepresentable without a heap allocation or pointer indirection per row.
pub type MessageSecretBytes = [u8; crate::reporting_token::MESSAGE_SECRET_SIZE];

pub use crate::types::message_secret::{InvalidMessageSecret, MessageSecret};

/// A validated stored secret and the parent's event time in Unix seconds.
/// `None` means unknown, not the Unix epoch. Retention deadlines remain a
/// separate persisted property and are not changed by this read projection.
#[derive(Debug, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub struct StoredMessageSecret {
    pub secret: MessageSecret,
    pub message_ts: Option<i64>,
}

impl StoredMessageSecret {
    pub fn new(secret: MessageSecret, message_ts: Option<i64>) -> Self {
        Self {
            secret,
            message_ts: message_ts.filter(|ts| *ts != 0),
        }
    }

    /// Validate bytes read from storage, mapping malformed rows to a typed
    /// invalid-data error without truncating, padding or logging their contents.
    pub fn from_stored_bytes(secret: &[u8], message_ts: i64) -> Result<Self> {
        Ok(Self::new(
            MessageSecret::try_from(secret)
                .map_err(crate::store::error::StoreError::InvalidMessageSecret)?,
            Some(message_ts),
        ))
    }
}

/// App state synchronization key for WhatsApp's app state protocol.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AppStateSyncKey {
    pub key_data: Vec<u8>,
    pub fingerprint: Vec<u8>,
    pub timestamp: i64,
}

/// Entry representing a LID to Phone Number mapping.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LidPnMappingEntry {
    /// The LID user part (e.g., "100000012345678")
    pub lid: String,
    /// The phone number user part (e.g., "559980000001")
    pub phone_number: String,
    /// Unix timestamp when the mapping was first learned
    pub created_at: i64,
    /// Unix timestamp when the mapping was last updated
    pub updated_at: i64,
    /// The source from which this mapping was learned (e.g., "usync", "peer_pn_message")
    pub learning_source: String,
}

/// Trusted contact privacy token entry.
///
/// Matches WhatsApp Web's Chat.tcToken / tcTokenTimestamp / tcTokenSenderTimestamp.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TcTokenEntry {
    /// Raw token bytes received from the server.
    pub token: Vec<u8>,
    /// Unix timestamp (seconds) when the token was received.
    pub token_timestamp: i64,
    /// Unix timestamp (seconds) when we last issued our token to this contact.
    pub sender_timestamp: Option<i64>,
}

/// Message-secret write entry keyed by chat, sender, and message ID.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MsgSecretEntry {
    /// Canonical non-AD chat JID. Shared across entries from the same history
    /// conversation instead of allocating one identical string per message.
    pub chat: Arc<str>,
    /// Canonical non-AD sender JID. Often aliases `chat` for direct messages.
    pub sender: Arc<str>,
    /// Message identifier. `Arc<str>` keeps entry clones used by buffered
    /// persistence cheap without changing the serialized representation.
    pub msg_id: Arc<str>,
    pub secret: MessageSecretBytes,
    /// Absolute unix-seconds retention deadline. `0` means never expire.
    /// Computed by the caller from the parent message's event time plus a
    /// per-add-on-kind horizon (see `MsgSecretRetention`). The store prunes
    /// rows whose deadline has passed; it does not know the horizon itself.
    #[serde(default)]
    pub expires_at: i64,
    /// Parent message event time (unix seconds), or `0` when unknown. Kept so
    /// the receive path can enforce the edit-processing window
    /// (`editTs < message_ts + window`) the same way WhatsApp Web does.
    #[serde(default)]
    pub message_ts: i64,
}

impl MsgSecretEntry {
    /// The canonical non-AD sender identifier for a row whose chat identifier is
    /// already in hand, sharing that allocation whenever both JIDs address the
    /// same user. That covers every direct message (chat and sender are the peer)
    /// and every self-authored history row, which is what the `sender` field doc
    /// means by "often aliases `chat`".
    pub fn sender_id_for(chat: &Jid, chat_id: &Arc<str>, sender: &Jid) -> Arc<str> {
        if sender.is_same_chat_as(chat) {
            Arc::clone(chat_id)
        } else {
            Arc::from(sender.to_non_ad().to_string())
        }
    }

    /// Build a row from the JIDs the send and receive paths already carry.
    ///
    /// Single chokepoint for how a row's three identifiers are derived, so the
    /// inbound capture and the outbound persist cannot drift on either the
    /// canonicalisation or the aliasing above.
    pub fn new(
        chat: &Jid,
        sender: &Jid,
        msg_id: &str,
        secret: MessageSecretBytes,
        expires_at: i64,
        message_ts: i64,
    ) -> Self {
        let chat_id: Arc<str> = Arc::from(chat.to_non_ad().to_string());
        let sender_id = Self::sender_id_for(chat, &chat_id, sender);
        Self {
            chat: chat_id,
            sender: sender_id,
            msg_id: Arc::from(msg_id),
            secret,
            expires_at,
            message_ts,
        }
    }
}

/// Device information for registry tracking.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceInfo {
    /// The device ID (0 = primary device, 1+ = companion devices)
    pub device_id: u32,
    /// The key index, if known
    pub key_index: Option<u32>,
}

impl DeviceInfo {
    pub const fn new(device_id: u32, key_index: Option<u32>) -> Self {
        Self {
            device_id,
            key_index,
        }
    }

    pub const fn device_id(&self) -> u32 {
        self.device_id
    }

    pub const fn key_index(&self) -> Option<u32> {
        self.key_index
    }

    /// Compatibility shim: the current device record has no hosting bit.
    pub const fn with_hosting(self, _is_hosted: bool) -> Self {
        self
    }

    /// Compatibility shim: the current device record has no hosting bit.
    pub const fn is_hosted(&self) -> bool {
        false
    }
}

/// Device list record matching WhatsApp Web's DeviceListRecord structure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceListRecord {
    /// The user part of the JID (phone number or LID)
    pub user: String,
    /// List of known devices for this user
    pub devices: Vec<DeviceInfo>,
    /// Timestamp when this record was last updated
    pub timestamp: i64,
    /// Participant hash from usync, if available
    pub phash: Option<String>,
}

/// Signal protocol cryptographic storage operations.
///
/// Handles identity keys, sessions, pre-keys, signed pre-keys, and sender keys
/// for end-to-end encryption.
#[async_trait]
pub trait SignalStore: Send + Sync {
    // --- Identity Operations ---

    /// Store an identity key for a remote address.
    async fn put_identity(&self, address: &str, key: [u8; 32]) -> Result<()>;

    /// Load an identity key for a remote address.
    async fn load_identity(&self, address: &str) -> Result<Option<Vec<u8>>>;

    /// Delete an identity key.
    async fn delete_identity(&self, address: &str) -> Result<()>;

    // --- Session Operations ---

    /// Get an encrypted session for an address.
    async fn get_session(&self, address: &str) -> Result<Option<Vec<u8>>>;

    /// Store an encrypted session.
    async fn put_session(&self, address: &str, session: &[u8]) -> Result<()>;

    /// Delete a session.
    async fn delete_session(&self, address: &str) -> Result<()>;

    /// Check if a session exists. Default implementation uses `get_session`.
    async fn has_session(&self, address: &str) -> Result<bool> {
        Ok(self.get_session(address).await?.is_some())
    }

    // --- PreKey Operations ---

    /// Store a pre-key.
    async fn store_prekey(&self, id: u32, record: &[u8], uploaded: bool) -> Result<()>;

    /// Load a pre-key by ID.
    async fn load_prekey(&self, id: u32) -> Result<Option<Vec<u8>>>;

    /// Remove a pre-key.
    async fn remove_prekey(&self, id: u32) -> Result<()>;

    /// Get the highest stored pre-key ID, if any.
    ///
    /// Used to allocate new pre-key IDs without colliding with existing keys.
    /// Consumed one-time pre-keys are removed, so stored IDs are not contiguous:
    /// implementations must return the true maximum, not the last ID before the
    /// first gap.
    async fn max_prekey_id(&self) -> Result<Option<u32>>;

    // --- Signed PreKey Operations ---

    /// Store a signed pre-key.
    async fn store_signed_prekey(&self, id: u32, record: &[u8]) -> Result<()>;

    /// Load a signed pre-key by ID.
    async fn load_signed_prekey(&self, id: u32) -> Result<Option<Vec<u8>>>;

    /// Load all signed pre-keys. Returns (id, record) pairs.
    async fn load_all_signed_prekeys(&self) -> Result<Vec<(u32, Vec<u8>)>>;

    /// Remove a signed pre-key.
    async fn remove_signed_prekey(&self, id: u32) -> Result<()>;

    // --- Sender Key Operations ---

    /// Store a sender key for group messaging.
    async fn put_sender_key(&self, address: &str, record: &[u8]) -> Result<()>;

    /// Get a sender key.
    async fn get_sender_key(&self, address: &str) -> Result<Option<Vec<u8>>>;

    /// Delete a sender key.
    async fn delete_sender_key(&self, address: &str) -> Result<()>;
}

/// WhatsApp app state synchronization storage.
///
/// Handles sync keys, version tracking, and mutation MACs for the app state protocol.
#[async_trait]
pub trait AppSyncStore: Send + Sync {
    /// Get an app state sync key by ID.
    async fn get_sync_key(&self, key_id: &[u8]) -> Result<Option<AppStateSyncKey>>;

    /// Set an app state sync key.
    async fn set_sync_key(&self, key_id: &[u8], key: AppStateSyncKey) -> Result<()>;

    /// Get the app state version for a collection.
    async fn get_version(&self, name: &str) -> Result<HashState>;

    /// Set the app state version for a collection.
    async fn set_version(&self, name: &str, state: HashState) -> Result<()>;

    /// Store mutation MACs for a version.
    async fn put_mutation_macs(
        &self,
        name: &str,
        version: u64,
        mutations: &[AppStateMutationMAC],
    ) -> Result<()>;

    /// Get a mutation MAC by index.
    async fn get_mutation_mac(&self, name: &str, index_mac: &[u8]) -> Result<Option<Vec<u8>>>;

    /// Batch variant of [`get_mutation_mac`](Self::get_mutation_mac): fetch many
    /// previous-MAC values in a single backend round-trip. The default delegates
    /// to per-item lookups; backends with a set-membership query should override
    /// it to avoid an N+1.
    async fn get_mutation_macs(
        &self,
        name: &str,
        index_macs: &[[u8; 32]],
    ) -> Result<std::collections::HashMap<[u8; 32], Vec<u8>>> {
        let mut out = std::collections::HashMap::with_capacity(index_macs.len());
        for index_mac in index_macs {
            if let Some(mac) = self.get_mutation_mac(name, index_mac.as_slice()).await? {
                out.insert(*index_mac, mac);
            }
        }
        Ok(out)
    }

    /// Delete mutation MACs by their index MACs.
    async fn delete_mutation_macs(&self, name: &str, index_macs: &[Vec<u8>]) -> Result<()>;

    /// Persist one applied patch as a unit: the collection's new version, the
    /// index MACs the patch removed and the MACs it added.
    ///
    /// The default implementation applies the three existing primitives in
    /// order. It is not atomic; a backend that can do better should override it
    /// with a transaction so the version and MAC changes become visible
    /// together. Removals are applied before additions, as an index may occur
    /// in both.
    async fn commit_patch(
        &self,
        name: &str,
        state: HashState,
        removed_index_macs: &[Vec<u8>],
        added: &[AppStateMutationMAC],
    ) -> Result<()> {
        if !removed_index_macs.is_empty() {
            self.delete_mutation_macs(name, removed_index_macs).await?;
        }
        if !added.is_empty() {
            self.put_mutation_macs(name, state.version, added).await?;
        }
        self.set_version(name, state).await
    }

    /// Delete every mutation MAC for a collection.
    ///
    /// Called on snapshot re-sync so the MAC store is rebuilt from the snapshot,
    /// matching the ltHash baseline; leftover entries would corrupt the next
    /// patch's ltHash. Backends that cannot clear a collection fail closed
    /// rather than silently leaving stale MACs behind.
    async fn clear_mutation_macs(&self, _name: &str) -> Result<()> {
        Err(crate::store::error::StoreError::Database(
            "backend does not support clearing app state mutation MACs".to_string(),
        ))
    }

    /// Get the most recently stored app state sync key ID.
    ///
    /// Outbound patch building encrypts under the newest usable key; `None`
    /// means the account has no app-state key yet.
    async fn get_latest_sync_key_id(&self) -> Result<Option<Vec<u8>>>;
}

/// WhatsApp Web protocol alignment storage.
///
/// Handles SKDM tracking, LID-PN mapping, base key collision detection,
/// device registry, and sender key status.
#[async_trait]
pub trait ProtocolStore: Send + Sync {
    // --- SKDM Tracking ---

    /// Get device JIDs that have received SKDM for a group.
    async fn get_skdm_recipients(&self, group_jid: &str) -> Result<Vec<Jid>>;

    /// Record devices that have received SKDM for a group.
    async fn add_skdm_recipients(&self, group_jid: &str, device_jids: &[Jid]) -> Result<()>;

    /// Clear SKDM recipients for a group (call when sender key is rotated).
    async fn clear_skdm_recipients(&self, group_jid: &str) -> Result<()>;

    // --- LID-PN Mapping ---

    /// Get a mapping by LID.
    async fn get_lid_mapping(&self, lid: &str) -> Result<Option<LidPnMappingEntry>>;

    /// Get a mapping by phone number (returns the most recent LID for that phone).
    async fn get_pn_mapping(&self, phone: &str) -> Result<Option<LidPnMappingEntry>>;

    /// Store or update a LID-PN mapping.
    async fn put_lid_mapping(&self, entry: &LidPnMappingEntry) -> Result<()>;

    /// Get all LID-PN mappings (for cache warm-up).
    async fn get_all_lid_mappings(&self) -> Result<Vec<LidPnMappingEntry>>;

    // --- Base Key Collision Detection ---

    /// Save the base key for a session address during retry collision detection.
    async fn save_base_key(&self, address: &str, message_id: &str, base_key: &[u8]) -> Result<()>;

    /// Check if the current session has the same base key as the saved one.
    async fn has_same_base_key(
        &self,
        address: &str,
        message_id: &str,
        current_base_key: &[u8],
    ) -> Result<bool>;

    /// Delete a base key entry.
    async fn delete_base_key(&self, address: &str, message_id: &str) -> Result<()>;

    // --- Device Registry ---

    /// Update the device list for a user (called after usync responses).
    async fn update_device_list(&self, record: DeviceListRecord) -> Result<()>;

    /// Get all known devices for a user.
    async fn get_devices(&self, user: &str) -> Result<Option<DeviceListRecord>>;

    // --- Sender Key Status (Lazy Deletion) ---

    /// Mark a participant's sender key as needing regeneration for a group.
    async fn mark_forget_sender_key(&self, group_jid: &str, participant: &str) -> Result<()>;

    /// Get participants that need fresh SKDM (marked for forget).
    /// Consumes the marks (deletes them after reading).
    async fn consume_forget_marks(&self, group_jid: &str) -> Result<Vec<String>>;

    // --- TcToken Storage ---

    /// Get a trusted contact token for a JID (stored under LID).
    async fn get_tc_token(&self, jid: &str) -> Result<Option<TcTokenEntry>>;

    /// Store or update a trusted contact token for a JID.
    async fn put_tc_token(&self, jid: &str, entry: &TcTokenEntry) -> Result<()>;

    /// Delete a trusted contact token for a JID.
    async fn delete_tc_token(&self, jid: &str) -> Result<()>;

    /// Get all JIDs that have stored tc tokens.
    async fn get_all_tc_token_jids(&self) -> Result<Vec<String>>;

    /// Delete tc tokens with token_timestamp older than cutoff. Returns count deleted.
    async fn delete_expired_tc_tokens(&self, cutoff_timestamp: i64) -> Result<u32>;
}

/// Device data persistence operations.
#[async_trait]
pub trait DeviceStore: Send + Sync {
    /// Save device data.
    async fn save(&self, device: &crate::store::Device) -> Result<()>;

    /// Load device data.
    async fn load(&self) -> Result<Option<crate::store::Device>>;

    /// Check if a device exists.
    async fn exists(&self) -> Result<bool>;

    /// Create a new device row and return its generated device_id.
    async fn create(&self) -> Result<i32>;

    /// Create a snapshot of the database state.
    /// The argument `name` can be used to label the snapshot file.
    /// `extra_content` can be used to save a related binary blob (e.g. the message that caused the failure).
    async fn snapshot_db(&self, _name: &str, _extra_content: Option<&[u8]>) -> Result<()> {
        Ok(())
    }
}

/// Per-outbound-message secret storage for addon-style decryption.
///
/// Persists the 32-byte `MessageContextInfo.messageSecret` we send out so that
/// later inbound replies (poll votes, reactions, msmsg bot responses, edits)
/// referencing the original message ID can be decrypted. Mirrors WA Web's
/// `WAWebMsmsgMsgSecretCache` + the `messageSecret` field on the DB message row.
#[async_trait]
pub trait MsgSecretStore: Send + Sync {
    /// Persist the protocol-sized `secret` under the composite key with NO
    /// expiry (`expires_at = 0`). Convenience wrapper over [`put_msg_secrets`].
    /// `chat`, `sender`, and `msg_id` are JID strings / message ID strings;
    /// callers should pass non-AD (no-device) form for the JIDs so lookups
    /// match regardless of which device echo'd the stanza back.
    ///
    /// Real call sites that compute a retention deadline build
    /// [`MsgSecretEntry`] directly and call [`put_msg_secrets`].
    ///
    /// [`put_msg_secrets`]: MsgSecretStore::put_msg_secrets
    async fn put_msg_secret(
        &self,
        chat: &str,
        sender: &str,
        msg_id: &str,
        secret: &[u8; crate::reporting_token::MESSAGE_SECRET_SIZE],
    ) -> Result<()> {
        self.put_msg_secrets(vec![MsgSecretEntry {
            chat: Arc::from(chat),
            sender: Arc::from(sender),
            msg_id: Arc::from(msg_id),
            secret: *secret,
            expires_at: 0,
            message_ts: 0,
        }])
        .await?;
        Ok(())
    }

    /// Batched upsert carrying a per-row `expires_at` deadline. On key conflict
    /// implementations merge deterministically via [`merge_msg_secret_expiry`]
    /// (later deadline wins, `0` = "never" = infinity) so a redelivery or edit
    /// re-persist never shortens a window, and via [`merge_msg_secret_message_ts`]
    /// (the later non-zero parent time wins; a `0` never clobbers a known one).
    async fn put_msg_secrets(&self, entries: Vec<MsgSecretEntry>) -> Result<usize>;

    /// Read a validated secret and all available parent timestamp metadata.
    ///
    /// Required deliberately: an adapter migration must choose whether its
    /// timestamp is known, rather than inheriting a default that silently loses
    /// it. Backends without timestamps explicitly return `message_ts: None`.
    /// Malformed persisted bytes must return `StoreError::InvalidMessageSecret`.
    async fn get_stored_msg_secret(
        &self,
        chat: &str,
        sender: &str,
        msg_id: &str,
    ) -> Result<Option<StoredMessageSecret>>;

    /// Explicit byte-only interoperability projection. Prefer
    /// [`Self::get_stored_msg_secret`] when the parent time matters. Even this
    /// projection validates the persisted bytes through the named read.
    async fn get_msg_secret(
        &self,
        chat: &str,
        sender: &str,
        msg_id: &str,
    ) -> Result<Option<Vec<u8>>> {
        Ok(self
            .get_stored_msg_secret(chat, sender, msg_id)
            .await?
            .map(|stored| stored.secret.into_bytes().to_vec()))
    }

    /// Delete rows whose non-zero `expires_at` is at or before
    /// `cutoff_timestamp` (absolute unix seconds; callers pass "now"). Rows
    /// with `expires_at = 0` (never) are kept. Returns the number removed so
    /// the keepalive cleanup can log/throttle.
    async fn delete_expired_msg_secrets(&self, cutoff_timestamp: i64) -> Result<u32>;
}

/// Merge two `expires_at` deadlines on key conflict: `0` ("never") wins,
/// otherwise the later (larger) deadline is kept so windows never shrink.
pub fn merge_msg_secret_expiry(existing: i64, incoming: i64) -> i64 {
    if existing == 0 || incoming == 0 {
        0
    } else {
        existing.max(incoming)
    }
}

/// Merge two parent `message_ts` values on key conflict: the later (larger)
/// non-zero value wins, so a `0` ("unknown") never clobbers a known parent
/// time. `max` already yields this because every real timestamp is `> 0`.
pub fn merge_msg_secret_message_ts(existing: i64, incoming: i64) -> i64 {
    existing.max(incoming)
}

/// Combined storage backend trait.
///
/// Any type implementing all four domain traits automatically implements `Backend`.
pub trait Backend: SignalStore + AppSyncStore + ProtocolStore + DeviceStore + Send + Sync {}

impl<T> Backend for T where T: SignalStore + AppSyncStore + ProtocolStore + DeviceStore + Send + Sync
{}
