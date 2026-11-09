//! Phase 1 pair-code contract: rejection classification and companion crypto.

use wa_rs::pair_code::{PairCodeRejection, PairCodeUtils};
use wacore::pair_code::PairCodeError;

#[test]
fn rejection_codes_match_wa_web() {
    assert_eq!(PairCodeRejection::BadRequest.code(), 400);
    assert_eq!(PairCodeRejection::Forbidden.code(), 403);
    assert_eq!(PairCodeRejection::RateOverlimit.code(), 429);
    assert_eq!(PairCodeRejection::FeatureNotAvailable.code(), 452);
    assert_eq!(PairCodeRejection::InternalServerError.code(), 500);
    assert_eq!(
        PairCodeRejection::from(418),
        PairCodeRejection::Unknown(418)
    );
}

#[test]
fn rejection_throttle_and_text_helpers_are_stable() {
    assert!(PairCodeRejection::BadRequest.is_throttled());
    assert!(PairCodeRejection::RateOverlimit.is_throttled());
    assert!(!PairCodeRejection::Forbidden.is_throttled());
    assert_eq!(
        PairCodeRejection::RateOverlimit.text(),
        Some("rate-overlimit")
    );
    assert_eq!(PairCodeRejection::Unknown(418).text(), None);
    assert_eq!(
        PairCodeRejection::from_server(429, "rate-overlimit"),
        Some(PairCodeRejection::RateOverlimit)
    );
    assert_eq!(PairCodeRejection::from_server(429, "wrong"), None);
}

#[test]
fn pair_code_validation_and_ephemeral_roundtrip() {
    assert!(PairCodeUtils::validate_code("ABCD1234"));
    assert!(!PairCodeUtils::validate_code("ABCD0123"));

    let ephemeral = [0x42u8; 32];
    let wrapped = PairCodeUtils::encrypt_ephemeral_pub(&ephemeral, "ABCD1234");
    assert_eq!(wrapped.len(), 80);
    let decrypted =
        PairCodeUtils::decrypt_primary_ephemeral_pub(&wrapped, "ABCD1234").expect("roundtrip");
    assert_eq!(decrypted, ephemeral);
    assert!(matches!(
        PairCodeUtils::decrypt_primary_ephemeral_pub(&[0u8; 79], "ABCD1234"),
        Err(PairCodeError::InvalidWrappedData { .. })
    ));
}

#[test]
fn pair_error_wraps_validation_failures() {
    let error: wa_rs::pair_code::PairError = PairCodeError::PhoneNumberRequired.into();
    assert!(error.to_string().contains("Phone"));
}
