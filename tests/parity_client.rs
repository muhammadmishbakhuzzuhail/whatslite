//! Phase 1 client contract: builder, node filter/waiter and lifecycle errors.

use std::sync::Arc;

use wa_rs::Client;
use wa_rs::client::lifecycle::{ConnectError, ConnectStage};
use wa_rs::client::node_io::{NodeFilter, NodeWaiterRegistry};
use wa_rs::store::SqliteStore;
use wa_rs::store::persistence_manager::PersistenceManager;
use wa_rs::test_utils::MockHttpClient;
use wa_rs::transport::mock::MockTransportFactory;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::marshal::{marshal, unmarshal_ref};

async fn persistence() -> Arc<PersistenceManager> {
    // Each client gets its own in-memory database; a shared name makes
    // concurrently running tests race on the schema migration.
    static NEXT_DB: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
    let db = NEXT_DB.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
    let backend = Arc::new(
        SqliteStore::new(&format!(
            "file:memdb_parity_client_{db}?mode=memory&cache=shared"
        ))
        .await
        .expect("sqlite backend"),
    );
    Arc::new(
        PersistenceManager::new(backend)
            .await
            .expect("persistence manager"),
    )
}

#[tokio::test]
async fn builder_assembles_a_client_and_preserves_old_constructor() {
    let persistence = persistence().await;
    let build = Client::builder()
        .with_persistence_manager(persistence.clone())
        .with_transport_factory(MockTransportFactory::new())
        .with_http_client(MockHttpClient)
        .with_version_override((2, 3000, 1))
        .build()
        .await
        .expect("builder");
    let (client, _sync_rx) = build.into_parts();
    assert!(!client.is_logged_in());

    // Existing constructor remains available.
    let (_client, _sync_rx) = Client::new(
        persistence,
        Arc::new(MockTransportFactory::new()),
        Arc::new(MockHttpClient),
        None,
    )
    .await;
}

#[test]
fn node_filter_matches_tag_attributes_and_absence() {
    let node = NodeBuilder::new("notification")
        .attr("type", "w:gp2")
        .attr("from", "123@g.us")
        .build();
    let bytes = marshal(&node).expect("marshal");
    let nr = unmarshal_ref(&bytes[1..]).expect("unmarshal");

    assert!(NodeFilter::tag("notification").matches(&nr));
    assert!(
        NodeFilter::tag("notification")
            .attr("type", "w:gp2")
            .attr("from", "123@g.us")
            .matches(&nr)
    );
    assert!(
        !NodeFilter::tag("notification")
            .attr("type", "other")
            .matches(&nr)
    );
    assert!(
        NodeFilter::tag("notification")
            .without_attr("missing")
            .matches(&nr)
    );
    assert!(
        !NodeFilter::tag("notification")
            .without_attr("type")
            .matches(&nr)
    );
}

#[tokio::test]
async fn node_waiter_registry_resolves_matching_nodes_and_cancels_on_drop() {
    let registry = NodeWaiterRegistry::<wacore_binary::Node>::new();
    let node = Arc::new(
        NodeBuilder::new("notification")
            .attr("type", "w:gp2")
            .build(),
    );
    let bytes = marshal(&node).expect("marshal");
    let nr = unmarshal_ref(&bytes[1..]).expect("unmarshal");

    let mut waiter = registry.wait(NodeFilter::tag("notification").attr("type", "w:gp2"));
    assert_eq!(registry.active(), 1);
    registry.resolve(&node, &nr);
    let resolved = waiter.try_recv().expect("resolved").expect("node");
    assert!(Arc::ptr_eq(&resolved, &node));
    assert_eq!(registry.active(), 0);

    let dropped = registry.wait(NodeFilter::tag("notification"));
    assert_eq!(registry.active(), 1);
    drop(dropped);
    assert_eq!(registry.active(), 0);
}

#[test]
fn lifecycle_stages_and_transient_errors_are_stable() {
    let stages = [
        ConnectStage::Idle,
        ConnectStage::Connecting,
        ConnectStage::Handshaking,
        ConnectStage::Authenticating,
        ConnectStage::Ready,
        ConnectStage::Disconnecting,
        ConnectStage::Closed,
    ];
    assert_eq!(stages.len(), 7);
    assert!(ConnectError::Timeout.is_transient());
    assert!(ConnectError::Transport("x".into()).is_transient());
    assert!(!ConnectError::Shutdown.is_transient());
    assert!(!ConnectError::Paused.is_transient());
}

/// Builds a client backed by in-memory persistence for IQ-operation checks
/// that do not touch the network.
async fn iq_client() -> (Arc<Client>, Arc<PersistenceManager>) {
    let persistence = persistence().await;
    let build = Client::builder()
        .with_persistence_manager(persistence.clone())
        .with_transport_factory(MockTransportFactory::new())
        .with_http_client(MockHttpClient)
        .with_version_override((2, 3000, 1))
        .build()
        .await
        .expect("builder");
    let (client, _sync_rx) = build.into_parts();
    (client, persistence)
}

#[tokio::test]
async fn set_device_props_applies_override_through_persistence() {
    let (client, persistence) = iq_client().await;

    client
        .set_device_props(wacore::store::DevicePropsOverride::new().with_os("parity-os"))
        .await;

    let snapshot = persistence.get_device_snapshot().await;
    assert_eq!(snapshot.device_props.os.as_deref(), Some("parity-os"));
}

#[tokio::test]
async fn set_chat_disappearing_timer_rejects_non_one_to_one_chats() {
    let (client, _persistence) = iq_client().await;
    let group: wacore_binary::jid::Jid = "120363000000000000@g.us".parse().expect("group jid");

    let error = client
        .set_chat_disappearing_timer(group, 86_400)
        .await
        .expect_err("group chats use Groups::set_ephemeral");
    assert!(error.to_string().contains("1:1"), "{error}");
}

/// The IQ-operation surface is compile-checked: each method is referenced so a
/// rename or removal breaks the build. Network-backed paths cannot be asserted
/// without a server.
#[test]
fn client_iq_operations_are_exposed() {
    use wa_rs::wacore::iq::privacy::{DisallowedListUpdate, PrivacyCategory, PrivacyValue};

    let _ = Client::set_passive;
    let _ = Client::fetch_props;
    let _ = Client::fetch_privacy_settings;
    let _ = Client::set_privacy_setting;
    let _ = Client::set_privacy_disallowed_list;
    let _ = Client::set_default_disappearing_mode;
    let _ = Client::set_chat_disappearing_timer;
    let _ = Client::send_digest_key_bundle;
    let _ = Client::set_device_props;

    let update = DisallowedListUpdate {
        dhash: "dhash".to_string(),
        users: Vec::new(),
    };
    assert_eq!(PrivacyCategory::Last.as_str(), "last");
    assert_eq!(PrivacyValue::ContactBlacklist.as_str(), "contact_blacklist");
    assert!(PrivacyCategory::Last.supports_disallowed_list());
    assert_eq!(update.users.len(), 0);
}

// ---------------------------------------------------------------------------
// Config accessors: pure local state, no network involved.
// ---------------------------------------------------------------------------

/// Custom payload handler used to prove `has_enc_handler` reports the types the
/// client actually holds.
struct ParityEncHandler;

#[async_trait::async_trait]
impl wa_rs::types::enc_handler::EncHandler for ParityEncHandler {
    async fn handle(
        &self,
        _client: Arc<Client>,
        _enc_node: &wacore_binary::node::Node,
        _info: &wa_rs::types::message::MessageInfo,
    ) -> anyhow::Result<()> {
        Ok(())
    }
}

async fn config_client(http: Arc<dyn wa_rs::http::HttpClient>) -> Arc<Client> {
    let persistence = persistence().await;
    let build = Client::builder()
        .with_persistence_manager(persistence)
        .with_transport_factory(MockTransportFactory::new())
        .with_http_client_arc(http)
        .with_version_override((2, 3000, 1))
        .build()
        .await
        .expect("builder");
    build.into_parts().0
}

#[tokio::test]
async fn http_client_accessor_returns_the_injected_service() {
    let injected: Arc<dyn wa_rs::http::HttpClient> = Arc::new(MockHttpClient);
    let client = config_client(injected.clone()).await;

    assert!(Arc::ptr_eq(client.http_client(), &injected));
}

#[tokio::test]
async fn auto_reconnect_toggle_round_trips() {
    let client = config_client(Arc::new(MockHttpClient)).await;

    assert!(client.auto_reconnect_enabled(), "enabled by default");
    client.set_auto_reconnect(false);
    assert!(!client.auto_reconnect_enabled());
    client.set_auto_reconnect(true);
    assert!(client.auto_reconnect_enabled());
}

#[tokio::test]
async fn enc_handler_detection_tracks_registered_payload_types() {
    let client = config_client(Arc::new(MockHttpClient)).await;

    assert!(!client.has_enc_handler("frskmsg"));
    client
        .custom_enc_handlers
        .insert("frskmsg".to_string(), Arc::new(ParityEncHandler));
    assert!(client.has_enc_handler("frskmsg"));
    assert!(!client.has_enc_handler("other"));
}

#[tokio::test]
async fn skip_history_sync_toggle_round_trips() {
    let client = config_client(Arc::new(MockHttpClient)).await;

    assert!(!client.skip_history_sync_enabled(), "processed by default");
    client.set_skip_history_sync(true);
    assert!(client.skip_history_sync_enabled());
    client.set_skip_history_sync(false);
    assert!(!client.skip_history_sync_enabled());
}
