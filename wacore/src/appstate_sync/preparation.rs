//! Owned app-state preparation without storage access or task scheduling.

use super::IndexMac;
use crate::appstate::hash::HashState;
use crate::appstate::keys::ExpandedAppStateKeys;
use crate::appstate::processor::{PatchProcessingResult, ProcessedSnapshot};
use crate::appstate::{AppStateError, Mutation, process_patch, process_snapshot};
use std::{collections::HashMap, sync::Arc};
use waproto::whatsapp as wa;

fn lookup_app_state_key(
    keys_map: &HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>,
    key_id: &[u8],
) -> Result<ExpandedAppStateKeys, AppStateError> {
    // This fork's processor takes expanded keys by value, so the lookup clones
    // the ~160-byte key instead of handing back the Arc. The map itself stays
    // shared across the blocking handoff.
    keys_map
        .get(key_id)
        .map(|keys| (**keys).clone())
        .ok_or(AppStateError::KeyNotFound)
}

pub struct PreparedPatch {
    pub(super) patch: wa::SyncdPatch,
    pub(super) result: PatchProcessingResult,
}

impl PreparedPatch {
    pub fn mark_bootstrapped(&mut self) {
        self.result.state.bootstrapped = true;
    }

    pub fn result(&self) -> &PatchProcessingResult {
        &self.result
    }
    pub fn into_parts(self) -> (wa::SyncdPatch, PatchProcessingResult) {
        (self.patch, self.result)
    }
}

pub fn prepare_patch(
    patch: wa::SyncdPatch,
    mut state: HashState,
    keys: &HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>,
    previous_macs: &HashMap<IndexMac, Vec<u8>>,
    validate_macs: bool,
    collection: &str,
    final_page: bool,
) -> Result<PreparedPatch, AppStateError> {
    // WA Web: validatePatchVersion — strict monotonic version check. The patch
    // version must be exactly local_version + 1. Skip the check when there is
    // no prior state (version=0, empty hash), since there is no baseline to
    // validate against. This fork's pure processor does not enforce it, so the
    // shared engine does before the processor touches the cursor.
    let patch_version = patch
        .version
        .as_option()
        .and_then(|version| version.version)
        .unwrap_or(0);
    let had_no_prior_state = state.version == 0 && state.hash == [0u8; 128];
    let expected_version = state.version.saturating_add(1);
    if !had_no_prior_state && patch_version != expected_version {
        return Err(AppStateError::PatchVersionMismatch {
            expected: expected_version,
            got: patch_version,
        });
    }

    let mut result = process_patch(
        &patch,
        &mut state,
        |id| lookup_app_state_key(keys, id),
        |index| {
            Ok(<&IndexMac>::try_from(index)
                .ok()
                .and_then(|k| previous_macs.get(k))
                .cloned())
        },
        validate_macs,
        collection,
    )?;
    result.state.bootstrapped |= final_page;
    Ok(PreparedPatch { patch, result })
}

pub struct PreparedSnapshot {
    snapshot: wa::SyncdSnapshot,
    state: HashState,
    result: ProcessedSnapshot,
}

impl PreparedSnapshot {
    pub fn into_parts(self) -> (wa::SyncdSnapshot, HashState, ProcessedSnapshot) {
        (self.snapshot, self.state, self.result)
    }
}

pub fn prepare_snapshot(
    snapshot: wa::SyncdSnapshot,
    keys: &HashMap<Vec<u8>, Arc<ExpandedAppStateKeys>>,
    validate_macs: bool,
    collection: &str,
) -> Result<PreparedSnapshot, AppStateError> {
    let mut state = HashState::default();
    let result = process_snapshot(
        &snapshot,
        &mut state,
        |id| lookup_app_state_key(keys, id),
        validate_macs,
        collection,
    )?;
    Ok(PreparedSnapshot {
        snapshot,
        state,
        result,
    })
}

// Publication is a separate consuming step, called only after all required
// mutation effects and the backend cursor/MAC commit have succeeded.
pub(super) fn publish_patch(
    prepared: PreparedPatch,
    state: &mut HashState,
    committed_state: &mut HashState,
    mutations: &mut Vec<Mutation>,
) -> wa::SyncdPatch {
    let (patch, result) = prepared.into_parts();
    *state = result.state;
    *committed_state = state.clone();
    mutations.extend(result.mutations);
    patch
}
