//! Downstream contract for the typed group-operation error.
//!
//! Only public imports are used: this file must keep compiling for an external
//! consumer that matches on `GroupError` variants.

use std::collections::HashMap;
use std::error::Error as _;

use wa_rs::features::GroupError as FeatureGroupError;
use wa_rs::request::IqError;
use wa_rs::wacore_binary::jid::Jid;
use wa_rs::{
    CreateGroupResult, GroupCreateOptions, GroupDescription, GroupError, GroupMetadata,
    GroupSubject, Groups, ParticipantChangeResponse, wacore::client::context::GroupInfo,
};

#[test]
fn group_error_is_reachable_from_both_public_paths() {
    let from_features: GroupError = FeatureGroupError::DescriptionConflict;
    let from_crate: GroupError = wa_rs::GroupError::DescriptionConflict;
    assert_eq!(from_features.to_string(), from_crate.to_string());
}

#[test]
fn iq_error_display_forwards_and_exposes_source() {
    let error = GroupError::from(IqError::ServerError {
        code: 403,
        text: "not-authorized".to_string(),
    });

    assert_eq!(
        error.to_string(),
        "Received a server error response: code=403, text='not-authorized'"
    );

    let source = error.source().expect("Iq source is exposed");
    let iq = source
        .downcast_ref::<IqError>()
        .expect("source is the IqError");
    assert!(matches!(iq, IqError::ServerError { code: 403, .. }));
}

#[test]
fn unit_variant_messages_are_exact() {
    assert_eq!(
        GroupError::EmptyPicture.to_string(),
        "picture data cannot be empty; use remove_profile_picture instead"
    );
    assert_eq!(
        GroupError::InvalidRequest("empty invite code".to_string()).to_string(),
        "invalid group request: empty invite code"
    );
    assert_eq!(
        GroupError::DescriptionConflict.to_string(),
        "the group description changed since it was read"
    );
}

#[test]
fn internal_error_preserves_display_and_source() {
    let error = GroupError::from(anyhow::anyhow!("missing phone mapping"));

    assert_eq!(error.to_string(), "missing phone mapping");
    assert!(error.source().is_some(), "anyhow source is preserved");
}

/// Compile-only: every fallible group operation returns the typed error.
///
/// These wrappers fail to compile if a method signature regresses to
/// `anyhow::Error` or another error type.
#[allow(dead_code)]
mod signatures {
    use super::*;

    async fn query_info(groups: &Groups<'_>, jid: &Jid) -> Result<GroupInfo, GroupError> {
        groups.query_info(jid).await
    }

    async fn get_participating(
        groups: &Groups<'_>,
    ) -> Result<HashMap<String, GroupMetadata>, GroupError> {
        groups.get_participating().await
    }

    async fn get_metadata(groups: &Groups<'_>, jid: &Jid) -> Result<GroupMetadata, GroupError> {
        groups.get_metadata(jid).await
    }

    async fn create_group(
        groups: &Groups<'_>,
        options: GroupCreateOptions,
    ) -> Result<CreateGroupResult, GroupError> {
        groups.create_group(options).await
    }

    async fn set_subject(
        groups: &Groups<'_>,
        jid: &Jid,
        subject: GroupSubject,
    ) -> Result<(), GroupError> {
        groups.set_subject(jid, subject).await
    }

    async fn set_description(
        groups: &Groups<'_>,
        jid: &Jid,
        description: Option<GroupDescription>,
        prev: Option<String>,
    ) -> Result<(), GroupError> {
        groups.set_description(jid, description, prev).await
    }

    async fn leave(groups: &Groups<'_>, jid: &Jid) -> Result<(), GroupError> {
        groups.leave(jid).await
    }

    async fn add_participants(
        groups: &Groups<'_>,
        jid: &Jid,
        participants: &[Jid],
    ) -> Result<Vec<ParticipantChangeResponse>, GroupError> {
        groups.add_participants(jid, participants).await
    }

    async fn remove_participants(
        groups: &Groups<'_>,
        jid: &Jid,
        participants: &[Jid],
    ) -> Result<Vec<ParticipantChangeResponse>, GroupError> {
        groups.remove_participants(jid, participants).await
    }

    async fn promote_participants(
        groups: &Groups<'_>,
        jid: &Jid,
        participants: &[Jid],
    ) -> Result<(), GroupError> {
        groups.promote_participants(jid, participants).await
    }

    async fn demote_participants(
        groups: &Groups<'_>,
        jid: &Jid,
        participants: &[Jid],
    ) -> Result<(), GroupError> {
        groups.demote_participants(jid, participants).await
    }

    async fn get_invite_link(
        groups: &Groups<'_>,
        jid: &Jid,
        reset: bool,
    ) -> Result<String, GroupError> {
        groups.get_invite_link(jid, reset).await
    }

    fn all_signatures_referenced() {
        let _ = query_info;
        let _ = get_participating;
        let _ = get_metadata;
        let _ = create_group;
        let _ = set_subject;
        let _ = set_description;
        let _ = leave;
        let _ = add_participants;
        let _ = remove_participants;
        let _ = promote_participants;
        let _ = demote_participants;
        let _ = get_invite_link;
    }
}
