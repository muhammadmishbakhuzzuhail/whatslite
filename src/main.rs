use chrono::{Local, Utc};
use log::{error, info};
use std::io::Cursor;
use std::sync::Arc;
use wa_rs::bot::{Bot, MessageContext};
use wa_rs::pair_code::PairCodeOptions;
use wa_rs::store::SqliteStore;
use wa_rs::upload::UploadResponse;
use wa_rs_tokio_transport::TokioWebSocketTransportFactory;
use wa_rs_ureq_http::UreqHttpClient;
use wacore::download::{Downloadable, MediaType};
use wacore::proto_helpers::MessageExt;
use wacore::types::events::Event;
use waproto::whatsapp as wa;

const PING_TRIGGER: &str = "🦀ping";
const MEDIA_PING_TRIGGER: &str = "ping";
const PONG_TEXT: &str = "🏓 Pong!";
const MEDIA_PONG_TEXT: &str = "pong";
const REACTION_EMOJI: &str = "🏓";

// This is a demo of a simple ping-pong bot with every type of media.
//
// Usage:
//   cargo run                                      # QR code pairing only
//   cargo run -- --phone 15551234567               # Pair code + QR code (concurrent)
//   cargo run -- -p 15551234567                    # Short form
//   cargo run -- -p 15551234567 --code MYCODE12    # Custom 8-char pair code
//   cargo run -- -p 15551234567 -c MYCODE12        # Short form

fn main() {
    // Parse CLI arguments for phone number and optional custom code
    let args: Vec<String> = std::env::args().collect();
    let phone_number = parse_arg(&args, "--phone", "-p");
    let custom_code = parse_arg(&args, "--code", "-c");

    if let Some(ref phone) = phone_number {
        eprintln!("Phone number provided: {}", phone);
        if let Some(ref code) = custom_code {
            eprintln!("Custom pair code: {}", code);
        }
        eprintln!("Will use pair code authentication (concurrent with QR)");
    }
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info"))
        .format(|buf, record| {
            use std::io::Write;
            writeln!(
                buf,
                "{} [{:<5}] [{}] - {}",
                Local::now().format("%H:%M:%S"),
                record.level(),
                record.target(),
                record.args()
            )
        })
        .init();

    let rt = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .expect("Failed to build tokio runtime");

    rt.block_on(async {
        let backend = match SqliteStore::new("whatsapp.db").await {
            Ok(store) => Arc::new(store),
            Err(e) => {
                error!("Failed to create SQLite backend: {}", e);
                return;
            }
        };
        info!("SQLite backend initialized successfully.");

        let transport_factory = TokioWebSocketTransportFactory::new();
        let http_client = UreqHttpClient::new();

        let mut builder = Bot::builder()
            .with_backend(backend)
            .with_transport_factory(transport_factory)
            .with_http_client(http_client);
        // Optional: Override the WhatsApp version (normally auto-fetched)
        // builder = builder.with_version((2, 3000, 1027868167));

        // Add pair code authentication if phone number provided
        if let Some(phone) = phone_number {
            builder = builder.with_pair_code(PairCodeOptions {
                phone_number: phone,
                custom_code,
                ..Default::default()
            });
        }

        let mut bot = builder
            .on_event(move |event, client| {
                async move {
                    match event {
                        Event::PairingQrCode { code, timeout } => {
                            info!("----------------------------------------");
                            info!(
                                "QR code received (valid for {} seconds):",
                                timeout.as_secs()
                            );
                            info!("\n{}\n", code);
                            info!("----------------------------------------");
                        }
                        Event::PairingCode { code, timeout } => {
                            info!("========================================");
                            info!("PAIR CODE (valid for {} seconds):", timeout.as_secs());
                            info!("Enter this code on your phone:");
                            info!("WhatsApp > Linked Devices > Link a Device");
                            info!("> Link with phone number instead");
                            info!("");
                            info!("    >>> {} <<<", code);
                            info!("");
                            info!("========================================");
                        }

                        Event::Message(msg, info) => {
                            let ctx = MessageContext {
                                message: msg,
                                info,
                                client,
                            };

                            if let Some(media_ping_request) = get_pingable_media(&ctx.message) {
                                handle_media_ping(&ctx, media_ping_request).await;
                            }

                            if let Some(text) = ctx.message.text_content()
                                && text == PING_TRIGGER
                            {
                                info!("Received text ping, sending pong...");

                                let mut message_key = wa::MessageKey::default();
                                message_key.remote_jid = Some(ctx.info.source.chat.to_string());
                                message_key.id = Some(ctx.info.id.clone());
                                message_key.from_me = Some(ctx.info.source.is_from_me);
                                message_key.participant = if ctx.info.source.is_group {
                                    Some(ctx.info.source.sender.to_string())
                                } else {
                                    None
                                };

                                let mut reaction_message = wa::message::ReactionMessage::default();
                                reaction_message.key = message_key.into();
                                reaction_message.text = Some(REACTION_EMOJI.to_string());
                                reaction_message.sender_timestamp_ms =
                                    Some(Utc::now().timestamp_millis());

                                let mut final_message_to_send = wa::Message::default();
                                final_message_to_send.reaction_message = reaction_message.into();

                                if let Err(e) = ctx.send_message(final_message_to_send).await {
                                    error!("Failed to send reaction: {}", e);
                                }

                                let start = std::time::Instant::now();

                                let context_info = ctx.build_quote_context();

                                let mut reply_text = wa::message::ExtendedTextMessage::default();
                                reply_text.text = Some(PONG_TEXT.to_string());
                                reply_text.context_info = context_info.clone().into();
                                let mut reply_message = wa::Message::default();
                                reply_message.extended_text_message = reply_text.into();

                                let sent_msg_id = match ctx.send_message(reply_message).await {
                                    Ok(id) => id,
                                    Err(e) => {
                                        error!("Failed to send initial pong message: {}", e);
                                        return;
                                    }
                                };

                                let duration = start.elapsed();
                                let duration_str = format!("{:.2?}", duration);

                                info!(
                                    "Send took {}. Editing message {}...",
                                    duration_str, &sent_msg_id
                                );

                                let mut updated_text = wa::message::ExtendedTextMessage::default();
                                updated_text.text =
                                    Some(format!("{}\n`{}`", PONG_TEXT, duration_str));
                                updated_text.context_info = context_info.into();
                                let mut updated_content = wa::Message::default();
                                updated_content.extended_text_message = updated_text.into();

                                if let Err(e) =
                                    ctx.edit_message(sent_msg_id.clone(), updated_content).await
                                {
                                    error!("Failed to edit message {}: {}", sent_msg_id, e);
                                } else {
                                    info!("Successfully sent edit for message {}.", sent_msg_id);
                                }
                            }
                        }
                        Event::Connected(_) => {
                            info!("✅ Bot connected successfully!");
                        }
                        Event::Receipt(receipt) => {
                            info!(
                                "Got receipt for message(s) {:?}, type: {:?}",
                                receipt.message_ids, receipt.r#type
                            );
                        }
                        Event::LoggedOut(_) => {
                            error!("❌ Bot was logged out!");
                        }
                        _ => {
                            // debug!("Received unhandled event: {:?}", event);
                        }
                    }
                }
            })
            .build()
            .await
            .expect("Failed to build bot");

        // If you want and need, you can get the client:
        // let client = bot.client();

        let bot_handle = match bot.run().await {
            Ok(handle) => handle,
            Err(e) => {
                error!("Bot failed to start: {}", e);
                return;
            }
        };

        bot_handle
            .await
            .expect("Bot task should complete without panicking");
    });
}

trait MediaPing: Downloadable {
    fn media_type(&self) -> MediaType;

    fn build_pong_reply(&self, upload: UploadResponse) -> wa::Message;
}

impl MediaPing for wa::message::ImageMessage {
    fn media_type(&self) -> MediaType {
        MediaType::Image
    }

    fn build_pong_reply(&self, upload: UploadResponse) -> wa::Message {
        let mut image = wa::message::ImageMessage::default();
        image.mimetype = self.mimetype.clone();
        image.caption = Some(MEDIA_PONG_TEXT.to_string());
        image.url = Some(upload.url);
        image.direct_path = Some(upload.direct_path);
        image.media_key = Some(upload.media_key);
        image.file_enc_sha256 = Some(upload.file_enc_sha256);
        image.file_sha256 = Some(upload.file_sha256);
        image.file_length = Some(upload.file_length);
        let mut message = wa::Message::default();
        message.image_message = image.into();
        message
    }
}

impl MediaPing for wa::message::VideoMessage {
    fn media_type(&self) -> MediaType {
        MediaType::Video
    }

    fn build_pong_reply(&self, upload: UploadResponse) -> wa::Message {
        let mut video = wa::message::VideoMessage::default();
        video.mimetype = self.mimetype.clone();
        video.caption = Some(MEDIA_PONG_TEXT.to_string());
        video.url = Some(upload.url);
        video.direct_path = Some(upload.direct_path);
        video.media_key = Some(upload.media_key);
        video.file_enc_sha256 = Some(upload.file_enc_sha256);
        video.file_sha256 = Some(upload.file_sha256);
        video.file_length = Some(upload.file_length);
        video.gif_playback = self.gif_playback;
        video.height = self.height;
        video.width = self.width;
        video.seconds = self.seconds;
        video.gif_attribution = self.gif_attribution;
        let mut message = wa::Message::default();
        message.video_message = video.into();
        message
    }
}

fn get_pingable_media<'a>(message: &'a wa::Message) -> Option<&'a (dyn MediaPing + 'a)> {
    let base_message = message.get_base_message();

    if let Some(msg) = base_message.image_message.as_option()
        && msg.caption.as_deref() == Some(MEDIA_PING_TRIGGER)
    {
        return Some(msg);
    }
    if let Some(msg) = base_message.video_message.as_option()
        && msg.caption.as_deref() == Some(MEDIA_PING_TRIGGER)
    {
        return Some(msg);
    }

    None
}

async fn handle_media_ping(ctx: &MessageContext, media: &(dyn MediaPing + '_)) {
    info!(
        "Received {:?} ping from {}",
        media.media_type(),
        ctx.info.source.sender
    );

    let mut data_buffer = Cursor::new(Vec::new());
    if let Err(e) = ctx.client.download_to_file(media, &mut data_buffer).await {
        error!("Failed to download media: {}", e);
        let mut error_message = wa::Message::default();
        error_message.conversation = Some("Failed to download your media.".to_string());
        let _ = ctx.send_message(error_message).await;
        return;
    }

    info!(
        "Successfully downloaded media. Size: {} bytes. Now uploading...",
        data_buffer.get_ref().len()
    );
    let plaintext_data = data_buffer.into_inner();
    let upload_response = match ctx.client.upload(plaintext_data, media.media_type()).await {
        Ok(resp) => resp,
        Err(e) => {
            error!("Failed to upload media: {}", e);
            let mut error_message = wa::Message::default();
            error_message.conversation = Some("Failed to re-upload the media.".to_string());
            let _ = ctx.send_message(error_message).await;
            return;
        }
    };

    info!("Successfully uploaded media. Constructing reply message...");
    let reply_msg = media.build_pong_reply(upload_response);

    if let Err(e) = ctx.send_message(reply_msg).await {
        error!("Failed to send media pong reply: {}", e);
    } else {
        info!("Media pong reply sent successfully.");
    }
}

/// Parse a CLI argument by its long and short flags.
/// Supports: --flag VALUE, -f VALUE, --flag=VALUE
fn parse_arg(args: &[String], long: &str, short: &str) -> Option<String> {
    let long_prefix = format!("{}=", long);
    let mut iter = args.iter().skip(1); // Skip program name
    while let Some(arg) = iter.next() {
        if arg == long || arg == short {
            return iter.next().cloned();
        }
        if let Some(value) = arg.strip_prefix(&long_prefix) {
            return Some(value.to_string());
        }
    }
    None
}
