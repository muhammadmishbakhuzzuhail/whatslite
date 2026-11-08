use wacore_derive::WireEnum;

#[derive(Debug, Clone, Copy, PartialEq, Eq, WireEnum)]
enum MemberAddMode {
    #[wire = "admin_add"]
    AdminAdd,
    #[wire = "all_member_add"]
    AllMemberAdd,
}

#[test]
fn unit_string_wire_enum_round_trips() {
    assert_eq!(MemberAddMode::AdminAdd.as_str(), "admin_add");
    assert_eq!(MemberAddMode::AllMemberAdd.as_str(), "all_member_add");
    assert_eq!(
        MemberAddMode::try_from("admin_add").expect("known tag"),
        MemberAddMode::AdminAdd
    );
    assert!(MemberAddMode::try_from("unknown").is_err());
    assert_eq!(MemberAddMode::default(), MemberAddMode::AdminAdd);
    assert_eq!(MemberAddMode::AdminAdd.to_string(), "admin_add");
}

#[test]
fn unit_string_wire_enum_serde_round_trips() {
    let json = serde_json::to_string(&MemberAddMode::AllMemberAdd).expect("serialize");
    assert_eq!(json, "\"all_member_add\"");
    let parsed: MemberAddMode = serde_json::from_str(&json).expect("deserialize");
    assert_eq!(parsed, MemberAddMode::AllMemberAdd);
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, WireEnum)]
#[wire(kind = "int")]
enum ReceiptType {
    #[wire = 0]
    Delivered,
    #[wire = 1]
    Read,
}

#[test]
fn int_wire_enum_uses_code() {
    assert_eq!(ReceiptType::Delivered.code(), 0);
    assert_eq!(ReceiptType::Read.code(), 1);
    assert_eq!(
        ReceiptType::try_from(1).expect("known code"),
        ReceiptType::Read
    );
    assert!(ReceiptType::try_from(9).is_err());
}
