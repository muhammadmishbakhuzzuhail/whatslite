use wa_rs::features::creation::{CreatedEvent, CreatedPoll, EventRef, PollRef};
use wa_rs::features::group_history::{
    GroupHistoryRetryToken, GroupHistoryShareOutcome, GroupHistorySkipReason,
};
use wa_rs::features::signal::{Signal, SignalError, SignalSessionInfo, SignalSessionMigration};
use wa_rs::features::stanza::{RetryRequestOptions, StanzaRejection};

#[test]
fn creation_and_group_history_types_are_stable() {
    assert_eq!(CreatedPoll { id: "p".into() }.id, "p");
    assert_eq!(CreatedEvent { id: "e".into() }.id, "e");
    assert_eq!(PollRef { id: "p".into() }.id, "p");
    assert_eq!(EventRef { id: "e".into() }.id, "e");
    assert_eq!(
        GroupHistoryShareOutcome::Skipped(GroupHistorySkipReason::NoHistory),
        GroupHistoryShareOutcome::Skipped(GroupHistorySkipReason::NoHistory)
    );
    assert_eq!(
        GroupHistoryShareOutcome::Retry(GroupHistoryRetryToken("t".into())),
        GroupHistoryShareOutcome::Retry(GroupHistoryRetryToken("t".into()))
    );
}

#[test]
fn signal_and_stanza_surfaces_are_reachable() {
    let _ = Signal;
    assert_eq!(
        SignalSessionMigration::Migrated { sessions: 1 },
        SignalSessionMigration::Migrated { sessions: 1 }
    );
    assert!(SignalError::SessionNotFound.to_string().contains("session"));
    let info = SignalSessionInfo {
        jid: "a".into(),
        has_session: true,
        device_count: 1,
    };
    assert!(info.has_session);
    assert_eq!(
        RetryRequestOptions {
            count: 1,
            force_include_keys: false
        }
        .count,
        1
    );
    assert_eq!(
        StanzaRejection {
            code: 400,
            text: "bad".into()
        }
        .code,
        400
    );
}
