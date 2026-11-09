//! Phase 1 sync-engine contract: scope admission, history admission and
//! publication, activity accounting and retry budgets.

use std::time::Duration;

use wa_rs_core::appstate::patch_decode::WAPatchName;
use wa_rs_core::sync_engine::app_state::{
    BatchedSyncOutcome, CriticalSyncPlan, RetryBudget, ScopeLost, SyncScope,
};
use wa_rs_core::sync_engine::history::{
    HistoryActivity, HistoryAdmission, HistoryPublication, HistoryReceiptPoint, HistoryRecovery,
};
use wa_rs_core::time::Instant;

#[test]
fn sync_scope_rejects_retired_and_expired_runs() {
    let scope = SyncScope {
        generation: 4,
        deadline: Some(Instant::ZERO.checked_add(Duration::from_secs(5)).unwrap()),
    };
    assert!(scope.admits(4, || Instant::ZERO).is_ok());
    assert_eq!(
        scope.admits(3, || Instant::ZERO).unwrap_err(),
        ScopeLost::Retired
    );
    assert_eq!(
        scope
            .admits(4, || Instant::ZERO
                .checked_add(Duration::from_secs(5))
                .unwrap())
            .unwrap_err(),
        ScopeLost::Expired
    );
}

#[test]
fn history_skip_takes_precedence_over_consumer_policy() {
    assert_eq!(
        HistoryAdmission::decide(true, || panic!("policy must not run")),
        HistoryAdmission::RejectAndAcknowledge
    );
    assert_eq!(
        HistoryAdmission::decide(false, || true),
        HistoryAdmission::RejectAndAcknowledge
    );
    assert_eq!(
        HistoryAdmission::decide(false, || false),
        HistoryAdmission::Process
    );
}

#[test]
fn history_receipt_point_depends_on_recovery_and_capture() {
    let durable = HistoryPublication::new(HistoryRecovery::Durable, false);
    assert!(durable.receipt_at(HistoryReceiptPoint::Parsed));
    assert!(!durable.receipt_at(HistoryReceiptPoint::BeforePayload));

    let captured = HistoryPublication::new(HistoryRecovery::Session, true);
    assert!(captured.receipt_at(HistoryReceiptPoint::Captured));

    let uncaptured = HistoryPublication::new(HistoryRecovery::Session, false);
    assert!(uncaptured.receipt_at(HistoryReceiptPoint::BeforePayload));
}

#[test]
fn history_activity_tracks_peak_and_release() {
    let mut activity = HistoryActivity::default();
    let generation = activity.begin(100);
    activity.begin(50);
    let peak = activity.snapshot();
    assert_eq!(peak.tasks, 2);
    assert_eq!(peak.payload_bytes, 150);

    assert!(!activity.release(generation, 50));
    assert!(activity.release(generation, 100));
    let snapshot = activity.snapshot();
    assert_eq!(snapshot.tasks, 0);
    assert_eq!(snapshot.payload_bytes, 0);
    assert_eq!(snapshot.tasks_peak, 2);
    assert_eq!(snapshot.payload_bytes_peak, 150);
}

#[test]
fn critical_sync_plan_reports_outstanding_collections() {
    let mut outcome = BatchedSyncOutcome::default();
    outcome.fatal.push(WAPatchName::CriticalBlock);
    outcome.retryable.push(WAPatchName::Regular);
    let plan = CriticalSyncPlan::from_outcome(&outcome);
    assert!(plan.outstanding());

    let mut complete = BatchedSyncOutcome::default();
    complete.synced.push(WAPatchName::CriticalBlock);
    assert!(!CriticalSyncPlan::from_outcome(&complete).outstanding());
}

#[test]
fn retry_budget_is_finite_and_backs_off() {
    let mut budget = RetryBudget::default();
    assert!(budget.next_delay().is_some());
    budget.record_attempt();
    assert_eq!(budget.attempts(), 1);
    assert!(budget.next_delay().unwrap() >= Duration::from_secs(1));
}
