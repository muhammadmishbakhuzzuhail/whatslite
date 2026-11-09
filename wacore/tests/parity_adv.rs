//! Phase 1 ADV contract: signature-chain validation, key-index filtering and
//! hosted/companion prefix selection.

use wa_rs_core::adv::{
    AdvValidation, DecodedKeyIndex, filter_devices_by_key_index, is_key_index_valid,
    validate_adv_with_identity_key,
};
use wa_rs_core::libsignal::protocol::KeyPair;
use wa_rs_core::store::traits::DeviceInfo;
use wacore_binary::Jid;
use waproto::buffa::Message as _;
use waproto::whatsapp as wa;

fn key_bytes(kp: &KeyPair) -> Vec<u8> {
    kp.public_key.public_key_bytes().to_vec()
}

#[test]
fn valid_adv_chain_is_trusted_with_known_account_key() {
    // Build with the device public key exposed separately.
    let mut rng = rand::rng();
    let account = KeyPair::generate(&mut rng);
    let device = KeyPair::generate(&mut rng);
    let mut details = wa::ADVDeviceIdentity::default();
    details.raw_id = Some(1);
    details.account_type = Some(wa::ADVEncryptionType::E2EE);
    details.device_type = Some(wa::ADVEncryptionType::E2EE);
    let details = details.encode_to_vec();
    let account_key = key_bytes(&account);
    let device_key = key_bytes(&device);
    let account_sig = account
        .private_key
        .calculate_signature(
            &[&[6, 0], details.as_slice(), device_key.as_slice()].concat(),
            &mut rng,
        )
        .unwrap();
    let device_sig = device
        .private_key
        .calculate_signature(
            &[
                &[6, 1],
                details.as_slice(),
                device_key.as_slice(),
                account_key.as_slice(),
            ]
            .concat(),
            &mut rng,
        )
        .unwrap();
    let mut signed = wa::ADVSignedDeviceIdentity::default();
    signed.details = Some(details);
    signed.account_signature_key = Some(account_key.clone());
    signed.account_signature = Some(account_sig.to_vec());
    signed.device_signature = Some(device_sig.to_vec());

    let device_pub: [u8; 32] = device_key.try_into().unwrap();
    let account_pub: [u8; 32] = account_key.try_into().unwrap();
    let jid = Jid::pn_device("12025550111", 1);
    assert_eq!(
        validate_adv_with_identity_key(
            &signed.encode_to_vec(),
            &device_pub,
            Some(&account_pub),
            &jid,
        ),
        AdvValidation::Valid
    );
}

#[test]
fn malformed_or_unverifiable_adv_is_invalid() {
    let device_pub = [7u8; 32];
    let jid = Jid::pn_device("12025550111", 1);
    assert_eq!(
        validate_adv_with_identity_key(b"not-a-proto", &device_pub, None, &jid),
        AdvValidation::Invalid
    );

    let mut signed = wa::ADVSignedDeviceIdentity::default();
    signed.details = Some(vec![1, 2, 3]);
    signed.account_signature = Some(vec![0u8; 64]);
    signed.device_signature = Some(vec![0u8; 64]);
    assert_eq!(
        validate_adv_with_identity_key(&signed.encode_to_vec(), &device_pub, None, &jid),
        AdvValidation::Invalid
    );
}

#[test]
fn missing_account_key_is_reported_without_a_fallback() {
    let mut rng = rand::rng();
    let device = KeyPair::generate(&mut rng);
    let mut details = wa::ADVDeviceIdentity::default();
    details.raw_id = Some(1);
    details.account_type = Some(wa::ADVEncryptionType::E2EE);
    details.device_type = Some(wa::ADVEncryptionType::E2EE);
    let mut signed = wa::ADVSignedDeviceIdentity::default();
    signed.details = Some(details.encode_to_vec());
    signed.account_signature = Some(vec![0u8; 64]);
    signed.device_signature = Some(vec![0u8; 64]);

    let device_pub: [u8; 32] = key_bytes(&device).try_into().unwrap();
    let jid = Jid::pn_device("12025550111", 1);
    assert_eq!(
        validate_adv_with_identity_key(&signed.encode_to_vec(), &device_pub, None, &jid),
        AdvValidation::NoAccountKey
    );
}

#[test]
fn key_index_filter_keeps_primary_and_newer_or_listed_devices() {
    let decoded = DecodedKeyIndex {
        raw_id: 1,
        timestamp: 0,
        current_index: 2,
        valid_indexes: vec![1, 2],
    };
    assert!(is_key_index_valid(Some(1), &decoded));
    assert!(is_key_index_valid(Some(3), &decoded));
    assert!(!is_key_index_valid(Some(0), &decoded));
    assert!(!is_key_index_valid(None, &decoded));

    let devices = vec![
        DeviceInfo::new(0, None),
        DeviceInfo::new(1, Some(1)),
        DeviceInfo::new(2, Some(0)),
        DeviceInfo::new(3, Some(3)),
    ];
    let retained = filter_devices_by_key_index(&devices, &decoded);
    let ids: Vec<u32> = retained.iter().map(DeviceInfo::device_id).collect();
    assert_eq!(ids, vec![0, 1, 3]);
}
