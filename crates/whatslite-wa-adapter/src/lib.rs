use wa_rs::proto_helpers::MessageExt;
use wa_rs::types::events::Event;
use whatslite_services::{BackendEvent, ConnectionState};

pub fn map_event(event: &Event) -> Option<BackendEvent> {
    match event {
        Event::Connected(_) => Some(BackendEvent::ConnectionChanged(ConnectionState::Connected)),
        Event::Disconnected(_) => Some(BackendEvent::ConnectionChanged(
            ConnectionState::Disconnected,
        )),
        Event::PairSuccess(success) => Some(BackendEvent::PairingSucceeded {
            account_id: success.id.to_string(),
        }),
        Event::PairError(error) => Some(BackendEvent::Failure {
            message: error.error.clone(),
        }),
        Event::PairingQrCode(qr) => Some(BackendEvent::QrPairingReady {
            code: qr.code.clone(),
            valid_for_seconds: qr.timeout.as_secs(),
        }),
        Event::PairingCode(pair_code) => Some(BackendEvent::PairingCodeReady {
            code: pair_code.code.clone(),
            valid_for_seconds: pair_code.timeout.as_secs(),
        }),
        Event::Messages(batch) => batch.first().and_then(|inbound| {
            inbound
                .message
                .text_content()
                .map(|text| BackendEvent::MessageReceived {
                    chat_id: inbound.info.source.chat.to_string(),
                    sender_id: inbound.info.source.sender.to_string(),
                    text: text.to_owned(),
                    timestamp_millis: inbound.info.timestamp.timestamp_millis(),
                })
        }),
        _ => None,
    }
}
