#![cfg(feature = "mobile-core")]
//! Phase 1 mobile contract: environment validation/binding, login profile
//! validation and primary companion approval state.

use std::sync::Arc;

use wa_rs::mobile::primary::{PendingCompanion, PrimaryLinkError, SignedCompanionIdentity};
use wa_rs::mobile::{
    EnvironmentBinding, MobileConfigurationError, MobileEnvironment, MobilePlatform, MobileTrust,
    RegistrationEndpoint, TcpEndpoint,
};
use wa_rs::mobile_session::{
    MobileAppStatePolicy, MobileLoginOptions, MobileLoginProfile, RegisteredMobileSession,
    RegistrationLoginMetadata, RegistrationLoginPhoneIdMapping,
    validate_registration_login_profile,
};
use wacore_binary::Jid;

fn environment() -> MobileEnvironment {
    let registration =
        RegistrationEndpoint::new("https://registration.example.test").expect("https endpoint");
    let connection = TcpEndpoint::new("tcp.example.test", 443).expect("tcp endpoint");
    let trust = MobileTrust::new([1u8; 32], [2u8; 32]).expect("trust");
    MobileEnvironment::new("prod", registration, connection, trust).expect("environment")
}

#[test]
fn registration_endpoint_requires_https_and_rejects_credentials() {
    assert!(RegistrationEndpoint::new("https://example.test").is_ok());
    assert_eq!(
        RegistrationEndpoint::new("http://example.test").unwrap_err(),
        MobileConfigurationError::InvalidRegistrationEndpoint
    );
    assert!(RegistrationEndpoint::new("https://user:pass@example.test").is_err());
    assert!(RegistrationEndpoint::new("https://example.test?query=1").is_err());
    assert!(RegistrationEndpoint::for_test_http("http://127.0.0.1:8080").is_ok());
}

#[test]
fn tcp_endpoint_and_trust_validate_structure() {
    assert_eq!(
        TcpEndpoint::new("tcp.example.test", 443).unwrap().port(),
        443
    );
    assert!(TcpEndpoint::new("", 443).is_err());
    assert!(TcpEndpoint::new("host/path", 443).is_err());
    assert!(TcpEndpoint::new("host", 0).is_err());
    assert_eq!(
        MobileTrust::new([0u8; 32], [1u8; 32]).unwrap_err(),
        MobileConfigurationError::InvalidTrust
    );
}

#[test]
fn environment_binding_is_deterministic_and_guards_bound_values() {
    let first = environment();
    let second = environment();
    assert_eq!(first.binding(), second.binding());
    let bound = first.bind(42u32);
    assert_eq!(*bound.get(&second).unwrap(), 42);

    let other = MobileEnvironment::new(
        "staging",
        RegistrationEndpoint::new("https://staging.example.test").unwrap(),
        TcpEndpoint::new("tcp.example.test", 443).unwrap(),
        MobileTrust::new([1u8; 32], [2u8; 32]).unwrap(),
    )
    .unwrap();
    assert_eq!(
        bound.get(&other).unwrap_err(),
        MobileConfigurationError::EnvironmentMismatch
    );
}

#[test]
fn login_profile_validation_checks_platform_metadata() {
    let profile = MobileLoginProfile {
        version: vec![2, 3000, 1],
        platform: MobilePlatform::Android,
        user_agent: "WhatsApp/2.3000.1".to_string(),
    };
    let valid = RegistrationLoginMetadata {
        platform: MobilePlatform::Android,
        app_version: &[2, 3000, 1],
        manufacturer: "Example",
    };
    assert!(
        validate_registration_login_profile(
            &profile,
            &valid,
            RegistrationLoginPhoneIdMapping::IndependentHostId,
        )
        .is_ok()
    );

    let missing_manufacturer = RegistrationLoginMetadata {
        manufacturer: "",
        ..valid
    };
    assert!(
        validate_registration_login_profile(
            &profile,
            &missing_manufacturer,
            RegistrationLoginPhoneIdMapping::IndependentHostId,
        )
        .is_err()
    );
}

#[test]
fn registered_session_carries_policy_and_options() {
    let session = RegisteredMobileSession::new(
        environment(),
        123456789,
        vec![7u8; 32],
        MobileLoginProfile {
            version: vec![2, 3000, 1],
            platform: MobilePlatform::Android,
            user_agent: "WhatsApp/2.3000.1".to_string(),
        },
        MobileLoginOptions::default(),
    )
    .with_app_state_policy(MobileAppStatePolicy::AuthoritativeZapo);
    assert_eq!(session.username(), 123456789);
    assert_eq!(session.identity(), &[7u8; 32]);
    assert_eq!(
        session.app_state_policy(),
        Some(MobileAppStatePolicy::AuthoritativeZapo)
    );
}

#[test]
fn primary_companion_requires_reference_and_matching_device() {
    let account = Jid::pn("12025550111");
    let device = Jid::pn_device("12025550111", 1);
    let issued = SignedCompanionIdentity {
        device_identity: vec![1, 2, 3],
        key_index_list: vec![4, 5, 6],
    };
    let pending = PendingCompanion::new(
        b"reference",
        std::slice::from_ref(&account),
        &[9u8; 32],
        issued,
    )
    .expect("pending");
    let acknowledged = pending.acknowledge(&device).expect("acknowledged");

    assert_eq!(
        acknowledged
            .verify(b"wrong", &device, b"proof")
            .unwrap_err(),
        PrimaryLinkError::Authentication
    );
    assert_eq!(
        acknowledged.verify(b"reference", &device, b"").unwrap_err(),
        PrimaryLinkError::Authentication
    );
    let verified = acknowledged
        .verify(b"reference", &device, b"proof")
        .expect("verified");
    assert_eq!(verified.device_jid(), &device);
    assert_eq!(verified.peer_identity(), &[9u8; 32]);
    assert_eq!(verified.proof(), b"proof");
}

#[test]
fn mobile_ephemeral_key_provider_trait_is_reachable() {
    let _: Option<Arc<dyn wa_rs::mobile_session::MobileEphemeralKeyProvider>> = None;
    let _: Option<EnvironmentBinding> = None;
}
