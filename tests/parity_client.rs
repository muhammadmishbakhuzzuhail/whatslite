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
    let backend = Arc::new(
        SqliteStore::new("file:memdb_parity_client?mode=memory&cache=shared")
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
