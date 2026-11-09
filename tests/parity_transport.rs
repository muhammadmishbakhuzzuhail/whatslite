//! Phase 2 transport contract: factory/http trait seams and media connection types.

use std::sync::Arc;

use wa_rs::http::HttpClient;
use wa_rs::test_utils::MockHttpClient;
use wa_rs::transport::TransportFactory;
use wa_rs::transport::mock::MockTransportFactory;

#[test]
fn transport_and_http_factories_are_object_safe() {
    let _transport: Arc<dyn TransportFactory> = Arc::new(MockTransportFactory::new());
    let _http: Arc<dyn HttpClient> = Arc::new(MockHttpClient);
}

#[test]
fn transport_events_and_media_types_are_reachable() {
    let _: fn() = || {
        let _ = std::any::type_name::<wa_rs::transport::TransportEvent>();
        let _ = std::any::type_name::<wa_rs::mediaconn::MediaConn>();
        let _ = std::any::type_name::<dyn wa_rs::wacore::net::Transport>();
    };
}
