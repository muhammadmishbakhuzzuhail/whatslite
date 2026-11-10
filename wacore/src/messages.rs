use crate::libsignal::crypto::CryptographicHash;
use anyhow::{Result, anyhow};
use base64::Engine as _;
use waproto::whatsapp as wa;

pub struct MessageUtils;

impl MessageUtils {
    pub fn pad_message_v2(mut plaintext: Vec<u8>) -> Vec<u8> {
        use rand::Rng;
        let mut rng = rand::rng();

        let mut pad_val = rng.random::<u8>() & 0x0F;
        if pad_val == 0 {
            pad_val = 0x0F;
        }

        let padding = vec![pad_val; pad_val as usize];
        plaintext.extend_from_slice(&padding);
        plaintext
    }

    pub fn participant_list_hash(devices: &[wacore_binary::jid::Jid]) -> Result<String> {
        let mut jids: Vec<String> = devices.iter().map(|j| j.to_ad_string()).collect();
        jids.sort();

        let concatenated_jids = jids.join("");

        // Use finalize_sha256_array() for zero-allocation hash finalization
        let full_hash = {
            let mut h = CryptographicHash::new("SHA-256")
                .map_err(|e| anyhow!("failed to initialize SHA-256 hasher: {:?}", e))?;
            h.update(concatenated_jids.as_bytes());
            h.finalize_sha256_array()
                .map_err(|e| anyhow!("failed to finalize hash: {:?}", e))?
        };

        let truncated_hash = &full_hash[..6];

        Ok(format!(
            "2:{hash}",
            hash = base64::prelude::BASE64_URL_SAFE_NO_PAD.encode(truncated_hash)
        ))
    }

    pub fn unpad_message_ref(plaintext: &[u8], version: u8) -> Result<&[u8], anyhow::Error> {
        if version == 3 {
            return Ok(plaintext);
        }
        if plaintext.is_empty() {
            return Err(anyhow::anyhow!("plaintext is empty, cannot unpad"));
        }
        let pad_len = plaintext[plaintext.len() - 1] as usize;
        if pad_len == 0 || pad_len > plaintext.len() {
            return Err(anyhow::anyhow!("invalid padding length: {}", pad_len));
        }
        let (data, padding) = plaintext.split_at(plaintext.len() - pad_len);
        for &byte in padding {
            if byte != pad_len as u8 {
                return Err(anyhow::anyhow!("invalid padding bytes"));
            }
        }
        Ok(data)
    }
}

/// Decode padded ciphertext into a `wa::Message`.
///
/// Unpads the plaintext (using the given padding version) and decodes the
/// protobuf bytes into a WhatsApp Message. This is the pure,
/// runtime-independent portion of `handle_decrypted_plaintext`.
pub fn decode_plaintext(padded_plaintext: &[u8], padding_version: u8) -> Result<wa::Message> {
    let plaintext_slice = MessageUtils::unpad_message_ref(padded_plaintext, padding_version)?;
    // Route through the pinned codec entry point so the Message decode tree
    // (BotMetadata/ProtocolMessage/ContextInfo merge_field, etc.) is
    // instantiated once in waproto instead of copied into every calling crate.
    waproto::codec::message_decode(plaintext_slice)
        .map_err(|e| anyhow!("Failed to decode decrypted plaintext: {e}"))
}

/// Unwrap a DeviceSentMessage wrapper, returning the inner message.
///
/// When a message is sent from our own device, the actual content is nested
/// inside `device_sent_message.message`.  This function extracts that inner
/// message (preserving `message_context_info`), or returns the original
/// message unchanged when there is no wrapper or the wrapper has no inner
/// message.
pub fn unwrap_device_sent(mut msg: wa::Message) -> wa::Message {
    if let Some(mut dsm) = msg.device_sent_message.take() {
        if let Some(mut inner) = dsm.message.take() {
            inner.message_context_info = crate::proto_helpers::merge_dsm_context(
                inner.message_context_info.take(),
                msg.message_context_info.as_option(),
            )
            .map(buffa::MessageField::some)
            .unwrap_or_default();
            return inner;
        }
        msg.device_sent_message = buffa::MessageField::some(dsm);
    }
    msg
}

/// Returns `true` if the message contains only a SenderKey distribution
/// (internal key-exchange for group encryption) and no user-visible content.
///
/// When sending a group message, WhatsApp includes the SKDM in a separate
/// `pkmsg` enc node.  We must process it (store the sender key) but should
/// not surface it as a user event.
pub fn is_sender_key_distribution_only(msg: &mut wa::Message) -> bool {
    if msg.sender_key_distribution_message.is_unset()
        && msg
            .fast_ratchet_key_sender_key_distribution_message
            .is_unset()
    {
        return false;
    }

    // Fast path: most common user-visible fields (avoids the slow path for the typical case).
    if msg.conversation.is_some()
        || msg.extended_text_message.is_set()
        || msg.image_message.is_set()
        || msg.video_message.is_set()
        || msg.audio_message.is_set()
        || msg.document_message.is_set()
        || msg.reaction_message.is_set()
        || msg.protocol_message.is_set()
        || msg.sticker_message.is_set()
        || msg.contact_message.is_set()
        || msg.location_message.is_set()
        || msg.live_location_message.is_set()
    {
        return false;
    }

    // Slow path: temporarily take out the carrier fields and compare the rest to
    // default to catch all current and future fields, then restore them. This
    // avoids deep-cloning the whole Message just to clear three fields.
    let skdm = msg.sender_key_distribution_message.take();
    let fast = msg.fast_ratchet_key_sender_key_distribution_message.take();
    let ctx = msg.message_context_info.take();

    // proto fields only encode when non-default, so encoded length 0 means all
    // remaining fields are at default — i.e. the message has no user content.
    let mut cache = buffa::SizeCache::new();
    let only = waproto::codec::message_compute_size(msg, &mut cache) == 0;

    msg.sender_key_distribution_message = skdm.map(buffa::MessageField::some).unwrap_or_default();
    msg.fast_ratchet_key_sender_key_distribution_message =
        fast.map(buffa::MessageField::some).unwrap_or_default();
    msg.message_context_info = ctx.map(buffa::MessageField::some).unwrap_or_default();

    only
}

#[cfg(test)]
#[allow(clippy::disallowed_methods)]
mod tests {
    use super::*;

    /// Fixture stand-in for the send-path `wrap_device_sent`: builds the same
    /// wire shape the helper tests upstream exercise.
    fn wrap_in_dsm(inner: wa::Message, destination_jid: &str) -> wa::Message {
        {
            let mut proto = wa::Message::default();
            proto.device_sent_message = buffa::MessageField::some({
                let mut dsm = wa::message::DeviceSentMessage::default();
                dsm.destination_jid = Some(destination_jid.to_string());
                dsm.message = buffa::MessageField::some(inner);
                dsm
            });
            proto
        }
    }

    fn skdm(bytes: &[u8]) -> wa::message::SenderKeyDistributionMessage {
        {
            let mut proto = wa::message::SenderKeyDistributionMessage::default();
            proto.group_id = Some("group".into());
            proto.axolotl_sender_key_distribution_message = Some(bytes.to_vec());
            proto
        }
    }

    #[test]
    fn test_decode_plaintext_round_trips_padded_message() {
        use buffa::Message as _;

        let msg = {
            let mut proto = wa::Message::default();
            proto.conversation = Some("hello".to_string());
            proto
        };
        let padded = MessageUtils::pad_message_v2(msg.encode_to_vec());

        let decoded = decode_plaintext(&padded, 2).unwrap();
        assert_eq!(decoded.conversation.as_deref(), Some("hello"));
    }

    #[test]
    fn test_decode_plaintext_rejects_bad_padding() {
        let err = decode_plaintext(&[0x02], 2)
            .expect_err("padding length greater than available bytes must fail");
        assert!(err.to_string().contains("invalid padding length"));
    }

    #[test]
    fn test_is_sender_key_distribution_only() {
        let skdm = skdm(&[1, 2, 3]);

        // Empty message → false (no SKDM)
        assert!(!is_sender_key_distribution_only(&mut wa::Message::default()));

        // SKDM only → true
        assert!(is_sender_key_distribution_only(&mut {
            let mut proto = wa::Message::default();
            proto.sender_key_distribution_message = buffa::MessageField::some(skdm.clone());
            proto
        }));

        // SKDM + message_context_info → still true (context_info is metadata)
        assert!(is_sender_key_distribution_only(&mut {
            let mut proto = wa::Message::default();
            proto.sender_key_distribution_message = buffa::MessageField::some(skdm.clone());
            proto.message_context_info = buffa::MessageField::some(Default::default());
            proto
        }));

        // SKDM + sticker → false (has user content)
        assert!(!is_sender_key_distribution_only(&mut {
            let mut proto = wa::Message::default();
            proto.sender_key_distribution_message = buffa::MessageField::some(skdm.clone());
            proto.sticker_message =
                buffa::MessageField::some(wa::message::StickerMessage::default());
            proto
        }));

        // SKDM + text → false (has user content)
        assert!(!is_sender_key_distribution_only(&mut {
            let mut proto = wa::Message::default();
            proto.sender_key_distribution_message = buffa::MessageField::some(skdm.clone());
            proto.conversation = Some("hello".into());
            proto
        }));

        // protocol_message only (no SKDM) → false
        assert!(!is_sender_key_distribution_only(&mut {
            let mut proto_ = wa::Message::default();
            proto_.protocol_message =
                buffa::MessageField::some(wa::message::ProtocolMessage::default());
            proto_
        }));
    }

    #[test]
    fn skdm_only_detection_restores_carrier_fields() {
        // The slow path takes the carrier fields out to compare the rest against
        // default; it must restore them so callers still see the original message.
        let mut msg = {
            let mut proto = wa::Message::default();
            proto.sender_key_distribution_message = buffa::MessageField::some(skdm(&[1, 2, 3]));
            proto.fast_ratchet_key_sender_key_distribution_message =
                buffa::MessageField::some(skdm(&[4, 5, 6]));
            proto.message_context_info = buffa::MessageField::some({
                let mut proto = wa::MessageContextInfo::default();
                proto.message_secret = Some(vec![9, 8, 7]);
                proto
            });
            proto
        };

        assert!(is_sender_key_distribution_only(&mut msg));

        // Pin the exact payloads of all three taken/restored carrier fields, not
        // just presence: a buggy restore that put back a fresh default (losing the
        // original contents) must fail here.
        assert_eq!(
            msg.sender_key_distribution_message
                .as_option()
                .and_then(|s| s.axolotl_sender_key_distribution_message.as_deref()),
            Some([1, 2, 3].as_slice()),
            "sender_key_distribution_message payload must be restored unchanged"
        );
        assert_eq!(
            msg.fast_ratchet_key_sender_key_distribution_message
                .as_option()
                .and_then(|s| s.axolotl_sender_key_distribution_message.as_deref()),
            Some([4, 5, 6].as_slice()),
            "fast_ratchet carrier payload must be restored unchanged"
        );
        assert_eq!(
            msg.message_context_info
                .as_option()
                .and_then(|c| c.message_secret.as_deref()),
            Some([9, 8, 7].as_slice()),
            "message_context_info payload must be restored unchanged"
        );
    }

    /// Test: unwrap_device_sent extracts a reaction from a DeviceSentMessage wrapper.
    #[test]
    fn test_unwrap_device_sent_extracts_reaction() {
        let wrapped = wrap_in_dsm(
            {
                let mut proto = wa::Message::default();
                proto.reaction_message = buffa::MessageField::some({
                    let mut proto = wa::message::ReactionMessage::default();
                    proto.text = Some("\u{2764}".to_string());
                    proto
                });
                proto
            },
            "5511999999999@s.whatsapp.net",
        );

        let mut unwrapped = unwrap_device_sent(wrapped);
        assert!(
            unwrapped.device_sent_message.is_unset(),
            "DSM wrapper should be removed"
        );
        assert_eq!(
            unwrapped
                .reaction_message
                .as_option()
                .and_then(|r| r.text.as_deref()),
            Some("\u{2764}"),
            "reaction should be accessible after unwrapping"
        );
        assert!(
            !is_sender_key_distribution_only(&mut unwrapped),
            "unwrapped reaction should not be filtered as SKDM-only"
        );
    }

    /// Test: unwrap_device_sent preserves the wrapper when inner message is None.
    #[test]
    fn test_unwrap_device_sent_preserves_empty_wrapper() {
        let wrapped = {
            let mut proto = wa::Message::default();
            proto.device_sent_message = buffa::MessageField::some({
                let mut proto = wa::message::DeviceSentMessage::default();
                proto.destination_jid = Some("5511999999999@s.whatsapp.net".to_string());
                proto.message = Default::default();
                proto.phash = None;
                proto
            });
            proto
        };

        let result = unwrap_device_sent(wrapped);
        assert!(
            result.device_sent_message.is_set(),
            "empty DSM wrapper should be preserved"
        );
    }

    /// Test: unwrap_device_sent passes through a plain message unchanged.
    #[test]
    fn test_unwrap_device_sent_passthrough() {
        let msg = {
            let mut proto = wa::Message::default();
            proto.conversation = Some("hello".to_string());
            proto
        };

        let result = unwrap_device_sent(msg);
        assert_eq!(result.conversation.as_deref(), Some("hello"));
    }

    /// Test: unwrap_device_sent merges messageContextInfo from outer and inner,
    /// matching WAWebDeviceSentMessageProtoUtils.unwrapDeviceSentMessage.
    #[test]
    fn test_unwrap_device_sent_merges_context_info() {
        let mut wrapped = wrap_in_dsm(
            {
                let mut proto = wa::Message::default();
                proto.conversation = Some("hello".to_string()); // Inner has its own message_secret but no limit_sharing_v2
                proto.message_context_info = buffa::MessageField::some({
                    let mut proto = wa::MessageContextInfo::default();
                    proto.message_secret = Some(vec![1, 2, 3]);
                    proto
                });
                proto
            },
            "5511999999999@s.whatsapp.net",
        );
        // Outer message_context_info (from the DSM envelope)
        wrapped.message_context_info = buffa::MessageField::some({
            let mut proto = wa::MessageContextInfo::default();
            proto.message_secret = Some(vec![10, 20, 30]);
            proto.limit_sharing_v2 = buffa::MessageField::some(wa::LimitSharing::default());
            proto
        });

        let result = unwrap_device_sent(wrapped);
        let ctx = result.message_context_info.as_option().unwrap();

        assert_eq!(
            ctx.message_secret,
            Some(vec![1, 2, 3]),
            "inner message_secret should be preferred"
        );
        assert!(
            ctx.limit_sharing_v2.is_set(),
            "limit_sharing_v2 should come from outer (always)"
        );
    }

    /// Test: unwrap_device_sent falls back to outer message_secret when inner has none.
    #[test]
    fn test_unwrap_device_sent_secret_fallback() {
        let mut wrapped = wrap_in_dsm(
            {
                let mut proto = wa::Message::default();
                proto.conversation = Some("hello".to_string());
                proto
            },
            "5511999999999@s.whatsapp.net",
        );
        wrapped.message_context_info = buffa::MessageField::some({
            let mut proto = wa::MessageContextInfo::default();
            proto.message_secret = Some(vec![10, 20, 30]);
            proto
        });

        let result = unwrap_device_sent(wrapped);
        let ctx = result.message_context_info.as_option().unwrap();
        assert_eq!(
            ctx.message_secret,
            Some(vec![10, 20, 30]),
            "should fall back to outer message_secret"
        );
    }
}
