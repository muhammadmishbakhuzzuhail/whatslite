//! Phase 2 types contract: JIDs, message metadata, presence and LID learning.

use wa_rs::types::message::{AddressingMode, MessageSource};
use wa_rs::wacore::types::lid_pn::{LearningSource, LidPnEntry};
use wa_rs::wacore::types::presence::ReceiptType;
use wacore_binary::jid::Jid;

#[test]
fn message_source_and_addressing_mode_are_stable() {
    let source = MessageSource {
        chat: Jid::pn("15551234567"),
        sender: Jid::pn("15551234568"),
        is_from_me: false,
        is_group: false,
        ..Default::default()
    };
    assert!(!source.is_from_me);
    assert_eq!(AddressingMode::Pn, AddressingMode::Pn);
    assert_ne!(AddressingMode::Pn, AddressingMode::Lid);
}

#[test]
fn receipt_type_parses_known_and_unknown_values() {
    assert_eq!(ReceiptType::from("read".to_string()), ReceiptType::Read);
    assert_eq!(
        ReceiptType::from("future".to_string()),
        ReceiptType::Other("future".to_string())
    );
}

#[test]
fn lid_pn_entry_preserves_learning_source() {
    let entry = LidPnEntry {
        lid: "100000000000001".to_string(),
        phone_number: "15551234567".to_string(),
        created_at: 1,
        learning_source: LearningSource::Usync,
    };
    assert_eq!(entry.learning_source, LearningSource::Usync);
    assert_eq!(entry.learning_source.as_str(), "usync");
}
