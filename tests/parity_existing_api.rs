//! Phase 0 compatibility contract: the existing `wa-rs` public API keeps its
//! signatures while upstream-style APIs are added alongside it.

use std::sync::Arc;

use wa_rs::http::HttpClient;
use wa_rs::store::persistence_manager::PersistenceManager;
use wa_rs::transport::TransportFactory;

#[allow(dead_code)]
async fn existing_client_surface(
    persistence: Arc<PersistenceManager>,
    transport: Arc<dyn TransportFactory>,
    http: Arc<dyn HttpClient>,
) {
    let (client, _sync_rx) = wa_rs::Client::new(persistence, transport, http, None).await;

    client.run().await;
    client.connect().await.expect("connect signature");
    client.disconnect().await;

    let node = wa_rs::wacore_binary::builder::NodeBuilder::new("iq").build();
    client.send_node(node).await.expect("send_node signature");
}

#[allow(dead_code)]
async fn existing_bot_and_storage_surface() {
    let _builder = wa_rs::bot::Bot::builder();
    let _options = wa_rs::pair_code::PairCodeOptions::default();
    let _store = wa_rs::store::SqliteStore::new(":memory:").await;
}

#[test]
fn existing_signatures_type_check() {
    // The functions above are never called; compiling them is the assertion.
}
