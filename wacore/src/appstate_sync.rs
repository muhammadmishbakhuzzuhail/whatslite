pub mod preparation;

use std::collections::{HashMap, HashSet, VecDeque};
use std::sync::Arc;

use anyhow::{Context, Result, anyhow};
use async_lock::Mutex;
use async_trait::async_trait;
use thiserror::Error;

use crate::appstate::hash::HashState;
use crate::appstate::hash::generate_index_mac;
use crate::appstate::keys::ExpandedAppStateKeys;
use crate::appstate::patch_decode::{CollectionSyncError, PatchList, WAPatchName};
use crate::appstate::processor::AppStateMutationMAC;
use crate::appstate::{collect_key_ids_from_patch_list, expand_app_state_keys};
use crate::store::traits::Backend;
use waproto::whatsapp as wa;

// Re-export Mutation from appstate for convenience
pub use crate::appstate::Mutation;

/// Syncd index names the processor inspects directly. They live in the
/// generated schema catalog upstream; this fork's appstate crate does not carry
/// that catalog yet, so the two verbs the engine needs are named here.
const STATUS_PRIVACY_INDEX: &str = "status_privacy";
const PN_FOR_LID_CHAT_INDEX: &str = "pnForLidChat";

#[derive(Error)]
#[error("{source}")]
/// An error after earlier mutations in the same collection were committed.
///
/// Callers must dispatch the returned mutations even though a later patch
/// failed. The state is the cursor that was committed with those mutations.
pub struct CommittedMutationsError {
    mutations: Vec<Mutation>,
    state: HashState,
    collection: WAPatchName,
    #[source]
    source: anyhow::Error,
}

impl std::fmt::Debug for CommittedMutationsError {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("CommittedMutationsError")
            .field("mutations", &self.mutations.len())
            .field("state", &self.state)
            .field("collection", &self.collection)
            .field("source", &self.source)
            .finish()
    }
}

impl CommittedMutationsError {
    /// Split the committed work from the error that stopped the next patch.
    pub fn into_parts(self) -> (Vec<Mutation>, HashState, WAPatchName, anyhow::Error) {
        (self.mutations, self.state, self.collection, self.source)
    }
}

#[derive(Error)]
#[error("{source}")]
/// An error after earlier collections in a batch were committed.
///
/// Callers must consume the returned results even though a later collection
/// failed. The source may itself be a [`CommittedMutationsError`].
pub struct CommittedPatchListsError {
    results: Vec<(Vec<Mutation>, HashState, PatchList)>,
    #[source]
    source: anyhow::Error,
}

impl std::fmt::Debug for CommittedPatchListsError {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("CommittedPatchListsError")
            .field("results", &self.results.len())
            .field("source", &self.source)
            .finish()
    }
}

impl CommittedPatchListsError {
    /// Split the committed collection results from the later error.
    pub fn into_parts(self) -> (Vec<(Vec<Mutation>, HashState, PatchList)>, anyhow::Error) {
        (self.results, self.source)
    }
}

fn committed_mutations_error(
    source: anyhow::Error,
    mutations: Vec<Mutation>,
    state: &HashState,
    collection: WAPatchName,
) -> anyhow::Error {
    if mutations.is_empty() {
        source
    } else {
        CommittedMutationsError {
            mutations,
            state: state.clone(),
            collection,
            source,
        }
        .into()
    }
}

#[cfg_attr(target_arch = "wasm32", async_trait(?Send))]
#[cfg_attr(not(target_arch = "wasm32"), async_trait)]
/// Persists mutation-derived state that must not lag its app-state cursor.
///
/// The processor calls this hook before committing the cursor for a patch,
/// snapshot, or snapshot recovery. A hook error leaves that cursor unchanged
/// so the mutation can be replayed.
pub trait AppStateMutationPersistence: crate::sync_marker::MaybeSendSync {
    /// Persist the complete status privacy action, including unknown modes.
    async fn persist_status_privacy(
        &self,
        action: &wa::sync_action_value::StatusPrivacyAction,
    ) -> crate::store::error::Result<()>;

    /// Persist validated pnForLidChat pairs before advancing the collection cursor.
    /// The default preserves compatibility with status-privacy-only adapters.
    async fn persist_lid_pn_mappings(
        &self,
        _mappings: Vec<(String, String)>,
    ) -> crate::store::error::Result<()> {
        Ok(())
    }
}

/// Return a complete `status_privacy` SET action with a mode.
///
/// Missing modes and other mutation indexes are not valid audience updates.
pub fn status_privacy_action(
    mutation: &Mutation,
) -> Option<&wa::sync_action_value::StatusPrivacyAction> {
    if mutation.operation != wa::syncd_mutation::SyncdOperation::SET
        || mutation.index.as_slice() != [STATUS_PRIVACY_INDEX]
    {
        return None;
    }
    mutation
        .action_value
        .as_ref()?
        .status_privacy
        .as_option()
        .filter(|action| action.mode.is_some())
}

/// A validated pnForLidChat SET pair, with user addresses in LID and PN namespaces.
// WAWebPnForLidChatSync uses createUserLidOrThrow/createUserWidOrThrow.
// Reject other namespaces and device addresses rather than turning them into
// a pair of bare user strings that the ledger would accept without validation.
pub fn pn_for_lid_chat_pair(m: &Mutation) -> Option<(wacore_binary::Jid, wacore_binary::Jid)> {
    let [command, lid] = m.index.as_slice() else {
        return None;
    };
    if command != PN_FOR_LID_CHAT_INDEX || m.operation != wa::syncd_mutation::SyncdOperation::SET {
        return None;
    }
    let lid: wacore_binary::Jid = lid.parse().ok()?;
    let pn: wacore_binary::Jid = m
        .action_value
        .as_ref()?
        .pn_for_lid_chat_action
        .as_option()?
        .pn_jid
        .as_ref()?
        .parse()
        .ok()?;
    let valid_user = |jid: &wacore_binary::Jid| {
        !jid.user.is_empty()
            && jid.user != "0"
            && jid.user.bytes().all(|byte| byte.is_ascii_digit())
            && jid.device == 0
            && jid.agent == 0
            && jid.integrator == 0
    };
    (lid.is_lid() && pn.is_pn() && valid_user(&lid) && valid_user(&pn)).then_some((lid, pn))
}

/// Index MAC carried by a mutation's record, if present.
fn mutation_index_mac(m: &wa::SyncdMutation) -> Option<&[u8]> {
    m.record.as_option()?.index.as_option()?.blob.as_deref()
}

/// A mutation's index MAC as the fixed-size HMAC-SHA256 array; `None` for a
/// missing OR malformed (non-32-byte) blob. A malformed MAC could never match a
/// stored row, so dropping it from the batch lookup is equivalent to the miss
/// the lookup would return anyway.
fn mutation_index_mac_array(m: &wa::SyncdMutation) -> Option<IndexMac> {
    mutation_index_mac(m).and_then(|b| b.try_into().ok())
}

/// An appstate mutation index MAC: a full HMAC-SHA256 output, so always 32
/// bytes. Inline (`Copy`) so batch lookups sort/hash flat arrays with zero
/// per-MAC heap allocations.
pub type IndexMac = [u8; 32];

/// Distinct index MACs of a patch's mutations, feeding the batched
/// previous-value-MAC backend lookup. Both callers pass the result straight to a
/// `get_mutation_macs` HashMap fetch, so the order is unspecified.
///
/// Small patches dedup with a linear scan; larger patches sort + dedup the
/// inline arrays in place. Either way the only allocation is the returned `Vec`
/// itself.
pub fn collect_unique_index_macs(mutations: &[wa::SyncdMutation]) -> Vec<IndexMac> {
    if mutations.len() <= MAC_DEDUP_SCAN_LIMIT {
        let mut out: Vec<IndexMac> = Vec::with_capacity(mutations.len());
        for m in mutations {
            if let Some(mac) = mutation_index_mac_array(m)
                && !out.contains(&mac)
            {
                out.push(mac);
            }
        }
        return out;
    }

    let mut macs: Vec<IndexMac> = Vec::with_capacity(mutations.len());
    macs.extend(mutations.iter().filter_map(mutation_index_mac_array));
    macs.sort_unstable();
    macs.dedup();
    macs
}

/// Mutation count at or below which [`collect_unique_index_macs`] dedups with a
/// linear scan instead of a sort; above it the sort wins despite allocating
/// every MAC before deduping.
const MAC_DEDUP_SCAN_LIMIT: usize = 64;

// Only one owner holds the collection state across snapshot and patch phases.
// Their futures borrow this small state instead of embedding each other's locals.
struct PatchListProgress {
    state: HashState,
    committed_state: HashState,
    new_mutations: Vec<Mutation>,
}

/// Validate collection metadata before admitting external downloads.
/// Large lists belong on a blocking worker, as in [`AppStateProcessor::prepare_patch_list`].
#[inline(never)]
pub fn validate_patch_list_metadata(pl: &PatchList) -> Result<()> {
    let name = pl.name;
    // Validate the untrusted protobuf boundary before handing references to a
    // downloader or advancing any collection state.
    let collection = name.as_str();
    if let Some(reference) = &pl.snapshot_ref {
        crate::appstate::validation::validate_external_reference(collection, reference)?;
    }
    if let Some(snapshot) = &pl.snapshot {
        validate_snapshot_for_processing(collection, snapshot)?;
    }
    for patch in &pl.patches {
        crate::appstate::validation::validate_patch(collection, patch)?;
    }
    Ok(())
}

#[inline(never)]
fn validate_snapshot_for_processing(collection: &str, snapshot: &wa::SyncdSnapshot) -> Result<()> {
    crate::appstate::validation::validate_snapshot(collection, snapshot).map_err(|error| {
        if matches!(
            error.metric.as_deref(),
            Some("MISSING_SNAPSHOT_MAC" | "MISSING_SNAPSHOT_KEY_ID")
        ) {
            anyhow::Error::new(error).context(crate::appstate::AppStateError::SnapshotMACMissing)
        } else {
            anyhow::Error::new(error)
        }
    })
}

#[cfg(test)]
fn download_external_blobs(pl: &mut PatchList, download: &BlobDownloadFn<'_>) -> Result<()> {
    validate_patch_list_metadata(pl)?;
    decode_external_blobs(pl, download, true)
}

#[inline(never)]
fn decode_external_blobs(
    pl: &mut PatchList,
    download: &BlobDownloadFn<'_>,
    validate: bool,
) -> Result<()> {
    let name = pl.name;
    let collection = name.as_str();
    if pl.snapshot.is_none()
        && let Some(ext) = &pl.snapshot_ref
    {
        let data =
            download(ext).with_context(|| format!("download external snapshot for {name:?}"))?;
        let snapshot = waproto::codec::syncd_snapshot_decode(&data)
            .with_context(|| format!("decode external snapshot for {name:?}"))?;
        if validate {
            validate_snapshot_for_processing(collection, &snapshot)?;
        }
        pl.snapshot = Some(snapshot);
    }

    for patch in &mut pl.patches {
        if let Some(ext) = patch.external_mutations.as_option() {
            let v = patch
                .version
                .as_option()
                .and_then(|x| x.version)
                .unwrap_or(0);
            let data = download(ext)
                .with_context(|| format!("download external mutations for {name:?} v{v}"))?;
            let ext_mutations = waproto::codec::syncd_mutations_decode(&data)
                .with_context(|| format!("decode external mutations for {name:?} v{v}"))?;
            if validate {
                for mutation in &ext_mutations.mutations {
                    crate::appstate::validation::validate_mutation(collection, mutation)?;
                }
            }
            patch.mutations = ext_mutations.mutations;
            // The validated external arm omits exitCode.
            patch.exit_code = buffa::MessageField::none();
            // Consumed: the reference is what marks a patch as still external,
            // so leaving it would make the next pass download and decode the
            // blob a second time.
            patch.external_mutations = buffa::MessageField::none();
        }
    }
    if validate {
        crate::appstate::validation::validate_no_duplicate_patch_version(
            pl.patches.iter().filter_map(|patch| {
                patch
                    .version
                    .as_option()
                    .and_then(|version| version.version)
            }),
        )?;
    }
    Ok(())
}

/// An owned collection whose wire metadata has been checked before downloads.
///
/// The private fields prevent replacing or mutating the checked protobufs before
/// preparation. Metadata failures travel with their original collection so batch
/// processing can retain its existing per-collection error policy.
#[derive(Debug)]
pub struct MetadataValidatedPatchList {
    list: PatchList,
    error: Option<anyhow::Error>,
}
impl MetadataValidatedPatchList {
    /// Validate an owned list. Large collections belong on a blocking worker.
    pub fn validate(list: PatchList) -> Self {
        let error = if list.error.is_none() {
            validate_patch_list_metadata(&list).err()
        } else {
            None
        };
        Self { list, error }
    }
    pub fn list(&self) -> &PatchList {
        &self.list
    }
    pub fn can_download(&self) -> bool {
        self.error.is_none() && self.list.error.is_none()
    }
}

/// A collection whose wire metadata and downloaded content have been checked.
/// Owns preparation failures so batched callers can preserve per-collection policy.
#[derive(Debug)]
pub struct PreparedPatchList {
    list: PatchList,
    error: Option<anyhow::Error>,
}
impl PreparedPatchList {
    pub fn list(&self) -> &PatchList {
        &self.list
    }
    pub fn is_ready(&self) -> bool {
        self.error.is_none() && self.list.error.is_none()
    }
}

/// External-blob resolver as a trait object, so the large download/decode/apply
/// bodies below instantiate once instead of per closure type. Returns `Bytes`
/// so a resolver serving from a prefetch map hands out a refcount instead of
/// copying the blob.
pub type BlobDownloadFn<'a> =
    dyn Fn(&wa::ExternalBlobReference) -> Result<bytes::Bytes> + Send + Sync + 'a;

#[derive(Debug, Error)]
#[non_exhaustive]
pub enum AppStateSyncError {
    #[error("app state key not found: {0}")]
    KeyNotFound(String),
    #[error("store error")]
    Store(#[from] crate::store::error::StoreError),
    #[error("{0}")]
    Other(#[from] anyhow::Error),
}

/// One outstanding ask, and the id the answer will carry back.
///
/// The id is what lets a reply that cannot be decoded still be recognised as
/// the answer to a request: the collection name is inside the payload, so a
/// truncated or corrupt one names nothing.
struct RecoveryRequest {
    asked_at: crate::time::Instant,
    request_id: Option<String>,
    /// Set once an answer carrying this id has been taken up, so a repeat of
    /// that answer is refused before it costs a decode.
    answering: bool,
    /// The caller's monotonic connection generation when the ask was made.
    /// It stops an older sender from replacing a newer request, since the
    /// marker survives a reconnect.
    generation: u64,
}

/// What became of a collection the primary sent back.
///
/// Discarding is neither a failure nor a success, and a caller that could not
/// tell the two apart would log a recovery that did not happen.
#[derive(Debug)]
pub enum RecoveryOutcome {
    /// The collection was replaced; these are its records.
    Applied(Vec<Mutation>),
    /// The collection had already moved past what the primary offered, so it was
    /// left alone.
    Stale { held: u64, offered: u64 },
    /// The caller stopped standing behind the write before it began, so nothing
    /// was touched.
    Retired,
}

/// Which ask, if any, a recovery reply answers.
///
/// Three answers rather than an `Option`, because "nothing asked for this" and
/// "this is a copy of an answer already being handled" lead to opposite ends:
/// the first is decoded and applied if it moves the collection forward, the
/// second is refused before it costs a decode.
#[derive(Debug, PartialEq, Eq)]
pub enum RecoveryClaim {
    /// The reply answers this outstanding ask, now claimed for it.
    Claimed(String),
    /// The ask this id was made under is already being answered by another
    /// copy of the reply.
    AlreadyAnswering(String),
    /// No outstanding ask carries this id: a reply to an ask made by an earlier
    /// process, to one a newer ask has since replaced, or to none at all.
    Unrequested,
}

/// Bookkeeping for recovery asks, separated from the processor so the claim
/// rules are testable without a backend.
#[derive(Default)]
struct RecoveryRegistry {
    outstanding: HashMap<String, RecoveryRequest>,
}

impl RecoveryRegistry {
    /// How long one outstanding request suppresses another for the same
    /// collection.
    ///
    /// Bounded on both sides by what has to happen inside it. Longer than the
    /// reply takes (WA Web waits 60s) and shorter than the retry budget that
    /// produces the asks (1, 2, 4, ... 128 seconds), so a phone that stays
    /// silent is asked again within the same stuck sync.
    const REQUEST_TTL: core::time::Duration = core::time::Duration::from_secs(120);

    /// How long a request that is *being answered* survives instead.
    ///
    /// Measured from the claim, not from the ask: expiry is lazy, so a phone
    /// that answers long after the window closed still claims successfully.
    /// Still an upper bound: a task that dies without ending its claim would
    /// otherwise keep the collection from ever being asked about again.
    const ANSWER_TTL: core::time::Duration = core::time::Duration::from_secs(900);

    fn mark(&mut self, collection: &str, generation: u64, request_id: Option<&str>) -> bool {
        if let Some(request) = self.outstanding.get(collection)
            && (request.generation > generation
                || (request.generation == generation
                    && request.asked_at.elapsed()
                        < if request.answering {
                            Self::ANSWER_TTL
                        } else {
                            Self::REQUEST_TTL
                        }))
        {
            return false;
        }
        self.outstanding.insert(
            collection.to_string(),
            RecoveryRequest {
                asked_at: crate::time::Instant::now(),
                request_id: request_id.map(str::to_string),
                answering: false,
                generation,
            },
        );
        true
    }

    fn note_id(&mut self, collection: &str, request_id: &str) {
        if let Some(request) = self.outstanding.get_mut(collection) {
            request.request_id = Some(request_id.to_string());
        }
    }

    fn claim_with_generation(&mut self, request_id: &str) -> Option<(String, u64)> {
        let (name, request) = self
            .outstanding
            .iter_mut()
            .find(|(_, request)| request.request_id.as_deref() == Some(request_id))?;
        if request.answering {
            return None;
        }
        request.answering = true;
        // The clock restarts here: from here the entry protects a handler
        // rather than recording an unanswered ask.
        request.asked_at = crate::time::Instant::now();
        Some((name.clone(), request.generation))
    }

    fn claim(&mut self, request_id: &str) -> RecoveryClaim {
        let Some((name, request)) = self
            .outstanding
            .iter_mut()
            .find(|(_, request)| request.request_id.as_deref() == Some(request_id))
        else {
            return RecoveryClaim::Unrequested;
        };
        if request.answering {
            return RecoveryClaim::AlreadyAnswering(name.clone());
        }
        request.answering = true;
        request.asked_at = crate::time::Instant::now();
        RecoveryClaim::Claimed(name.clone())
    }

    fn end_answer(&mut self, collection: &str, request_id: Option<&str>) -> bool {
        let settled = self.outstanding.get(collection).is_some_and(|request| {
            !request.answering
                || request_id.is_some_and(|id| request.request_id.as_deref() == Some(id))
        });
        if settled {
            self.outstanding.remove(collection);
        }
        settled
    }

    fn take_by_id(&mut self, request_id: &str) -> Option<(String, u64)> {
        let collection = self
            .outstanding
            .iter()
            .find(|(_, request)| request.request_id.as_deref() == Some(request_id))
            .map(|(name, _)| name.clone())?;
        let request = self.outstanding.remove(&collection)?;
        Some((collection, request.generation))
    }

    fn has(&self, collection: &str) -> bool {
        self.outstanding.contains_key(collection)
    }

    fn take(&mut self, collection: &str) -> bool {
        self.outstanding.remove(collection).is_some()
    }

    fn len(&self) -> usize {
        self.outstanding.len()
    }
}

#[derive(Clone)]
pub struct AppStateProcessor {
    pub backend: Arc<dyn Backend>,
    pub runtime: Arc<dyn crate::runtime::Runtime>,
    mutation_persistence: Option<Arc<dyn AppStateMutationPersistence>>,
    /// Expanded app-state keys, keyed by the raw key id: the lookup runs once
    /// per mutation, and a base64 key meant encoding (and allocating) the id for
    /// every one of them.
    key_cache: Arc<Mutex<KeyCache>>,
    /// Collections a recovery has been asked of the primary for.
    ///
    /// Held here rather than beside the connection because it has to outlive the
    /// sync that raised it: the phone answers whenever it answers, and by then
    /// the run that asked is long over.
    recovery_requested: Arc<Mutex<RecoveryRegistry>>,
}

/// Expanded app-state keys, bounded and keyed by raw key id.
///
/// An expanded key is a pure function of its key id — HKDF over bytes the
/// backend stores and never rewrites — so a cached entry can never go stale and
/// the only reason to drop one is memory.
#[derive(Default)]
struct KeyCache {
    keys: HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>,
    /// Insertion order, oldest first. Only ids currently in `keys`.
    order: VecDeque<Vec<u8>>,
}

impl KeyCache {
    const CAPACITY: usize = 32;

    fn get(&self, key_id: &[u8]) -> Option<Arc<ExpandedAppStateKeys>> {
        self.keys.get(key_id).cloned()
    }

    fn insert(&mut self, key_id: Vec<u8>, expanded: Arc<ExpandedAppStateKeys>) {
        if self.keys.insert(key_id.clone(), expanded).is_none() {
            self.order.push_back(key_id);
            while self.order.len() > Self::CAPACITY
                && let Some(oldest) = self.order.pop_front()
            {
                self.keys.remove(&oldest);
            }
        }
    }

    fn len(&self) -> usize {
        self.keys.len()
    }
}

impl AppStateProcessor {
    /// Create a processor without a mutation persistence hook.
    ///
    /// This preserves the original processor contract. Mutations are returned
    /// to the caller, but mutation-derived state is not persisted separately.
    pub fn new(backend: Arc<dyn Backend>, runtime: Arc<dyn crate::runtime::Runtime>) -> Self {
        Self {
            runtime,
            backend,
            mutation_persistence: None,
            key_cache: Arc::new(Mutex::new(KeyCache::default())),
            recovery_requested: Arc::new(Mutex::new(RecoveryRegistry::default())),
        }
    }

    /// Install durable storage for mutation-derived state.
    ///
    /// Hook failures stop the affected cursor from advancing. Earlier work may
    /// still be returned through [`CommittedMutationsError`] or
    /// [`CommittedPatchListsError`].
    pub fn with_mutation_persistence(
        mut self,
        persistence: Arc<dyn AppStateMutationPersistence>,
    ) -> Self {
        self.mutation_persistence = Some(persistence);
        self
    }

    async fn persist_mutation_effects(&self, mutations: &[Mutation]) -> Result<()> {
        let Some(persistence) = self.mutation_persistence.as_ref() else {
            return Ok(());
        };
        if let Some(action) = mutations.iter().rev().find_map(status_privacy_action) {
            persistence.persist_status_privacy(action).await?;
        }
        let mappings: Vec<_> = mutations
            .iter()
            .filter_map(pn_for_lid_chat_pair)
            .map(|(lid, pn)| (lid.user.to_string(), pn.user.to_string()))
            .collect();
        if !mappings.is_empty() {
            persistence.persist_lid_pn_mappings(mappings).await?;
        }
        Ok(())
    }

    /// Records that the primary is being asked for this collection, answering
    /// whether the request is a new one.
    ///
    /// `false` means one is already outstanding and the caller should not send:
    /// the reply that is already coming answers this ask too.
    pub async fn mark_recovery_requested(&self, collection: &str) -> bool {
        self.mark_recovery_requested_on_generation(collection, 0)
            .await
    }

    /// Records the caller's connection generation for response validation.
    pub async fn mark_recovery_requested_on_generation(
        &self,
        collection: &str,
        generation: u64,
    ) -> bool {
        self.recovery_requested
            .lock()
            .await
            .mark(collection, generation, None)
    }

    /// Records the id and generation atomically before a peer send.
    pub async fn mark_recovery_requested_with_id(
        &self,
        collection: &str,
        generation: u64,
        request_id: &str,
    ) -> bool {
        self.recovery_requested
            .lock()
            .await
            .mark(collection, generation, Some(request_id))
    }

    /// How many recovery requests are outstanding, for the memory report.
    pub async fn outstanding_recovery_requests(&self) -> usize {
        self.recovery_requested.lock().await.len()
    }

    /// Records the id the primary's answer will carry.
    ///
    /// Set before the send, like the marker itself: the id is minted by the
    /// caller rather than by the send, and a reply that beats the send's return
    /// would otherwise find the request recorded with no id at all.
    pub async fn note_recovery_request_id(&self, collection: &str, request_id: &str) {
        self.recovery_requested
            .lock()
            .await
            .note_id(collection, request_id);
    }

    /// The collection an answer's id was asked about, claimed for this answer.
    pub async fn claim_recovery_request_by_id(&self, request_id: &str) -> Option<String> {
        self.claim_recovery_request_with_generation_by_id(request_id)
            .await
            .map(|(name, _)| name)
    }

    /// Claims a reply together with its request generation, before decoding.
    pub async fn claim_recovery_request_with_generation_by_id(
        &self,
        request_id: &str,
    ) -> Option<(String, u64)> {
        self.recovery_requested
            .lock()
            .await
            .claim_with_generation(request_id)
    }

    /// [`Self::claim_recovery_request_by_id`], telling a reply nobody here is
    /// waiting for apart from a copy of an answer already being handled.
    pub async fn claim_recovery_reply(&self, request_id: &str) -> RecoveryClaim {
        self.recovery_requested.lock().await.claim(request_id)
    }

    /// Ends an answer for `collection`, releasing the ask it settled.
    pub async fn end_recovery_answer(&self, collection: &str, request_id: Option<&str>) -> bool {
        self.recovery_requested
            .lock()
            .await
            .end_answer(collection, request_id)
    }

    /// Takes the request an id answers, whatever its payload turned out to be.
    pub async fn take_recovery_request_by_id(&self, request_id: &str) -> Option<String> {
        self.take_recovery_request_with_generation_by_id(request_id)
            .await
            .map(|(collection, _)| collection)
    }

    /// [`Self::take_recovery_request_by_id`], also returning the generation the
    /// ask was marked with.
    pub async fn take_recovery_request_with_generation_by_id(
        &self,
        request_id: &str,
    ) -> Option<(String, u64)> {
        self.recovery_requested.lock().await.take_by_id(request_id)
    }

    /// Whether a recovery for this collection is outstanding, without taking it.
    pub async fn has_recovery_request(&self, collection: &str) -> bool {
        self.recovery_requested.lock().await.has(collection)
    }

    pub async fn take_recovery_request(&self, collection: &str) -> bool {
        self.recovery_requested.lock().await.take(collection)
    }

    pub async fn get_app_state_key(
        &self,
        key_id: &[u8],
    ) -> std::result::Result<Arc<ExpandedAppStateKeys>, AppStateSyncError> {
        use base64::Engine;
        use base64::engine::general_purpose::STANDARD_NO_PAD;
        if let Some(cached) = self.key_cache.lock().await.get(key_id) {
            return Ok(cached);
        }
        let key_opt = self.backend.get_sync_key(key_id).await?;
        let key = key_opt
            .ok_or_else(|| AppStateSyncError::KeyNotFound(STANDARD_NO_PAD.encode(key_id)))?;
        let expanded = Arc::new(expand_app_state_keys(&key.key_data));
        self.key_cache
            .lock()
            .await
            .insert(key_id.to_vec(), expanded.clone());
        Ok(expanded)
    }

    /// Rebuild a collection from what the primary device sent back.
    ///
    /// The escalation for a snapshot this side cannot validate. The server's
    /// snapshot is signed with a MAC we recompute, and when the two disagree
    /// there is no way forward through the server. The records here carry their
    /// `SyncActionData` in the clear, so nothing is decrypted; the app-state key
    /// is needed only to re-derive each index MAC.
    ///
    /// `still_current` is asked once more, immediately before the first write.
    pub async fn apply_snapshot_recovery(
        &self,
        recovery: wa::SyncdSnapshotRecovery,
        expected_collection: &str,
        still_current: &(dyn Fn() -> bool + Send + Sync),
    ) -> Result<RecoveryOutcome> {
        let name = recovery
            .collection_name
            .as_deref()
            .ok_or_else(|| anyhow!("snapshot recovery names no collection"))?;
        if name != expected_collection {
            return Err(anyhow!(
                "snapshot recovery is for {name}, not the {expected_collection} that was asked for"
            ));
        }

        let version = recovery
            .version
            .as_option()
            .and_then(|v| v.version)
            .ok_or_else(|| anyhow!("snapshot recovery for {name} carries no version"))?;

        // Asked here, before the ltHash is read and before a single record is
        // looked at. Nothing waits for this reply, so the collection can have
        // moved on while the phone was answering; rolling it back would undo
        // real state and leave the next patch conflicting with a baseline
        // nobody is at.
        let held = self.backend.get_version(name).await?.version;
        if snapshot_is_stale(held, version) {
            return Ok(RecoveryOutcome::Stale {
                held,
                offered: version,
            });
        }

        let lthash = recovery
            .collection_lthash
            .as_deref()
            .ok_or_else(|| anyhow!("snapshot recovery for {name} carries no ltHash"))?;
        let hash: [u8; 128] = lthash.try_into().map_err(|_| {
            anyhow!(
                "snapshot recovery for {name} carries a {}-byte ltHash, not 128",
                lthash.len()
            )
        })?;

        // The keys first, because looking one up may reach the store and the
        // work below may not. Distinct ids only.
        let mut keys_by_id: HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>> = HashMap::new();
        for (i, record) in recovery.mutation_records.iter().enumerate() {
            let key_id = record
                .key_id
                .as_deref()
                .ok_or_else(|| anyhow!("recovery record {i} of {name} carries no key id"))?;
            if !keys_by_id.contains_key(key_id) {
                let keys = self.get_app_state_key(key_id).await?;
                keys_by_id.insert(key_id.to_vec(), keys);
            }
        }

        // And the rest off the caller's thread: a primary may send a whole
        // collection, and this is a JSON parse, an action clone and an HMAC per
        // record.
        let owned_name = name.to_string();
        let (mutations, macs) = crate::runtime::blocking(&*self.runtime, move || {
            // One winner per index -- the last record for it -- which is the
            // rule the snapshot path already applies: the MAC store keeps a
            // single value per index. Keyed by the index *and the key id*,
            // because that pair is what the stored index MAC is derived from.
            let winners: Vec<bool> = {
                let mut winners = vec![true; recovery.mutation_records.len()];
                let mut seen: HashSet<(&[u8], &[u8])> = HashSet::new();
                // Backwards, so the first hit for an index is its last record.
                for (i, record) in recovery.mutation_records.iter().enumerate().rev() {
                    let Some(index) = record.value.as_option().and_then(|v| v.index.as_deref())
                    else {
                        continue;
                    };
                    let Some(key_id) = record.key_id.as_deref() else {
                        continue;
                    };
                    if !seen.insert((index, key_id)) {
                        winners[i] = false;
                    }
                }
                winners
            };

            let mut mutations = Vec::with_capacity(recovery.mutation_records.len());
            let mut macs = Vec::with_capacity(recovery.mutation_records.len());
            for (i, record) in recovery.mutation_records.into_iter().enumerate() {
                if !winners[i] {
                    continue;
                }
                let action = record.value.into_option().ok_or_else(|| {
                    anyhow!("recovery record {i} of {owned_name} carries no value")
                })?;
                // The payload itself, not just the envelope around it. A record
                // whose `SyncActionData` omits its value produces a mutation
                // with nothing in it, and the update is gone while the MAC and
                // version are committed.
                if action.value.as_option().is_none() {
                    return Err(anyhow!(
                        "recovery record {i} of {owned_name} carries no action value"
                    ));
                }
                let key_id = record.key_id.ok_or_else(|| {
                    anyhow!("recovery record {i} of {owned_name} carries no key id")
                })?;
                // The record's own MAC is the value MAC the store keeps: the
                // primary sends what it computed over the encrypted form.
                let value_mac = record
                    .mac
                    .ok_or_else(|| anyhow!("recovery record {i} of {owned_name} carries no MAC"))?;
                if value_mac.len() != 32 {
                    return Err(anyhow!(
                        "recovery record {i} of {owned_name} carries a {}-byte MAC, not 32",
                        value_mac.len()
                    ));
                }
                let index = action.index.ok_or_else(|| {
                    anyhow!("recovery record {i} of {owned_name} carries no index")
                })?;
                let keys = keys_by_id.get(&key_id).ok_or_else(|| {
                    anyhow!("recovery record {i} of {owned_name} names an unexpanded key")
                })?;

                let index_mac = generate_index_mac(&index, &keys.index);
                macs.push(AppStateMutationMAC {
                    index_mac: index_mac.clone(),
                    value_mac: value_mac.clone(),
                });
                // Not `unwrap_or_default()`. An index that does not parse would
                // become an empty vec, the dispatcher returns early on one, and
                // the record would reach no consumer -- while its MAC was
                // committed and the version written.
                let parsed_index: Vec<String> = serde_json::from_slice(&index).map_err(|e| {
                    anyhow!("recovery record {i} of {owned_name} has an unreadable index: {e}")
                })?;
                if parsed_index.is_empty() {
                    return Err(anyhow!(
                        "recovery record {i} of {owned_name} carries an empty index"
                    ));
                }

                mutations.push(Mutation {
                    action_value: action.value.into_option(),
                    index_mac,
                    value_mac,
                    index: parsed_index,
                    // A recovered collection is what exists, so every record is
                    // a SET -- there is nothing left to remove.
                    operation: wa::syncd_mutation::SyncdOperation::SET,
                });
            }
            Ok::<_, anyhow::Error>((mutations, macs))
        })
        .await?;

        // Asked again here, with the reads and the record work behind us and the
        // first write in front.
        if !still_current() {
            return Ok(RecoveryOutcome::Retired);
        }

        // Same order the snapshot path commits in: the version goes last, so a
        // store error part-way leaves a collection that will be recovered again
        // rather than one whose version says it is current over MACs that are
        // gone.
        self.backend.clear_mutation_macs(name).await?;
        if !macs.is_empty() {
            self.backend.put_mutation_macs(name, version, &macs).await?;
        }
        self.persist_mutation_effects(&mutations).await?;
        self.backend
            .set_version(
                name,
                HashState {
                    version,
                    hash,
                    bootstrapped: true,
                    // The snapshot that could not be validated is the reason we
                    // are here; the collection the primary just handed over is
                    // not in that state.
                    mac_mismatch_fatal: false,
                    ..Default::default()
                },
            )
            .await?;

        Ok(RecoveryOutcome::Applied(mutations))
    }

    /// Drop every cached key, so the next access re-expands from the backend.
    pub async fn clear_key_cache(&self) {
        *self.key_cache.lock().await = KeyCache::default();
    }

    /// Expanded app-state keys held in memory, for the memory report.
    /// Bounded by `KeyCache::CAPACITY`.
    pub async fn cached_key_count(&self) -> usize {
        self.key_cache.lock().await.len()
    }

    /// Every key a patch list references, expanded, as the one map its blocking
    /// snapshot and patch closures look keys up in.
    ///
    /// Returned rather than read back out of `key_cache`: that cache is bounded,
    /// so a list referencing more keys than it holds would have evicted its first
    /// keys by the time its last was fetched.
    async fn prefetch_keys(
        &self,
        pl: &PatchList,
    ) -> Result<HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>> {
        let key_ids = collect_key_ids_from_patch_list(pl.snapshot.as_ref(), &pl.patches);
        let mut keys = HashMap::with_capacity(key_ids.len());
        for key_id in key_ids {
            match self.get_app_state_key(&key_id).await {
                Ok(expanded) => {
                    keys.insert(key_id, expanded);
                }
                Err(AppStateSyncError::KeyNotFound(_)) => {}
                Err(error) => return Err(error.into()),
            }
        }
        Ok(keys)
    }

    /// Process an already-parsed single PatchList by downloading external blobs,
    /// then decoding and applying it.
    pub async fn process_parsed_patch_list(
        &self,
        pl: PatchList,
        download: &BlobDownloadFn<'_>,
        validate_macs: bool,
    ) -> Result<(Vec<Mutation>, HashState, PatchList)> {
        let prepared = self.prepare_patch_list(pl, download).await;
        self.process_prepared_patch_list(prepared, validate_macs)
            .await
    }

    /// Validate and inline an owned list on the runtime's blocking executor.
    /// Cache lookups remain borrowed callbacks; parsing and validation own their
    /// inputs across the worker handoff and never clone encrypted payloads.
    pub async fn prepare_patch_list(
        &self,
        pl: PatchList,
        download: &BlobDownloadFn<'_>,
    ) -> PreparedPatchList {
        if pl.error.is_some() {
            return PreparedPatchList {
                list: pl,
                error: None,
            };
        }
        let preparing = {
            let validated = crate::runtime::blocking(&*self.runtime, move || {
                MetadataValidatedPatchList::validate(pl)
            })
            .await;
            Box::pin(self.prepare_metadata_validated_patch_list(validated, download))
        };
        preparing.await
    }

    /// Inline downloaded content without revalidating the immutable wire metadata.
    /// Decoded snapshots/mutations and batch rules are still checked on the worker.
    pub async fn prepare_metadata_validated_patch_list(
        &self,
        validated: MetadataValidatedPatchList,
        download: &BlobDownloadFn<'_>,
    ) -> PreparedPatchList {
        if !validated.can_download() {
            return PreparedPatchList {
                list: validated.list,
                error: validated.error,
            };
        }
        let MetadataValidatedPatchList { list: pl, .. } = validated;
        let mut blobs = Vec::new();
        for reference in pl
            .snapshot_ref
            .iter()
            .filter(|_| pl.snapshot.is_none())
            .chain(
                pl.patches
                    .iter()
                    .filter_map(|patch| patch.external_mutations.as_option()),
            )
        {
            match download(reference) {
                Ok(bytes) => {
                    blobs.push(bytes);
                }
                Err(error) => {
                    return PreparedPatchList {
                        list: pl,
                        error: Some(error),
                    };
                }
            }
        }
        crate::runtime::blocking(&*self.runtime, move || {
            let mut pl = pl;
            // Snapshot, then patch references, in the same order used above.
            // Do not merge distinct references merely because their paths match.
            let position = std::sync::atomic::AtomicUsize::new(0);
            let resolver = |_: &wa::ExternalBlobReference| {
                blobs
                    .get(position.fetch_add(1, std::sync::atomic::Ordering::Relaxed))
                    .cloned()
                    .ok_or_else(|| anyhow!("external blob was not resolved"))
            };
            let error = decode_external_blobs(&mut pl, &resolver, true).err();
            PreparedPatchList { list: pl, error }
        })
        .await
    }

    pub async fn process_prepared_patch_list(
        &self,
        prepared: PreparedPatchList,
        validate_macs: bool,
    ) -> Result<(Vec<Mutation>, HashState, PatchList)> {
        let PreparedPatchList { list, error } = prepared;
        if let Some(error) = error {
            return Err(error);
        }
        if list.error.is_some() {
            let state = self.backend.get_version(list.name.as_str()).await?;
            return Ok((Vec::new(), state, list));
        }
        Box::pin(self.process_patch_list(list, validate_macs)).await
    }

    /// Process already-parsed patch lists and download any external blobs.
    ///
    /// If a later collection fails after earlier collections were committed,
    /// the error downcasts to [`CommittedPatchListsError`]. Its results still
    /// require dispatch. The source retains any [`CommittedMutationsError`]
    /// raised within the failed collection.
    pub async fn process_patch_lists(
        &self,
        patch_lists: Vec<PatchList>,
        download: &BlobDownloadFn<'_>,
        validate_macs: bool,
    ) -> Result<Vec<(Vec<Mutation>, HashState, PatchList)>> {
        let mut results = Vec::with_capacity(patch_lists.len());

        for pl in patch_lists {
            match self
                .process_one_patch_list(pl, download, validate_macs)
                .await
            {
                Ok(result) => results.push(result),
                Err(source) if results.is_empty() => return Err(source),
                Err(source) => {
                    return Err(CommittedPatchListsError { results, source }.into());
                }
            }
        }

        Ok(results)
    }

    /// One collection's share of [`Self::process_patch_lists`], exposed on its own.
    ///
    /// Each call persists what it applies, so a caller that must not write past
    /// some boundary needs to be able to stop between collections rather than
    /// hand over a batch it can no longer take back.
    pub async fn process_one_patch_list(
        &self,
        pl: PatchList,
        download: &BlobDownloadFn<'_>,
        validate_macs: bool,
    ) -> Result<(Vec<Mutation>, HashState, PatchList)> {
        let prepared = self.prepare_patch_list(pl, download).await;
        self.process_one_prepared_patch_list(prepared, validate_macs)
            .await
    }

    pub async fn process_one_prepared_patch_list(
        &self,
        prepared: PreparedPatchList,
        validate_macs: bool,
    ) -> Result<(Vec<Mutation>, HashState, PatchList)> {
        let PreparedPatchList {
            list: mut pl,
            error,
        } = prepared;
        // Skip collections with errors — caller handles them via pl.error
        if pl.error.is_some() {
            let state = self.backend.get_version(pl.name.as_str()).await?;
            return Ok((Vec::new(), state, pl));
        }

        // A failed external-blob fetch must not advance the version with an empty
        // patch (silent data loss). Mark the collection retryable and skip it, so
        // the caller re-fetches it instead of persisting partial state.
        if let Some(e) = error {
            if matches!(
                e.downcast_ref::<crate::appstate::AppStateError>(),
                Some(
                    crate::appstate::AppStateError::SnapshotMACMissing
                        | crate::appstate::AppStateError::SnapshotMACMismatch
                )
            ) {
                return Err(e);
            }
            log::warn!(target: "AppState", "App-state preparation failed for {:?}, will refetch: {e:#}", pl.name);
            pl.error = Some(CollectionSyncError::Retry {
                code: 0,
                text: e.to_string(),
            });
            let state = self.backend.get_version(pl.name.as_str()).await?;
            return Ok((Vec::new(), state, pl));
        }

        Box::pin(self.process_patch_list(pl, validate_macs)).await
    }

    /// Process one patch list whose external blobs have already been inlined.
    ///
    /// If a later patch fails after earlier mutations were committed, the error
    /// downcasts to [`CommittedMutationsError`]. Its mutations still require
    /// dispatch, and its state is the last committed cursor.
    pub async fn process_patch_list(
        &self,
        mut pl: PatchList,
        validate_macs: bool,
    ) -> Result<(Vec<Mutation>, HashState, PatchList)> {
        // Arc so each blocking closure's handoff is a refcount bump, not a map copy.
        let keys_map = Arc::new(self.prefetch_keys(&pl).await?);

        let state = self.backend.get_version(pl.name.as_str()).await?;
        let mut progress = Box::new(PatchListProgress {
            committed_state: state.clone(),
            state,
            new_mutations: Vec::new(),
        });
        let collection = pl.name;
        let collection_name = pl.name.as_str();

        // Process snapshot if present, unless it is stale. WA Web applies a
        // snapshot only when it is strictly newer than the persisted version; a
        // stale or replayed snapshot is discarded so it can't roll the
        // collection backward.
        let snapshot_fresh = pl.snapshot.as_ref().is_some_and(|snapshot| {
            let snapshot_version = snapshot
                .version
                .as_option()
                .and_then(|v| v.version)
                .unwrap_or(0);
            if snapshot_is_stale(progress.state.version, snapshot_version) {
                log::warn!(
                    target: "AppState",
                    "Skipping stale snapshot for {collection_name}: incoming v{snapshot_version} <= persisted v{}",
                    progress.state.version
                );
                return false;
            }
            true
        });
        if snapshot_fresh && let Some(snapshot) = pl.snapshot.take() {
            Box::pin(self.apply_patch_list_snapshot(
                snapshot,
                &mut pl,
                keys_map.clone(),
                &mut progress,
                validate_macs,
            ))
            .await?;
        }

        // WA Web AntiTampering: an unsynced collection (empty ltHash) can only be
        // seeded by a snapshot or the genesis patch (version 1). If no snapshot
        // was applied and the first patch is non-genesis, applying it would
        // anchor the aggregate ltHash to nothing and persist unverified
        // mutations, then advance the version so the next sync no longer
        // requests a snapshot.
        if progress.state.version == 0 && progress.state.hash == [0u8; 128] {
            let first_version = pl
                .patches
                .first()
                .and_then(|p| p.version.as_option())
                .and_then(|v| v.version)
                .unwrap_or(0);
            if !pl.patches.is_empty() && first_version != 1 {
                log::warn!(
                    target: "AppState",
                    "Collection {collection_name} has empty ltHash and a non-genesis first patch v{first_version} without a snapshot; will refetch"
                );
                pl.error = Some(CollectionSyncError::Retry {
                    code: 0,
                    text: "empty lthash".to_string(),
                });
                return Ok((progress.new_mutations, progress.state, pl));
            }
            // Reached here with an empty baseline and no snapshot applied: a
            // genesis patch (v1), or no patches. Any mutation MACs still on disk
            // are from a prior, now-reset state, so wipe them before the genesis
            // patch runs, or its ltHash would be anchored to stale
            // index->value entries.
            if let Err(error) = self.backend.clear_mutation_macs(collection_name).await {
                return Err(committed_mutations_error(
                    error.into(),
                    progress.new_mutations,
                    &progress.committed_state,
                    collection,
                ));
            }
        }

        let patches = std::mem::take(&mut pl.patches);
        let mut processed_patches = Vec::with_capacity(patches.len());
        for patch in patches {
            match Box::pin(self.apply_patch_list_patch(
                patch,
                collection_name,
                keys_map.clone(),
                &mut progress,
                validate_macs,
                !pl.has_more_patches,
            ))
            .await
            {
                Ok(patch) => processed_patches.push(patch),
                Err(error) => {
                    return Err(committed_mutations_error(
                        error,
                        progress.new_mutations,
                        &progress.committed_state,
                        collection,
                    ));
                }
            }
        }
        pl.patches = processed_patches;

        // Handle case where we only have a snapshot and no patches
        if pl.patches.is_empty() && pl.snapshot.is_some() {
            progress.state.bootstrapped |= !pl.has_more_patches;
            if let Err(error) = self
                .backend
                .set_version(collection_name, progress.state.clone())
                .await
            {
                return Err(committed_mutations_error(
                    error.into(),
                    progress.new_mutations,
                    &progress.committed_state,
                    collection,
                ));
            }
        } else if pl.patches.is_empty()
            && !pl.has_more_patches
            && pl.snapshot_ref.is_none()
            && !progress.state.bootstrapped
            && pl.error.is_none()
        {
            // The server answered with nothing to apply and nothing still to
            // come, so the collection is at its head, and that finishes a
            // bootstrap whether or not there was a baseline under it.
            //
            // Without one, this is WA Web's "sync X but there are no updates"
            // branch; the record is what stops the next sync asking for the
            // snapshot again. With one, only the flag is missing: a row written
            // before it existed, or one a migration reset.
            //
            // `has_more_patches` and an undownloaded `snapshot_ref` both mean
            // this is a page rather than the whole answer, so neither records
            // zero.
            progress.state.bootstrapped = true;
            self.backend
                .set_version(collection_name, progress.state.clone())
                .await?;
        }

        Ok((progress.new_mutations, progress.state, pl))
    }

    /// Build and encode a `SyncdPatch` for sending mutations to the server.
    ///
    /// Takes a list of pre-encoded mutations (from `appstate::encode_record`)
    /// and produces the protobuf-encoded patch bytes ready for inclusion in an
    /// IQ stanza.
    ///
    /// Returns `(patch_bytes, base_version)` where `base_version` is the
    /// collection version before the patch (for the IQ `version` attribute).
    /// Does NOT persist state — the caller must only persist after the server
    /// acknowledges the patch.
    pub async fn build_patch(
        &self,
        collection_name: &str,
        mutations: Vec<wa::SyncdMutation>,
    ) -> Result<(Vec<u8>, u64)> {
        use crate::appstate::hash::generate_patch_mac;

        // Get active key
        let key_id = self
            .backend
            .get_latest_sync_key_id()
            .await?
            .ok_or_else(|| anyhow!("No app state sync key available"))?;
        let keys = self.get_app_state_key(&key_id).await?;

        // Get current hash state — save base version for the caller
        let mut state = self.backend.get_version(collection_name).await?;
        let base_version = state.version;

        // Pre-fetch previous value MACs in one backend round-trip, mirroring
        // the inbound patch path: one batched query instead of a
        // spawn_blocking + single-row SELECT per mutation.
        let need_db_lookup = collect_unique_index_macs(&mutations);
        let db_prev: HashMap<IndexMac, Vec<u8>> = self
            .backend
            .get_mutation_macs(collection_name, &need_db_lookup)
            .await?;

        // Update hash state
        let (_, hash_result) = state.update_hash(&mutations, |index_mac, _| {
            Ok(<&IndexMac>::try_from(index_mac)
                .ok()
                .and_then(|k| db_prev.get(k))
                .cloned())
        });
        hash_result?;

        state.version += 1;

        // Generate snapshot MAC
        let snapshot_mac = state.generate_snapshot_mac(collection_name, &keys.snapshot_mac);

        // Build the patch — matching whatsmeow: no Version or DeviceIndex fields
        let mut patch = wa::SyncdPatch::default();
        patch.snapshot_mac = Some(snapshot_mac);
        let mut id = wa::KeyId::default();
        id.id = Some(key_id.clone());
        patch.key_id = id.into();
        patch.mutations = mutations;

        // Generate and set patch MAC
        let patch_mac = generate_patch_mac(&patch, collection_name, &keys.patch_mac, state.version);
        patch.patch_mac = Some(patch_mac);

        // Encode to protobuf
        let patch_bytes = waproto::codec::syncd_patch_to_vec(&patch);

        Ok((patch_bytes, base_version))
    }

    pub async fn get_missing_key_ids(&self, pl: &PatchList) -> Result<Vec<Vec<u8>>> {
        let key_ids = collect_key_ids_from_patch_list(pl.snapshot.as_ref(), &pl.patches);
        let mut missing = Vec::with_capacity(key_ids.len());
        for id in key_ids {
            if self.backend.get_sync_key(&id).await?.is_none() {
                missing.push(id);
            }
        }
        Ok(missing)
    }

    async fn apply_patch_list_snapshot(
        &self,
        snapshot: wa::SyncdSnapshot,
        pl: &mut PatchList,
        keys_map: Arc<HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>>,
        progress: &mut PatchListProgress,
        validate_macs: bool,
    ) -> Result<()> {
        let collection_name = pl.name.as_str();
        let collection_name_owned = collection_name.to_string();

        // Offload CPU-intensive snapshot processing to a blocking thread. The
        // snapshot moves into the closure (its 'static bound used to force a
        // multi-MB deep clone on bootstrap) and comes back via the return tuple
        // because the caller still reads pl.snapshot (get_missing_key_ids).
        let result = crate::runtime::blocking(&*self.runtime, move || {
            preparation::prepare_snapshot(
                snapshot,
                &keys_map,
                validate_macs,
                &collection_name_owned,
            )
        })
        .await
        // Carried as itself rather than flattened to a string: a snapshot MAC
        // that does not match is the one error with an answer -- asking the
        // primary for the collection.
        .map_err(anyhow::Error::new)?;

        let (snapshot, snapshot_state, snapshot_result) = result.into_parts();
        pl.snapshot = Some(snapshot);
        progress.state = snapshot_state;

        // A snapshot is a fresh baseline, so wipe the collection's prior mutation
        // MACs first, even when the snapshot has none. Leftover index->value
        // entries would corrupt the next patch's ltHash.
        //
        // Commit the version LAST. If clear/put fails on a transient store error,
        // an already-advanced version would make the retry treat this same
        // snapshot as stale and skip it, stranding the old MACs forever.
        self.backend.clear_mutation_macs(collection_name).await?;
        if !snapshot_result.mutation_macs.is_empty() {
            self.backend
                .put_mutation_macs(
                    collection_name,
                    progress.state.version,
                    &snapshot_result.mutation_macs,
                )
                .await?;
        }
        progress.state.bootstrapped |= !pl.has_more_patches;
        self.persist_mutation_effects(&snapshot_result.mutations)
            .await?;
        self.backend
            .set_version(collection_name, progress.state.clone())
            .await?;
        // Snapshot owns the whole collection: move its Vec into the empty
        // accumulator rather than extend, which would allocate + copy a second
        // collection-sized buffer at the memory peak.
        if progress.new_mutations.is_empty() {
            progress.new_mutations = snapshot_result.mutations;
        } else {
            progress.new_mutations.extend(snapshot_result.mutations);
        }
        progress.committed_state = progress.state.clone();
        Ok(())
    }

    async fn apply_patch_list_patch(
        &self,
        patch: wa::SyncdPatch,
        collection_name: &str,
        keys_map: Arc<HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>>,
        progress: &mut PatchListProgress,
        validate_macs: bool,
        final_page: bool,
    ) -> Result<wa::SyncdPatch> {
        let need_db_lookup = collect_unique_index_macs(&patch.mutations);

        // Fetch previous value MACs in one backend round-trip instead of a
        // spawn_blocking + query per mutation (N+1).
        let db_prev: HashMap<IndexMac, Vec<u8>> = self
            .backend
            .get_mutation_macs(collection_name, &need_db_lookup)
            .await?;

        let state_clone = progress.state.clone();
        let keys = keys_map;
        let coll = collection_name.to_string();

        // Offload CPU-intensive patch processing to a blocking thread
        let processed = crate::runtime::blocking(&*self.runtime, move || {
            preparation::prepare_patch(
                patch,
                state_clone,
                &keys,
                &db_prev,
                validate_macs,
                &coll,
                false,
            )
        })
        .await;
        let mut prepared = match processed {
            Ok(prepared) => prepared,
            Err(error) => return Err(anyhow!("{}", error)),
        };
        if final_page {
            prepared.mark_bootstrapped();
        }
        self.persist_mutation_effects(&prepared.result().mutations)
            .await?;
        self.backend
            .commit_patch(
                collection_name,
                prepared.result().state.clone(),
                &prepared.result().removed_index_macs,
                &prepared.result().added_macs,
            )
            .await?;
        Ok(preparation::publish_patch(
            prepared,
            &mut progress.state,
            &mut progress.committed_state,
            &mut progress.new_mutations,
        ))
    }

    /// Inline the patch list's external blobs, then report which referenced decode keys
    /// are absent. Inlining first is load-bearing: the SNAPSHOT's `key_id` lives inside
    /// its external blob, so [`get_missing_key_ids`](Self::get_missing_key_ids) alone (called
    /// before download) can't see it and would miss the snapshot's key.
    pub async fn missing_key_ids_after_inline(
        &self,
        pl: &mut PatchList,
        download: &BlobDownloadFn<'_>,
    ) -> Result<Vec<Vec<u8>>> {
        // Check the original metadata before inlining can erase conflicting
        // inline mutations. Owned runtime callers use blocking preparation.
        validate_patch_list_metadata(pl)?;
        decode_external_blobs(pl, download, false)?;
        self.get_missing_key_ids(pl).await
    }
}

/// A snapshot is stale when the collection already holds a version at or beyond the
/// incoming snapshot's; WA Web discards it ("skip applying syncd old version") rather
/// than rolling the collection backward. The `persisted_version > 0` guard keeps the
/// benign first-sync path (snapshots are requested only at version 0) unaffected.
pub fn snapshot_is_stale(persisted_version: u64, snapshot_version: u64) -> bool {
    persisted_version > 0 && snapshot_version <= persisted_version
}

#[cfg(test)]
mod snapshot_guard_tests {
    use super::snapshot_is_stale;

    #[test]
    fn first_sync_is_never_stale() {
        // Benign path: nothing persisted yet (version 0), so any snapshot applies.
        assert!(!snapshot_is_stale(0, 1));
        assert!(!snapshot_is_stale(0, 0));
    }

    #[test]
    fn newer_snapshot_applies() {
        assert!(!snapshot_is_stale(5, 6));
    }

    #[test]
    fn equal_or_older_snapshot_is_stale() {
        // WA Web's `a.version >= t` skips equal versions too.
        assert!(snapshot_is_stale(5, 5));
        assert!(snapshot_is_stale(5, 3));
        assert!(snapshot_is_stale(5, 0));
    }
}

#[cfg(test)]
mod external_blob_tests {
    use super::*;
    use crate::appstate::patch_decode::WAPatchName;

    fn valid_reference(path: &str) -> wa::ExternalBlobReference {
        let mut reference = wa::ExternalBlobReference::default();
        reference.direct_path = Some(path.into());
        reference.media_key = Some(vec![]);
        reference.file_sha256 = Some(vec![]);
        reference.file_enc_sha256 = Some(vec![]);
        reference
    }
    fn valid_external_patch() -> wa::SyncdPatch {
        let mut patch = wa::SyncdPatch::default();
        let mut version = wa::SyncdVersion::default();
        version.version = Some(1);
        patch.version = version.into();
        patch.snapshot_mac = Some(vec![]);
        patch.patch_mac = Some(vec![]);
        let mut key = wa::KeyId::default();
        key.id = Some(vec![]);
        patch.key_id = key.into();
        patch.external_mutations = valid_reference("/mutations").into();
        patch
    }

    fn pl_with_snapshot_ref(snapshot_ref: Option<wa::ExternalBlobReference>) -> PatchList {
        PatchList {
            name: WAPatchName::Regular,
            has_more_patches: false,
            patches: Vec::new(),
            snapshot: None,
            snapshot_ref,
            error: None,
        }
    }

    #[test]
    fn external_snapshot_download_failure_propagates() {
        // A referenced blob that fails to download must error, not be swallowed
        // (which would apply an empty patch and advance the version).
        let mut pl = pl_with_snapshot_ref(Some(valid_reference("/blob")));
        let download = |_: &wa::ExternalBlobReference| -> Result<bytes::Bytes> {
            Err(anyhow!("simulated failure"))
        };
        let error = download_external_blobs(&mut pl, &download).unwrap_err();
        assert!(format!("{error:#}").contains("simulated failure"));
    }

    #[test]
    fn external_snapshot_decode_failure_propagates() {
        // Download succeeds but the bytes aren't a valid SyncdSnapshot: the decode
        // error must propagate too, not just download errors.
        let mut pl = pl_with_snapshot_ref(Some(valid_reference("/blob")));
        let download = |_: &wa::ExternalBlobReference| -> Result<bytes::Bytes> {
            Ok(bytes::Bytes::from_static(&[0xFF, 0xFF, 0xFF]))
        };
        let error = download_external_blobs(&mut pl, &download).unwrap_err();
        assert!(format!("{error:#}").contains("decode external snapshot"));
    }

    #[test]
    fn external_mutation_download_failure_propagates() {
        // The patch-level external_mutations path must propagate failures as well.
        let mut pl = PatchList {
            name: WAPatchName::Regular,
            has_more_patches: false,
            patches: vec![valid_external_patch()],
            snapshot: None,
            snapshot_ref: None,
            error: None,
        };
        let download = |_: &wa::ExternalBlobReference| -> Result<bytes::Bytes> {
            Err(anyhow!("simulated failure"))
        };
        let error = download_external_blobs(&mut pl, &download).unwrap_err();
        assert!(format!("{error:#}").contains("simulated failure"));
    }

    /// The missing-key probe inlines before `process_one_patch_list` inlines
    /// again; the second pass must find nothing left to fetch.
    #[test]
    fn external_mutations_are_fetched_once() {
        let mut pl = PatchList {
            name: WAPatchName::Regular,
            has_more_patches: false,
            patches: vec![valid_external_patch()],
            snapshot: None,
            snapshot_ref: None,
            error: None,
        };
        let fetches = std::sync::atomic::AtomicUsize::new(0);
        // An empty buffer decodes as a `SyncdMutations` with no mutations.
        let download = |_: &wa::ExternalBlobReference| -> Result<bytes::Bytes> {
            fetches.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
            Ok(bytes::Bytes::new())
        };
        download_external_blobs(&mut pl, &download).expect("first inline");
        download_external_blobs(&mut pl, &download).expect("second inline");
        assert_eq!(fetches.load(std::sync::atomic::Ordering::Relaxed), 1);
        assert!(pl.patches[0].external_mutations.is_unset());
    }

    #[test]
    fn no_external_refs_is_ok() {
        let mut pl = pl_with_snapshot_ref(None);
        let download =
            |_: &wa::ExternalBlobReference| -> Result<bytes::Bytes> { Ok(bytes::Bytes::new()) };
        assert!(download_external_blobs(&mut pl, &download).is_ok());
    }

    #[test]
    fn conflicting_mutation_sources_are_rejected_before_downloading() {
        let mut pl = pl_with_snapshot_ref(None);
        let mut patch = valid_external_patch();
        patch.mutations.push(wa::SyncdMutation::default());
        pl.patches.push(patch);
        let download = |_: &wa::ExternalBlobReference| -> Result<bytes::Bytes> {
            panic!("invalid patch reached downloader")
        };
        let error = download_external_blobs(&mut pl, &download).unwrap_err();
        assert_eq!(
            error
                .downcast_ref::<crate::appstate::validation::ValidationError>()
                .unwrap()
                .metric
                .as_deref(),
            Some("PATCH_WITH_BOTH_INLINE_AND_EXTERNAL_MUTATIONS")
        );
        assert!(pl.patches[0].external_mutations.is_set());
    }
}

#[cfg(test)]
mod dedup_tests {
    use super::*;

    fn mutation(index_mac: &[u8]) -> wa::SyncdMutation {
        let mut proto = wa::SyncdMutation::default();
        proto.record = buffa::MessageField::some({
            let mut proto = wa::SyncdRecord::default();
            proto.index = buffa::MessageField::some({
                let mut proto = wa::SyncdIndex::default();
                proto.blob = Some(index_mac.to_vec());
                proto
            });
            proto
        });
        proto
    }

    /// Builds `n` mutations whose index MACs repeat every `distinct` values, so
    /// the distinct set is the first `distinct` MACs.
    fn build(n: usize, distinct: usize) -> Vec<wa::SyncdMutation> {
        (0..n)
            .map(|i| {
                let mut mac = vec![0u8; 32];
                mac[..8].copy_from_slice(&((i % distinct) as u64).to_le_bytes());
                mutation(&mac)
            })
            .collect()
    }

    fn mac_bytes(i: usize) -> IndexMac {
        let mut mac = [0u8; 32];
        mac[..8].copy_from_slice(&(i as u64).to_le_bytes());
        mac
    }

    fn expected(distinct: usize) -> Vec<IndexMac> {
        (0..distinct).map(mac_bytes).collect()
    }

    /// Dedups to the distinct index MACs across small and large N, dropping
    /// repeats. Order is unspecified (callers feed a HashMap lookup), so compare
    /// as sorted sets.
    #[test]
    fn dedups_to_distinct_macs() {
        for &n in &[8usize, 64, 65, 1000] {
            let distinct = (n / 2).max(1);
            let mut got = collect_unique_index_macs(&build(n, distinct));
            got.sort_unstable();
            let mut want = expected(distinct);
            want.sort_unstable();
            assert_eq!(got, want, "n = {n}");
        }
    }

    #[test]
    fn skips_mutations_without_index_blob() {
        let mutations = vec![
            mutation(b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
            wa::SyncdMutation::default(),
            mutation(b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
            mutation(b"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
        ];
        let mut macs = collect_unique_index_macs(&mutations);
        macs.sort_unstable();
        assert_eq!(
            macs,
            vec![
                *b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                *b"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
            ]
        );
    }
}

#[cfg(test)]
mod key_cache_tests {
    use super::*;

    fn expanded(byte: u8) -> Arc<ExpandedAppStateKeys> {
        Arc::new(expand_app_state_keys(&[byte; 32]))
    }

    /// The cap is the whole bound: entries are never invalidated, only crowded
    /// out, and the oldest is the one that goes.
    #[test]
    fn the_key_cache_keeps_the_newest_entries_up_to_its_capacity() {
        let mut cache = KeyCache::default();
        for i in 0..(KeyCache::CAPACITY + 4) {
            cache.insert(vec![i as u8], expanded(i as u8));
        }

        assert_eq!(cache.len(), KeyCache::CAPACITY);
        for evicted in 0..4u8 {
            assert!(
                cache.get(&[evicted]).is_none(),
                "key {evicted} is one of the four oldest and must have been evicted"
            );
        }
        for kept in 4..(KeyCache::CAPACITY + 4) {
            assert!(
                cache.get(&[kept as u8]).is_some(),
                "key {kept} must be kept"
            );
        }
    }

    /// Re-expanding a key already held must not consume a second slot: the same
    /// key id can be looked up any number of times.
    #[test]
    fn re_inserting_a_key_id_does_not_grow_the_cache() {
        let mut cache = KeyCache::default();
        for _ in 0..(KeyCache::CAPACITY * 2) {
            cache.insert(vec![7], expanded(7));
        }

        assert_eq!(cache.len(), 1);
        assert!(cache.get(&[7]).is_some());
    }
}

#[cfg(test)]
mod metadata_validation_tests {
    use super::*;
    use crate::appstate::patch_decode::WAPatchName;

    fn key() -> wa::KeyId {
        let mut key = wa::KeyId::default();
        key.id = Some(vec![]);
        key
    }

    fn version(value: u64) -> wa::SyncdVersion {
        let mut version = wa::SyncdVersion::default();
        version.version = Some(value);
        version
    }

    fn reference() -> wa::ExternalBlobReference {
        let mut reference = wa::ExternalBlobReference::default();
        reference.direct_path = Some("/synthetic".into());
        reference.media_key = Some(vec![1; 32]);
        reference.file_sha256 = Some(vec![2; 32]);
        reference.file_enc_sha256 = Some(vec![3; 32]);
        reference
    }

    fn snapshot() -> wa::SyncdSnapshot {
        let mut snapshot = wa::SyncdSnapshot::default();
        snapshot.version = version(1).into();
        snapshot.mac = Some(vec![0; 32]);
        snapshot.key_id = key().into();
        snapshot
    }

    fn patch(value: u64) -> wa::SyncdPatch {
        let mut patch = wa::SyncdPatch::default();
        patch.version = version(value).into();
        patch.snapshot_mac = Some(vec![0; 32]);
        patch.patch_mac = Some(vec![0; 32]);
        patch.key_id = key().into();
        patch
    }

    fn list() -> PatchList {
        PatchList {
            name: WAPatchName::Regular,
            has_more_patches: false,
            patches: vec![patch(2)],
            snapshot: Some(snapshot()),
            snapshot_ref: None,
            error: None,
        }
    }

    fn metric(error: &anyhow::Error) -> Option<&str> {
        error
            .downcast_ref::<crate::appstate::validation::ValidationError>()
            .and_then(|error| error.metric.as_deref())
    }

    #[test]
    fn a_complete_patch_list_passes_metadata_validation() {
        validate_patch_list_metadata(&list()).expect("valid metadata");
    }

    #[test]
    fn conflicting_mutation_sources_are_rejected() {
        let mut pl = list();
        pl.patches[0].mutations.push(wa::SyncdMutation::default());
        pl.patches[0].external_mutations = reference().into();
        let error = validate_patch_list_metadata(&pl).unwrap_err();
        assert_eq!(
            metric(&error),
            Some("PATCH_WITH_BOTH_INLINE_AND_EXTERNAL_MUTATIONS")
        );
    }

    #[test]
    fn missing_patch_integrity_fields_are_rejected_in_order() {
        let mut pl = list();
        pl.patches[0].version = Default::default();
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_PATCH_VERSION")
        );

        pl = list();
        pl.patches[0].snapshot_mac = None;
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_PATCH_SNAPSHOT_MAC")
        );

        pl = list();
        pl.patches[0].patch_mac = None;
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_PATCH_MAC")
        );

        pl = list();
        pl.patches[0].key_id = Default::default();
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_PATCH_KEY_ID")
        );
    }

    #[test]
    fn missing_snapshot_integrity_fields_are_rejected() {
        let mut pl = list();
        pl.snapshot.as_mut().expect("snapshot").mac = None;
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_SNAPSHOT_MAC")
        );

        pl = list();
        pl.snapshot.as_mut().expect("snapshot").key_id = Default::default();
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_SNAPSHOT_KEY_ID")
        );
    }

    #[test]
    fn external_reference_requires_its_identity_fields() {
        let mut pl = list();
        pl.snapshot = None;
        pl.patches.clear();
        pl.snapshot_ref = Some(reference());
        pl.snapshot_ref.as_mut().expect("reference").media_key = None;
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_EXTERNAL_BLOB_REFERENCE_MEDIA_KEY")
        );

        pl.snapshot_ref = Some(reference());
        pl.snapshot_ref.as_mut().expect("reference").direct_path = None;
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_EXTERNAL_BLOB_REFERENCE_DIRECT_PATH")
        );

        pl.snapshot_ref = Some(reference());
        pl.snapshot_ref.as_mut().expect("reference").file_sha256 = None;
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_EXTERNAL_BLOB_REFERENCE_FILE_SHA256")
        );

        pl.snapshot_ref = Some(reference());
        pl.snapshot_ref.as_mut().expect("reference").file_enc_sha256 = None;
        assert_eq!(
            metric(&validate_patch_list_metadata(&pl).unwrap_err()),
            Some("MISSING_EXTERNAL_BLOB_REFERENCE_FILE_ENC_SHA256")
        );
    }

    #[test]
    fn duplicate_patch_versions_are_rejected_after_inlining() {
        let mut pl = list();
        pl.snapshot = None;
        pl.patches = vec![patch(2), patch(2)];
        // Metadata validation alone admits the list; the batch rule runs on the
        // decoded content, after inlining.
        assert!(validate_patch_list_metadata(&pl).is_ok());
        let download =
            |_: &wa::ExternalBlobReference| -> Result<bytes::Bytes> { Ok(bytes::Bytes::new()) };
        let error = download_external_blobs(&mut pl, &download).unwrap_err();
        assert_eq!(
            metric(&error),
            Some("DUPLICATE_PATCH_VERSION_IN_COLLECTION")
        );
    }

    #[test]
    fn metadata_validation_token_retains_the_error() {
        let mut pl = list();
        pl.patches[0].patch_mac = None;
        let validated = MetadataValidatedPatchList::validate(pl);
        assert!(!validated.can_download());
    }

    #[test]
    fn collection_errors_are_not_downloaded() {
        let mut pl = list();
        pl.error = Some(CollectionSyncError::Retry {
            code: 503,
            text: "synthetic retry".into(),
        });
        let validated = MetadataValidatedPatchList::validate(pl);
        assert!(!validated.can_download());
    }
}

#[cfg(test)]
mod committed_error_tests {
    use super::*;
    use crate::appstate::patch_decode::WAPatchName;

    fn mutation() -> Mutation {
        Mutation {
            action_value: None,
            index_mac: vec![1; 32],
            value_mac: vec![2; 32],
            index: vec!["archive".to_string()],
            operation: wa::syncd_mutation::SyncdOperation::SET,
        }
    }

    #[test]
    fn committed_mutations_error_parts_round_trip() {
        let state = HashState {
            version: 4,
            ..HashState::default()
        };
        let error = committed_mutations_error(
            anyhow!("later patch failed"),
            vec![mutation()],
            &state,
            WAPatchName::Regular,
        );
        let error = error
            .downcast::<CommittedMutationsError>()
            .expect("committed mutations error");
        let (mutations, committed, collection, source) = error.into_parts();
        assert_eq!(mutations.len(), 1);
        assert_eq!(committed.version, 4);
        assert_eq!(collection, WAPatchName::Regular);
        assert_eq!(source.to_string(), "later patch failed");
    }

    #[test]
    fn empty_mutations_leave_the_source_error_alone() {
        let error = committed_mutations_error(
            anyhow!("nothing committed"),
            Vec::new(),
            &HashState::default(),
            WAPatchName::Regular,
        );
        assert!(error.downcast_ref::<CommittedMutationsError>().is_none());
        assert_eq!(error.to_string(), "nothing committed");
    }

    #[test]
    fn committed_patch_lists_error_parts_round_trip() {
        let error = CommittedPatchListsError {
            results: vec![(
                vec![mutation()],
                HashState::default(),
                PatchList {
                    name: WAPatchName::Regular,
                    has_more_patches: false,
                    patches: Vec::new(),
                    snapshot: None,
                    snapshot_ref: None,
                    error: None,
                },
            )],
            source: anyhow!("later collection failed"),
        };
        let error: anyhow::Error = error.into();
        let error = error
            .downcast::<CommittedPatchListsError>()
            .expect("committed patch lists error");
        let (results, source) = error.into_parts();
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].0.len(), 1);
        assert_eq!(source.to_string(), "later collection failed");
    }
}

#[cfg(test)]
mod preparation_tests {
    use super::preparation::{PreparedPatch, publish_patch};
    use super::*;
    use crate::appstate::processor::PatchProcessingResult;

    #[test]
    fn invalid_version_does_not_prepare_a_patch_or_change_the_held_cursor() {
        let held = HashState {
            version: 7,
            ..HashState::default()
        };
        let patch = wa::SyncdPatch::default();
        assert!(
            preparation::prepare_patch(
                patch,
                held.clone(),
                &HashMap::new(),
                &HashMap::new(),
                true,
                "regular",
                true
            )
            .is_err()
        );
        assert_eq!(held.version, 7);
        assert!(!held.bootstrapped);
    }

    fn encrypted_record(
        keys: &ExpandedAppStateKeys,
        key_id: &[u8],
        index: &[&str],
    ) -> wa::SyncdRecord {
        use crate::libsignal::crypto::aes_256_cbc_encrypt_into;

        let mut action = wa::SyncActionData::default();
        action.index = Some(serde_json::to_vec(index).expect("index json"));
        action.value = wa::SyncActionValue::default().into();
        let plaintext = buffa::Message::encode_to_vec(&action);

        let iv = vec![0u8; 16];
        let mut ciphertext = Vec::new();
        aes_256_cbc_encrypt_into(&plaintext, &keys.value_encryption, &iv, &mut ciphertext)
            .expect("AES-CBC encryption");
        let mut value_blob = iv;
        value_blob.extend_from_slice(&ciphertext);
        let value_mac = crate::appstate::hash::generate_content_mac(
            wa::syncd_mutation::SyncdOperation::SET,
            &value_blob,
            key_id,
            &keys.value_mac,
        );
        value_blob.extend_from_slice(&value_mac);

        let mut record = wa::SyncdRecord::default();
        let mut record_index = wa::SyncdIndex::default();
        record_index.blob = Some(crate::appstate::hash::generate_index_mac(
            &serde_json::to_vec(index).expect("index json"),
            &keys.index,
        ));
        record.index = record_index.into();
        let mut value = wa::SyncdValue::default();
        value.blob = Some(value_blob);
        record.value = value.into();
        let mut id = wa::KeyId::default();
        id.id = Some(key_id.to_vec());
        record.key_id = id.into();
        record
    }

    fn keys() -> (HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>, Vec<u8>) {
        let key_id = b"synthetic-key".to_vec();
        let mut map = HashMap::new();
        map.insert(
            key_id.clone(),
            Arc::new(crate::appstate::expand_app_state_keys(&[7u8; 32])),
        );
        (map, key_id)
    }

    #[test]
    fn snapshot_preparation_decodes_records_and_reports_macs() {
        let (keys, key_id) = keys();
        let expanded = keys.get(&key_id).expect("key").clone();

        let mut snapshot = wa::SyncdSnapshot::default();
        let mut version = wa::SyncdVersion::default();
        version.version = Some(3);
        snapshot.version = version.into();
        snapshot
            .records
            .push(encrypted_record(&expanded, &key_id, &["archive"]));

        let prepared = preparation::prepare_snapshot(snapshot, &keys, false, "regular")
            .expect("snapshot prepares");
        let (snapshot, state, result) = prepared.into_parts();
        assert_eq!(snapshot.records.len(), 1);
        assert_eq!(state.version, 3);
        assert_eq!(result.mutations.len(), 1);
        assert_eq!(result.mutations[0].index, vec!["archive".to_string()]);
        assert_eq!(result.mutation_macs.len(), 1);
    }

    #[test]
    fn patch_preparation_advances_the_cursor_and_reports_macs() {
        let (keys, key_id) = keys();
        let expanded = keys.get(&key_id).expect("key").clone();

        let mut patch = wa::SyncdPatch::default();
        let mut version = wa::SyncdVersion::default();
        version.version = Some(1);
        patch.version = version.into();
        let mut id = wa::KeyId::default();
        id.id = Some(key_id.clone());
        patch.key_id = id.into();
        patch.mutations.push({
            let mut mutation = wa::SyncdMutation::default();
            mutation.operation = Some(wa::syncd_mutation::SyncdOperation::SET.into());
            mutation.record = encrypted_record(&expanded, &key_id, &["archive"]).into();
            mutation
        });

        let prepared = preparation::prepare_patch(
            patch,
            HashState::default(),
            &keys,
            &HashMap::new(),
            false,
            "regular",
            true,
        )
        .expect("patch prepares");
        assert_eq!(prepared.result().state.version, 1);
        assert!(prepared.result().state.bootstrapped);
        assert_eq!(prepared.result().mutations.len(), 1);
        assert_eq!(prepared.result().added_macs.len(), 1);
    }

    #[test]
    fn publication_moves_only_confirmed_results() {
        let mut state = HashState::default();
        let mut committed = state.clone();
        let mut mutations = Vec::new();
        let mut next = state.clone();
        next.version = 4;
        next.bootstrapped = true;
        let prepared = PreparedPatch {
            patch: wa::SyncdPatch::default(),
            result: PatchProcessingResult {
                state: next,
                mutations: Vec::new(),
                added_macs: Vec::new(),
                removed_index_macs: Vec::new(),
            },
        };
        assert_eq!(state.version, 0);
        publish_patch(prepared, &mut state, &mut committed, &mut mutations);
        assert_eq!(state.version, 4);
        assert_eq!(committed.version, 4);
        assert!(committed.bootstrapped);
    }
}

#[cfg(test)]
mod recovery_claim_tests {
    use super::*;

    fn registry() -> RecoveryRegistry {
        RecoveryRegistry::default()
    }

    #[test]
    fn a_new_ask_is_recorded_and_suppresses_another() {
        let mut registry = registry();
        assert!(registry.mark("regular", 0, None));
        assert!(!registry.mark("regular", 0, None));
        assert!(registry.has("regular"));
    }

    #[test]
    fn a_newer_generation_replaces_an_older_ask() {
        let mut registry = registry();
        assert!(registry.mark("regular", 1, None));
        assert!(registry.mark("regular", 2, None));
        assert!(registry.take("regular"));
        assert!(!registry.has("regular"));
    }

    #[test]
    fn an_ask_is_claimed_once_by_its_id() {
        let mut registry = registry();
        assert!(registry.mark("regular", 0, Some("req-1")));
        assert_eq!(
            registry.claim("req-1"),
            RecoveryClaim::Claimed("regular".into())
        );
        assert_eq!(
            registry.claim("req-1"),
            RecoveryClaim::AlreadyAnswering("regular".into())
        );
        assert_eq!(registry.claim("unknown"), RecoveryClaim::Unrequested);
    }

    #[test]
    fn claiming_with_generation_returns_the_marked_generation() {
        let mut registry = registry();
        assert!(registry.mark("regular", 9, Some("req-2")));
        assert_eq!(
            registry.claim_with_generation("req-2"),
            Some(("regular".into(), 9))
        );
        assert_eq!(registry.claim_with_generation("req-2"), None);
    }

    #[test]
    fn ending_an_answer_releases_the_ask() {
        let mut registry = registry();
        assert!(registry.mark("regular", 0, Some("req-3")));
        assert!(registry.claim("req-3") == RecoveryClaim::Claimed("regular".into()));
        assert!(registry.end_answer("regular", Some("req-3")));
        assert!(!registry.has("regular"));
    }

    #[test]
    fn ending_another_replys_claim_leaves_it_in_place() {
        let mut registry = registry();
        assert!(registry.mark("regular", 0, Some("req-4")));
        assert!(registry.claim("req-4") == RecoveryClaim::Claimed("regular".into()));
        assert!(!registry.end_answer("regular", Some("someone-else")));
        assert!(registry.has("regular"));
    }

    #[test]
    fn taking_an_ask_by_id_removes_it_even_when_undecodable() {
        let mut registry = registry();
        assert!(registry.mark("regular", 3, Some("req-5")));
        assert_eq!(registry.take_by_id("req-5"), Some(("regular".into(), 3)));
        assert!(!registry.has("regular"));
        assert_eq!(registry.take_by_id("req-5"), None);
    }

    #[test]
    fn a_request_id_can_be_recorded_after_the_ask() {
        let mut registry = registry();
        assert!(registry.mark("regular", 0, None));
        registry.note_id("regular", "req-6");
        assert_eq!(
            registry.claim("req-6"),
            RecoveryClaim::Claimed("regular".into())
        );
    }
}

#[cfg(test)]
mod async_engine_tests {
    use super::*;
    use crate::runtime::{AbortHandle, Runtime};
    use crate::store::error::Result as StoreResult;
    use crate::store::traits::{
        AppStateSyncKey, AppSyncStore, DeviceListRecord, DeviceStore, LidPnMappingEntry,
        MessageSecret, MsgSecretEntry, MsgSecretStore, ProtocolStore, SignalStore,
        StoredMessageSecret, merge_msg_secret_expiry, merge_msg_secret_message_ts,
    };
    use std::future::Future;
    use std::pin::Pin;
    use std::time::Duration;

    type MockMacMap = Arc<Mutex<HashMap<(String, Vec<u8>), Vec<u8>>>>;
    /// `(chat, sender, msg_id)` -> `(secret, expires_at, message_ts)`.
    type MockMsgSecretMap = Arc<Mutex<HashMap<(String, String, String), ([u8; 32], i64, i64)>>>;

    #[derive(Default, Clone)]
    struct MockBackend {
        versions: Arc<Mutex<HashMap<String, HashState>>>,
        macs: MockMacMap,
        keys: Arc<Mutex<HashMap<Vec<u8>, AppStateSyncKey>>>,
        latest: Arc<Mutex<Option<Vec<u8>>>>,
        msg_secrets: MockMsgSecretMap,
    }

    impl MockBackend {
        async fn store_key(&self, key_id: &[u8], master: [u8; 32]) {
            self.keys.lock().await.insert(
                key_id.to_vec(),
                AppStateSyncKey {
                    key_data: master.to_vec(),
                    ..Default::default()
                },
            );
            *self.latest.lock().await = Some(key_id.to_vec());
        }
    }

    #[async_trait]
    impl SignalStore for MockBackend {
        async fn put_identity(&self, _: &str, _: [u8; 32]) -> StoreResult<()> {
            Ok(())
        }
        async fn load_identity(&self, _: &str) -> StoreResult<Option<Vec<u8>>> {
            Ok(None)
        }
        async fn delete_identity(&self, _: &str) -> StoreResult<()> {
            Ok(())
        }
        async fn get_session(&self, _: &str) -> StoreResult<Option<Vec<u8>>> {
            Ok(None)
        }
        async fn put_session(&self, _: &str, _: &[u8]) -> StoreResult<()> {
            Ok(())
        }
        async fn delete_session(&self, _: &str) -> StoreResult<()> {
            Ok(())
        }
        async fn store_prekey(&self, _: u32, _: &[u8], _: bool) -> StoreResult<()> {
            Ok(())
        }
        async fn load_prekey(&self, _: u32) -> StoreResult<Option<Vec<u8>>> {
            Ok(None)
        }
        async fn remove_prekey(&self, _: u32) -> StoreResult<()> {
            Ok(())
        }
        async fn max_prekey_id(&self) -> StoreResult<Option<u32>> {
            Ok(None)
        }
        async fn store_signed_prekey(&self, _: u32, _: &[u8]) -> StoreResult<()> {
            Ok(())
        }
        async fn load_signed_prekey(&self, _: u32) -> StoreResult<Option<Vec<u8>>> {
            Ok(None)
        }
        async fn load_all_signed_prekeys(&self) -> StoreResult<Vec<(u32, Vec<u8>)>> {
            Ok(vec![])
        }
        async fn remove_signed_prekey(&self, _: u32) -> StoreResult<()> {
            Ok(())
        }
        async fn put_sender_key(&self, _: &str, _: &[u8]) -> StoreResult<()> {
            Ok(())
        }
        async fn get_sender_key(&self, _: &str) -> StoreResult<Option<Vec<u8>>> {
            Ok(None)
        }
        async fn delete_sender_key(&self, _: &str) -> StoreResult<()> {
            Ok(())
        }
    }

    #[async_trait]
    impl AppSyncStore for MockBackend {
        async fn get_sync_key(&self, key_id: &[u8]) -> StoreResult<Option<AppStateSyncKey>> {
            Ok(self.keys.lock().await.get(key_id).cloned())
        }
        async fn set_sync_key(&self, key_id: &[u8], key: AppStateSyncKey) -> StoreResult<()> {
            self.keys.lock().await.insert(key_id.to_vec(), key);
            Ok(())
        }
        async fn get_version(&self, name: &str) -> StoreResult<HashState> {
            Ok(self
                .versions
                .lock()
                .await
                .get(name)
                .cloned()
                .unwrap_or_default())
        }
        async fn set_version(&self, name: &str, state: HashState) -> StoreResult<()> {
            self.versions.lock().await.insert(name.to_string(), state);
            Ok(())
        }
        async fn put_mutation_macs(
            &self,
            name: &str,
            _version: u64,
            mutations: &[AppStateMutationMAC],
        ) -> StoreResult<()> {
            let mut macs = self.macs.lock().await;
            for m in mutations {
                macs.insert((name.to_string(), m.index_mac.clone()), m.value_mac.clone());
            }
            Ok(())
        }
        async fn get_mutation_mac(
            &self,
            name: &str,
            index_mac: &[u8],
        ) -> StoreResult<Option<Vec<u8>>> {
            Ok(self
                .macs
                .lock()
                .await
                .get(&(name.to_string(), index_mac.to_vec()))
                .cloned())
        }
        async fn delete_mutation_macs(
            &self,
            name: &str,
            index_macs: &[Vec<u8>],
        ) -> StoreResult<()> {
            let mut macs = self.macs.lock().await;
            for index_mac in index_macs {
                macs.remove(&(name.to_string(), index_mac.clone()));
            }
            Ok(())
        }
        async fn clear_mutation_macs(&self, name: &str) -> StoreResult<()> {
            let mut macs = self.macs.lock().await;
            macs.retain(|(collection, _), _| collection != name);
            Ok(())
        }
        async fn get_latest_sync_key_id(&self) -> StoreResult<Option<Vec<u8>>> {
            Ok(self.latest.lock().await.clone())
        }
    }

    #[async_trait]
    impl ProtocolStore for MockBackend {
        async fn get_skdm_recipients(&self, _: &str) -> StoreResult<Vec<wacore_binary::jid::Jid>> {
            Ok(vec![])
        }
        async fn add_skdm_recipients(
            &self,
            _: &str,
            _: &[wacore_binary::jid::Jid],
        ) -> StoreResult<()> {
            Ok(())
        }
        async fn clear_skdm_recipients(&self, _: &str) -> StoreResult<()> {
            Ok(())
        }
        async fn get_lid_mapping(&self, _: &str) -> StoreResult<Option<LidPnMappingEntry>> {
            Ok(None)
        }
        async fn get_pn_mapping(&self, _: &str) -> StoreResult<Option<LidPnMappingEntry>> {
            Ok(None)
        }
        async fn put_lid_mapping(&self, _: &LidPnMappingEntry) -> StoreResult<()> {
            Ok(())
        }
        async fn get_all_lid_mappings(&self) -> StoreResult<Vec<LidPnMappingEntry>> {
            Ok(vec![])
        }
        async fn save_base_key(&self, _: &str, _: &str, _: &[u8]) -> StoreResult<()> {
            Ok(())
        }
        async fn has_same_base_key(&self, _: &str, _: &str, _: &[u8]) -> StoreResult<bool> {
            Ok(false)
        }
        async fn delete_base_key(&self, _: &str, _: &str) -> StoreResult<()> {
            Ok(())
        }
        async fn update_device_list(&self, _: DeviceListRecord) -> StoreResult<()> {
            Ok(())
        }
        async fn get_devices(&self, _: &str) -> StoreResult<Option<DeviceListRecord>> {
            Ok(None)
        }
        async fn mark_forget_sender_key(&self, _: &str, _: &str) -> StoreResult<()> {
            Ok(())
        }
        async fn consume_forget_marks(&self, _: &str) -> StoreResult<Vec<String>> {
            Ok(vec![])
        }
        async fn get_tc_token(
            &self,
            _: &str,
        ) -> StoreResult<Option<crate::store::traits::TcTokenEntry>> {
            Ok(None)
        }
        async fn put_tc_token(
            &self,
            _: &str,
            _: &crate::store::traits::TcTokenEntry,
        ) -> StoreResult<()> {
            Ok(())
        }
        async fn delete_tc_token(&self, _: &str) -> StoreResult<()> {
            Ok(())
        }
        async fn get_all_tc_token_jids(&self) -> StoreResult<Vec<String>> {
            Ok(vec![])
        }
        async fn delete_expired_tc_tokens(&self, _: i64) -> StoreResult<u32> {
            Ok(0)
        }
    }

    #[async_trait]
    impl MsgSecretStore for MockBackend {
        async fn put_msg_secrets(&self, entries: Vec<MsgSecretEntry>) -> StoreResult<usize> {
            let stored = entries.len();
            let mut secrets = self.msg_secrets.lock().await;
            for entry in entries {
                let key = (
                    entry.chat.to_string(),
                    entry.sender.to_string(),
                    entry.msg_id.to_string(),
                );
                match secrets.entry(key) {
                    std::collections::hash_map::Entry::Occupied(mut occupied) => {
                        let (secret, expires_at, message_ts) = occupied.get_mut();
                        *secret = entry.secret;
                        *expires_at = merge_msg_secret_expiry(*expires_at, entry.expires_at);
                        *message_ts = merge_msg_secret_message_ts(*message_ts, entry.message_ts);
                    }
                    std::collections::hash_map::Entry::Vacant(vacant) => {
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
        ) -> StoreResult<Option<StoredMessageSecret>> {
            let secrets = self.msg_secrets.lock().await;
            Ok(secrets
                .get(&(chat.to_string(), sender.to_string(), msg_id.to_string()))
                .map(|(secret, _, message_ts)| {
                    StoredMessageSecret::new(MessageSecret::from_bytes(*secret), Some(*message_ts))
                }))
        }

        async fn delete_expired_msg_secrets(&self, cutoff_timestamp: i64) -> StoreResult<u32> {
            let mut secrets = self.msg_secrets.lock().await;
            let before = secrets.len();
            secrets
                .retain(|_, (_, expires_at, _)| *expires_at == 0 || *expires_at > cutoff_timestamp);
            Ok((before - secrets.len()) as u32)
        }
    }

    #[async_trait]
    impl DeviceStore for MockBackend {
        async fn save(&self, _: &crate::store::Device) -> StoreResult<()> {
            Ok(())
        }
        async fn load(&self) -> StoreResult<Option<crate::store::Device>> {
            Ok(Some(crate::store::Device::new()))
        }
        async fn exists(&self) -> StoreResult<bool> {
            Ok(true)
        }
        async fn create(&self) -> StoreResult<i32> {
            Ok(1)
        }
    }

    struct TestRuntime;

    #[async_trait]
    impl Runtime for TestRuntime {
        fn spawn(&self, _: Pin<Box<dyn Future<Output = ()> + Send + 'static>>) -> AbortHandle {
            AbortHandle::noop()
        }
        fn sleep(&self, _: Duration) -> Pin<Box<dyn Future<Output = ()> + Send>> {
            Box::pin(std::future::ready(()))
        }
        fn spawn_blocking(
            &self,
            f: Box<dyn FnOnce() + Send + 'static>,
        ) -> Pin<Box<dyn Future<Output = ()> + Send>> {
            f();
            Box::pin(std::future::ready(()))
        }
        fn yield_now(&self) -> Option<Pin<Box<dyn Future<Output = ()> + Send>>> {
            None
        }
    }

    fn empty_list() -> PatchList {
        PatchList {
            name: WAPatchName::Regular,
            has_more_patches: false,
            patches: Vec::new(),
            snapshot: None,
            snapshot_ref: None,
            error: None,
        }
    }

    fn processor(backend: Arc<MockBackend>) -> AppStateProcessor {
        AppStateProcessor::new(backend, Arc::new(TestRuntime))
    }

    fn no_download(_: &wa::ExternalBlobReference) -> Result<bytes::Bytes> {
        Ok(bytes::Bytes::new())
    }

    #[tokio::test]
    async fn an_empty_terminal_answer_records_a_bootstrap() {
        let backend = Arc::new(MockBackend::default());
        let processor = processor(backend.clone());

        let (mutations, state, list) = processor
            .process_patch_list(empty_list(), false)
            .await
            .expect("empty answer processes");

        assert!(mutations.is_empty());
        assert!(state.bootstrapped, "a terminal empty answer is a bootstrap");
        assert!(list.error.is_none());
        let stored = backend.get_version("regular").await.expect("version read");
        assert!(stored.bootstrapped, "the flag is persisted");
        assert_eq!(stored.version, 0);
    }

    #[tokio::test]
    async fn a_collection_error_is_not_recorded_as_a_bootstrap() {
        let backend = Arc::new(MockBackend::default());
        let processor = processor(backend.clone());
        let mut pl = empty_list();
        pl.error = Some(CollectionSyncError::Fatal {
            code: 404,
            text: "synthetic fatal".into(),
        });

        let (mutations, state, list) = processor
            .process_one_patch_list(pl, &no_download, false)
            .await
            .expect("errored collection is skipped");

        assert!(mutations.is_empty());
        assert!(!state.bootstrapped);
        assert!(matches!(
            list.error,
            Some(CollectionSyncError::Fatal { .. })
        ));
        let stored = backend.get_version("regular").await.expect("version read");
        assert!(!stored.bootstrapped);
        assert_eq!(stored.version, 0);
    }

    fn recovery(version: u64, key_id: &[u8]) -> wa::SyncdSnapshotRecovery {
        let mut recovery = wa::SyncdSnapshotRecovery::default();
        recovery.collection_name = Some("regular".into());
        let mut syncd_version = wa::SyncdVersion::default();
        syncd_version.version = Some(version);
        recovery.version = syncd_version.into();
        recovery.collection_lthash = Some(vec![9u8; 128]);

        let mut action = wa::SyncActionData::default();
        action.index = Some(serde_json::to_vec(&["archive"]).expect("index json"));
        let mut value = wa::SyncActionValue::default();
        value.timestamp = Some(1);
        action.value = value.into();

        let mut record = wa::SyncdPlainTextRecord::default();
        record.value = action.into();
        record.key_id = Some(key_id.to_vec());
        record.mac = Some(vec![7u8; 32]);
        recovery.mutation_records.push(record);
        recovery
    }

    #[tokio::test]
    async fn snapshot_recovery_replaces_the_collection_and_its_macs() {
        let backend = Arc::new(MockBackend::default());
        let key_id = b"synthetic-key".to_vec();
        backend.store_key(&key_id, [7u8; 32]).await;
        let processor = processor(backend.clone());

        let outcome = processor
            .apply_snapshot_recovery(recovery(5, &key_id), "regular", &|| true)
            .await
            .expect("recovery applies");

        let RecoveryOutcome::Applied(mutations) = outcome else {
            panic!("expected an applied recovery");
        };
        assert_eq!(mutations.len(), 1);
        assert_eq!(mutations[0].index, vec!["archive".to_string()]);

        let stored = backend.get_version("regular").await.expect("version read");
        assert_eq!(stored.version, 5);
        assert_eq!(stored.hash, [9u8; 128]);
        assert!(stored.bootstrapped);

        let index_mac = crate::appstate::hash::generate_index_mac(
            &serde_json::to_vec(&["archive"]).expect("index json"),
            &crate::appstate::expand_app_state_keys(&[7u8; 32]).index,
        );
        let stored_mac = backend
            .get_mutation_mac("regular", &index_mac)
            .await
            .expect("mac read");
        assert_eq!(stored_mac, Some(vec![7u8; 32]));
    }

    #[tokio::test]
    async fn a_stale_recovery_is_discarded_without_writing() {
        let backend = Arc::new(MockBackend::default());
        let key_id = b"synthetic-key".to_vec();
        backend.store_key(&key_id, [7u8; 32]).await;
        backend
            .set_version(
                "regular",
                HashState {
                    version: 10,
                    ..HashState::default()
                },
            )
            .await
            .expect("seed version");
        let processor = processor(backend.clone());

        let outcome = processor
            .apply_snapshot_recovery(recovery(5, &key_id), "regular", &|| true)
            .await
            .expect("stale recovery is not an error");

        assert!(matches!(
            outcome,
            RecoveryOutcome::Stale {
                held: 10,
                offered: 5
            }
        ));
        let stored = backend.get_version("regular").await.expect("version read");
        assert_eq!(stored.version, 10);
        assert!(!stored.bootstrapped);
    }

    #[tokio::test]
    async fn a_retired_recovery_writes_nothing() {
        let backend = Arc::new(MockBackend::default());
        let key_id = b"synthetic-key".to_vec();
        backend.store_key(&key_id, [7u8; 32]).await;
        let processor = processor(backend.clone());

        let outcome = processor
            .apply_snapshot_recovery(recovery(5, &key_id), "regular", &|| false)
            .await
            .expect("retired recovery is not an error");

        assert!(matches!(outcome, RecoveryOutcome::Retired));
        let stored = backend.get_version("regular").await.expect("version read");
        assert_eq!(stored.version, 0);
        assert!(!stored.bootstrapped);
    }

    #[tokio::test]
    async fn a_recovery_for_another_collection_is_refused() {
        let backend = Arc::new(MockBackend::default());
        let key_id = b"synthetic-key".to_vec();
        backend.store_key(&key_id, [7u8; 32]).await;
        let processor = processor(backend.clone());

        let error = processor
            .apply_snapshot_recovery(recovery(5, &key_id), "regular_low", &|| true)
            .await
            .expect_err("mismatched collection is refused");
        assert!(error.to_string().contains("not the regular_low"));
    }

    #[tokio::test]
    async fn build_patch_uses_the_latest_key_and_advances_the_version() {
        let backend = Arc::new(MockBackend::default());
        let key_id = b"synthetic-key".to_vec();
        backend.store_key(&key_id, [7u8; 32]).await;
        let processor = processor(backend.clone());

        let mut mutation = wa::SyncdMutation::default();
        mutation.operation = Some(wa::syncd_mutation::SyncdOperation::SET.into());
        let mut record = wa::SyncdRecord::default();
        let mut index = wa::SyncdIndex::default();
        index.blob = Some(vec![1; 32]);
        record.index = index.into();
        let mut value = wa::SyncdValue::default();
        value.blob = Some(vec![3; 64]);
        record.value = value.into();
        let mut record_key = wa::KeyId::default();
        record_key.id = Some(key_id.clone());
        record.key_id = record_key.into();
        mutation.record = record.into();

        let (bytes, base_version) = processor
            .build_patch("regular", vec![mutation])
            .await
            .expect("patch builds");

        assert_eq!(base_version, 0, "nothing was persisted before the ack");
        let patch = waproto::codec::syncd_patch_decode(&bytes).expect("patch decodes");
        assert_eq!(patch.mutations.len(), 1);
        assert!(patch.snapshot_mac.is_some());
        assert!(patch.patch_mac.is_some());
        assert_eq!(
            patch.key_id.as_option().and_then(|key| key.id.as_deref()),
            Some(key_id.as_slice())
        );
    }
}
