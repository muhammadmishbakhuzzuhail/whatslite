//! Local HTTP sidecar bridging wa-rs to the Homun engine channel gateway.
//!
//! The Homun engine (or any local consumer) talks to this process over
//! 127.0.0.1 HTTP; the sidecar owns the WhatsApp Web session and pushes
//! inbound messages to a callback URL.
//!
//! Environment:
//!   WA_BRIDGE_PORT            HTTP port (default 8902, binds 127.0.0.1 only)
//!   WA_BRIDGE_DB              SQLite session path (default wa-bridge.db)
//!   WA_BRIDGE_PHONE           Optional phone number for pair-code linking
//!   WA_BRIDGE_PAIR_CODE       Optional custom 8-char pair code
//!   WA_BRIDGE_CALLBACK_URL    Engine inbound URL for received messages
//!   WA_BRIDGE_CALLBACK_TOKEN  Bearer token sent with the callback
//!
//! Pairing starts automatically at boot when no session exists: the QR
//! payload (and, with WA_BRIDGE_PHONE, the pair code) surfaces via
//! GET /status and POST /pair/start until the phone completes linking.

use std::sync::Arc;
use std::time::Duration;

use axum::extract::State;
use axum::routing::{get, post};
use axum::{Json, Router};
use chrono::{DateTime, Utc};
use serde_json::{Value, json};
use tokio::sync::RwLock;
use wa_rs::bot::MessageContext;
use wa_rs::client::Client;
use wacore::proto_helpers::MessageExt;
use waproto::whatsapp as wa;

#[derive(Clone)]
pub struct Callback {
    pub url: String,
    pub token: String,
}

pub struct AppState {
    pub client: RwLock<Option<Arc<Client>>>,
    pub qr: RwLock<Option<(String, DateTime<Utc>)>>,
    pub pair_code: RwLock<Option<(String, DateTime<Utc>)>>,
    pub jid: RwLock<Option<String>>,
    pub lid: RwLock<Option<String>>,
    pub self_chat_peer: RwLock<Option<String>>,
    pub pending_own: RwLock<std::collections::HashMap<String, (String, std::time::Instant)>>,
    pub recent_self: RwLock<Vec<(String, std::time::Instant)>>,
    pub connected: RwLock<bool>,
    pub logged_out: RwLock<bool>,
    pub last_error: RwLock<Option<String>>,
    pub last_pair_error: RwLock<Option<String>>,
    pub started_at: DateTime<Utc>,
    pub db_path: String,
    pub callback: Callback,
}

impl AppState {
    pub fn new(callback: Callback, db_path: String) -> Arc<Self> {
        Arc::new(Self {
            client: RwLock::new(None),
            qr: RwLock::new(None),
            pair_code: RwLock::new(None),
            jid: RwLock::new(None),
            lid: RwLock::new(None),
            self_chat_peer: RwLock::new(None),
            pending_own: RwLock::new(std::collections::HashMap::new()),
            recent_self: RwLock::new(Vec::new()),
            connected: RwLock::new(false),
            logged_out: RwLock::new(false),
            last_error: RwLock::new(None),
            last_pair_error: RwLock::new(None),
            started_at: Utc::now(),
            db_path,
            callback,
        })
    }

    pub async fn status_payload(self: &Arc<Self>) -> Value {
        let jid = self.jid.read().await.clone();
        let lid = self.lid.read().await.clone();
        let connected = *self.connected.read().await;
        let logged_out = *self.logged_out.read().await;
        let qr = self.qr.read().await.clone();
        let pair_code = self.pair_code.read().await.clone();
        let last_error = self.last_error.read().await.clone();
        let last_pair_error = self.last_pair_error.read().await.clone();
        let paired = connected && !logged_out;
        json!({
            "ok": true,
            "service": "wa-rs-bridge",
            "version": env!("CARGO_PKG_VERSION"),
            "paired": paired,
            "jid": jid,
            "lid": lid,
            "connected": connected,
            "logged_out": logged_out,
            "started_at": self.started_at.to_rfc3339(),
            "last_error": last_error,
            "last_pair_error": last_pair_error,
            "qr": qr.as_ref().map(|(payload, expires)| json!({
                "payload": payload,
                "expires_at": expires.to_rfc3339(),
            })),
            "pair_code": pair_code.as_ref().map(|(code, expires)| json!({
                "code": code,
                "expires_at": expires.to_rfc3339(),
            })),
        })
    }
}

/// Persist the paired identities next to the session db, and restore them at
/// boot: PairSuccess only fires on first linking, never on session resume.
fn identity_path(db_path: &str) -> std::path::PathBuf {
    std::path::Path::new(db_path).with_extension("identity.json")
}

pub fn persist_identity(db_path: &str, jid: &str, lid: &str, self_chat_peer: Option<&str>) {
    let doc = json!({"jid": jid, "lid": lid, "self_chat_peer": self_chat_peer});
    if let Err(e) = std::fs::write(identity_path(db_path), doc.to_string()) {
        log::warn!("failed to persist bridge identity: {e}");
    }
}

pub async fn restore_identity(state: &Arc<AppState>, db_path: &str) {
    let Ok(raw) = std::fs::read_to_string(identity_path(db_path)) else {
        return;
    };
    let Ok(doc) = serde_json::from_str::<Value>(&raw) else {
        return;
    };
    if let Some(jid) = doc.get("jid").and_then(Value::as_str) {
        *state.jid.write().await = Some(jid.to_string());
    }
    if let Some(lid) = doc.get("lid").and_then(Value::as_str) {
        *state.lid.write().await = Some(lid.to_string());
    }
    if let Some(peer) = doc.get("self_chat_peer").and_then(Value::as_str) {
        *state.self_chat_peer.write().await = Some(peer.to_string());
    }
    log::info!("restored paired identity from session store");
}

pub fn expires_in(timeout: Duration) -> DateTime<Utc> {
    Utc::now() + chrono::TimeDelta::from_std(timeout).unwrap_or(chrono::TimeDelta::seconds(60))
}

/// Deliver an inbound message to the engine callback. Blocking ureq call on
/// the tokio blocking pool; failures are logged, never fatal for the session.
async fn push_callback(callback: &Callback, body: Value) {
    if callback.url.is_empty() {
        return;
    }
    let url = callback.url.clone();
    let token = callback.token.clone();
    let payload = body.to_string();
    tokio::task::spawn_blocking(move || {
        // Same policy as the workspace ureq client: bounded connect and
        // global timeouts so a stuck engine can't pile up callback threads.
        let mut req = ureq::post(&url)
            .config()
            .timeout_connect(Some(Duration::from_secs(10)))
            .timeout_global(Some(Duration::from_secs(30)))
            .build();
        req = req.header("Content-Type", "application/json");
        if !token.is_empty() {
            req = req.header("Authorization", format!("Bearer {token}"));
        }
        if let Err(e) = req.send(payload.as_bytes()) {
            log::warn!("callback to {url} failed: {e}");
        }
    })
    .await
    .ok();
}

pub async fn health() -> Json<Value> {
    Json(json!({"ok": true, "service": "wa-rs-bridge", "version": env!("CARGO_PKG_VERSION")}))
}

pub async fn status(State(state): State<Arc<AppState>>) -> Json<Value> {
    Json(state.status_payload().await)
}

pub async fn pair_start(State(state): State<Arc<AppState>>) -> Json<Value> {
    // Pairing begins at boot; this endpoint reports the live QR/pair code.
    Json(state.status_payload().await)
}

pub async fn send(
    State(state): State<Arc<AppState>>,
    Json(body): Json<Value>,
) -> (axum::http::StatusCode, Json<Value>) {
    let to = body
        .get("to")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let text = body
        .get("text")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    if to.is_empty() || text.is_empty() {
        return (
            axum::http::StatusCode::BAD_REQUEST,
            Json(json!({"ok": false, "error": "missing_to_or_text"})),
        );
    }
    // Accept bare phone numbers as well as full JIDs.
    let mut jid_str = if to.contains('@') {
        to
    } else {
        format!("{to}@s.whatsapp.net")
    };
    // Self-chat routing: the self-chat only renders notes sent to the
    // account's phone-number JID. The companion JID and the account's LID
    // form are accepted by the server but land in invisible chats, so every
    // self-directed destination is rewritten to the bare PN form.
    {
        let bare = |s: &str| {
            s.split('@')
                .next()
                .unwrap_or("")
                .split(':')
                .next()
                .unwrap_or("")
                .to_string()
        };
        let own_jid = state.jid.read().await.clone();
        let own_lid = state.lid.read().await.clone();
        let peer = state.self_chat_peer.read().await.clone();
        let own_pn = own_jid.as_deref().map(bare).unwrap_or_default();
        let own_lid_user = own_lid.as_deref().map(bare).unwrap_or_default();
        let peer_user = peer.as_deref().map(bare).unwrap_or_default();
        let to_user = bare(&jid_str);
        let is_self_destination = !own_pn.is_empty()
            && (to_user == own_pn
                || (!own_lid_user.is_empty() && to_user == own_lid_user)
                || (!peer_user.is_empty() && to_user == peer_user));
        if is_self_destination {
            jid_str = format!("{own_pn}@s.whatsapp.net");
        }
    }
    let client = state.client.read().await.clone();
    let Some(client) = client else {
        return (
            axum::http::StatusCode::SERVICE_UNAVAILABLE,
            Json(json!({"ok": false, "error": "bridge_not_ready"})),
        );
    };
    let Ok(jid) = jid_str.parse() else {
        return (
            axum::http::StatusCode::BAD_REQUEST,
            Json(json!({"ok": false, "error": "invalid_jid"})),
        );
    };
    let mut extended_text = wa::message::ExtendedTextMessage::default();
    extended_text.text = Some(text);
    let mut message = wa::Message::default();
    message.extended_text_message = extended_text.into();
    match client.send_message(jid, message).await {
        Ok(id) => (
            axum::http::StatusCode::OK,
            Json(json!({"ok": true, "id": id})),
        ),
        Err(e) => (
            axum::http::StatusCode::BAD_GATEWAY,
            Json(json!({"ok": false, "error": e.to_string()})),
        ),
    }
}

impl AppState {
    /// Router-only state for HTTP surface tests: no live WhatsApp client.
    pub fn for_test() -> Arc<Self> {
        Self::new(
            Callback {
                url: String::new(),
                token: String::new(),
            },
            ":memory:".to_string(),
        )
    }
}

/// Build the sidecar HTTP router around a state handle.
pub fn build_router(state: Arc<AppState>) -> Router {
    Router::new()
        .route("/health", get(health))
        .route("/status", get(status))
        .route("/pair/start", post(pair_start))
        .route("/send", post(send))
        .with_state(state)
}

pub async fn handle_message(state: &Arc<AppState>, ctx: MessageContext) {
    let info = &ctx.info;
    let Some(text) = ctx.message.text_content() else {
        log::info!("dropped message without text content");
        return; // v1 bridges text messages only.
    };
    if text.trim().is_empty() {
        log::info!("dropped empty message");
        return;
    }
    let chat = info.source.chat.to_string();
    let sender = info.source.sender.to_string();
    let jid = state.jid.read().await.clone();
    let lid = state.lid.read().await.clone();
    let bare = |s: &str| {
        s.split('@')
            .next()
            .unwrap_or("")
            .split(':')
            .next()
            .unwrap_or("")
            .to_string()
    };
    let own_users: Vec<String> = [jid.clone(), lid.clone()]
        .into_iter()
        .flatten()
        .map(|full| bare(&full))
        .filter(|u| !u.is_empty())
        .collect();
    let chat_bare = bare(&chat);

    // Self-notes reach linked devices in two shapes: delivered directly to
    // the account's own chat (is_from_me), or echoed by the self-chat
    // companion as ordinary inbound traffic. Both must produce exactly one
    // engine turn.
    let mut self_note = false;
    if info.source.is_from_me {
        if own_users.contains(&chat_bare) {
            self_note = true;
        } else {
            // Own message to someone else: remember it only as the seed for
            // companion-echo discovery, never as engine input.
            state
                .pending_own
                .write()
                .await
                .insert(chat.clone(), (text.to_string(), std::time::Instant::now()));
            log::info!("dropped own message outside the self-chat");
            return;
        }
    } else {
        let peer = state.self_chat_peer.read().await.clone();
        let from_peer = peer
            .as_deref()
            .map(|p| bare(p) == bare(&sender))
            .unwrap_or(false);
        if from_peer {
            self_note = true;
        } else {
            let mut pending = state.pending_own.write().await;
            let matched = pending
                .get(&chat)
                .map(|(t, at)| t == &text && at.elapsed() < std::time::Duration::from_secs(30))
                .unwrap_or(false);
            if matched {
                pending.remove(&chat);
                drop(pending);
                *state.self_chat_peer.write().await = Some(sender.clone());
                persist_identity(
                    &state.db_path,
                    jid.as_deref().unwrap_or(""),
                    lid.as_deref().unwrap_or(""),
                    Some(&sender),
                );
                log::info!("learned self-chat companion {sender}");
                self_note = true;
            }
        }
    }

    if self_note {
        // The direct copy and the companion echo can both arrive: forward the
        // first, drop its duplicate.
        let mut recent = state.recent_self.write().await;
        if recent
            .iter()
            .any(|(t, at)| t == &text && at.elapsed() < std::time::Duration::from_secs(25))
        {
            log::info!("dropped duplicate self-note copy");
            return;
        }
        recent.push((text.to_string(), std::time::Instant::now()));
        recent.retain(|(_, at)| at.elapsed() < std::time::Duration::from_secs(60));
    }

    let mut body = json!({
        "source": "wa-rs-bridge",
        "message": {
            "id": info.id.clone(),
            "chat": chat,
            "sender": sender,
            "push_name": info.push_name.clone(),
            "is_group": info.source.is_group,
            "text": text,
            "timestamp": info.timestamp.to_rfc3339(),
        }
    });
    if self_note {
        // Attribute self-notes to the account owner so the engine's
        // authorization gate and conversation identity see the human.
        if let Some(obj) = body.get_mut("message").and_then(Value::as_object_mut) {
            obj.insert("self_echo".into(), json!(true));
            obj.insert("owner".into(), json!({"jid": jid, "lid": lid}));
        }
    }
    log::info!("forwarding message to engine callback (self_note={self_note})");
    push_callback(&state.callback, body).await;
}
