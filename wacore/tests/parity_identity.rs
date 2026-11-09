//! Phase 1 identity contract: numeric security codes, companion registration
//! platform mapping and client profile defaults.

use wa_rs_core::client_profile::ClientProfile;
use wa_rs_core::companion_reg::{CompanionWebClientType, companion_web_client_type_for_platform};
use wa_rs_core::identity_verification::{
    SecurityCodeError, SecurityCodeVersion, numeric_security_code,
};
use wa_rs_core::libsignal::protocol::{IdentityKey, KeyPair};
use wacore_binary::Jid;
use waproto::whatsapp as wa;

fn identity(kp: &KeyPair) -> IdentityKey {
    IdentityKey::new(kp.public_key)
}

#[test]
fn numeric_security_code_is_sixty_digits_and_deterministic() {
    let mut rng = rand::rng();
    let local = Jid::pn("12025550111");
    let remote = Jid::pn("12025550112");
    let local_keys = vec![identity(&KeyPair::generate(&mut rng))];
    let remote_keys = vec![identity(&KeyPair::generate(&mut rng))];

    let code = numeric_security_code(
        SecurityCodeVersion::V3,
        &local,
        &local_keys,
        &remote,
        &remote_keys,
    )
    .expect("valid PN identities");
    assert_eq!(code.len(), 60);
    assert!(code.bytes().all(|b| b.is_ascii_digit()));

    let again = numeric_security_code(
        SecurityCodeVersion::V3,
        &local,
        &local_keys,
        &remote,
        &remote_keys,
    )
    .expect("deterministic");
    assert_eq!(code, again);
}

#[test]
fn numeric_security_code_v4_uses_lid_identifiers() {
    let mut rng = rand::rng();
    let local = Jid::lid("100000000000001");
    let remote = Jid::lid("100000000000002");
    let code = numeric_security_code(
        SecurityCodeVersion::V4,
        &local,
        &[identity(&KeyPair::generate(&mut rng))],
        &remote,
        &[identity(&KeyPair::generate(&mut rng))],
    )
    .expect("valid LID identities");
    assert_eq!(code.len(), 60);
}

#[test]
fn security_code_rejects_non_numeric_or_missing_keys() {
    let mut rng = rand::rng();
    let key = identity(&KeyPair::generate(&mut rng));
    let local = Jid::pn("12025550111");
    let remote = Jid::pn("12025550112");

    assert_eq!(
        numeric_security_code(
            SecurityCodeVersion::V3,
            &Jid::pn("alice"),
            &[key],
            &remote,
            &[key],
        )
        .unwrap_err(),
        SecurityCodeError::InvalidIdentifier
    );
    assert_eq!(
        numeric_security_code(SecurityCodeVersion::V3, &local, &[], &remote, &[key],).unwrap_err(),
        SecurityCodeError::MissingIdentity
    );
}

#[test]
fn companion_registration_maps_platform_types() {
    assert_eq!(
        companion_web_client_type_for_platform(wa::device_props::PlatformType::CHROME),
        CompanionWebClientType::Chrome
    );
    assert_eq!(
        companion_web_client_type_for_platform(wa::device_props::PlatformType::FIREFOX),
        CompanionWebClientType::Firefox
    );
    assert_eq!(
        companion_web_client_type_for_platform(wa::device_props::PlatformType::SAFARI),
        CompanionWebClientType::Safari
    );
}

#[test]
fn client_profile_defaults_to_web_browser() {
    let profile = ClientProfile::web();
    assert_eq!(
        profile.user_agent_platform,
        wa::client_payload::user_agent::Platform::WEB
    );
    assert!(profile.include_web_info);
    assert!(!profile.passive_login);
    assert_eq!(profile.locale_language, "en");
    assert_eq!(profile.locale_country, "US");

    let android = ClientProfile::android("14");
    assert_eq!(
        android.user_agent_platform,
        wa::client_payload::user_agent::Platform::ANDROID
    );
    assert!(!android.include_web_info);
}
