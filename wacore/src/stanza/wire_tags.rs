//! Auto-generated stanza and notification vocabulary (WhatsApp 2.3000.1052559453). DO NOT EDIT.
//!
//! The tag a stanza arrives under, and the type of a `<notification>`.
//!
//! Regenerate with `cargo run -p whatspec-codegen`; never edit by hand.

/// The tag a stanza arrives under.
///
/// The union of WA Web's stanza dispatcher, the requests the server
/// sends us, and the types this client sends: no one document lists
/// them all.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, crate::WireEnum)]
pub enum StanzaTag {
    #[wire = "ack"]
    Ack,
    #[wire = "call"]
    Call,
    #[wire = "chatstate"]
    ChatState,
    #[wire = "error"]
    ErrorStanza,
    #[wire = "failure"]
    Failure,
    #[wire = "ib"]
    InfoBanner,
    #[wire = "iq"]
    Iq,
    #[wire = "message"]
    Message,
    #[wire = "notification"]
    Notification,
    #[wire = "presence"]
    Presence,
    #[wire = "receipt"]
    Receipt,
    #[wire = "status"]
    Status,
    #[wire = "stream:error"]
    StreamError,
    #[wire = "success"]
    Success,
    #[wire = "xmlstreamend"]
    XmlStreamEnd,
}

/// The `type` attribute of an incoming `<notification>`.
///
/// Every value WA Web routes. This client handles a subset and
/// forwards the rest as a raw event, so a variant here is a value the
/// protocol carries, not a promise that anything acts on it.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, crate::WireEnum)]
pub enum NotificationType {
    #[wire = "account_sync"]
    AccountSync,
    #[wire = "biz_thread_data"]
    BizThreadData,
    #[wire = "biz_thread_data_bulk"]
    BizThreadDataBulk,
    #[wire = "business"]
    Business,
    #[wire = "companion_reg_refresh"]
    CompanionRegRefresh,
    #[wire = "contacts"]
    Contacts,
    #[wire = "crsc_continuation"]
    CrscContinuation,
    #[wire = "devices"]
    Devices,
    #[wire = "digital_commerce_subscription"]
    DigitalCommerceSubscription,
    #[wire = "disappearing_mode"]
    DisappearingMode,
    #[wire = "encrypt"]
    Encrypt,
    #[wire = "fb:update"]
    FbUpdate,
    #[wire = "hosted"]
    Hosted,
    #[wire = "link_code_companion_reg"]
    LinkCodeCompanionReg,
    #[wire = "mediaretry"]
    MediaRetry,
    #[wire = "mex"]
    Mex,
    #[wire = "newsletter"]
    Newsletter,
    #[wire = "passkey_prologue_request"]
    PasskeyPrologueRequest,
    #[wire = "pay"]
    Pay,
    #[wire = "picture"]
    Picture,
    #[wire = "privacy_token"]
    PrivacyToken,
    #[wire = "psa"]
    Psa,
    #[wire = "registration"]
    Registration,
    #[wire = "server"]
    Server,
    #[wire = "server_sync"]
    ServerSync,
    #[wire = "status"]
    Status,
    #[wire = "w:gp2"]
    WGp2,
    #[wire = "w:growth"]
    WGrowth,
    #[wire = "waffle"]
    Waffle,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stanza_tag_wire_values() {
        let cases = [
            (StanzaTag::Ack, "ack"),
            (StanzaTag::Call, "call"),
            (StanzaTag::ChatState, "chatstate"),
            (StanzaTag::ErrorStanza, "error"),
            (StanzaTag::Failure, "failure"),
            (StanzaTag::InfoBanner, "ib"),
            (StanzaTag::Iq, "iq"),
            (StanzaTag::Message, "message"),
            (StanzaTag::Notification, "notification"),
            (StanzaTag::Presence, "presence"),
            (StanzaTag::Receipt, "receipt"),
            (StanzaTag::Status, "status"),
            (StanzaTag::StreamError, "stream:error"),
            (StanzaTag::Success, "success"),
            (StanzaTag::XmlStreamEnd, "xmlstreamend"),
        ];
        for (variant, wire) in cases {
            assert_eq!(variant.as_str(), wire);
            assert_eq!(
                StanzaTag::try_from(wire).expect("known stanza tag"),
                variant
            );
        }
        assert_eq!(StanzaTag::default(), StanzaTag::Ack);
    }

    #[test]
    fn stanza_tag_unknown_is_rejected() {
        assert!(StanzaTag::try_from("bogus").is_err());
        assert!(StanzaTag::try_from("").is_err());
        assert!(StanzaTag::try_from("Message").is_err());
    }

    #[test]
    fn notification_type_wire_values() {
        let cases = [
            (NotificationType::AccountSync, "account_sync"),
            (NotificationType::BizThreadData, "biz_thread_data"),
            (NotificationType::BizThreadDataBulk, "biz_thread_data_bulk"),
            (NotificationType::Business, "business"),
            (
                NotificationType::CompanionRegRefresh,
                "companion_reg_refresh",
            ),
            (NotificationType::Contacts, "contacts"),
            (NotificationType::CrscContinuation, "crsc_continuation"),
            (NotificationType::Devices, "devices"),
            (
                NotificationType::DigitalCommerceSubscription,
                "digital_commerce_subscription",
            ),
            (NotificationType::DisappearingMode, "disappearing_mode"),
            (NotificationType::Encrypt, "encrypt"),
            (NotificationType::FbUpdate, "fb:update"),
            (NotificationType::Hosted, "hosted"),
            (
                NotificationType::LinkCodeCompanionReg,
                "link_code_companion_reg",
            ),
            (NotificationType::MediaRetry, "mediaretry"),
            (NotificationType::Mex, "mex"),
            (NotificationType::Newsletter, "newsletter"),
            (
                NotificationType::PasskeyPrologueRequest,
                "passkey_prologue_request",
            ),
            (NotificationType::Pay, "pay"),
            (NotificationType::Picture, "picture"),
            (NotificationType::PrivacyToken, "privacy_token"),
            (NotificationType::Psa, "psa"),
            (NotificationType::Registration, "registration"),
            (NotificationType::Server, "server"),
            (NotificationType::ServerSync, "server_sync"),
            (NotificationType::Status, "status"),
            (NotificationType::WGp2, "w:gp2"),
            (NotificationType::WGrowth, "w:growth"),
            (NotificationType::Waffle, "waffle"),
        ];
        for (variant, wire) in cases {
            assert_eq!(variant.as_str(), wire);
            assert_eq!(
                NotificationType::try_from(wire).expect("known notification type"),
                variant
            );
        }
        assert_eq!(NotificationType::default(), NotificationType::AccountSync);
    }

    #[test]
    fn notification_type_unknown_is_rejected() {
        assert!(NotificationType::try_from("bogus").is_err());
        assert!(NotificationType::try_from("").is_err());
        assert!(NotificationType::try_from("server_sync ").is_err());
    }
}
