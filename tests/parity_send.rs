//! Phase 2 send/error contract.

use wa_rs::ErrorChainExt;
use wa_rs::send::{EditOptions, PinDuration, SendError, SendOptions, SendResult};

#[test]
fn send_options_and_pin_durations_are_stable() {
    let options = SendOptions::default();
    assert!(options.extra_stanza_nodes.is_empty());
    let edit = EditOptions {
        id: Some("id".to_string()),
    };
    assert_eq!(edit.id.as_deref(), Some("id"));
    assert_eq!(PinDuration::Hours24.hours(), 24);
    assert_eq!(PinDuration::Days7.hours(), 168);
    assert_eq!(PinDuration::Days30.hours(), 720);
    assert_eq!(PinDuration::Custom(5).hours(), 5);
    assert_eq!(
        SendResult {
            id: "ABC".to_string()
        }
        .id,
        "ABC"
    );
}

#[test]
fn send_error_renders_domain_categories() {
    assert!(
        SendError::Transport("offline".into())
            .to_string()
            .contains("offline")
    );
    assert!(
        SendError::Rejected("403".into())
            .to_string()
            .contains("403")
    );
    assert!(
        SendError::InvalidRequest("missing target".into())
            .to_string()
            .contains("missing target")
    );
}

#[test]
fn error_chain_exposes_server_rejection() {
    let error = anyhow::Error::new(wacore::request::IqError::ServerError {
        code: 403,
        text: "forbidden".to_string(),
    });
    let cause: &(dyn std::error::Error + 'static) = error.as_ref();
    let rejection = cause.server_rejection().expect("server rejection");
    assert_eq!(rejection.code, 403);
    assert_eq!(rejection.text, "forbidden");
    assert_eq!(rejection.error_type, None);
    assert_eq!(rejection.backoff, None);
}
