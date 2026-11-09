use wa_rs::features::{MemberAddMode, MembershipApprovalMode, ParticipantChangeResponse};
#[test]
fn membership_modes_are_stable() {
    assert_eq!(MemberAddMode::AdminAdd.as_str(), "admin_add");
    assert_eq!(MemberAddMode::AllMemberAdd.as_str(), "all_member_add");
    assert_eq!(MembershipApprovalMode::Off.as_str(), "off");
    assert_eq!(MembershipApprovalMode::On.as_str(), "on");
    let _ = ParticipantChangeResponse {
        jid: "a@s.whatsapp.net".parse().unwrap(),
        status: Some("200".into()),
        error: None,
    };
}
