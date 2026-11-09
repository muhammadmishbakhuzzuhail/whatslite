//! Phase 2 reliability contract: retry surface, keepalive predicates and LID learning.

use std::time::Duration;

use wa_rs::retry::{
    MessageRetransmission, RetryRequestError, RetryRequestOptions, RetryRequestOutcome,
    StanzaRejection, StanzaResponseError,
};
use wa_rs::wacore::protocol::keepalive::{
    DEAD_SOCKET_TIME, KEEP_ALIVE_INTERVAL_MAX, KEEP_ALIVE_INTERVAL_MIN, is_dead_socket_at,
};
use wa_rs::wacore::time::Instant;
use wa_rs::wacore::types::lid_pn::LearningSource;

#[test]
fn retry_surface_is_stable() {
    let options = RetryRequestOptions {
        count: 2,
        force_include_keys: true,
    };
    assert_eq!(options.count, 2);
    assert!(options.force_include_keys);
    assert_eq!(RetryRequestOutcome::Accepted, RetryRequestOutcome::Accepted);
    assert!(
        RetryRequestError::NotConnected
            .to_string()
            .contains("connected")
    );
    assert_eq!(
        StanzaResponseError {
            rejection: Some(StanzaRejection {
                code: 403,
                text: "forbidden".to_string(),
            }),
        }
        .rejection
        .unwrap()
        .code,
        403
    );
    assert_eq!(
        MessageRetransmission {
            message_id: "M1".to_string(),
            retry_count: 1,
        }
        .retry_count,
        1
    );
}

#[test]
fn keepalive_dead_socket_uses_deadline() {
    assert_eq!(KEEP_ALIVE_INTERVAL_MIN, Duration::from_secs(15));
    assert_eq!(KEEP_ALIVE_INTERVAL_MAX, Duration::from_secs(30));
    let armed = Instant::ZERO;
    let before = Instant::ZERO.checked_add(DEAD_SOCKET_TIME).unwrap();
    let after = before.checked_add(Duration::from_millis(1)).unwrap();
    assert!(!is_dead_socket_at(Some(armed), None, before));
    assert!(is_dead_socket_at(Some(armed), None, after));
}

#[test]
fn lid_learning_sources_roundtrip_strings() {
    for source in [
        LearningSource::Usync,
        LearningSource::PeerPnMessage,
        LearningSource::PeerLidMessage,
        LearningSource::Pairing,
        LearningSource::DeviceNotification,
        LearningSource::Other,
    ] {
        assert!(!source.as_str().is_empty());
    }
    assert_eq!(LearningSource::Usync.as_str(), "usync");
}
