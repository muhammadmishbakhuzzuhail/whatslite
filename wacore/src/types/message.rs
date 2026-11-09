use chrono::{DateTime, Utc};
use serde::Deserialize;
use serde::Serialize;
use wacore_binary::jid::{Jid, JidExt, MessageId, MessageServerId};
use waproto::whatsapp as wa;

/// Unique identifier for a message stanza within a chat.
/// Used for deduplication and retry tracking.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct StanzaKey {
    pub chat: Jid,
    pub id: MessageId,
}

impl StanzaKey {
    pub fn new(chat: Jid, id: MessageId) -> Self {
        Self { chat, id }
    }
}

/// Addressing mode for a group (phone number vs LID).
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum AddressingMode {
    #[default]
    Pn,
    Lid,
}

impl AddressingMode {
    pub fn as_str(&self) -> &'static str {
        match self {
            AddressingMode::Pn => "pn",
            AddressingMode::Lid => "lid",
        }
    }
}

impl std::fmt::Display for AddressingMode {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.as_str())
    }
}

impl TryFrom<&str> for AddressingMode {
    type Error = anyhow::Error;

    fn try_from(value: &str) -> Result<Self, Self::Error> {
        match value {
            "lid" => Ok(AddressingMode::Lid),
            "pn" | "" => Ok(AddressingMode::Pn),
            _ => Err(anyhow::anyhow!("unknown addressing_mode: {value}")),
        }
    }
}

#[derive(Debug, Clone, Default, Serialize)]
pub struct MessageSource {
    pub chat: Jid,
    pub sender: Jid,
    pub is_from_me: bool,
    pub is_group: bool,
    pub addressing_mode: Option<AddressingMode>,
    pub sender_alt: Option<Jid>,
    pub recipient_alt: Option<Jid>,
    pub broadcast_list_owner: Option<Jid>,
    pub recipient: Option<Jid>,
}

impl MessageSource {
    pub fn is_incoming_broadcast(&self) -> bool {
        (!self.is_from_me || self.broadcast_list_owner.is_some()) && self.chat.is_broadcast_list()
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct DeviceSentMeta {
    pub destination_jid: String,
    pub phash: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize)]
pub enum EditAttribute {
    #[default]
    Empty,
    MessageEdit,
    PinInChat,
    AdminEdit,
    SenderRevoke,
    AdminRevoke,
    Unknown(String),
}

impl From<String> for EditAttribute {
    fn from(s: String) -> Self {
        match s.as_str() {
            "" => Self::Empty,
            "1" => Self::MessageEdit,
            "2" => Self::PinInChat,
            "3" => Self::AdminEdit,
            "7" => Self::SenderRevoke,
            "8" => Self::AdminRevoke,
            _ => Self::Unknown(s),
        }
    }
}

impl EditAttribute {
    pub fn to_string_val(&self) -> &'static str {
        match self {
            Self::Empty => "",
            Self::MessageEdit => "1",
            Self::PinInChat => "2",
            Self::AdminEdit => "3",
            Self::SenderRevoke => "7",
            Self::AdminRevoke => "8",
            Self::Unknown(_) => "",
        }
    }

    /// Wire `edit` value derived from a fully-constructed message body.
    /// Mirrors WAWebSendMsgCommonApi.editAttribute.
    pub fn infer_from_message(msg: &wa::Message) -> Option<Self> {
        use wa::KeepType;
        use wa::message::protocol_message::Type as ProtocolType;
        use wa::message::secret_encrypted_message::SecretEncType;

        let msg = unwrap_message(msg);

        if msg.pin_in_chat_message.is_set() {
            return Some(Self::PinInChat);
        }
        if msg.edited_message.is_set() {
            return Some(Self::MessageEdit);
        }
        if let Some(pm) = msg.protocol_message.as_option() {
            if pm.r#type == Some(ProtocolType::REVOKE) {
                let from_me = pm.key.as_option().and_then(|k| k.from_me).unwrap_or(false);
                return Some(if from_me {
                    Self::SenderRevoke
                } else {
                    Self::AdminRevoke
                });
            }
            if pm.r#type == Some(ProtocolType::MESSAGE_EDIT) || pm.edited_message.is_set() {
                return Some(Self::MessageEdit);
            }
        }
        if let Some(sec) = msg.secret_encrypted_message.as_option()
            && let Some(enc_type) = sec.secret_enc_type
            && (enc_type == SecretEncType::MESSAGE_EDIT || enc_type == SecretEncType::EVENT_EDIT)
        {
            return Some(Self::MessageEdit);
        }
        // Reaction with empty text == sender-revoke of a previous reaction.
        if let Some(react) = msg.reaction_message.as_option()
            && react.text.as_deref() == Some("")
        {
            return Some(Self::SenderRevoke);
        }
        // KeepInChat UNDO_KEEP_FOR_ALL is a sender-revoke at the wire level.
        if let Some(keep) = msg.keep_in_chat_message.as_option()
            && keep.key.as_option().and_then(|k| k.from_me) == Some(true)
            && keep.keep_type == Some(KeepType::UNDO_KEEP_FOR_ALL)
        {
            return Some(Self::SenderRevoke);
        }
        None
    }
}

/// Unwrap wrapper message types to reach the inner message. Does not unwrap
/// `edited_message`; that field is itself a signal callers may need.
fn unwrap_message(msg: &wa::Message) -> &wa::Message {
    macro_rules! try_unwrap_one {
        ($field:ident) => {
            if let Some(wrapper) = msg.$field.as_option()
                && let Some(inner) = wrapper.message.as_option()
            {
                return unwrap_message(inner);
            }
        };
    }
    try_unwrap_one!(ephemeral_message);
    try_unwrap_one!(view_once_message);
    try_unwrap_one!(view_once_message_v2);
    try_unwrap_one!(view_once_message_v2_extension);
    try_unwrap_one!(document_with_caption_message);
    try_unwrap_one!(group_mentioned_message);
    try_unwrap_one!(bot_invoke_message);
    try_unwrap_one!(associated_child_message);
    try_unwrap_one!(poll_creation_option_image_message);
    try_unwrap_one!(event_cover_image);
    try_unwrap_one!(group_status_message);
    try_unwrap_one!(group_status_message_v2);
    try_unwrap_one!(group_status_mention_message);
    try_unwrap_one!(status_add_yours);
    try_unwrap_one!(status_mention_message);
    try_unwrap_one!(question_message);
    try_unwrap_one!(question_reply_message);
    try_unwrap_one!(spoiler_message);
    try_unwrap_one!(lottie_sticker_message);
    try_unwrap_one!(limit_sharing_message);
    try_unwrap_one!(newsletter_admin_profile_message);
    try_unwrap_one!(newsletter_admin_profile_message_v2);
    try_unwrap_one!(poll_creation_message_v4);
    try_unwrap_one!(bot_forwarded_message);
    if let Some(dsm) = msg.device_sent_message.as_option()
        && let Some(inner) = dsm.message.as_option()
    {
        return unwrap_message(inner);
    }
    msg
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub enum BotEditType {
    First,
    Inner,
    Last,
}

#[derive(Debug, Clone, Serialize)]
pub struct MsgBotInfo {
    pub edit_type: Option<BotEditType>,
    pub edit_target_id: Option<MessageId>,
    pub edit_sender_timestamp_ms: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Default, Serialize)]
pub struct MsgMetaInfo {
    pub target_id: Option<MessageId>,
    pub target_sender: Option<Jid>,
    pub deprecated_lid_session: Option<bool>,
    pub thread_message_id: Option<MessageId>,
    pub thread_message_sender_jid: Option<Jid>,
}

#[derive(Debug, Clone, Default, Serialize)]
pub struct MessageInfo {
    pub source: MessageSource,
    pub id: MessageId,
    pub server_id: MessageServerId,
    pub r#type: String,
    pub push_name: String,
    pub timestamp: DateTime<Utc>,
    pub category: String,
    pub multicast: bool,
    pub media_type: String,
    pub edit: EditAttribute,
    pub bot_info: Option<MsgBotInfo>,
    pub meta_info: MsgMetaInfo,
    pub verified_name: Option<wa::VerifiedNameCertificate>,
    pub device_sent_meta: Option<DeviceSentMeta>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_edit_attribute_parsing_and_serialization() {
        // Test all known edit attribute values
        let attrs = vec![
            ("", EditAttribute::Empty),
            ("1", EditAttribute::MessageEdit),
            ("2", EditAttribute::PinInChat),
            ("3", EditAttribute::AdminEdit),
            ("7", EditAttribute::SenderRevoke),
            ("8", EditAttribute::AdminRevoke),
        ];

        for (string_val, expected_attr) in attrs {
            let parsed = EditAttribute::from(string_val.to_string());
            assert_eq!(parsed, expected_attr);
            assert_eq!(parsed.to_string_val(), string_val);
        }

        // Unknown values should be preserved
        assert_eq!(
            EditAttribute::from("99".to_string()),
            EditAttribute::Unknown("99".to_string())
        );
        assert_eq!(
            EditAttribute::Unknown("anything".to_string()).to_string_val(),
            ""
        );
    }

    #[test]
    fn test_decrypt_fail_hide_logic_for_edits() {
        // Documents the logic used in prepare_group_stanza (wacore/src/send.rs).
        // The decrypt-fail="hide" attribute is added for edited messages to hide
        // failed decryption attempts. However, admin revokes should NOT have it
        // because WhatsApp Web doesn't include it, and the server rejects it.

        fn should_add_decrypt_fail_hide(edit: &EditAttribute) -> bool {
            *edit != EditAttribute::Empty && *edit != EditAttribute::AdminRevoke
        }

        // Should add decrypt-fail="hide"
        assert!(should_add_decrypt_fail_hide(&EditAttribute::MessageEdit));
        assert!(should_add_decrypt_fail_hide(&EditAttribute::PinInChat));
        assert!(should_add_decrypt_fail_hide(&EditAttribute::AdminEdit));
        assert!(should_add_decrypt_fail_hide(&EditAttribute::SenderRevoke));

        // Should NOT add decrypt-fail="hide"
        assert!(!should_add_decrypt_fail_hide(&EditAttribute::Empty));
        assert!(!should_add_decrypt_fail_hide(&EditAttribute::AdminRevoke));
    }
}
