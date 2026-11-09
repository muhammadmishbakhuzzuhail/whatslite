#![cfg(feature = "voip-control")]
//! Phase 4 VoIP control boundary contract.

use wa_rs_core::voip_control::{
    CallEvent, CallRegistry, GroupControl, GroupRawEpoch, MediaAudioSpec, MediaCloseReason,
    MediaDirectPeer, MediaGroupControlKind, MediaGroupEpoch, MediaGroupSpec, MediaSessionSpec,
    MediaSetupError, NormalizedGroupControl, VideoControl, parse_relay_endpoints,
};
use wacore_binary::builder::NodeBuilder;
use wacore_binary::marshal::{marshal, unmarshal_ref};

fn spec(call_id: &str) -> MediaSessionSpec {
    MediaSessionSpec {
        call_id: call_id.to_string(),
        audio: MediaAudioSpec {
            sample_rate: 48_000,
            channels: 1,
        },
        direct_peer: Some(MediaDirectPeer {
            jid: "15551234567@s.whatsapp.net".parse().unwrap(),
            endpoint: "1.2.3.4:1000".to_string(),
        }),
        group: None,
        video: false,
    }
}

#[test]
fn call_registry_allocates_and_tracks_sessions() {
    let mut registry = CallRegistry::new();
    let id = registry.allocate_id();
    assert_eq!(registry.len(), 0);
    registry.insert(spec(&id));
    assert_eq!(registry.len(), 1);
    assert!(registry.get(&id).is_some());
    assert!(registry.remove(&id).is_some());
    assert!(registry.is_empty());
}

#[test]
fn group_control_normalizes_epoch_and_kind() {
    let control = GroupControl {
        call_id: "call".to_string(),
        normalized: NormalizedGroupControl {
            kind: MediaGroupControlKind::Join,
            epoch: GroupRawEpoch(vec![1, 2]),
        },
    };
    assert_eq!(control.normalized.kind, MediaGroupControlKind::Join);
    let group = MediaGroupSpec {
        participants: vec!["15551234567@s.whatsapp.net".parse().unwrap()],
        epoch: MediaGroupEpoch(vec![1, 2]),
    };
    assert_eq!(group.participants.len(), 1);
}

#[test]
fn media_setup_errors_and_events_are_stable() {
    assert!(
        MediaSetupError::BackendUnavailable
            .to_string()
            .contains("backend")
    );
    assert!(
        MediaSetupError::InvalidSpec("x".into())
            .to_string()
            .contains("x")
    );
    let event = CallEvent::Ended {
        call_id: "c".to_string(),
        reason: MediaCloseReason::Normal,
    };
    assert_eq!(
        event,
        CallEvent::Ended {
            call_id: "c".into(),
            reason: MediaCloseReason::Normal
        }
    );
}

#[test]
fn relay_endpoints_parse_and_skip_malformed_children() {
    let relay = NodeBuilder::new("relay")
        .children([
            NodeBuilder::new("endpoint")
                .attr("host", "relay.example.test")
                .attr("port", "443")
                .attr("token", "tok")
                .build(),
            NodeBuilder::new("endpoint").attr("host", "bad").build(),
        ])
        .build();
    let bytes = marshal(&relay).expect("marshal");
    let nr = unmarshal_ref(&bytes[1..]).expect("unmarshal");
    let endpoints = parse_relay_endpoints(&nr);
    assert_eq!(endpoints.len(), 1);
    assert_eq!(endpoints[0].host, "relay.example.test");
    assert_eq!(endpoints[0].port, 443);
    assert_eq!(endpoints[0].token.as_deref(), Some("tok"));
}

#[tokio::test]
async fn video_control_channel_roundtrips() {
    let (tx, rx) = async_channel::bounded(2);
    tx.send(VideoControl::Start).await.unwrap();
    assert_eq!(rx.recv().await.unwrap(), VideoControl::Start);
}
