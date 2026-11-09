#![cfg(feature = "passkey")]

//! Phase 1 passkey contract: request parsing, assertion packaging and the
//! callback authenticator seam. Gated by the `passkey` feature.

use std::sync::Arc;

use wa_rs::passkey::{
    Assertion, CallbackAuthenticator, PasskeyAuthenticator, UserVerification,
    build_webauthn_assertion_json, parse_request_options,
};

#[test]
fn request_options_parse_challenge_and_credentials() {
    let json = serde_json::json!({
        "challenge": "AQID",
        "rpId": "web.whatsapp.com",
        "userVerification": "required",
        "timeout": 60000,
        "allowCredentials": [{"type": "public-key", "id": "BAUG"}]
    })
    .to_string();
    let request = parse_request_options(&json).expect("valid options");
    assert_eq!(request.challenge, vec![1, 2, 3]);
    assert_eq!(request.rp_id.as_deref(), Some("web.whatsapp.com"));
    assert_eq!(request.user_verification, UserVerification::Required);
    assert_eq!(request.timeout_ms, Some(60000));
    assert_eq!(request.allow_credentials, vec![vec![4, 5, 6]]);
    assert_eq!(request.raw_options_json, json);
}

#[test]
fn malformed_options_fail_closed() {
    assert!(parse_request_options("{}").is_err());
    assert!(parse_request_options(r#"{"challenge":"AQID","userVerification":"bogus"}"#).is_err());
    assert!(parse_request_options(r#"{"challenge":"AQID","allowCredentials":{}}"#).is_err());
}

#[test]
fn assertion_json_uses_whatsapp_shape() {
    let json = build_webauthn_assertion_json(
        b"cred",
        b"client-data",
        b"auth-data",
        b"signature",
        Some(b"user"),
    );
    let value: serde_json::Value = serde_json::from_slice(&json).expect("json");
    assert_eq!(value["type"], "public-key");
    assert_eq!(value["id"], "Y3JlZA");
    assert_eq!(value["response"]["signature"], "c2lnbmF0dXJl");
    assert_eq!(value["response"]["userHandle"], "dXNlcg");
}

#[tokio::test]
async fn callback_authenticator_returns_host_assertion() {
    let authenticator = CallbackAuthenticator::new(|request| {
        Box::pin(async move {
            Ok(Assertion::new(
                build_webauthn_assertion_json(b"cred", b"c", b"a", b"s", None),
                request.challenge,
            ))
        })
    });
    let request = parse_request_options(r#"{"challenge":"AQID"}"#).unwrap();
    let assertion = authenticator.get_assertion(&request).await.unwrap();
    assert_eq!(assertion.credential_id, vec![1, 2, 3]);
    assert!(!assertion.assertion_json.is_empty());
    let _: Arc<dyn PasskeyAuthenticator> = Arc::new(authenticator);
}
