//! Phase 2 secret-envelope contract: reactions, edits, bot messages and media
//! retry encryption.

use wa_rs_core::bot_message::{BotMessageContext, decrypt_bot_message, encrypt_bot_message};
use wa_rs_core::media_retry::{decrypt_media_retry_notification, encrypt_media_retry_receipt};
use wa_rs_core::message_edit::{MessageEditContext, decrypt_message_edit, encrypt_message_edit};
use wa_rs_core::reaction::{decrypt_reaction_with_secret, encrypt_reaction_with_secret};
use waproto::buffa::Message as _;
use waproto::whatsapp as wa;

const SECRET: [u8; 32] = [0x55; 32];

#[test]
fn reaction_secret_envelope_roundtrips() {
    let (payload, iv) = encrypt_reaction_with_secret(
        "🔥",
        1_700_000_000_000,
        &SECRET,
        "PARENT",
        "creator@s.whatsapp.net",
        "reactor@s.whatsapp.net",
    )
    .expect("encrypt reaction");
    let reaction = decrypt_reaction_with_secret(
        &payload,
        &iv,
        &SECRET,
        "PARENT",
        "creator@s.whatsapp.net",
        "reactor@s.whatsapp.net",
    )
    .expect("decrypt reaction");
    assert_eq!(reaction.text.as_deref(), Some("🔥"));
    assert_eq!(reaction.sender_timestamp_ms, Some(1_700_000_000_000));

    assert!(
        decrypt_reaction_with_secret(
            &payload,
            &iv,
            &[0x11; 32],
            "PARENT",
            "creator@s.whatsapp.net",
            "reactor@s.whatsapp.net",
        )
        .is_err()
    );
}

#[test]
fn message_edit_secret_envelope_roundtrips() {
    let mut inner = wa::Message::default();
    inner.conversation = Some("edited text".to_string());
    let ctx = MessageEditContext {
        original_msg_id: "ORIGINAL",
        original_sender_jid: "author@s.whatsapp.net",
        editor_jid: "editor@s.whatsapp.net",
    };
    let (payload, iv) = encrypt_message_edit(&inner, &SECRET, &ctx).expect("encrypt edit");
    let decrypted = decrypt_message_edit(&payload, &iv, &SECRET, &ctx).expect("decrypt edit");
    assert_eq!(decrypted.conversation.as_deref(), Some("edited text"));

    let bad_iv = [0u8; 11];
    assert!(decrypt_message_edit(&payload, &bad_iv, &SECRET, &ctx).is_err());
}

#[test]
fn bot_message_secret_envelope_roundtrips() {
    let ctx = BotMessageContext {
        msg_id: "BOTMSG1",
        target_sender_user_jid: "111@s.whatsapp.net",
        bot_user_jid: "867051314767696@bot",
    };
    let plaintext = b"bot payload";
    let (payload, iv) = encrypt_bot_message(plaintext, &SECRET, &ctx).expect("encrypt bot");
    let decrypted = decrypt_bot_message(&SECRET, &iv, &payload, &ctx).expect("decrypt bot");
    assert_eq!(decrypted, plaintext);

    let wrong_ctx = BotMessageContext {
        msg_id: "OTHER",
        ..ctx
    };
    assert!(decrypt_bot_message(&SECRET, &iv, &payload, &wrong_ctx).is_err());
}

#[test]
fn media_retry_envelope_roundtrips_and_wrong_stanza_fails() {
    let media_key = [0x42u8; 32];
    let (ciphertext, iv) =
        encrypt_media_retry_receipt(&media_key, "STANZA").expect("encrypt retry");
    let plaintext = decrypt_media_retry_notification(&media_key, "STANZA", &iv, &ciphertext)
        .expect("decrypt retry");
    let receipt = wa::ServerErrorReceipt::decode_from_slice(&plaintext).expect("proto");
    assert_eq!(receipt.stanza_id.as_deref(), Some("STANZA"));

    assert!(decrypt_media_retry_notification(&media_key, "OTHER", &iv, &ciphertext).is_err());
}
