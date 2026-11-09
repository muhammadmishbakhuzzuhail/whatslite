//! HTTP-surface smoke tests: the router must answer without a live session.

use axum::body::{Body, to_bytes};
use axum::http::{Request, StatusCode};
use tower::ServiceExt;

use wa_rs_bridge::{AppState, build_router};

async fn json_response(response: axum::response::Response) -> serde_json::Value {
    let body = to_bytes(response.into_body(), usize::MAX)
        .await
        .expect("response body");
    serde_json::from_slice(&body).expect("json response")
}

#[tokio::test]
async fn health_reports_service_identity() {
    let app = build_router(AppState::for_test());
    let response = app
        .oneshot(
            Request::builder()
                .uri("/health")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let json = json_response(response).await;
    assert_eq!(json["ok"], true);
    assert_eq!(json["service"], "wa-rs-bridge");
}

#[tokio::test]
async fn status_and_pair_start_are_available_without_client() {
    let app = build_router(AppState::for_test());
    for (method, uri) in [("GET", "/status"), ("POST", "/pair/start")] {
        let response = app
            .clone()
            .oneshot(
                Request::builder()
                    .method(method)
                    .uri(uri)
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(response.status(), StatusCode::OK, "{uri}");
        let json = json_response(response).await;
        assert_eq!(json["ok"], true, "{uri}");
        assert_eq!(json["connected"], false, "{uri}");
    }
}

#[tokio::test]
async fn send_validates_body_and_reports_unready_bridge() {
    let app = build_router(AppState::for_test());

    let response = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/send")
                .header("content-type", "application/json")
                .body(Body::from(r#"{"to":"","text":""}"#))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::BAD_REQUEST);
    let json = json_response(response).await;
    assert_eq!(json["error"], "missing_to_or_text");

    let response = app
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/send")
                .header("content-type", "application/json")
                .body(Body::from(r#"{"to":"15551234567","text":"hello"}"#))
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(response.status(), StatusCode::SERVICE_UNAVAILABLE);
    let json = json_response(response).await;
    assert_eq!(json["error"], "bridge_not_ready");
}
