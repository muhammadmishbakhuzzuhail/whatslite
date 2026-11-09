use wa_rs::features::{CreateGroupResult, GroupMetadata, GroupParticipant};
#[test]
fn group_types_are_reachable() {
    let metadata = GroupMetadata {
        id: "123@g.us".parse().unwrap(),
        subject: "Group".to_string(),
        participants: vec![GroupParticipant {
            jid: "15551234567@s.whatsapp.net".parse().unwrap(),
            phone_number: None,
            is_admin: false,
        }],
        addressing_mode: Default::default(),
    };
    assert_eq!(metadata.subject, "Group");
    assert_eq!(
        CreateGroupResult {
            gid: "123@g.us".parse().unwrap()
        }
        .gid
        .to_string(),
        "123@g.us"
    );
}
