use async_channel::{Receiver, Sender, unbounded};
use whatslite_services::{
    AppSettings, AuthService, BackendCommand, BackendError, BackendEvent, BackendService,
    CallOptions, CallRecord, CallService, CallState, ChatId, ChatService, ChatSummary,
    ConnectionState, ContactId, ContactService, ContactSummary, GroupService, GroupSummary,
    MediaId, MediaItem, MediaService, MessageDraft, MessageId, NotificationService,
    PairingStatus, ServiceError, SettingsStore, ThemePreference,
};

fn sample_attachment_name() -> String {
    let tokens: serde_json::Value = serde_json::from_str(include_str!("../../../specs/design-system/design-tokens.json"))
        .expect("design tokens must be valid JSON");
    tokens["content_fixture"]["sample_attachment"].as_str()
        .expect("sample_attachment token is required").to_owned()
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum MockScenario { #[default] Success, Offline, Failure }

#[derive(Clone, Copy)]
pub struct DeterministicServices { pub scenario: MockScenario }

impl DeterministicServices {
    pub fn new(scenario: MockScenario) -> Self { Self { scenario } }
}

impl Default for DeterministicServices { fn default() -> Self { Self::new(MockScenario::Success) } }

impl DeterministicServices {
    fn result<T>(&self, value: T) -> Result<T, ServiceError> {
        match self.scenario {
            MockScenario::Success => Ok(value),
            MockScenario::Offline => Err(ServiceError::Offline),
            MockScenario::Failure => Err(ServiceError::Failed("scripted failure".into())),
        }
    }
}

#[async_trait::async_trait]
impl AuthService for DeterministicServices {
    async fn start_qr_pairing(&self) -> Result<PairingStatus, ServiceError> {
        self.result(PairingStatus { code: "WHATS-LITE-MOCK-QR".into(), expires_in_seconds: 300 })
    }
    async fn start_phone_linking(&self, phone_number: &str) -> Result<PairingStatus, ServiceError> {
        if phone_number.trim().is_empty() {
            return Err(ServiceError::Failed("phone number is required".into()));
        }
        self.result(PairingStatus {
            code: "MOCK-123-456".into(),
            expires_in_seconds: 300,
        })
    }
    async fn poll_link_status(&self) -> Result<ConnectionState, ServiceError> { self.result(ConnectionState::WaitingForPairing) }
    async fn logout(&self) -> Result<(), ServiceError> { self.result(()) }
}

#[async_trait::async_trait]
impl ChatService for DeterministicServices {
    async fn list_chats(&self) -> Result<Vec<ChatSummary>, ServiceError> {
        self.result(vec![
            ChatSummary { id: ChatId::new("maya-chat").expect("valid fixture id"), title: "Maya Putri".into(), preview: "Morning! I sent the updated brief.".into(), timestamp: "2026-11-02T09:41:00+07:00".into(), unread: true, favorite: true, is_group: false, archived: false, pinned: true },
            ChatSummary { id: ChatId::new("product-team").expect("valid fixture id"), title: "Product Team".into(), preview: "Rizky: build is ready for review".into(), timestamp: "2026-11-02T09:18:00+07:00".into(), unread: true, favorite: false, is_group: true, archived: false, pinned: false },
            ChatSummary { id: ChatId::new("budi-chat").expect("valid fixture id"), title: "Budi Santoso".into(), preview: "Let’s meet after lunch.".into(), timestamp: "2026-11-01T12:10:00+07:00".into(), unread: false, favorite: false, is_group: false, archived: false, pinned: false },
            ChatSummary { id: ChatId::new("family-chat").expect("valid fixture id"), title: "Family".into(), preview: "Photo · 3 attachments".into(), timestamp: "2026-11-01T16:25:00+07:00".into(), unread: false, favorite: true, is_group: true, archived: false, pinned: false },
            ChatSummary { id: ChatId::new("rizky-chat").expect("valid fixture id"), title: "Rizky Pratama".into(), preview: "Thanks for the notes.".into(), timestamp: "2026-10-31T10:05:00+07:00".into(), unread: false, favorite: false, is_group: false, archived: false, pinned: false },
        ])
    }
    async fn list_messages(&self, chat_id: &ChatId) -> Result<Vec<String>, ServiceError> {
        self.result(if chat_id.as_str() == "maya-chat" { vec!["Morning! I sent the updated brief.".into()] } else { Vec::new() })
    }
    async fn send(&self, _draft: MessageDraft) -> Result<MessageId, ServiceError> {
        self.result(MessageId::new("m002").expect("valid fixture id"))
    }
    async fn edit(&self, _message_id: &MessageId, _text: &str) -> Result<(), ServiceError> { self.result(()) }
    async fn delete(&self, _message_id: &MessageId) -> Result<(), ServiceError> { self.result(()) }
    async fn forward(&self, _message_id: &MessageId, _target: &ChatId) -> Result<MessageId, ServiceError> {
        self.result(MessageId::new("m003").expect("valid fixture id"))
    }
    async fn react(&self, _message_id: &MessageId, _reaction: &str) -> Result<(), ServiceError> { self.result(()) }
    async fn search(&self, chat_id: Option<&ChatId>, query: &str) -> Result<Vec<String>, ServiceError> {
        let messages = self.list_messages(chat_id.unwrap_or(&ChatId::new("maya-chat").expect("valid fixture id"))).await?;
        let query = query.to_lowercase();
        self.result(messages.into_iter().filter(|message| message.to_lowercase().contains(&query)).collect())
    }
}

#[async_trait::async_trait]
impl ContactService for DeterministicServices {
    async fn list_contacts(&self) -> Result<Vec<ContactSummary>, ServiceError> {
        self.result(vec![
            ContactSummary { id: ContactId::new("maya").expect("valid fixture id"), name: "Maya Putri".into() },
            ContactSummary { id: ContactId::new("budi").expect("valid fixture id"), name: "Budi Santoso".into() },
            ContactSummary { id: ContactId::new("rizky").expect("valid fixture id"), name: "Rizky Pratama".into() },
        ])
    }
    async fn search_contacts(&self, query: &str) -> Result<Vec<ContactSummary>, ServiceError> {
        let contacts = self.list_contacts().await?;
        let query = query.to_lowercase();
        Ok(contacts.into_iter().filter(|contact| contact.name.to_lowercase().contains(&query)).collect())
    }
    async fn get_contact(&self, id: &ContactId) -> Result<ContactSummary, ServiceError> {
        let contact = self.list_contacts().await?.into_iter().find(|contact| &contact.id == id).ok_or(ServiceError::NotFound)?;
        self.result(contact)
    }
    async fn block(&self, _id: &ContactId, _blocked: bool) -> Result<(), ServiceError> { self.result(()) }
}

#[async_trait::async_trait]
impl GroupService for DeterministicServices {
    async fn create(&self, name: &str, members: &[ContactId]) -> Result<GroupSummary, ServiceError> {
        if name.trim().is_empty() { return Err(ServiceError::Failed("group name is required".into())); }
        self.result(GroupSummary { id: ChatId::new("new-group").expect("valid fixture id"), name: name.to_owned(), member_count: members.len() })
    }
    async fn update(&self, id: &ChatId, name: &str) -> Result<GroupSummary, ServiceError> {
        self.result(GroupSummary { id: id.clone(), name: name.to_owned(), member_count: 3 })
    }
    async fn members(&self, _id: &ChatId) -> Result<Vec<ContactSummary>, ServiceError> { self.list_contacts().await }
    async fn set_permissions(&self, _id: &ChatId, _admins_only: bool) -> Result<(), ServiceError> { self.result(()) }
    async fn invite(&self, _id: &ChatId, _contact_id: &ContactId) -> Result<(), ServiceError> { self.result(()) }
}

#[async_trait::async_trait]
impl MediaService for DeterministicServices {
    async fn list_media(&self, _chat_id: &ChatId) -> Result<Vec<MediaItem>, ServiceError> {
        self.result(vec![MediaItem { id: MediaId::new("design-update-v2-pdf").expect("valid fixture id"), name: sample_attachment_name() }])
    }
    async fn upload(&self, _chat_id: &ChatId, name: &str) -> Result<MediaId, ServiceError> {
        if name.is_empty() { return Err(ServiceError::Failed("file name is required".into())); }
        self.result(MediaId::new("uploaded-brief").expect("valid fixture id"))
    }
    async fn choose_files(&self) -> Result<Vec<MediaItem>, ServiceError> { self.list_media(&ChatId::new("maya-chat").expect("valid fixture id")).await }
    async fn download(&self, id: &MediaId) -> Result<Vec<u8>, ServiceError> {
        if id.as_str() != "design-update-v2-pdf" && id.as_str() != "uploaded-brief" { return Err(ServiceError::NotFound); }
        self.result(b"mock attachment bytes".to_vec())
    }
    async fn media_index(&self, chat_id: &ChatId) -> Result<Vec<MediaItem>, ServiceError> { self.list_media(chat_id).await }
}

#[async_trait::async_trait]
impl CallService for DeterministicServices {
    async fn history(&self) -> Result<Vec<CallRecord>, ServiceError> {
        self.result(vec![CallRecord { id: "call-001".into(), contact_id: ContactId::new("maya").expect("valid fixture id"), timestamp: "2026-11-02T08:30:00+07:00".into(), video: false, missed: false }])
    }
    async fn place(&self, _contact_id: &ContactId, _options: CallOptions) -> Result<CallState, ServiceError> { self.result(CallState::Ringing) }
    async fn accept(&self) -> Result<CallState, ServiceError> { self.result(CallState::Connected) }
    async fn reject(&self) -> Result<(), ServiceError> { self.result(()) }
    async fn hangup(&self) -> Result<CallState, ServiceError> { self.result(CallState::Ended) }
    async fn set_mute(&self, _muted: bool) -> Result<(), ServiceError> { self.result(()) }
    async fn set_video(&self, _enabled: bool) -> Result<(), ServiceError> { self.result(()) }
    async fn share(&self) -> Result<(), ServiceError> { self.result(()) }
}

#[async_trait::async_trait]
impl SettingsStore for DeterministicServices {
    async fn load(&self) -> Result<AppSettings, ServiceError> { self.result(AppSettings { theme: ThemePreference::System, notifications_enabled: true }) }
    async fn update(&self, _settings: AppSettings) -> Result<(), ServiceError> { self.result(()) }
    async fn set_theme(&self, _theme: ThemePreference) -> Result<(), ServiceError> { self.result(()) }
}

#[async_trait::async_trait]
impl NotificationService for DeterministicServices {
    async fn show_in_app(&self, _title: &str, _body: &str) -> Result<(), ServiceError> { self.result(()) }
    async fn dismiss(&self, _id: &str) -> Result<(), ServiceError> { self.result(()) }
}

pub struct MockBackend {
    event_sender: Sender<BackendEvent>,
    event_receiver: Receiver<BackendEvent>,
}

impl Default for MockBackend {
    fn default() -> Self {
        Self::new()
    }
}

impl MockBackend {
    pub fn new() -> Self {
        let (event_sender, event_receiver) = unbounded();
        Self {
            event_sender,
            event_receiver,
        }
    }

    fn emit(&self, event: BackendEvent) -> Result<(), BackendError> {
        self.event_sender
            .try_send(event)
            .map_err(|_| BackendError::EventReceiverClosed)
    }
}

impl BackendService for MockBackend {
    fn dispatch(&self, command: BackendCommand) -> Result<(), BackendError> {
        match command {
            BackendCommand::StartQrPairing => {
                self.emit(BackendEvent::ConnectionChanged(ConnectionState::Connecting))?;
                self.emit(BackendEvent::ConnectionChanged(
                    ConnectionState::WaitingForPairing,
                ))?;
                self.emit(BackendEvent::QrPairingReady {
                    code: "WHATS-LITE-MOCK-QR".to_owned(),
                    valid_for_seconds: 300,
                })
            }
            BackendCommand::SendTextMessage { chat_id, text } => {
                self.emit(BackendEvent::MessageSent { chat_id, text })
            }
            BackendCommand::Disconnect => {
                self.emit(BackendEvent::ConnectionChanged(ConnectionState::Disconnected))
            }
        }
    }

    fn subscribe(&self) -> Receiver<BackendEvent> {
        self.event_receiver.clone()
    }
}

#[cfg(test)]
mod async_service_contract_tests {
    use super::*;
    use std::sync::Arc;
    use std::task::{Context, Poll, Wake, Waker};

    struct Ready;
    impl Wake for Ready { fn wake(self: Arc<Self>) {} }
    fn block_on<T>(future: impl std::future::Future<Output = T>) -> T {
        let waker = Waker::from(Arc::new(Ready));
        let mut context = Context::from_waker(&waker);
        let mut future = std::pin::pin!(future);
        loop { if let Poll::Ready(value) = future.as_mut().poll(&mut context) { return value; } }
    }

    #[test]
    fn scripted_scenarios_are_repeatable_and_expose_failures() {
        let success = DeterministicServices::new(MockScenario::Success);
        let first = block_on(success.list_chats()).unwrap();
        let second = block_on(success.list_chats()).unwrap();
        assert_eq!(first, second);
        assert_eq!(first[0].id.as_str(), "maya-chat");

        let offline = DeterministicServices::new(MockScenario::Offline);
        assert_eq!(block_on(offline.list_chats()), Err(ServiceError::Offline));

        let failure = DeterministicServices::new(MockScenario::Failure);
        assert_eq!(block_on(failure.send(MessageDraft { chat_id: first[0].id.clone(), text: "retry me".into() })), Err(ServiceError::Failed("scripted failure".into())));

        let pairing = block_on(success.start_phone_linking("+6281200000000")).unwrap();
        assert_eq!(pairing.code, "MOCK-123-456");
        assert_eq!(block_on(success.start_phone_linking(" ")), Err(ServiceError::Failed("phone number is required".into())));
        let media = block_on(success.list_media(&first[0].id)).unwrap();
        assert_eq!(media[0].name, "Design-Update-v2.pdf");
    }

    #[test]
    fn mock_service_fixtures_are_stable_across_service_families() {
        let services = DeterministicServices::default();
        let chats = block_on(services.list_chats()).unwrap();
        assert_eq!(chats, block_on(services.list_chats()).unwrap());
        assert_eq!(chats.len(), 5);
        assert!(chats.iter().all(|chat| chat.timestamp.contains('T')));

        let groups = block_on(services.members(&ChatId::new("product-team").unwrap())).unwrap();
        assert_eq!(groups, block_on(services.members(&ChatId::new("product-team").unwrap())).unwrap());
        assert_eq!(groups.iter().map(|contact| contact.id.as_str()).collect::<Vec<_>>(), ["maya", "budi", "rizky"]);

        let calls = block_on(services.history()).unwrap();
        assert_eq!(calls, block_on(services.history()).unwrap());
        assert!(calls[0].timestamp.contains('T'));
        assert_eq!(block_on(services.load()).unwrap(), block_on(services.load()).unwrap());
        assert_eq!(block_on(services.choose_files()).unwrap(), block_on(services.choose_files()).unwrap());
    }
}
