use super::*;
use std::sync::Arc;

#[test]
fn message_reference_conversion_errors_are_typed() {
    assert_eq!(
        MessageId::new("").unwrap_err(),
        MessageRefError::EmptyMessageId
    );
    assert_eq!(
        StanzaId::try_from(String::new()).unwrap_err(),
        MessageRefError::EmptyStanzaId
    );
    let group: Jid = "120363000000000001@g.us".parse().unwrap();
    let channel: Jid = "120363000000000001@newsletter".parse().unwrap();
    assert_eq!(
        MessageRef::new(&group, MessageId::new("ID").unwrap(), None, false).unwrap_err(),
        MessageRefError::MissingSender
    );
    assert_eq!(
        MessageRef::new(&channel, MessageId::new("ID").unwrap(), None, true).unwrap_err(),
        MessageRefError::ExpectedChat
    );
    assert_eq!(
        NewsletterMessageRef::new(&group, None, None).unwrap_err(),
        MessageRefError::ExpectedNewsletter
    );
    let missing = NewsletterMessageRef::new(&channel, None, None).unwrap();
    assert_eq!(
        missing.require_message_id().unwrap_err(),
        MessageRefError::MissingMessageId
    );
    assert_eq!(
        missing.require_server_id().unwrap_err(),
        MessageRefError::MissingServerMessageId
    );
    assert_eq!(
        MessageId::try_from("ID_ünïcødé_✅".to_string())
            .unwrap()
            .as_str(),
        "ID_ünïcødé_✅"
    );
}

#[test]
fn newsletter_envelope_reference_preserves_numbers_and_absence() {
    let own: Jid = "15550000001@s.whatsapp.net".parse().unwrap();
    let chat: Jid = "120363000000000001@newsletter".parse().unwrap();
    for value in [None, Some(0), Some(42), Some(u64::MAX)] {
        // The wacore crate has no stanza parser; the envelope parser that
        // populates `newsletter_server_id` lives in the runtime crate. This
        // covers the reference projection of an already-parsed envelope.
        let info = MessageInfo {
            source: MessageSource {
                chat: chat.clone(),
                sender: own.clone(),
                ..Default::default()
            },
            id: "CLIENT_CONTENT".into(),
            newsletter_server_id: value,
            ..Default::default()
        };
        assert_eq!(info.newsletter_server_id, value);
        let message = InboundMessage::builder()
            .info(Arc::new(info))
            .message(Arc::new(wa::Message::default()))
            .build();
        let reference = message.newsletter_ref().unwrap();
        assert_eq!(reference.message_id().unwrap().as_str(), "CLIENT_CONTENT");
        assert_eq!(reference.server_id().map(ServerMessageId::get), value);
        assert_eq!(reference.from_me(), None);
        assert!(!message.info.source.is_from_me);
        assert_eq!(
            message.message_ref().unwrap_err(),
            MessageRefError::ExpectedChat
        );
    }
}

#[test]
fn newsletter_live_update_has_only_server_content_id() {
    let chat: Jid = "120363000000000001@newsletter".parse().unwrap();
    let update = super::super::events::NewsletterLiveUpdateMessage::builder()
        .server_id(0)
        .reactions(vec![])
        .build();
    let reference = update.message_ref(&chat).unwrap();
    assert_eq!(reference.server_id().unwrap().get(), 0);
    assert_eq!(reference.message_id(), None);
    assert_eq!(reference.from_me(), None);
}

#[test]
fn message_ack_correlation_requires_message_class_not_delivery() {
    let operation = StanzaId::new("OPERATION").unwrap();
    let ack = ServerAck::builder()
        .id("OPERATION".into())
        .class("receipt".into())
        .build();
    assert!(!operation.matches_message_ack(&ack));
    let nack = ServerAck::builder()
        .id("OPERATION".into())
        .class("message".into())
        .error("479".into())
        .build();
    assert!(operation.matches_message_ack(&nack));
    assert_eq!(nack.stanza_id().unwrap(), operation);
    assert!(nack.error.is_some());
}
