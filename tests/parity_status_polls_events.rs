use wa_rs::features::events::{EventCreationParams, EventResponseType};
use wa_rs::features::polls::{PollOptionResult, PollVoteCiphertext};
use wa_rs::features::status::{StatusPrivacySetting, StatusSendOptions};
#[test]
fn status_polls_events_types_are_stable() {
    assert_eq!(
        StatusPrivacySetting::Contacts,
        StatusPrivacySetting::Contacts
    );
    assert!(StatusSendOptions::default().background_color.is_none());
    assert_eq!(
        PollOptionResult {
            name: "A".into(),
            votes: 2
        }
        .votes,
        2
    );
    assert_eq!(
        PollVoteCiphertext {
            ciphertext: vec![1]
        }
        .ciphertext,
        vec![1]
    );
    assert_eq!(EventResponseType::Going, EventResponseType::Going);
    assert_eq!(EventCreationParams::default().name, "");
}
