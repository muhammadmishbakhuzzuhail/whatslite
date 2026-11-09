use wa_rs::features::chat_actions::{
    ChatActions, ClearChatOptions, MediaDeletion, SyncActionMessageRange,
};
#[test]
fn chat_action_types_are_stable() {
    assert!(!ClearChatOptions::default().delete_media);
    assert!(!MediaDeletion::default().delete_media);
    assert_eq!(SyncActionMessageRange::default().start, None);
    let _ = ChatActions;
}
