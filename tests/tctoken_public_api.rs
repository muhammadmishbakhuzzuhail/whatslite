use std::error::Error as _;
use wa_rs::features::{TcToken, TcTokenError};
use wa_rs::request::IqError;
use wa_rs::wacore::store::error::StoreError;

fn tc_token<'a>(client: &'a wa_rs::Client) -> TcToken<'a> {
    client.tc_token()
}

#[test]
fn token_error_is_public_and_keeps_typed_sources() {
    let _ = tc_token;

    let iq = TcTokenError::from(IqError::Timeout);
    assert_eq!(iq.to_string(), "IQ request timed out");
    assert!(iq.source().is_some());

    let store = TcTokenError::from(StoreError::NotFound("sentinel".into()));
    assert_eq!(store.to_string(), "Item not found: sentinel");
    assert!(store.source().is_some());
}
