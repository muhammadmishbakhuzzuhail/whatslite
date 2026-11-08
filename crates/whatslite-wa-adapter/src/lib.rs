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
        Event::PairingQrCode { code, timeout } => Some(BackendEvent::QrPairingReady {
            code: code.clone(),
            valid_for_seconds: timeout.as_secs(),
        }),
        Event::PairingCode { code, timeout } => Some(BackendEvent::PairingCodeReady {
            code: code.clone(),
            valid_for_seconds: timeout.as_secs(),
        }),
        Event::Message(message, info) => {
            message
                .text_content()
                .map(|text| BackendEvent::MessageReceived {
                    chat_id: info.source.chat.to_string(),
                    sender_id: info.source.sender.to_string(),
                    text: text.to_owned(),
                    timestamp_millis: info.timestamp.timestamp_millis(),
                })
        }
        _ => None,
    }
}
