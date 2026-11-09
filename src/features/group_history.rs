//! Group history sharing tokens and outcomes.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GroupHistorySkipReason {
    NoHistory,
    NotMember,
    Disabled,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupHistoryRetryToken(pub String);

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum GroupHistoryShareOutcome {
    Shared,
    Skipped(GroupHistorySkipReason),
    Retry(GroupHistoryRetryToken),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupHistoryAddResult {
    pub added: usize,
    pub outcome: GroupHistoryShareOutcome,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct HistorySharePreparation {
    pub messages: usize,
}

macro_rules! history_field_whitelist {
    ($predicate:ident, $ty:ident, [$($allowed:ident),*], [$($field:ident => $empty:ident),* $(,)?]) => {
        fn $predicate(value: &waproto::whatsapp::$ty) -> bool {
            $(let _ = &value.$allowed;)*
            true $(&& value.$field.$empty())*
        }
    };
}

history_field_whitelist!(
    history_message_fields_are_empty, Message, [conversation, message_context_info], [
                sender_key_distribution_message => is_unset,
                image_message => is_unset,
                contact_message => is_unset,
                location_message => is_unset,
                extended_text_message => is_unset,
                document_message => is_unset,
                audio_message => is_unset,
                video_message => is_unset,
                call => is_unset,
                chat => is_unset,
                protocol_message => is_unset,
                contacts_array_message => is_unset,
                highly_structured_message => is_unset,
                fast_ratchet_key_sender_key_distribution_message => is_unset,
                send_payment_message => is_unset,
                live_location_message => is_unset,
                request_payment_message => is_unset,
                decline_payment_request_message => is_unset,
                cancel_payment_request_message => is_unset,
                template_message => is_unset,
                sticker_message => is_unset,
                group_invite_message => is_unset,
                template_button_reply_message => is_unset,
                product_message => is_unset,
                device_sent_message => is_unset,
                list_message => is_unset,
                view_once_message => is_unset,
                order_message => is_unset,
                list_response_message => is_unset,
                ephemeral_message => is_unset,
                invoice_message => is_unset,
                buttons_message => is_unset,
                buttons_response_message => is_unset,
                payment_invite_message => is_unset,
                interactive_message => is_unset,
                reaction_message => is_unset,
                sticker_sync_rmr_message => is_unset,
                interactive_response_message => is_unset,
                poll_creation_message => is_unset,
                poll_update_message => is_unset,
                keep_in_chat_message => is_unset,
                document_with_caption_message => is_unset,
                request_phone_number_message => is_unset,
                view_once_message_v2 => is_unset,
                enc_reaction_message => is_unset,
                edited_message => is_unset,
                view_once_message_v2_extension => is_unset,
                poll_creation_message_v2 => is_unset,
                scheduled_call_creation_message => is_unset,
                group_mentioned_message => is_unset,
                pin_in_chat_message => is_unset,
                poll_creation_message_v3 => is_unset,
                scheduled_call_edit_message => is_unset,
                ptv_message => is_unset,
                bot_invoke_message => is_unset,
                call_log_messsage => is_unset,
                message_history_bundle => is_unset,
                enc_comment_message => is_unset,
                bcall_message => is_unset,
                lottie_sticker_message => is_unset,
                event_message => is_unset,
                enc_event_response_message => is_unset,
                comment_message => is_unset,
                newsletter_admin_invite_message => is_unset,
                placeholder_message => is_unset,
                secret_encrypted_message => is_unset,
                album_message => is_unset,
                event_cover_image => is_unset,
                sticker_pack_message => is_unset,
                status_mention_message => is_unset,
                poll_result_snapshot_message => is_unset,
                poll_creation_option_image_message => is_unset,
                associated_child_message => is_unset,
                group_status_mention_message => is_unset,
                poll_creation_message_v4 => is_unset,
                status_add_yours => is_unset,
                group_status_message => is_unset,
                rich_response_message => is_unset,
                status_notification_message => is_unset,
                limit_sharing_message => is_unset,
                bot_task_message => is_unset,
                question_message => is_unset,
                message_history_notice => is_unset,
                group_status_message_v2 => is_unset,
                bot_forwarded_message => is_unset,
                status_question_answer_message => is_unset,
                question_reply_message => is_unset,
                question_response_message => is_unset,
                status_quoted_message => is_unset,
                status_sticker_interaction_message => is_unset,
                poll_creation_message_v5 => is_unset,
                newsletter_follower_invite_message_v2 => is_unset,
                poll_result_snapshot_message_v3 => is_unset,
                newsletter_admin_profile_message => is_unset,
                newsletter_admin_profile_message_v2 => is_unset,
                spoiler_message => is_unset,
                poll_creation_message_v6 => is_unset,
                conditional_reveal_message => is_unset,
                poll_add_option_message => is_unset,
                event_invite_message => is_unset,
                group_root_key_share => is_unset,
                payment_reminder_message => is_unset,
                split_payment_message => is_unset,
                newsletter_admin_profile_status_message => is_unset,
                root_secret_distribute_message => is_unset,
                split_payment_update_message => is_unset,
                music_message => is_unset,
                status_link_preview_metadata => is_unset,
                bot_platform_registration_success_message => is_unset,
                newsletter_scheduled_message => is_unset,
                acp2_setting_message => is_unset,
                audio_sticker_message => is_unset,
                instant_image_message => is_unset,
                request_location_message => is_unset,
                bot_group_participant_message => is_unset,
                request_location_update_message => is_unset,
            ]
);

history_field_whitelist!(
    history_context_fields_are_empty, MessageContextInfo, [message_secret, reporting_token_version], [
                        device_list_metadata => is_unset,
                        device_list_metadata_version => is_none,
                        padding_bytes => is_none,
                        message_add_on_duration_in_secs => is_none,
                        bot_message_secret => is_none,
                        bot_metadata => is_unset,
                        message_add_on_expiry_type => is_none,
                        message_association => is_unset,
                        capi_created_group => is_none,
                        support_payload => is_none,
                        limit_sharing => is_unset,
                        limit_sharing_v2 => is_unset,
                        thread_id => is_empty,
                        weblink_render_config => is_none,
                        tee_bot_metadata => is_none,
                        account_encryption_attestation => is_unset,
                        associated_primary_identity_key => is_none,
                        tee_context_anchor_message_id => is_none,
                        acp2_setting => is_unset,
                        light_e2_ee => is_none,
                    ]
);

pub fn is_shareable_history_text(content: &waproto::whatsapp::Message) -> bool {
    content
        .conversation
        .as_deref()
        .is_some_and(|text| !text.is_empty())
        && history_message_fields_are_empty(content)
        && content
            .message_context_info
            .as_option()
            .is_none_or(history_context_fields_are_empty)
}
