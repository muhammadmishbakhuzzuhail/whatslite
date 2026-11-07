//! Deterministic fixture model for the WhatsLite desktop mock.
//!
//! The UI is intentionally frontend-only: every value below is local sample
//! data, and no WhatsApp connection or network call is made.

#[derive(Clone, Debug)]
pub struct Chat {
    pub id: usize,
    pub name: String,
    pub initials: String,
    pub preview: String,
    pub time: String,
    pub unread: u32,
    pub muted: bool,
    pub pinned: bool,
    pub online: bool,
    pub is_group: bool,
    pub archived: bool,
    pub favorite: bool,
    pub draft: Option<String>,
    pub typing: Option<String>,
    pub verified: bool,
    pub business: bool,
}

impl Chat {
    pub fn preview_text(&self) -> String {
        if let Some(draft) = &self.draft {
            format!("Draft: {draft}")
        } else if let Some(typing) = &self.typing {
            typing.clone()
        } else {
            self.preview.clone()
        }
    }

    pub fn is_unread(&self) -> bool {
        self.unread > 0
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Sender {
    Me,
    Contact,
    System,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Delivery {
    Sending,
    Sent,
    Delivered,
    Read,
    Failed,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewOnceState {
    Sealed,
    Opening,
    Opened,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PollState {
    Open,
    Voted,
    Closed,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum EventState {
    Invited,
    Going,
    Declined,
    Updated,
    Cancelled,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum LiveLocationState {
    Sharing,
    Paused,
    Ended,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum MediaState {
    TapToDownload,
    Downloading,
    Failed,
    Expired,
    Ready,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Notice {
    Encrypted,
    Disappearing,
    System,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum TextRunStyle {
    Normal,
    Bold,
    Italic,
    Underline,
    Strike,
    Code,
    Link,
}

#[derive(Clone, Debug)]
pub struct TextRun {
    pub text: String,
    pub style: TextRunStyle,
}

#[derive(Clone, Debug)]
pub enum MessageBody {
    Text(String),
    Rich(Vec<TextRun>),
    Image {
        caption: Option<String>,
        state: MediaState,
        ratio: f32,
    },
    Video {
        caption: Option<String>,
        duration: String,
        state: MediaState,
    },
    Voice {
        duration: String,
        transcript: Option<String>,
    },
    Document {
        name: String,
        meta: String,
    },
    Link {
        title: String,
        domain: String,
        description: String,
    },
    Album {
        count: usize,
        caption: Option<String>,
    },
    Emoji(String),
    Poll {
        question: String,
        options: Vec<(String, u32)>,
        state: PollState,
        multiple: bool,
    },
    Event {
        title: String,
        time: String,
        location: String,
        state: EventState,
    },
    ContactCard {
        name: String,
        phone: String,
    },
    Location {
        name: String,
        address: String,
    },
    Sticker {
        symbol: String,
        pack_missing: bool,
    },
    Mention {
        text: String,
        mention: String,
    },
    ViewOnce {
        caption: Option<String>,
        state: ViewOnceState,
    },
    Gif {
        caption: Option<String>,
        failed: bool,
    },
    Upload {
        name: String,
        progress: u32,
        failed: bool,
    },
    Unread(String),
    LiveLocation {
        name: String,
        address: String,
        remaining: String,
        state: LiveLocationState,
    },
    Call {
        missed: bool,
        video: bool,
        duration: Option<String>,
    },
    Deleted,
    System(String),
    Notice(Notice),
    Date(String),
}

#[derive(Clone, Debug)]
pub struct ReplyPreview {
    pub sender: String,
    pub text: String,
    pub deleted: bool,
    pub message_id: Option<usize>,
}

#[derive(Clone, Debug)]
pub struct Reaction {
    pub emoji: String,
    pub count: u32,
}

#[derive(Clone, Debug)]
pub struct Message {
    pub id: usize,
    pub sender: Sender,
    pub sender_name: Option<String>,
    pub body: MessageBody,
    pub time: String,
    pub delivery: Delivery,
    pub edited: bool,
    pub reply: Option<ReplyPreview>,
    pub reactions: Vec<Reaction>,
    pub starred: bool,
    pub countdown: Option<String>,
    pub forwarded: u32,
}

impl Message {
    pub fn outgoing(&self) -> bool {
        self.sender == Sender::Me
    }

    pub fn text(&self) -> Option<&str> {
        match &self.body {
            MessageBody::Text(text) => Some(text),
            _ => None,
        }
    }
}

#[derive(Clone, Debug)]
pub struct CallEntry {
    pub id: usize,
    pub name: String,
    pub initials: String,
    pub time: String,
    pub video: bool,
    pub missed: bool,
    pub incoming: bool,
    pub duration: Option<String>,
    pub group: bool,
}

#[derive(Clone, Debug)]
pub struct StatusUpdate {
    pub id: usize,
    pub name: String,
    pub initials: String,
    pub time: String,
    pub seen: bool,
    pub muted: bool,
    pub items: usize,
    pub caption: String,
    pub accent: usize,
}

#[derive(Clone, Debug)]
pub struct Community {
    pub id: usize,
    pub name: String,
    pub initials: String,
    pub members: String,
    pub groups: String,
    pub description: String,
    pub joined: bool,
    pub unread: u32,
}

#[derive(Clone, Debug)]
pub enum MediaKind {
    Image,
    Video,
    Document,
    Link,
}

#[derive(Clone, Debug)]
pub struct MediaItem {
    pub id: usize,
    pub name: String,
    pub meta: String,
    pub kind: MediaKind,
    pub duration: Option<String>,
    pub chat: String,
    pub accent: usize,
}

#[derive(Clone, Debug)]
pub struct StarredMessage {
    pub id: usize,
    pub chat: String,
    pub sender: String,
    pub preview: String,
    pub date: String,
}

#[derive(Clone, Debug)]
pub struct Account {
    pub id: usize,
    pub name: String,
    pub initials: String,
    pub label: String,
    pub active: bool,
    pub unread: u32,
}

#[derive(Clone, Debug)]
pub struct Broadcast {
    pub id: usize,
    pub name: String,
    pub recipients: String,
    pub time: String,
    pub preview: String,
}

#[derive(Clone, Debug)]
pub struct Contact {
    pub id: usize,
    pub name: String,
    pub initials: String,
    pub about: String,
    pub phone: String,
    pub online: bool,
    pub blocked: bool,
    pub favorite: bool,
}

pub fn chats() -> Vec<Chat> {
    vec![
        Chat {
            id: 0,
            name: "Rizky Pratama".into(),
            initials: "RP".into(),
            preview: "Foto produknya sudah".into(),
            time: "21:14".into(),
            unread: 0,
            muted: false,
            pinned: false,
            online: true,
            is_group: false,
            archived: false,
            favorite: true,
            draft: None,
            typing: None,
            verified: false,
            business: false,
        },
        Chat {
            id: 1,
            name: "Tim Produk".into(),
            initials: "TP".into(),
            preview: "Deck sudah aku update".into(),
            time: "09:41".into(),
            unread: 3,
            muted: false,
            pinned: true,
            online: false,
            is_group: true,
            archived: false,
            favorite: true,
            draft: None,
            typing: None,
            verified: false,
            business: false,
        },
        Chat {
            id: 2,
            name: "Maya".into(),
            initials: "MA".into(),
            preview: "Voice note 0:12".into(),
            time: "08:22".into(),
            unread: 0,
            muted: false,
            pinned: false,
            online: true,
            is_group: false,
            archived: false,
            favorite: true,
            draft: None,
            typing: Some("typing…".into()),
            verified: false,
            business: false,
        },
        Chat {
            id: 3,
            name: "Jakarta Crew".into(),
            initials: "JC".into(),
            preview: "Rina: besok jadi?".into(),
            time: "Yesterday".into(),
            unread: 0,
            muted: true,
            pinned: false,
            online: false,
            is_group: true,
            archived: false,
            favorite: false,
            draft: None,
            typing: None,
            verified: false,
            business: false,
        },
        Chat {
            id: 4,
            name: "Dewi Lestari Kusumaningrum".into(),
            initials: "DL".into(),
            preview: "Terima kasih ya!".into(),
            time: "Tue".into(),
            unread: 0,
            muted: true,
            pinned: false,
            online: false,
            is_group: false,
            archived: false,
            favorite: false,
            draft: Some("Besok aku kirim invoice ya".into()),
            typing: None,
            verified: false,
            business: false,
        },
        Chat {
            id: 5,
            name: "Bank BCA".into(),
            initials: "BC".into(),
            preview: "Kode OTP kamu: 4821".into(),
            time: "Mon".into(),
            unread: 0,
            muted: false,
            pinned: false,
            online: false,
            is_group: false,
            archived: false,
            favorite: false,
            draft: None,
            typing: None,
            verified: true,
            business: true,
        },
        Chat {
            id: 6,
            name: "Promo Grosir".into(),
            initials: "PG".into(),
            preview: "Katalog grosir September sudah rilis".into(),
            time: "Mon".into(),
            unread: 128,
            muted: false,
            pinned: false,
            online: false,
            is_group: false,
            archived: false,
            favorite: false,
            draft: None,
            typing: None,
            verified: true,
            business: true,
        },
    ]
}

pub fn archived_chats() -> Vec<Chat> {
    let mut archived = chats()
        .into_iter()
        .filter(|chat| matches!(chat.id, 1 | 3 | 4 | 5))
        .collect::<Vec<_>>();
    for chat in &mut archived {
        chat.archived = true;
        chat.muted = true;
        chat.pinned = false;
    }
    archived
}

pub fn contacts() -> Vec<Contact> {
    vec![
        Contact { id: 0, name: "Rizky Pratama".into(), initials: "RP".into(), about: "Available".into(), phone: "+62 812 3456 7890".into(), online: true, blocked: false, favorite: true },
        Contact { id: 1, name: "Maya Putri".into(), initials: "MP".into(), about: "Designing calm interfaces".into(), phone: "+62 811 2233 4455".into(), online: true, blocked: false, favorite: true },
        Contact { id: 2, name: "Dewi Lestari".into(), initials: "DL".into(), about: "Linen, always".into(), phone: "+62 813 9988 7766".into(), online: false, blocked: false, favorite: false },
        Contact { id: 3, name: "Budi Santoso".into(), initials: "BS".into(), about: "At the studio".into(), phone: "+62 812 0000 1111".into(), online: false, blocked: false, favorite: false },
        Contact { id: 4, name: "Rina".into(), initials: "RI".into(), about: "Production schedule".into(), phone: "+62 815 4433 2211".into(), online: true, blocked: false, favorite: false },
        Contact { id: 5, name: "Bayu".into(), initials: "BA".into(), about: "Camera and lighting".into(), phone: "+62 817 6655 4433".into(), online: false, blocked: false, favorite: false },
        Contact { id: 6, name: "Sari".into(), initials: "SA".into(), about: "Studio manager".into(), phone: "+62 818 2211 0099".into(), online: false, blocked: false, favorite: false },
    ]
}

pub fn accounts() -> Vec<Account> {
    vec![
        Account { id: 0, name: "You".into(), initials: "Y".into(), label: "Personal".into(), active: true, unread: 0 },
        Account { id: 1, name: "Sari".into(), initials: "SA".into(), label: "Studio".into(), active: false, unread: 3 },
    ]
}

pub fn broadcasts() -> Vec<Broadcast> {
    vec![
        Broadcast { id: 0, name: "Resellers".into(), recipients: "12 recipients".into(), time: "Yesterday".into(), preview: "Katalog September sudah rilis, harga grosir ada di halaman 3.".into() },
        Broadcast { id: 1, name: "Studio clients".into(), recipients: "8 recipients".into(), time: "Tue".into(), preview: "Jadwal produksi minggu depan sudah aku kirim.".into() },
        Broadcast { id: 2, name: "Workshop crew".into(), recipients: "4 recipients".into(), time: "Mon".into(), preview: "Bawa sampel kain warna hitam besok.".into() },
    ]
}

pub fn calls() -> Vec<CallEntry> {
    vec![
        CallEntry { id: 0, name: "Rizky Pratama".into(), initials: "RP".into(), time: "Yesterday, 21:08".into(), video: false, missed: true, incoming: true, duration: None, group: false },
        CallEntry { id: 1, name: "Tim Produk".into(), initials: "TP".into(), time: "Today, 09:41".into(), video: true, missed: false, incoming: true, duration: Some("12 min".into()), group: true },
        CallEntry { id: 2, name: "Maya".into(), initials: "MA".into(), time: "Today, 08:15".into(), video: false, missed: false, incoming: false, duration: Some("4 min".into()), group: false },
        CallEntry { id: 3, name: "Dewi Lestari".into(), initials: "DL".into(), time: "Tue, 19:30".into(), video: true, missed: true, incoming: true, duration: None, group: false },
        CallEntry { id: 4, name: "Bank BCA".into(), initials: "BC".into(), time: "Mon, 14:02".into(), video: false, missed: false, incoming: true, duration: Some("1 min".into()), group: false },
    ]
}

pub fn statuses() -> Vec<StatusUpdate> {
    vec![
        StatusUpdate { id: 0, name: "My status".into(), initials: "Y".into(), time: "Tap to add status update".into(), seen: true, muted: false, items: 0, caption: String::new(), accent: 0 },
        StatusUpdate { id: 1, name: "Rizky Pratama".into(), initials: "RP".into(), time: "Today, 09:12".into(), seen: false, muted: false, items: 3, caption: "Sampel kain linen — koleksi September.".into(), accent: 0 },
        StatusUpdate { id: 2, name: "Maya".into(), initials: "MA".into(), time: "Today, 08:40".into(), seen: false, muted: false, items: 2, caption: "Studio lighting walkthrough".into(), accent: 1 },
        StatusUpdate { id: 3, name: "Jakarta Crew".into(), initials: "JC".into(), time: "Yesterday, 22:10".into(), seen: false, muted: false, items: 5, caption: "Revisi halaman 3 jam lima".into(), accent: 2 },
        StatusUpdate { id: 4, name: "Dewi Lestari".into(), initials: "DL".into(), time: "Yesterday, 19:02".into(), seen: true, muted: false, items: 1, caption: "Terima kasih sudah menunggu.".into(), accent: 3 },
        StatusUpdate { id: 5, name: "Bank BCA".into(), initials: "BC".into(), time: "Mon, 14:20".into(), seen: true, muted: true, items: 1, caption: "Promo Grosir".into(), accent: 4 },
    ]
}

pub fn communities() -> Vec<Community> {
    vec![
        Community { id: 0, name: "Kain Nusantara".into(), initials: "KN".into(), members: "1.2k members".into(), groups: "18 groups".into(), description: "A community for weavers, studios and shops working with Indonesian textiles.".into(), joined: true, unread: 4 },
        Community { id: 1, name: "Studio Linen".into(), initials: "SL".into(), members: "240 members".into(), groups: "6 groups".into(), description: "Linen notes, production schedules and calm interfaces.".into(), joined: true, unread: 0 },
        Community { id: 2, name: "Desain Tenang".into(), initials: "DT".into(), members: "90 members".into(), groups: "4 groups".into(), description: "A small community about quiet, useful product design.".into(), joined: true, unread: 0 },
        Community { id: 3, name: "Bengkel Kayu".into(), initials: "BK".into(), members: "640 members".into(), groups: "8 groups".into(), description: "Woodworking, tools and shared workshop time.".into(), joined: false, unread: 0 },
        Community { id: 4, name: "Pasar Kreatif".into(), initials: "PK".into(), members: "2.1k members".into(), groups: "12 groups".into(), description: "A market for independent makers and small studios.".into(), joined: false, unread: 0 },
    ]
}

pub fn media_items() -> Vec<MediaItem> {
    vec![
        MediaItem { id: 0, name: "Linen detail".into(), meta: "2.4 MB".into(), kind: MediaKind::Image, duration: None, chat: "Rizky Pratama".into(), accent: 0 },
        MediaItem { id: 1, name: "Studio lighting".into(), meta: "18 MB".into(), kind: MediaKind::Video, duration: Some("0:51".into()), chat: "Rizky Pratama".into(), accent: 1 },
        MediaItem { id: 2, name: "Folded stack".into(), meta: "1.2 MB".into(), kind: MediaKind::Image, duration: None, chat: "Tim Produk".into(), accent: 2 },
        MediaItem { id: 3, name: "Scissors on coral".into(), meta: "2.1 MB".into(), kind: MediaKind::Image, duration: None, chat: "Studio Linen".into(), accent: 3 },
        MediaItem { id: 4, name: "Fabric close-up".into(), meta: "3.0 MB".into(), kind: MediaKind::Video, duration: Some("0:33".into()), chat: "Jakarta Crew".into(), accent: 4 },
        MediaItem { id: 5, name: "Invoice-2026-09.pdf".into(), meta: "PDF · 1.2 MB".into(), kind: MediaKind::Document, duration: None, chat: "Rizky Pratama".into(), accent: 5 },
        MediaItem { id: 6, name: "Designing calm interfaces".into(), meta: "fieldnotes.design".into(), kind: MediaKind::Link, duration: None, chat: "Rizky Pratama".into(), accent: 0 },
        MediaItem { id: 7, name: "Production schedule Q4".into(), meta: "Docs · 3 pages".into(), kind: MediaKind::Document, duration: None, chat: "Tim Produk".into(), accent: 1 },
    ]
}

pub fn starred_messages() -> Vec<StarredMessage> {
    vec![
        StarredMessage { id: 0, chat: "Rizky Pratama".into(), sender: "Rizky".into(), preview: "Aku kirim alamat studionya ya".into(), date: "19 Sep".into() },
        StarredMessage { id: 1, chat: "Tim Produk".into(), sender: "Rina".into(), preview: "Revisi halaman 3 jam lima".into(), date: "17 Sep".into() },
        StarredMessage { id: 2, chat: "Maya".into(), sender: "Maya".into(), preview: "Voice note 0:12".into(), date: "12 Sep".into() },
        StarredMessage { id: 3, chat: "Dewi Lestari".into(), sender: "Dewi".into(), preview: "Invoice sudah aku transfer".into(), date: "09 Sep".into() },
        StarredMessage { id: 4, chat: "Jakarta Crew".into(), sender: "Bayu".into(), preview: "Bawa sampel kain warna hitam".into(), date: "05 Sep".into() },
        StarredMessage { id: 5, chat: "Bank BCA".into(), sender: "BCA".into(), preview: "Kode OTP kamu: 4821".into(), date: "02 Sep".into() },
    ]
}

fn message(id: usize, sender: Sender, sender_name: Option<&str>, body: MessageBody, time: &str) -> Message {
    Message {
        id,
        sender,
        sender_name: sender_name.map(str::to_owned),
        body,
        time: time.into(),
        delivery: if sender == Sender::Me { Delivery::Read } else { Delivery::Delivered },
        edited: false,
        reply: None,
        reactions: Vec::new(),
        starred: false,
        countdown: None,
        forwarded: 0,
    }
}

pub fn messages_for(chat_id: usize) -> Vec<Message> {
    match chat_id {
        0 => direct_conversation(),
        1 => group_conversation(),
        2 => {
            let mut messages = vec![
                message(0, Sender::System, None, MessageBody::Date("Today".into()), ""),
                message(1, Sender::Contact, Some("Maya"), MessageBody::Text("Pagi! Voice note soal sampel kain sudah aku kirim.".into()), "08:20"),
                message(2, Sender::Me, None, MessageBody::Text("Oke, aku dengar dulu ya.".into()), "08:21"),
                message(3, Sender::Contact, Some("Maya"), MessageBody::Voice { duration: "0:12".into(), transcript: Some("Sampel linen yang hitam datang minggu depan.".into()) }, "08:22"),
                message(4, Sender::Contact, Some("Maya"), MessageBody::Image { caption: Some("Linen detail".into()), state: MediaState::TapToDownload, ratio: 1.0 }, "08:23"),
                message(5, Sender::Contact, Some("Maya"), MessageBody::Video { caption: Some("Studio lighting".into()), duration: "0:51".into(), state: MediaState::Downloading }, "08:24"),
                message(6, Sender::Contact, Some("Maya"), MessageBody::Document { name: "Invoice-2026-09.pdf".into(), meta: "PDF · 1.2 MB".into() }, "08:25"),
                message(7, Sender::Contact, Some("Maya"), MessageBody::Link { title: "Designing calm interfaces".into(), domain: "fieldnotes.design".into(), description: "Notes on quiet, useful product design.".into() }, "08:26"),
                message(8, Sender::Contact, Some("Maya"), MessageBody::Album { count: 5, caption: Some("Tiga warna, semua natural dye.".into()) }, "08:27"),
                message(9, Sender::Me, None, MessageBody::Emoji("♥".into()), "08:28"),
                message(10, Sender::System, None, MessageBody::System("Maya changed the subject to “Studio Linen”".into()), ""),
            ];
            let mut failed = message(11, Sender::Me, None, MessageBody::Text("Coba kirim ulang sampel yang tadi.".into()), "08:29");
            failed.delivery = Delivery::Failed;
            messages.push(failed);
            messages.push(message(12, Sender::Contact, Some("Maya"), MessageBody::Image { caption: Some("Expired sample".into()), state: MediaState::Expired, ratio: 1.0 }, "08:30"));
            messages.push(message(13, Sender::Contact, Some("Maya"), MessageBody::Image { caption: Some("Failed download".into()), state: MediaState::Failed, ratio: 1.0 }, "08:31"));
            messages.push(message(14, Sender::Contact, Some("Maya"), MessageBody::Poll {
                question: "Kapan review deck final?".into(),
                options: vec![("Jumat".into(), 6), ("Senin".into(), 3), ("Maybe".into(), 1)],
                state: PollState::Voted,
                multiple: false,
            }, "08:32"));
            messages.push(message(27, Sender::Contact, Some("Maya"), MessageBody::Poll {
                question: "Bahan untuk koleksi berikutnya?".into(),
                options: vec![("Linen".into(), 5), ("Tenun".into(), 3), ("Katun".into(), 2)],
                state: PollState::Closed,
                multiple: true,
            }, "08:32"));
            messages.push(message(15, Sender::Contact, Some("Maya"), MessageBody::Event {
                title: "Review Q4".into(),
                time: "Fri, 19 Sep · 15:00".into(),
                location: "Jl. Kemang Raya".into(),
                state: EventState::Going,
            }, "08:33"));
            messages.push(message(28, Sender::Contact, Some("Maya"), MessageBody::Event {
                title: "Foto produk batch 2".into(),
                time: "Sat, 20 Sep · 10:00".into(),
                location: "Jl. Kemang Raya".into(),
                state: EventState::Cancelled,
            }, "08:34"));
            messages.push(message(16, Sender::Contact, Some("Maya"), MessageBody::ContactCard {
                name: "Rizky Pratama".into(),
                phone: "+62 812 3456 7890".into(),
            }, "08:34"));
            messages.push(message(17, Sender::Contact, Some("Maya"), MessageBody::Location {
                name: "Studio Kain".into(),
                address: "Jl. Kemang Raya · 1.2 km".into(),
            }, "08:35"));
            messages.push(message(18, Sender::Contact, Some("Maya"), MessageBody::Sticker { symbol: "✿".into(), pack_missing: false }, "08:36"));
            messages.push(message(29, Sender::Contact, Some("Maya"), MessageBody::Sticker { symbol: "☁".into(), pack_missing: true }, "08:37"));
            messages.push(message(19, Sender::Contact, Some("Maya"), MessageBody::ViewOnce { caption: Some("Photo · view once".into()), state: ViewOnceState::Sealed }, "08:37"));
            messages.push(message(20, Sender::Contact, Some("Maya"), MessageBody::Gif { caption: Some("Celebrate".into()), failed: false }, "08:38"));
            messages.push(message(30, Sender::Contact, Some("Maya"), MessageBody::Gif { caption: Some("Celebrate".into()), failed: true }, "08:39"));
            messages.push(message(21, Sender::Contact, Some("Maya"), MessageBody::ViewOnce { caption: None, state: ViewOnceState::Opening }, "08:39"));
            messages.push(message(22, Sender::Contact, Some("Maya"), MessageBody::ViewOnce { caption: Some("Opened sample".into()), state: ViewOnceState::Opened }, "08:40"));
            messages.push(message(23, Sender::Me, None, MessageBody::Upload { name: "IMG_2241.jpg".into(), progress: 62, failed: false }, "08:41"));
            messages.push(message(24, Sender::Me, None, MessageBody::Upload { name: "Upload failed · 2.1 MB".into(), progress: 38, failed: true }, "08:42"));
            messages.push(message(25, Sender::Contact, Some("Maya"), MessageBody::LiveLocation { name: "Studio Kain".into(), address: "Jl. Kemang Raya".into(), remaining: "12 more minutes".into(), state: LiveLocationState::Sharing }, "08:43"));
            messages.push(message(26, Sender::Contact, Some("Maya"), MessageBody::Call { missed: true, video: false, duration: None }, "08:44"));
            messages.push(message(31, Sender::Contact, Some("Maya"), MessageBody::LiveLocation { name: "Studio Kain".into(), address: "Jl. Kemang Raya".into(), remaining: "Paused 4 minutes ago".into(), state: LiveLocationState::Paused }, "08:44"));
            messages.push(message(32, Sender::Contact, Some("Maya"), MessageBody::LiveLocation { name: "Studio Kain".into(), address: "Jl. Kemang Raya".into(), remaining: "Ended 08:58".into(), state: LiveLocationState::Ended }, "08:45"));
            messages.push(message(33, Sender::Contact, Some("Rina"), MessageBody::Mention { text: "Tolong cek sampel ini ya".into(), mention: "Rina".into() }, "08:46"));
            let mut forwarded = message(34, Sender::Contact, Some("Maya"), MessageBody::Text("Katalog grosir September sudah rilis.".into()), "08:47");
            forwarded.forwarded = 7;
            messages.push(forwarded);
            messages.push(message(35, Sender::Contact, Some("Maya"), MessageBody::Poll {
                question: "Kapan kita mulai produksi?".into(),
                options: vec![("Minggu ini".into(), 0), ("Minggu depan".into(), 0)],
                state: PollState::Open,
                multiple: true,
            }, "08:48"));
            messages.push(message(36, Sender::Contact, Some("Maya"), MessageBody::Event {
                title: "Sesi foto cadangan".into(),
                time: "Sun, 21 Sep · 09:00".into(),
                location: "Studio Kemang".into(),
                state: EventState::Declined,
            }, "08:49"));
            messages.push(message(37, Sender::Contact, Some("Maya"), MessageBody::Event {
                title: "Review koleksi September".into(),
                time: "Fri, 26 Sep · 16:00".into(),
                location: "Studio Kemang".into(),
                state: EventState::Updated,
            }, "08:50"));
            messages
        }
        3 => vec![
            message(0, Sender::System, None, MessageBody::Date("Yesterday".into()), ""),
            message(1, Sender::Contact, Some("Rina"), MessageBody::Text("Besok jadi ketemu di studio jam 10?".into()), "19:02"),
            message(2, Sender::Contact, Some("Bayu"), MessageBody::Text("Aku bawa kamera dan lampu.".into()), "19:04"),
            message(3, Sender::Me, None, MessageBody::Text("Siap, aku bawa sampel kainnya.".into()), "19:06"),
        ],
        4 => vec![
            message(0, Sender::System, None, MessageBody::Date("Tue".into()), ""),
            message(1, Sender::Contact, Some("Dewi"), MessageBody::Text("Invoice sudah aku transfer ya.".into()), "19:30"),
            message(2, Sender::Me, None, MessageBody::Text("Terima kasih ya!".into()), "19:31"),
        ],
        5 => vec![
            message(0, Sender::System, None, MessageBody::Date("Mon".into()), ""),
            message(1, Sender::Contact, Some("Bank BCA"), MessageBody::Notice(Notice::Encrypted), ""),
            message(2, Sender::Contact, Some("Bank BCA"), MessageBody::Text("Kode OTP kamu: 4821. Jangan bagikan kode ini.".into()), "14:02"),
        ],
        _ => Vec::new(),
    }
}

fn direct_conversation() -> Vec<Message> {
    let mut messages = Vec::new();
    messages.push(message(0, Sender::System, None, MessageBody::Date("19 Sep 2025".into()), ""));
    messages.push(message(1, Sender::System, None, MessageBody::Notice(Notice::Encrypted), ""));
    messages.push(message(2, Sender::System, None, MessageBody::Notice(Notice::Disappearing), ""));
    let mut first = message(3, Sender::Contact, Some("Rizky"), MessageBody::Text("Pagi! Foto produk yang kemarin sudah jadi belum?".into()), "09:12");
    first.starred = true;
    messages.push(first);
    let mut outgoing = message(4, Sender::Me, None, MessageBody::Text("Udah jadi kok, aku foto dari tiga sisi. Nanti aku kirim semua ya.".into()), "09:13");
    outgoing.delivery = Delivery::Read;
    outgoing.countdown = Some("23h".into());
    messages.push(outgoing);
    messages.push(message(5, Sender::Me, None, MessageBody::Image { caption: Some("Sampel kain linen — koleksi September.".into()), state: MediaState::Ready, ratio: 1.35 }, "21:16"));
    messages.push(message(6, Sender::Contact, Some("Rizky"), MessageBody::Text("Keren. Yang warna hitam ada?".into()), "09:15"));
    let mut voice = message(7, Sender::Me, None, MessageBody::Voice { duration: "0:12".into(), transcript: Some("Yang hitam masuk minggu depan, aku kabari kalau sudah sampai.".into()) }, "09:17");
    voice.delivery = Delivery::Read;
    messages.push(voice);
    messages.push(message(8, Sender::Contact, Some("Rizky"), MessageBody::Text("Oke siap, besok aku transfer ya.".into()), "09:18"));
    messages.push(message(9, Sender::System, None, MessageBody::Notice(Notice::System), ""));
    messages.push(message(10, Sender::Me, None, MessageBody::Rich(vec![
        TextRun { text: "Deck final ".into(), style: TextRunStyle::Bold },
        TextRun { text: "sudah ".into(), style: TextRunStyle::Italic },
        TextRun { text: "dikirim".into(), style: TextRunStyle::Underline },
        TextRun { text: ", versi lama ".into(), style: TextRunStyle::Normal },
        TextRun { text: "diarsipkan".into(), style: TextRunStyle::Strike },
        TextRun { text: " — cek ".into(), style: TextRunStyle::Normal },
        TextRun { text: "fieldnotes.design".into(), style: TextRunStyle::Link },
        TextRun { text: " · ".into(), style: TextRunStyle::Normal },
        TextRun { text: "npm run design".into(), style: TextRunStyle::Code },
        TextRun { text: "." .into(), style: TextRunStyle::Normal },
    ]), "09:20"));
    let mut reply = message(11, Sender::Me, None, MessageBody::Text("Ini yang aku maksud.".into()), "09:21");
    reply.reply = Some(ReplyPreview {
        sender: "Rizky".into(),
        text: "Pagi! Foto produk yang kemarin sudah jadi belum?".into(),
        deleted: false,
        message_id: Some(3),
    });
    messages.push(reply);
    messages
}

fn group_conversation() -> Vec<Message> {
    let mut messages = Vec::new();
    messages.push(message(0, Sender::System, None, MessageBody::Date("Today".into()), ""));
    messages.push(message(1, Sender::Contact, Some("Rina"), MessageBody::Text("Pagi semua, deck final aku kirim jam 3 ya.".into()), "09:40"));
    messages.push(message(2, Sender::Contact, Some("Bayu"), MessageBody::Text("Siap. Aku bawa kamera buat foto produknya.".into()), "09:42"));
    let mut outgoing = message(3, Sender::Me, None, MessageBody::Text("Oke, aku siapin ruangannya dulu.".into()), "09:44");
    outgoing.delivery = Delivery::Read;
    messages.push(outgoing);
    messages.push(message(4, Sender::Contact, Some("Sari"), MessageBody::Text("Terima kasih! Aku bawa sampel kain baru juga.".into()), "09:47"));
    messages.push(message(5, Sender::Contact, Some("Rina"), MessageBody::Text("Nanti kita review bareng sebelum dikirim ke klien.".into()), "10:01"));
    messages.push(message(50, Sender::System, None, MessageBody::Unread("3 new messages".into()), ""));
    let mut reply = message(6, Sender::Me, None, MessageBody::Text("Siap, aku tunggu di studio.".into()), "10:02");
    reply.delivery = Delivery::Read;
    reply.reactions = vec![
        Reaction { emoji: "👍".into(), count: 3 },
        Reaction { emoji: "❤️".into(), count: 2 },
        Reaction { emoji: "😂".into(), count: 1 },
        Reaction { emoji: "🙏".into(), count: 1 },
        Reaction { emoji: "🔥".into(), count: 1 },
        Reaction { emoji: "🎉".into(), count: 1 },
        Reaction { emoji: "👀".into(), count: 1 },
        Reaction { emoji: "✅".into(), count: 1 },
    ];
    messages.push(reply);
    messages
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fixture_chats_keep_the_reference_order_and_unread_badges() {
        let chats = chats();
        assert_eq!(chats.len(), 7);
        assert_eq!(chats[0].name, "Rizky Pratama");
        assert_eq!(chats[1].name, "Tim Produk");
        assert_eq!(chats[1].unread, 3);
        assert!(chats[1].pinned);
        assert!(chats[3].muted);
        assert!(chats[4].name.contains("Kusumaningrum"));
        assert!(chats[5].verified);
        assert!(chats[5].business);
        assert_eq!(chats[6].unread, 128);
    }

    #[test]
    fn direct_conversation_contains_the_reference_beats() {
        let messages = messages_for(0);
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Notice(Notice::Encrypted))));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Notice(Notice::Disappearing))));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Image { .. })));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Voice { .. })));
        assert!(messages.iter().any(|message| message.starred));
    }

    #[test]
    fn maya_chat_covers_media_states_and_extra_message_types() {
        let messages = messages_for(2);
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Image { state: MediaState::TapToDownload, .. })));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Video { state: MediaState::Downloading, .. })));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Document { .. })));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Link { .. })));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Album { count: 5, .. })));
        assert!(messages.iter().any(|message| matches!(message.body, MessageBody::Emoji(_))));
        assert!(messages.iter().any(|message| message.delivery == Delivery::Failed));
    }

    #[test]
    fn archived_fixtures_are_muted_and_archived() {
        let archived = archived_chats();
        assert_eq!(archived.len(), 4);
        assert!(archived.iter().all(|chat| chat.archived && chat.muted));
    }

    #[test]
    fn settings_sections_match_the_six_design_sections() {
        let titles = [
            "Account",
            "Privacy",
            "Chats",
            "Notifications",
            "Storage and data",
            "Help and feedback",
        ];
        assert_eq!(titles.len(), 6);
    }
}
