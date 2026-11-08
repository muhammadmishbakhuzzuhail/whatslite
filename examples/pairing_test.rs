//! Pairing smoke test: fresh QR pairing + inbound message verification.
//!
//! This is the acceptance test for the post-Apr-24 server alignment
//! (ADV HMAC enforcement, Noise IK/XXfallback). The reported symptom was
//! "pair succeeds, no inbound messages flow" — so this script's success
//! criteria are, in order:
//!
//!   1. "Handshake complete (XX)" on the first connect.
//!   2. A scannable QR below; scanning it produces Event::Connected.
//!   3. "INBOUND MESSAGE" lines when someone messages the paired number.
//!
//! Second run with the same database reconnects with the stored session and
//! should log "Handshake complete (IK)" — that exercises the cert-chain cache.
//!
//! Usage:
//!   cargo run --example pairing_test [--fresh] [db-path]
//!   RUST_LOG=debug cargo run --example pairing_test   # verbose protocol logs
//!
//! Flags:
//!   --fresh   delete the database before starting (new pairing)
//!   [db-path] SQLite file to use (default: pairing_test.db)

use chrono::Local;
use log::info;
use std::sync::Arc;
use std::sync::atomic::{AtomicU32, Ordering};
use wa_rs::bot::Bot;
use wa_rs::store::SqliteStore;
use wa_rs_tokio_transport::TokioWebSocketTransportFactory;
use wa_rs_ureq_http::UreqHttpClient;
use wacore::proto_helpers::MessageExt;
use wacore::types::events::Event;

static QR_COUNT: AtomicU32 = AtomicU32::new(0);
static INBOUND_COUNT: AtomicU32 = AtomicU32::new(0);

/// Terminal QR with near-square modules (half-block renderer), surrounded by
/// a quiet zone. Scan with WhatsApp > Settings > Linked Devices > Link a Device.
fn render_qr(code: &str) -> Option<String> {
    let qr = qrcode::QrCode::new(code.as_bytes()).ok()?;
    Some(
        qr.render::<qrcode::render::unicode::Dense1x2>()
            .quiet_zone(true)
            .build(),
    )
}

fn print_qr(code: &str, timeout_secs: u64) {
    let n = QR_COUNT.fetch_add(1, Ordering::Relaxed) + 1;
    println!();
    println!("══════ QR #{n} — scan with WhatsApp (valid {timeout_secs}s) ══════");
    match render_qr(code) {
        Some(qr) => println!("{qr}"),
        None => {
            println!("! QR rendering failed — raw pairing string:");
            println!("{code}");
        }
    }
    println!("If scanning fails: shrink the font until the code looks");
    println!("square and well-spaced, or use a light terminal background.");
    println!("Raw string: {code}");
    println!();
}

fn print_inbound(msg: &waproto::whatsapp::Message, info: &wa_rs::types::message::MessageInfo) {
    let n = INBOUND_COUNT.fetch_add(1, Ordering::Relaxed) + 1;
    let text = msg.text_content().unwrap_or("[non-text message]");
    println!();
    println!(
        "✅ INBOUND MESSAGE #{n} ─ id={} chat={} sender={}",
        info.id, info.source.chat, info.source.sender
    );
    println!("   {text}");
    println!();
}

#[tokio::main]
async fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.iter().any(|a| a == "--qr-demo") {
        // Rendering check without connecting: payload shaped like a real one.
        let demo = "2@DemoPayloadForTerminalQrRenderingCheck,Vl0xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx==,Vl0yyyy==,DESKTOP";
        print_qr(demo, 20);
        return;
    }
    let fresh = args.iter().any(|a| a == "--fresh");
    let db_path = args
        .iter()
        .skip(1)
        .find(|a| !a.starts_with("--"))
        .cloned()
        .unwrap_or_else(|| "pairing_test.db".to_string());

    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info"))
        .format(|buf, record| {
            use std::io::Write;
            writeln!(
                buf,
                "{} [{:<5}] {}",
                Local::now().format("%H:%M:%S"),
                record.level(),
                record.args()
            )
        })
        .init();

    if fresh {
        for suffix in ["", "-wal", "-shm"] {
            let f = format!("{db_path}{suffix}");
            match std::fs::remove_file(&f) {
                Ok(()) => info!("removed {f}"),
                Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
                Err(e) => {
                    log::error!("failed to remove {f}: {e}");
                    return;
                }
            }
        }
    }

    info!("pairing test starting — db: {db_path} (fresh={fresh})");
    info!("success criteria: Handshake complete (XX) → scan QR → CONNECTED → INBOUND MESSAGE");
    if !fresh && std::path::Path::new(&db_path).exists() {
        info!("note: existing db — if it holds a session, expect \"Handshake complete (IK)\"");
    }

    let backend = match SqliteStore::new(&db_path).await {
        Ok(store) => Arc::new(store),
        Err(e) => {
            log::error!("failed to open SQLite backend: {e}");
            return;
        }
    };
    let transport_factory = TokioWebSocketTransportFactory::new();
    let http_client = UreqHttpClient::new();
    let db_path_for_banner = db_path.clone();

    let mut bot = Bot::builder()
        .with_backend(backend)
        .with_transport_factory(transport_factory)
        .with_http_client(http_client)
        .on_event(move |event, _client| {
            let db_path_for_banner = db_path_for_banner.clone();
            async move {
                match event {
                    Event::PairingQrCode { code, timeout } => {
                        print_qr(&code, timeout.as_secs());
                    }
                    Event::Connected(_) => {
                        println!();
                        println!("🎉 CONNECTED — pairing complete, session stored in {db_path_for_banner}");
                        println!("   Now send a message to this number from another phone:");
                        println!("   watch for ✅ INBOUND MESSAGE lines below.");
                        println!("   Re-run without --fresh to test the IK reconnect path.");
                        println!();
                    }
                    Event::Message(msg, info) => {
                        print_inbound(&msg, &info);
                    }
                    Event::PairError(e) => {
                        println!("❌ PAIR ERROR: {} — {}", e.error, e.platform);
                    }
                    Event::LoggedOut(_) => {
                        println!("❌ LOGGED OUT by the server.");
                    }
                    Event::ClientOutdated(_) => {
                        println!("❌ CLIENT OUTDATED — the server rejected this version.");
                    }
                    _ => {}
                }
            }
        })
        .build()
        .await
        .expect("failed to build bot");

    let bot_handle = bot.run().await.expect("bot failed to start");

    println!("running — Ctrl-C to stop");
    tokio::select! {
        _ = bot_handle => {
            info!("bot task finished");
        }
        _ = tokio::signal::ctrl_c() => {
            info!("Ctrl-C received, disconnecting…");
            bot.client().disconnect().await;
        }
    }

    let inbound = INBOUND_COUNT.load(Ordering::Relaxed);
    if inbound > 0 {
        println!("RESULT: PASS — {inbound} inbound message(s) received");
    } else {
        println!(
            "RESULT: pairing phase only — no inbound messages received (send one to complete the test)"
        );
    }
}
