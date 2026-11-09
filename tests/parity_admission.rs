//! Phase 1 admission and durability contracts.

use std::sync::Arc;
use std::time::Duration;

use wa_rs::types::connect_admission::ConnectAdmission;
use wa_rs::types::decrypt_failure_admission::{
    DecryptFailureAdmission, DecryptFailureContext, DecryptFailureDecision,
};
use wa_rs::types::history_sync_admission::{
    HistorySyncAdmission, HistorySyncDecision, HistorySyncMetadata,
};
use wa_rs::types::retry_admission::{RetryAdmission, RetryContext, RetryDecision};
use wacore_binary::Jid;

struct ConnectPolicy;
impl ConnectAdmission for ConnectPolicy {
    fn delay(&self) -> Duration {
        Duration::from_millis(25)
    }
    fn recheck(&self) -> Duration {
        Duration::from_millis(5)
    }
}

struct DecryptPolicy;
impl DecryptFailureAdmission for DecryptPolicy {
    fn decide(&self, context: &DecryptFailureContext<'_>) -> DecryptFailureDecision {
        if context.retry_count == 0 {
            DecryptFailureDecision::RequestRetry
        } else {
            DecryptFailureDecision::AcknowledgeWithoutRetry
        }
    }
}

struct HistoryPolicy;
impl HistorySyncAdmission for HistoryPolicy {
    fn decide(&self, metadata: &HistorySyncMetadata<'_>) -> HistorySyncDecision {
        if metadata.file_length.unwrap_or(0) > 1_000_000 {
            HistorySyncDecision::RejectAndAcknowledge
        } else {
            HistorySyncDecision::Accept
        }
    }
}

struct RetryPolicy;
impl RetryAdmission for RetryPolicy {
    fn decide(&self, context: &RetryContext<'_>) -> RetryDecision {
        if context.retry_count > 3 {
            RetryDecision::Drop
        } else {
            RetryDecision::Accept
        }
    }
}

#[test]
fn connect_admission_exposes_delay_and_recheck() {
    let policy: Arc<dyn ConnectAdmission> = Arc::new(ConnectPolicy);
    assert_eq!(policy.delay(), Duration::from_millis(25));
    assert_eq!(policy.recheck(), Duration::from_millis(5));
}

#[test]
fn decrypt_and_retry_admissions_return_expected_decisions() {
    let chat = Jid::pn("12025550111");
    let sender = Jid::pn("12025550112");
    let decrypt = DecryptPolicy;
    assert_eq!(
        decrypt.decide(&DecryptFailureContext::new(&chat, &sender, 0)),
        DecryptFailureDecision::RequestRetry
    );
    assert_eq!(
        decrypt.decide(&DecryptFailureContext::new(&chat, &sender, 1)),
        DecryptFailureDecision::AcknowledgeWithoutRetry
    );

    let retry = RetryPolicy;
    assert_eq!(
        retry.decide(&RetryContext::new(&chat, &sender, 3)),
        RetryDecision::Accept
    );
    assert_eq!(
        retry.decide(&RetryContext::new(&chat, &sender, 4)),
        RetryDecision::Drop
    );
}

#[test]
fn history_admission_can_acknowledge_without_processing() {
    let policy = HistoryPolicy;
    assert_eq!(
        policy.decide(&HistorySyncMetadata::default()),
        HistorySyncDecision::Accept
    );
    assert_eq!(
        policy.decide(&{
            let mut metadata = HistorySyncMetadata::default();
            metadata.file_length = Some(2_000_000);
            metadata
        }),
        HistorySyncDecision::RejectAndAcknowledge
    );
}
