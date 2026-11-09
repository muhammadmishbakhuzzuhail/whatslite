use crate::AppStateError;
use crate::hash::{generate_content_mac, validate_index_mac};
use crate::keys::ExpandedAppStateKeys;
use wacore_libsignal::crypto::aes_256_cbc_decrypt_into;
use waproto::buffa::Message as _;
use waproto::whatsapp as wa;

/// A decoded mutation from an app state record.
#[derive(Debug, Clone)]
pub struct Mutation {
    /// The decoded action value.
    pub action_value: Option<wa::SyncActionValue>,
    /// The MAC of the index.
    pub index_mac: Vec<u8>,
    /// The MAC of the value.
    pub value_mac: Vec<u8>,
    /// The parsed index components (JSON array of strings).
    pub index: Vec<String>,
    /// The operation type (Set or Remove).
    pub operation: wa::syncd_mutation::SyncdOperation,
}

/// Decode a single encrypted record into a mutation.
///
/// This is a pure, synchronous function that takes the expanded keys directly,
/// avoiding any async key lookup.
///
/// # Arguments
/// * `operation` - The operation type (Set or Remove)
/// * `record` - The encrypted SyncdRecord to decode
/// * `keys` - The pre-expanded app state keys for decryption
/// * `key_id` - The key ID used for MAC validation
/// * `validate_macs` - Whether to validate MACs during decoding
///
/// # Returns
/// A decoded `Mutation` or an error if decoding/validation fails.
pub fn decode_record(
    operation: wa::syncd_mutation::SyncdOperation,
    record: &wa::SyncdRecord,
    keys: &ExpandedAppStateKeys,
    key_id: &[u8],
    validate_macs: bool,
) -> Result<Mutation, AppStateError> {
    let value_blob = record
        .value
        .as_option()
        .and_then(|v| v.blob.as_ref())
        .ok_or(AppStateError::MissingValueBlob)?;

    if value_blob.len() < 16 + 32 {
        return Err(AppStateError::ValueBlobTooShort);
    }

    let (iv, rest) = value_blob.split_at(16);
    let (ciphertext, value_mac) = rest.split_at(rest.len() - 32);

    if validate_macs {
        let expected = generate_content_mac(
            operation,
            &value_blob[..value_blob.len() - 32],
            key_id,
            &keys.value_mac,
        );
        if expected != value_mac {
            return Err(AppStateError::MismatchingContentMAC);
        }
    }

    let mut plaintext = Vec::new();
    aes_256_cbc_decrypt_into(ciphertext, &keys.value_encryption, iv, &mut plaintext)
        .map_err(|_| AppStateError::DecryptionFailed)?;

    let action = wa::SyncActionData::decode_from_slice(plaintext.as_slice())
        .map_err(|_| AppStateError::DecodeFailed)?;

    let mut index_list: Vec<String> = Vec::new();
    if let Some(idx_bytes) = action.index.as_ref() {
        if validate_macs {
            let stored = record
                .index
                .as_option()
                .and_then(|i| i.blob.as_ref())
                .ok_or(AppStateError::MissingIndexMAC)?;
            validate_index_mac(idx_bytes, stored, &keys.index)?;
        }
        if let Ok(parsed) = serde_json::from_slice::<Vec<String>>(idx_bytes) {
            index_list = parsed;
        }
    }

    Ok(Mutation {
        action_value: action.value.as_option().cloned(),
        index_mac: record
            .index
            .as_option()
            .and_then(|i| i.blob.clone())
            .unwrap_or_default(),
        value_mac: value_mac.to_vec(),
        index: index_list,
        operation,
    })
}

/// Extract all unique key IDs from a patch list that need to be fetched.
///
/// This is a pure function that collects key IDs from snapshots and patches
/// without checking against storage.
pub fn collect_key_ids_from_patch_list(
    snapshot: Option<&wa::SyncdSnapshot>,
    patches: &[wa::SyncdPatch],
) -> Vec<Vec<u8>> {
    use std::collections::HashSet;

    let mut seen = HashSet::new();
    let mut key_ids = Vec::new();

    let mut check = |key_id: Option<&Vec<u8>>| {
        if let Some(k) = key_id
            && seen.insert(k.clone())
        {
            key_ids.push(k.clone());
        }
    };

    if let Some(snapshot) = snapshot {
        check(snapshot.key_id.as_option().and_then(|k| k.id.as_ref()));
        for rec in &snapshot.records {
            check(rec.key_id.as_option().and_then(|k| k.id.as_ref()));
        }
    }

    for patch in patches {
        check(patch.key_id.as_option().and_then(|k| k.id.as_ref()));
        for mutation in &patch.mutations {
            if let Some(record) = mutation.record.as_option() {
                check(record.key_id.as_option().and_then(|k| k.id.as_ref()));
            }
        }
    }

    key_ids
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::hash::generate_content_mac;
    use crate::keys::expand_app_state_keys;
    use wacore_libsignal::crypto::aes_256_cbc_encrypt_into;

    fn make_key_id(id: Vec<u8>) -> wa::KeyId {
        let mut key_id = wa::KeyId::default();
        key_id.id = Some(id);
        key_id
    }

    fn create_test_record(
        op: wa::syncd_mutation::SyncdOperation,
        keys: &ExpandedAppStateKeys,
        key_id: &[u8],
        action_data: &wa::SyncActionData,
    ) -> wa::SyncdRecord {
        let plaintext = action_data.encode_to_vec();
        let iv = vec![0u8; 16];
        let mut ciphertext = Vec::new();
        aes_256_cbc_encrypt_into(&plaintext, &keys.value_encryption, &iv, &mut ciphertext)
            .expect("test encryption should succeed");

        let mut value_with_iv = iv;
        value_with_iv.extend_from_slice(&ciphertext);
        let value_mac = generate_content_mac(op, &value_with_iv, key_id, &keys.value_mac);
        let mut value_blob = value_with_iv;
        value_blob.extend_from_slice(&value_mac);

        let mut index = wa::SyncdIndex::default();
        index.blob = Some(vec![1; 32]);
        let mut value = wa::SyncdValue::default();
        value.blob = Some(value_blob);

        let mut record = wa::SyncdRecord::default();
        record.index = index.into();
        record.value = value.into();
        record.key_id = make_key_id(key_id.to_vec()).into();
        record
    }

    fn make_action_data(timestamp: i64) -> wa::SyncActionData {
        let mut value = wa::SyncActionValue::default();
        value.timestamp = Some(timestamp);
        let mut action_data = wa::SyncActionData::default();
        action_data.value = value.into();
        action_data
    }

    #[test]
    fn test_decode_record_basic() {
        let master_key = [7u8; 32];
        let keys = expand_app_state_keys(&master_key);
        let key_id = b"test_key_id".to_vec();

        let action_data = make_action_data(1234567890);

        let record = create_test_record(
            wa::syncd_mutation::SyncdOperation::Set,
            &keys,
            &key_id,
            &action_data,
        );

        let mutation = decode_record(
            wa::syncd_mutation::SyncdOperation::Set,
            &record,
            &keys,
            &key_id,
            false, // skip MAC validation for this test
        )
        .expect("test encryption should succeed");

        assert_eq!(
            mutation.action_value.as_ref().and_then(|v| v.timestamp),
            Some(1234567890)
        );
        assert_eq!(mutation.operation, wa::syncd_mutation::SyncdOperation::Set);
    }

    #[test]
    fn test_decode_record_with_mac_validation() {
        let master_key = [7u8; 32];
        let keys = expand_app_state_keys(&master_key);
        let key_id = b"test_key_id".to_vec();

        let action_data = make_action_data(1234567890);

        let record = create_test_record(
            wa::syncd_mutation::SyncdOperation::Set,
            &keys,
            &key_id,
            &action_data,
        );

        // With MAC validation enabled but no index in action_data, should succeed
        let result = decode_record(
            wa::syncd_mutation::SyncdOperation::Set,
            &record,
            &keys,
            &key_id,
            true,
        );
        assert!(result.is_ok());
    }

    #[test]
    fn test_collect_key_ids_from_patch_list() {
        let key_id_1 = vec![1, 2, 3];
        let key_id_2 = vec![4, 5, 6];
        let key_id_3 = vec![7, 8, 9];
        let key_id_4 = vec![10, 11, 12];

        let mut snapshot_record = wa::SyncdRecord::default();
        snapshot_record.key_id = make_key_id(key_id_2.clone()).into();
        let mut snapshot = wa::SyncdSnapshot::default();
        snapshot.key_id = make_key_id(key_id_1.clone()).into();
        snapshot.records = vec![snapshot_record];

        let mut mutation_record = wa::SyncdRecord::default();
        mutation_record.key_id = make_key_id(key_id_4.clone()).into();
        let mut mutation = wa::SyncdMutation::default();
        mutation.record = mutation_record.into();
        let mut patch = wa::SyncdPatch::default();
        patch.key_id = make_key_id(key_id_3.clone()).into();
        patch.mutations = vec![mutation];
        let patches = vec![patch];

        let key_ids = collect_key_ids_from_patch_list(Some(&snapshot), &patches);

        assert_eq!(key_ids.len(), 4);
        assert!(key_ids.contains(&key_id_1));
        assert!(key_ids.contains(&key_id_2));
        assert!(key_ids.contains(&key_id_3));
        assert!(key_ids.contains(&key_id_4));
    }

    #[test]
    fn test_collect_key_ids_deduplicates() {
        let key_id = vec![1, 2, 3];

        let mut snapshot_record = wa::SyncdRecord::default();
        snapshot_record.key_id = make_key_id(key_id.clone()).into();
        let mut snapshot = wa::SyncdSnapshot::default();
        snapshot.key_id = make_key_id(key_id.clone()).into();
        snapshot.records = vec![snapshot_record];

        let mut patch = wa::SyncdPatch::default();
        patch.key_id = make_key_id(key_id.clone()).into();
        let patches = vec![patch];

        let key_ids = collect_key_ids_from_patch_list(Some(&snapshot), &patches);

        // Should only have one entry since all key IDs are the same
        assert_eq!(key_ids.len(), 1);
        assert_eq!(key_ids[0], key_id);
    }
}
