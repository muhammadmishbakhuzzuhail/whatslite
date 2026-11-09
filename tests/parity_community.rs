use wa_rs::features::community::{CommunitySubgroup, SubgroupVisibility};
#[test]
fn community_types_are_stable() {
    assert_eq!(SubgroupVisibility::Visible, SubgroupVisibility::Visible);
    assert_eq!(CommunitySubgroup { jid: "a".into() }.jid, "a");
}
