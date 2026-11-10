use super::Client;
use crate::types::group_notification_facts::{
    GroupNotificationFactsAlreadySet, GroupNotificationFactsProvider,
};
use std::sync::Arc;
use wacore::stanza::groups::GroupNotification;
use wacore::stanza::groups::normalized::{GroupParseContext, GroupParseError};
use wacore_binary::{Jid, JidExt, NodeRef};

fn is_me_account(jid: &Jid, pn: Option<&Jid>, lid: Option<&Jid>) -> bool {
    pn.is_some_and(|me| jid.is_same_chat_as(me)) || lid.is_some_and(|me| jid.is_same_chat_as(me))
}

fn parse_with_provider(
    node: &NodeRef<'_>,
    pn: Option<&Jid>,
    lid: Option<&Jid>,
    provider: &dyn GroupNotificationFactsProvider,
) -> Result<GroupNotification, GroupParseError> {
    let facts = provider.snapshot();
    let is_me = |jid: &Jid| is_me_account(jid, pn, lid);
    let normalize_country = |code: &str| provider.normalize_country_code(code);
    let context = GroupParseContext {
        me_device_lid: lid,
        me_device_pn: pn,
        username_displayed: facts.username_displayed,
        history_after_join: facts.history_after_join,
        opus_enabled: facts.opus_enabled,
        is_me_account: &is_me,
        normalize_country_code: &normalize_country,
    };
    GroupNotification::try_from_node_ref_with_context(node, &context)
}

impl Client {
    /// Opt into contextual group parsing using this account's effective facts.
    /// The provider is fixed after registration. It may internally expose newer
    /// coherent snapshots as the account configuration changes.
    pub fn set_group_notification_facts_provider(
        &self,
        provider: Arc<dyn GroupNotificationFactsProvider>,
    ) -> Result<(), GroupNotificationFactsAlreadySet> {
        self.group_notification_facts_provider
            .set(provider)
            .map_err(|_| GroupNotificationFactsAlreadySet)
    }

    /// Keep the existing adapter when the host has not supplied effective facts.
    /// Both branches parse before cache, storage and ACK effects.
    pub(crate) async fn parse_group_notification(
        &self,
        node: &NodeRef<'_>,
    ) -> Result<Option<GroupNotification>, GroupParseError> {
        let Some(provider) = self.group_notification_facts_provider.get() else {
            return Ok(GroupNotification::try_from_node_ref(node));
        };
        // This fork's snapshot is an owned clone behind an async lock, so the
        // wrapper is async where upstream's cached snapshot is sync.
        let device = self.persistence_manager.get_device_snapshot().await;
        parse_with_provider(
            node,
            device.pn.as_ref(),
            device.lid.as_ref(),
            provider.as_ref(),
        )
        .map(Some)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::store::persistence_manager::PersistenceManager;
    use crate::types::group_notification_facts::GroupNotificationFacts;
    use std::sync::atomic::{AtomicUsize, Ordering};
    use wacore_binary::builder::NodeBuilder;

    struct Provider {
        facts: GroupNotificationFacts,
        reads: AtomicUsize,
    }
    impl GroupNotificationFactsProvider for Provider {
        fn snapshot(&self) -> GroupNotificationFacts {
            self.reads.fetch_add(1, Ordering::Relaxed);
            self.facts
        }
        fn normalize_country_code(&self, code: &str) -> Option<String> {
            (code == "US").then(|| "US".into())
        }
    }
    fn provider(enabled: bool) -> Provider {
        Provider {
            facts: GroupNotificationFacts::builder()
                .username_displayed(enabled)
                .history_after_join(false)
                .opus_enabled(false)
                .build(),
            reads: AtomicUsize::new(0),
        }
    }
    fn base() -> NodeBuilder {
        NodeBuilder::new("notification")
            .attr("from", "123456@g.us")
            .attr("id", "synthetic")
            .attr("t", "123")
    }

    async fn client() -> Arc<Client> {
        let backend = crate::test_utils::create_test_backend().await;
        let pm = Arc::new(
            PersistenceManager::new(backend)
                .await
                .expect("persistence manager should initialize"),
        );
        let (client, _rx) = Client::new(
            pm,
            Arc::new(crate::transport::mock::MockTransportFactory::new()),
            Arc::new(crate::test_utils::MockHttpClient),
            None,
        )
        .await;
        client
    }

    #[test]
    fn account_matching_retains_namespace_and_ignores_device_only() {
        let pn = Jid::pn_device("200002", 2);
        let lid = Jid::lid_device("100001", 3);
        assert!(is_me_account(&pn.with_device(0), Some(&pn), Some(&lid)));
        assert!(is_me_account(&lid.with_device(9), Some(&pn), Some(&lid)));
        assert!(!is_me_account(
            &Jid::lid_device("200002", 0),
            Some(&pn),
            Some(&lid)
        ));
        assert!(!is_me_account(
            &Jid::pn_device("100001", 0),
            Some(&pn),
            Some(&lid)
        ));
    }

    #[test]
    fn effective_facts_are_read_once_and_destination_uses_device_snapshot() {
        let provider = provider(true);
        let lid = Jid::lid_device("100001", 2);
        let pn = Jid::pn_device("200002", 3);
        let node = base()
            .attr("to", pn.to_string())
            .attr("participant_username", "synthetic_user")
            .build();
        let parsed = parse_with_provider(
            &node.as_node_ref(),
            Some(&pn),
            Some(&lid),
            &provider as &dyn GroupNotificationFactsProvider,
        )
        .unwrap();
        assert_eq!(
            parsed.participant_username.as_deref(),
            Some("synthetic_user")
        );
        assert_eq!(provider.reads.load(Ordering::Relaxed), 1);
    }

    #[test]
    fn no_lid_fact_is_needed_without_destination() {
        let provider = provider(false);
        let node = base()
            .attr("participant_username", "synthetic_user")
            .build();
        let parsed = parse_with_provider(
            &node.as_node_ref(),
            None,
            None,
            &provider as &dyn GroupNotificationFactsProvider,
        )
        .unwrap();
        assert!(parsed.participant_username.is_none());
        let node = base().attr("to", "100001:2@lid").build();
        assert!(matches!(
            parse_with_provider(
                &node.as_node_ref(),
                None,
                None,
                &provider as &dyn GroupNotificationFactsProvider
            ),
            Err(GroupParseError::MissingContext("me_device_lid"))
        ));
    }

    #[tokio::test]
    async fn provider_is_fixed_after_registration() {
        let client = client().await;
        assert!(
            client
                .set_group_notification_facts_provider(Arc::new(provider(true)))
                .is_ok()
        );
        assert_eq!(
            client.set_group_notification_facts_provider(Arc::new(provider(false))),
            Err(GroupNotificationFactsAlreadySet)
        );
    }

    fn future_action_node() -> wacore_binary::Node {
        base()
            .children([NodeBuilder::new("future_action").build()])
            .build()
    }

    #[tokio::test]
    async fn unregistered_provider_keeps_the_legacy_adapter() {
        let client = client().await;
        let node = future_action_node();
        let parsed = client
            .parse_group_notification(&node.as_node_ref())
            .await
            .expect("legacy parse should not fail")
            .expect("legacy adapter should accept the envelope");
        // The legacy adapter tolerates unknown actions; the contextual one rejects them.
        assert!(matches!(
            parsed.actions[0],
            wacore::stanza::groups::GroupNotificationAction::Unknown { .. }
        ));
    }

    #[tokio::test]
    async fn registered_provider_switches_to_contextual_parsing() {
        let client = client().await;
        assert!(
            client
                .set_group_notification_facts_provider(Arc::new(provider(true)))
                .is_ok()
        );
        let node = future_action_node();
        assert!(matches!(
            client.parse_group_notification(&node.as_node_ref()).await,
            Err(GroupParseError::UnknownAction(action)) if action == "future_action"
        ));
        // Contextual parsing validates the destination, which the device snapshot
        // cannot satisfy without a LID.
        let addressed = base().attr("to", "100001:2@lid").build();
        assert!(matches!(
            client
                .parse_group_notification(&addressed.as_node_ref())
                .await,
            Err(GroupParseError::MissingContext("me_device_lid"))
        ));
    }
}
