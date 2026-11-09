#![cfg(feature = "voip-runtime")]
//! Phase 4 VoIP facade contract.
use std::time::Duration;
use wa_rs::voip::{
    AcceptCall, CallError, CallEvents, CallHandle, CallLinkCall, CallState, CallTermination,
    GroupBoundCall, OutgoingCall, OutgoingGroupCall, VIDEO_UPGRADE_TIMEOUT, Voip,
};
#[test]
fn facade_types_and_timeout_are_stable() {
    assert_eq!(VIDEO_UPGRADE_TIMEOUT, Duration::from_secs(30));
    assert_eq!(CallState::Ringing, CallState::Ringing);
    assert_eq!(
        AcceptCall {
            call_id: "c".into()
        }
        .call_id,
        "c"
    );
    assert_eq!(
        CallHandle {
            call_id: "c".into(),
            state: CallState::Connected
        }
        .state,
        CallState::Connected
    );
    assert_eq!(
        OutgoingCall {
            call_id: "c".into(),
            to: "a".into()
        }
        .to,
        "a"
    );
    assert_eq!(
        OutgoingGroupCall {
            call_id: "c".into(),
            group: "g".into()
        }
        .group,
        "g"
    );
    assert_eq!(
        GroupBoundCall {
            call_id: "c".into(),
            group: "g".into()
        }
        .group,
        "g"
    );
    assert_eq!(CallLinkCall { token: "t".into() }.token, "t");
    assert_eq!(CallTermination::Timeout, CallTermination::Timeout);
    assert!(CallError::NotConnected.to_string().contains("connected"));
    let _ = Voip;
    let events = CallEvents {
        call_id: "c".into(),
        events: vec![],
    };
    assert!(events.events.is_empty());
}
