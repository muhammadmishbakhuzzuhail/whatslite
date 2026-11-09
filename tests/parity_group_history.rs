use wa_rs::features::group_history::{
    GroupHistoryAddResult, GroupHistoryShareOutcome, HistorySharePreparation,
};
#[test]
fn group_history_results_are_stable() {
    assert_eq!(HistorySharePreparation { messages: 2 }.messages, 2);
    let result = GroupHistoryAddResult {
        added: 1,
        outcome: GroupHistoryShareOutcome::Shared,
    };
    assert_eq!(result.added, 1);
}
