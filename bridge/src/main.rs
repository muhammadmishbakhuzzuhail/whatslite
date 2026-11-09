//! Thin binary for the wa-rs HTTP sidecar.

use std::sync::Arc;

use wa_rs::bot::{Bot, MessageContext};
use wa_rs::pair_code::PairCodeOptions;
use wa_rs::store::SqliteStore;
use wa_rs_bridge::Callback;
use wa_rs_tokio_transport::TokioWebSocketTransportFactory;
use wa_rs_ureq_http::UreqHttpClient;
use wacore::types::events::Event;

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|v| !v.trim().is_empty())
        .unwrap_or_else(|| default.to_string())
}

#[tokio::main]
async fn main() {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info")).init();

    let port: u16 = env_or("WA_BRIDGE_PORT", "8902").parse().unwrap_or(8902);
    let db_path = env_or("WA_BRIDGE_DB", "wa-bridge.db");
    let phone = std::env::var("WA_BRIDGE_PHONE")
        .ok()
        .filter(|v| !v.trim().is_empty());
    let custom_code = std::env::var("WA_BRIDGE_PAIR_CODE")
        .ok()
        .filter(|v| !v.trim().is_empty());
    let callback = Callback {
        url: env_or("WA_BRIDGE_CALLBACK_URL", ""),
        token: env_or("WA_BRIDGE_CALLBACK_TOKEN", ""),
    };

    let state = wa_rs_bridge::AppState::new(callback, db_path.clone());
    wa_rs_bridge::restore_identity(&state, &db_path).await;

    let backend = match SqliteStore::new(&db_path).await {
        Ok(store) => Arc::new(store),
        Err(e) => {
            log::error!("failed to open session db {db_path}: {e}");
            std::process::exit(1);
        }
    };

    let mut builder = Bot::builder()
        .with_backend(backend)
        .with_transport_factory(TokioWebSocketTransportFactory::new())
        .with_http_client(UreqHttpClient::new());
    if let Some(phone) = phone {
        log::info!("pair-code linking enabled for phone {phone}");
        builder = builder.with_pair_code(PairCodeOptions {
            phone_number: phone,
            custom_code,
            ..Default::default()
        });
    }

    let event_state = state.clone();
    let events_db_path = db_path.clone();
    let mut bot = builder
        .on_event(move |event, client| {
            let state = event_state.clone();
            let db_path = events_db_path.clone();
            async move {
                // Publish the client for /send as soon as it exists.
                {
                    let mut guard = state.client.write().await;
                    if guard.is_none() {
                        *guard = Some(client.clone());
                    }
                }
                match event {
                    Event::PairingQrCode(qr) => {
                        log::info!("pairing QR available (valid {}s)", qr.timeout.as_secs());
                        *state.qr.write().await = Some((qr.code, wa_rs_bridge::expires_in(qr.timeout)));
                    }
                    Event::PairingCode(pair_code) => {
                        log::info!("pair code available: {}", pair_code.code);
                        *state.pair_code.write().await =
                            Some((pair_code.code, wa_rs_bridge::expires_in(pair_code.timeout)));
                    }
                    Event::PairSuccess(success) => {
                        log::info!("paired as {} (lid {})", success.id, success.lid);
                        // A WhatsApp account has two wire identities: the phone
                        // Jid and an opaque LID. Keep both: incoming traffic is
                        // addressed with either, depending on the sender device.
                        // They survive restarts via a sidecar file: PairSuccess
                        // only fires on first linking, not on session resume.
                        *state.jid.write().await = Some(success.id.to_string());
                        *state.lid.write().await = Some(success.lid.to_string());
                        *state.logged_out.write().await = false;
                        *state.last_pair_error.write().await = None;
                        wa_rs_bridge::persist_identity(&db_path, &success.id.to_string(), &success.lid.to_string(), None);
                    }
                    Event::PairError(err) => {
                        // The phone showed an error: keep the server-provided
                        // reason for /status so the app can display it.
                        log::error!("pairing rejected: {} (platform {})", err.error, err.platform);
                        *state.last_pair_error.write().await =
                            Some(format!("{} (platform {})", err.error, err.platform));
                    }
                    Event::QrScannedWithoutMultidevice(_) => {
                        log::error!("QR scanned but the account has no multidevice support");
                        *state.last_pair_error.write().await = Some(
                            "QR scansionato ma l'account non supporta i dispositivi collegati (multidevice)".to_string(),
                        );
                    }
                    Event::Connected(_) => {
                        log::info!("WhatsApp session connected");
                        *state.connected.write().await = true;
                    }
                    Event::Disconnected(_) => {
                        log::warn!("WhatsApp session disconnected");
                        *state.connected.write().await = false;
                    }
                    Event::LoggedOut(logout) => {
                        log::error!("WhatsApp session logged out: {:?}", logout.reason);
                        *state.logged_out.write().await = true;
                        *state.connected.write().await = false;
                        *state.jid.write().await = None;
                    }
                    Event::Messages(batch) => {
                        for inbound in batch.iter() {
                            let ctx = MessageContext {
                                message: Box::new((*inbound.message).clone()),
                                info: (*inbound.info).clone(),
                                client: client.clone(),
                            };
                            wa_rs_bridge::handle_message(&state, ctx).await;
                        }
                    }
                    _ => {}
                }
            }
        })
        .build()
        .await
        .expect("failed to build bot");

    // The client exists right after build; publish it for /send.
    {
        let client = bot.client();
        *state.client.write().await = Some(client);
    }

    let run_state = state.clone();
    let handle = match bot.run().await {
        Ok(handle) => handle,
        Err(e) => {
            log::error!("bot failed to start: {e}");
            *run_state.last_error.write().await = Some(e.to_string());
            std::process::exit(1);
        }
    };

    let app = wa_rs_bridge::build_router(state);

    let addr = std::net::SocketAddr::from(([127, 0, 0, 1], port));
    let listener = match tokio::net::TcpListener::bind(addr).await {
        Ok(l) => l,
        Err(e) => {
            log::error!("failed to bind {addr}: {e}");
            std::process::exit(1);
        }
    };
    log::info!("wa-rs-bridge listening on http://{addr}");

    let server = async move { axum::serve(listener, app).await };
    tokio::pin!(server);
    tokio::select! {
        _ = &mut server => {}
        result = handle => {
            log::error!("bot task terminated: {:?}", result);
        }
    }
}
