use std::collections::HashSet;
use std::time::Duration;

use gpui_kit::Keystroke;
use gpui_kit::assets::IconName;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::avatar::{Avatar, AvatarGroup};
use gpui_kit::component::badge::Badge;
use gpui_kit::component::button::Button;
use gpui_kit::component::checkbox::Checkbox;
use gpui_kit::component::input::{Input, InputState, Textarea, TextareaState};
use gpui_kit::component::kbd::Kbd;
use gpui_kit::component::progress::Progress;
use gpui_kit::component::radio::Radio;
use gpui_kit::component::scroll::ScrollableElement;
use gpui_kit::prelude::*;
use gpui_kit::{
    Animation, AnimationExt, Context, Entity, FontWeight, IntoElement, KeyDownEvent, MouseButton,
    Render, Role, Window, div, px,
};

use crate::components as ui;
use crate::model::{
    self, Account, Broadcast, CallEntry, Chat as ChatModel, Community, Contact, Delivery,
    MediaItem, MediaKind, MediaState, Message, MessageBody, Notice, Reaction, Sender,
    StarredMessage, StatusUpdate,
};
use crate::theme::{
    CHAT_ROW_HEIGHT, COMPACT_BREAKPOINT, COMPOSER_HEIGHT, CONVERSATION_HEADER_HEIGHT,
    INFO_PANEL_WIDTH, LIST_WIDTH, PANEL_INLINE_BREAKPOINT, Palette, RADIUS_BUBBLE, RADIUS_LG,
    RADIUS_MD, RADIUS_SM, RADIUS_XL, RAIL_WIDTH, SETTINGS_NAV_WIDTH, SIDEBAR_WIDTH, TEXT_BODY,
    TEXT_CAPTION, TEXT_HEADING, TEXT_SMALL, TITLEBAR_HEIGHT, ThemeMode, palette,
};

pub const STATUS_ITEM_LIMIT: usize = 30;

pub const SHORTCUTS: [(&str, &[&str]); 8] = [
    ("Search chats", &["Ctrl", "K"]),
    ("New chat", &["Ctrl", "N"]),
    ("Send message", &["Enter"]),
    ("New line", &["Shift", "Enter"]),
    ("Close the top layer", &["Esc"]),
    ("Open in-chat search", &["Ctrl", "F"]),
    ("Next chat", &["Alt", "↓"]),
    ("Previous chat", &["Alt", "↑"]),
];

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Route {
    Onboarding,
    Chats,
    Calls,
    Updates,
    Communities,
    Media,
    Ai,
    Contacts,
    Settings,
    Archived,
    Starred,
    Locked,
    Broadcast,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ChatSearchFilter {
    All,
    Media,
    Links,
    Docs,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ChatFilter {
    All,
    Unread,
    Favorites,
    Groups,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SettingsPage {
    Account,
    Privacy,
    Chats,
    Notifications,
    Storage,
    Help,
}

impl SettingsPage {
    const ALL: [Self; 6] = [
        Self::Account,
        Self::Privacy,
        Self::Chats,
        Self::Notifications,
        Self::Storage,
        Self::Help,
    ];

    fn title(self) -> &'static str {
        match self {
            Self::Account => "Account",
            Self::Privacy => "Privacy",
            Self::Chats => "Chats",
            Self::Notifications => "Notifications",
            Self::Storage => "Storage and data",
            Self::Help => "Help and feedback",
        }
    }

    fn icon(self) -> IconName {
        match self {
            Self::Account => IconName::User,
            Self::Privacy => IconName::Lock,
            Self::Chats => IconName::MessageSquare,
            Self::Notifications => IconName::Bell,
            Self::Storage => IconName::HardDrive,
            Self::Help => IconName::CircleQuestionMark,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SettingsSubpage {
    LinkedDevices,
    LinkedDeviceLogout,
    Visibility,
    ChangeNumber,
    ChangeNumberOtp,
    RecoveryEmail,
    RecoveryEmailCode,
    ProfilePhoto,
    AutoLock,
    NotificationOverrides,
    NotificationSound,
    NotificationPreview,
    QuietHours,
    BlockedContacts,
    BlockedContactPicker,
    Wallpaper,
    FontSize,
    TwoStep,
    AccountInfo,
    DeleteAccount,
    StoragePerChat,
    HelpCentre,
    DesignReference,
    ContactUs,
    ReportProblem,
    SystemStatus,
    CallStates,
    NetworkStates,
    Primitives,
    Shortcuts,
    Language,
    Proxy,
    About,
}

impl SettingsSubpage {
    fn title(self) -> &'static str {
        match self {
            Self::LinkedDevices => "Linked devices",
            Self::LinkedDeviceLogout => "Log out of device",
            Self::Visibility => "Last seen and online",
            Self::ChangeNumber => "Change phone number",
            Self::ChangeNumberOtp => "Enter the 6-digit code",
            Self::RecoveryEmail => "Recovery email",
            Self::RecoveryEmailCode => "Verify your email",
            Self::ProfilePhoto => "Profile photo",
            Self::AutoLock => "Auto-lock after",
            Self::NotificationOverrides => "Per-chat overrides",
            Self::NotificationSound => "Notification sound",
            Self::NotificationPreview => "Message preview",
            Self::QuietHours => "Quiet hours",
            Self::BlockedContacts => "Blocked contacts",
            Self::BlockedContactPicker => "Block a contact",
            Self::Wallpaper => "Chat wallpaper",
            Self::FontSize => "Chat font size",
            Self::TwoStep => "Two-step verification",
            Self::AccountInfo => "Request account info",
            Self::DeleteAccount => "Delete account",
            Self::StoragePerChat => "Storage per chat",
            Self::HelpCentre => "Help centre",
            Self::DesignReference => "Design reference",
            Self::ContactUs => "Contact us",
            Self::ReportProblem => "Report a problem",
            Self::SystemStatus => "System status",
            Self::CallStates => "Call states",
            Self::NetworkStates => "Network states",
            Self::Primitives => "Primitives",
            Self::Shortcuts => "Keyboard shortcuts",
            Self::Language => "Language",
            Self::Proxy => "Proxy",
            Self::About => "About",
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum MediaTab {
    Media,
    Links,
    Docs,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum InfoPanel {
    Contact,
    Group,
    Community,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum NetworkState {
    Online,
    Connecting,
    Offline,
    Reconnecting,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SystemState {
    OsNotification,
    Permission,
    Tray,
    Update,
    Error,
    DragDrop,
    Paste,
    ToastStack,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CallState {
    Dialing,
    Ringing,
    Connecting,
    Connected,
    Reconnecting,
    Busy,
    NoAnswer,
    EndedByThem,
    Declined,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ProfilePhotoState {
    Empty,
    Cropping,
    Uploaded,
    Saved,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AiState {
    Idle,
    Suggested,
    Streaming,
    Error,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum StagedKind {
    File,
    Image,
    Contact,
}

#[derive(Clone, Debug)]
pub struct StagedAttachment {
    pub kind: StagedKind,
    pub label: String,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ShellStyle {
    Windows,
    MacOS,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum QrState {
    Loading,
    Active,
    Expired,
    Success,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ListState {
    Ready,
    Loading,
    Failed,
}

#[derive(Clone, Debug)]
pub enum Overlay {
    AttachmentMenu,
    HeaderMenu,
    ChatListMenu,
    ContactProfile(String),
    ContextMenu(usize),
    EmojiPicker,
    ConfirmDelete(usize),
    PinnedMessages,
    NewChat,
    NewGroup,
    Lightbox(usize),
    StatusComposer,
    StatusFont,
    StatusPrivacy,
    PollComposer,
    EventComposer,
    DisappearingPicker,
    StickerPicker,
    FileDialog,
    GroupMembers,
    UseCaseDemo,
    ClearChat,
    ExportChat,
    SecurityCode,
    ChannelInfo(String),
    CommunityMembers,
    MemberApprovals,
    LockedUnlock,
    StatusViewers,
    Forward(usize),
    PollVotes,
    Reactions(usize),
    GifPicker,
    NewBroadcast,
    NewCommunity,
    EditContact(usize),
    Accounts,
    AddAccount,
    AiDraft,
    ShortcutEditor(usize),
    SystemState(SystemState),
}

#[derive(Clone, Debug)]
pub struct Toast {
    pub id: u64,
    pub message: String,
}

#[derive(Clone, Debug)]
pub enum Action {
    Navigate(Route),
    SelectChat(usize),
    SetFilter(ChatFilter),
    SetTheme(ThemeMode),
    SetShell(ShellStyle),
    OpenInfo(InfoPanel),
    OpenContactProfile(String),
    ToggleHeaderMenu,
    OpenChatListMenu,
    OpenUseCaseDemo,
    StartDemo(usize),
    CloseInfo,
    CloseOverlay,
    CloseOverlayAndToast(String),
    CloseSubpageAndToast(String),
    Toast(String),
    Send,
    OpenAttachments,
    OpenContextMenu(usize),
    Reply(usize),
    Edit(usize),
    StarMessage(usize),
    OpenPinnedMessages,
    JumpToMessage(usize),
    OpenStarredChat(String),
    UnarchiveAll,
    Delete(usize),
    ConfirmDelete(usize),
    React(usize, String),
    StartCall(usize, bool),
    EndCall,
    SetCallState(CallState),
    PinCallParticipant(usize),
    ToggleMute,
    ToggleVideo,
    OpenStatus(usize),
    OpenStatusComposer,
    CloseStatus,
    ToggleStatusPause,
    SetMediaTab(MediaTab),
    OpenMedia(usize),
    CloseMedia,
    SelectMediaChat(String),
    MediaPrev,
    MediaNext,
    ZoomIn,
    ZoomOut,
    ToggleMediaSelectMode,
    ToggleMediaSelection(usize),
    MediaStar(usize),
    MediaDownload(usize),
    MediaForward(usize),
    MediaDelete(usize),
    MediaStarSelected,
    MediaForwardSelected,
    MediaDeleteSelected,
    SetTextScale(f32),
    SetRtl(bool),
    SetReducedMotion(bool),
    SetSystemTheme,
    SetOsTheme(bool),
    SettingsPage(SettingsPage),
    OpenSubpage(SettingsSubpage),
    CloseSubpage,
    ToggleSetting(&'static str),
    NewChat,
    NewGroup,
    CreateGroup,
    NewBroadcast,
    NewCommunity,
    CreateCommunity,
    SelectBroadcast(usize),
    OpenAccounts,
    SwitchAccount(usize),
    AddAccount,
    BroadcastNextStep,
    BroadcastBack,
    BroadcastCreate,
    ToggleBroadcastRecipient(usize),
    ClearDraft,
    DismissMissedCall,
    UnblockChat(usize),
    SetNetworkState(NetworkState),
    FlushOutbox,
    ClearMediaCache,
    InsertAiDraft,
    ToggleAiTranslation,
    AiSuggest,
    AiRegenerate,
    AiRetry,
    AiCancel,
    AiDiscard,
    AiPrompt,
    InsertAiSuggestion,
    OpenShortcutEditor(usize),
    ApplyShortcut(usize, String),
    ResetShortcuts,
    ToggleStatusItem(usize),
    AddStatusItem,
    RemoveStatusItem(usize),
    SendStatus,
    OpenStatusFont,
    SetStatusFont(usize),
    SetStatusFontSize(usize),
    OpenStatusPrivacy,
    SetStatusPrivacy(usize),
    StageAttachment(StagedKind, String),
    RemoveStagedAttachment(usize),
    SendStaged,
    OpenPollComposer,
    OpenEventComposer,
    CreatePoll,
    CreateEvent,
    OpenDisappearingPicker,
    SetDisappearing(usize),
    OpenStickerPicker,
    SendSticker(String),
    OpenFileDialog,
    SelectFile,
    AttachSelectedFile,
    OpenGroupMembers,
    OpenClearChat,
    ConfirmClearChat,
    OpenExportChat,
    OpenSecurityCode,
    OpenChannelInfo(String),
    OpenCommunityMembers,
    OpenMemberApprovals,
    ToggleChannelFollow,
    ApproveMember(usize),
    OpenStatusViewers,
    OpenForward(usize),
    ForwardTo(usize, usize),
    OpenPollVotes,
    OpenReactions(usize),
    EnterMessageSelection(usize),
    ToggleMessageSelection(usize),
    ExitMessageSelection,
    SelectAllMessages,
    SelectionStar,
    SelectionDelete,
    SelectionForward,
    SelectionCopy,
    SelectionInfo,
    OpenGifPicker,
    SetGifQuery(usize),
    SendGif(String),
    AcceptCall,
    DeclineCall,
    OpenLockedUnlock,
    SetGroupCallSize(usize),
    SendAnnouncement,
    ProfilePhotoUpload,
    ProfilePhotoCropCancel,
    ProfilePhotoCropConfirm,
    ProfilePhotoSave,
    ProfilePhotoRemove,
    ProfilePhotoZoomIn,
    ProfilePhotoZoomOut,
    OpenSystemState(SystemState),
    EditContact(usize),
    SelectContact(usize),
    SelectCommunity(usize),
    CycleQr,
    ToggleSearch,
    SearchNext,
    SearchPrev,
    SetChatSearchFilter(ChatSearchFilter),
    ScrollToBottom,
    ToggleTranscript,
    ToggleRecording,
    CycleVoiceSpeed,
    AppendEmoji(String),
    OpenEmoji,
    ShowComingSoon(&'static str),
    SetListState(ListState),
}

pub struct App {
    route: Route,
    theme: ThemeMode,
    shell: ShellStyle,
    collapsed: bool,
    selected_chat: usize,
    chat_filter: ChatFilter,
    search: Entity<InputState>,
    chat_search: Entity<InputState>,
    composer: Entity<TextareaState>,
    chats: Vec<ChatModel>,
    archived: Vec<ChatModel>,
    messages: Vec<Vec<Message>>,
    calls: Vec<CallEntry>,
    statuses: Vec<StatusUpdate>,
    communities: Vec<Community>,
    media: Vec<MediaItem>,
    starred: Vec<StarredMessage>,
    contacts: Vec<Contact>,
    broadcasts: Vec<Broadcast>,
    selected_broadcast: Option<usize>,
    accounts: Vec<Account>,
    active_account: usize,
    broadcast_step: u8,
    broadcast_recipients: HashSet<usize>,
    missed_banner: bool,
    blocked_chats: HashSet<usize>,
    settings_page: SettingsPage,
    settings_subpage: Option<SettingsSubpage>,
    info_panel: Option<InfoPanel>,
    header_hovered: bool,
    compact: bool,
    overlay: Option<Overlay>,
    toasts: Vec<Toast>,
    toast_seq: u64,
    call_active: Option<usize>,
    call_video: bool,
    call_muted: bool,
    call_state: CallState,
    call_incoming: bool,
    group_call_size: usize,
    call_pinned: Option<usize>,
    status_viewer: Option<usize>,
    status_paused: bool,
    media_tab: MediaTab,
    selected_media: Option<usize>,
    selected_media_chat: Option<String>,
    selected_messages: HashSet<usize>,
    message_select_mode: bool,
    drag_file_hover: Option<usize>,
    gif_query: usize,
    media_zoom: u32,
    media_starred: HashSet<usize>,
    media_deleted: HashSet<usize>,
    media_selected: HashSet<usize>,
    media_select_mode: bool,
    text_scale: f32,
    rtl: bool,
    reduced_motion: bool,
    system_theme: bool,
    os_theme_dark: bool,
    profile_photo: ProfilePhotoState,
    profile_photo_zoom: u32,
    shortcut_overrides: Vec<Option<String>>,
    status_items: Vec<usize>,
    status_drag: Option<usize>,
    staged_attachments: Vec<StagedAttachment>,
    disappearing: usize,
    file_selected: bool,
    channel_following: bool,
    member_requests: Vec<String>,
    community_admin: bool,
    status_font: usize,
    status_font_size: usize,
    status_privacy: usize,
    ai_state: AiState,
    ai_draft_english: bool,
    selected_contact: Option<usize>,
    selected_community: Option<usize>,
    reply_to: Option<usize>,
    editing: Option<usize>,
    pinned: Vec<usize>,
    transcript_open: bool,
    voice_speed: usize,
    searching: bool,
    search_hit: usize,
    chat_search_filter: ChatSearchFilter,
    network_state: NetworkState,
    outbox: usize,
    media_cache_mb: usize,
    recording: bool,
    recording_seconds: u32,
    toggles: HashSet<&'static str>,
    qr_state: QrState,
    list_state: ListState,
    conversation_scroll: gpui_kit::ScrollHandle,
}

impl App {
    pub fn new(window: &mut Window, cx: &mut Context<Self>) -> Self {
        let chats = model::chats();
        let archived = model::archived_chats();
        let messages = (0..chats.len()).map(model::messages_for).collect();
        let reduced_motion = std::env::var("WHATSLITE_REDUCED_MOTION").as_deref() == Ok("1");
        let mut toggles = HashSet::new();
        toggles.extend([
            "read_receipts",
            "enter_sends",
            "media_preview",
            "keep_archived",
            "mute_archived",
            "notifications",
            "sound",
            "auto_download",
            "reaction_notifications",
            "taskbar_badge",
            "spellcheck",
        ]);
        if reduced_motion {
            toggles.insert("reduce_motion");
        }

        let route = initial_route();
        let system_theme = std::env::var("WHATSLITE_THEME").as_deref() == Ok("system")
            || std::env::var("WHATSLITE_SYSTEM_THEME").as_deref() == Ok("1");
        let os_theme_dark = std::env::var("WHATSLITE_OS_THEME").as_deref() == Ok("dark");
        let theme = if std::env::var("WHATSLITE_THEME").as_deref() == Ok("dark")
            || (system_theme && os_theme_dark)
        {
            ThemeMode::Dark
        } else {
            ThemeMode::Light
        };
        let shell = if std::env::var("WHATSLITE_SHELL").as_deref() == Ok("macos") {
            ShellStyle::MacOS
        } else {
            ShellStyle::Windows
        };
        crate::wallpaper::prewarm(window.scale_factor(), theme == ThemeMode::Dark);
        let selected_chat = std::env::var("WHATSLITE_CHAT")
            .ok()
            .and_then(|value| value.parse::<usize>().ok())
            .unwrap_or(0)
            .min(chats.len().saturating_sub(1));
        let settings_page = match std::env::var("WHATSLITE_SETTINGS").as_deref() {
            Ok("privacy") => SettingsPage::Privacy,
            Ok("chats") => SettingsPage::Chats,
            Ok("notifications") => SettingsPage::Notifications,
            Ok("storage") => SettingsPage::Storage,
            Ok("help") => SettingsPage::Help,
            _ => SettingsPage::Account,
        };
        let selected_contact = (route == Route::Contacts).then_some(0);
        let selected_community = (route == Route::Communities).then_some(0);
        let settings_subpage = match std::env::var("WHATSLITE_SUBPAGE")
            .unwrap_or_default()
            .as_str()
        {
            "linked-devices" => Some(SettingsSubpage::LinkedDevices),
            "linked-logout" => Some(SettingsSubpage::LinkedDeviceLogout),
            "visibility" => Some(SettingsSubpage::Visibility),
            "change-number" => Some(SettingsSubpage::ChangeNumber),
            "change-number-otp" => Some(SettingsSubpage::ChangeNumberOtp),
            "recovery-email" => Some(SettingsSubpage::RecoveryEmail),
            "recovery-email-code" => Some(SettingsSubpage::RecoveryEmailCode),
            "profile-photo" => Some(SettingsSubpage::ProfilePhoto),
            "auto-lock" => Some(SettingsSubpage::AutoLock),
            "overrides" => Some(SettingsSubpage::NotificationOverrides),
            "sound" => Some(SettingsSubpage::NotificationSound),
            "preview" => Some(SettingsSubpage::NotificationPreview),
            "quiet-hours" => Some(SettingsSubpage::QuietHours),
            "blocked" => Some(SettingsSubpage::BlockedContacts),
            "block-picker" => Some(SettingsSubpage::BlockedContactPicker),
            "wallpaper" => Some(SettingsSubpage::Wallpaper),
            "font-size" => Some(SettingsSubpage::FontSize),
            "two-step" => Some(SettingsSubpage::TwoStep),
            "account-info" => Some(SettingsSubpage::AccountInfo),
            "delete-account" => Some(SettingsSubpage::DeleteAccount),
            "storage-per-chat" => Some(SettingsSubpage::StoragePerChat),
            "help-centre" => Some(SettingsSubpage::HelpCentre),
            "design" => Some(SettingsSubpage::DesignReference),
            "contact-us" => Some(SettingsSubpage::ContactUs),
            "report" => Some(SettingsSubpage::ReportProblem),
            "system" | "system-status" => Some(SettingsSubpage::SystemStatus),
            "call-states" => Some(SettingsSubpage::CallStates),
            "network" => Some(SettingsSubpage::NetworkStates),
            "primitives" => Some(SettingsSubpage::Primitives),
            "shortcuts" => Some(SettingsSubpage::Shortcuts),
            "language" => Some(SettingsSubpage::Language),
            "proxy" => Some(SettingsSubpage::Proxy),
            "about" => Some(SettingsSubpage::About),
            _ => None,
        };
        let overlay = match std::env::var("WHATSLITE_OVERLAY")
            .unwrap_or_default()
            .as_str()
        {
            "attachments" => Some(Overlay::AttachmentMenu),
            "demo" => Some(Overlay::UseCaseDemo),
            "header-menu" => Some(Overlay::HeaderMenu),
            "chat-list-menu" => Some(Overlay::ChatListMenu),
            "contact-profile" => Some(Overlay::ContactProfile("Rina".into())),
            "context" => Some(Overlay::ContextMenu(3)),
            "emoji" => Some(Overlay::EmojiPicker),
            "delete" => Some(Overlay::ConfirmDelete(3)),
            "newchat" => Some(Overlay::NewChat),
            "newgroup" => Some(Overlay::NewGroup),
            "lightbox" => Some(Overlay::Lightbox(0)),
            "status" => Some(Overlay::StatusComposer),
            "edit-contact" => Some(Overlay::EditContact(0)),
            "new-broadcast" => Some(Overlay::NewBroadcast),
            "new-community" => Some(Overlay::NewCommunity),
            "accounts" => Some(Overlay::Accounts),
            "add-account" => Some(Overlay::AddAccount),
            "ai-draft" => Some(Overlay::AiDraft),
            "status-font" => Some(Overlay::StatusFont),
            "status-privacy" => Some(Overlay::StatusPrivacy),
            "poll" => Some(Overlay::PollComposer),
            "event" => Some(Overlay::EventComposer),
            "disappearing" => Some(Overlay::DisappearingPicker),
            "sticker" => Some(Overlay::StickerPicker),
            "file-dialog" => Some(Overlay::FileDialog),
            "group-members" => Some(Overlay::GroupMembers),
            "clear-chat" => Some(Overlay::ClearChat),
            "export-chat" => Some(Overlay::ExportChat),
            "security-code" => Some(Overlay::SecurityCode),
            "channel-info" => Some(Overlay::ChannelInfo("Design Weekly".into())),
            "community-members" => Some(Overlay::CommunityMembers),
            "member-approvals" => Some(Overlay::MemberApprovals),
            "status-viewers" => Some(Overlay::StatusViewers),
            "locked-unlock" => Some(Overlay::LockedUnlock),
            "forward" => Some(Overlay::Forward(0)),
            "poll-votes" => Some(Overlay::PollVotes),
            "reactions" => Some(Overlay::Reactions(6)),
            "gif" => Some(Overlay::GifPicker),
            "pinned" => Some(Overlay::PinnedMessages),
            "shortcut-editor" => Some(Overlay::ShortcutEditor(0)),
            "system-os" => Some(Overlay::SystemState(SystemState::OsNotification)),
            "system-permission" => Some(Overlay::SystemState(SystemState::Permission)),
            "system-tray" => Some(Overlay::SystemState(SystemState::Tray)),
            "system-update" => Some(Overlay::SystemState(SystemState::Update)),
            "system-error" => Some(Overlay::SystemState(SystemState::Error)),
            "system-drop" => Some(Overlay::SystemState(SystemState::DragDrop)),
            "system-paste" => Some(Overlay::SystemState(SystemState::Paste)),
            "system-toasts" => Some(Overlay::SystemState(SystemState::ToastStack)),
            _ => None,
        };
        let info_panel = match std::env::var("WHATSLITE_INFO").unwrap_or_default().as_str() {
            "contact" => Some(InfoPanel::Contact),
            "group" => Some(InfoPanel::Group),
            "community" => Some(InfoPanel::Community),
            _ => None,
        };
        let status_viewer = if std::env::var("WHATSLITE_STATUS").as_deref() == Ok("1") {
            Some(1)
        } else {
            None
        };
        let call_active = match std::env::var("WHATSLITE_CALL").unwrap_or_default().as_str() {
            "1" => Some(0),
            "group" => Some(1),
            "video" => Some(2),
            _ => None,
        };
        let call_video = std::env::var("WHATSLITE_CALL_VIDEO").as_deref() == Ok("1")
            || std::env::var("WHATSLITE_CALL").as_deref() == Ok("video");
        let searching = std::env::var("WHATSLITE_SEARCH").as_deref() == Ok("1")
            || std::env::var("WHATSLITE_SEARCH_QUERY").is_ok();

        let app = Self {
            route,
            theme,
            shell,
            collapsed: false,
            selected_chat,
            chat_filter: ChatFilter::All,
            search: cx.new(|cx| InputState::new(window, cx).placeholder("Search unread chats")),
            chat_search: cx.new(|cx| {
                let mut state = InputState::new(window, cx).placeholder("Search in chat");
                if let Ok(query) = std::env::var("WHATSLITE_SEARCH_QUERY") {
                    state.set_value(query, window, cx);
                }
                state
            }),
            composer: cx.new(|cx| {
                TextareaState::new(window, cx)
                    .placeholder("Type a message")
                    .auto_grow(1, 6)
            }),
            chats,
            archived,
            messages,
            calls: model::calls(),
            statuses: model::statuses(),
            communities: model::communities(),
            media: model::media_items(),
            starred: model::starred_messages(),
            contacts: model::contacts(),
            broadcasts: model::broadcasts(),
            accounts: model::accounts(),
            active_account: 0,
            broadcast_step: if std::env::var("WHATSLITE_BROADCAST_STEP").as_deref() == Ok("2") {
                2
            } else {
                1
            },
            broadcast_recipients: (0..model::contacts().len()).collect(),
            missed_banner: true,
            blocked_chats: [6usize].into_iter().collect(),
            selected_broadcast: if std::env::var("WHATSLITE_BROADCAST").as_deref() == Ok("1") {
                Some(0)
            } else {
                None
            },
            settings_page,
            settings_subpage,
            info_panel,
            overlay,
            toasts: Vec::new(),
            toast_seq: 0,
            call_active,
            call_video,
            call_muted: false,
            call_pinned: None,
            call_incoming: std::env::var("WHATSLITE_CALL_INCOMING").as_deref() == Ok("1"),
            group_call_size: std::env::var("WHATSLITE_GROUP_CALL_SIZE")
                .ok()
                .and_then(|value| value.parse::<usize>().ok())
                .unwrap_or(4)
                .clamp(2, 6),
            call_state: match std::env::var("WHATSLITE_CALL_STATE")
                .unwrap_or_default()
                .as_str()
            {
                "dialing" => CallState::Dialing,
                "ringing" => CallState::Ringing,
                "connecting" => CallState::Connecting,
                "reconnecting" => CallState::Reconnecting,
                "busy" => CallState::Busy,
                "no-answer" => CallState::NoAnswer,
                "ended-by-them" => CallState::EndedByThem,
                "declined" => CallState::Declined,
                _ => CallState::Connected,
            },
            status_viewer,
            status_paused: false,
            media_tab: MediaTab::Media,
            selected_media: None,
            selected_media_chat: None,
            selected_messages: if std::env::var("WHATSLITE_MESSAGE_SELECT").as_deref() == Ok("1") {
                [3usize, 4].into_iter().collect()
            } else {
                HashSet::new()
            },
            message_select_mode: std::env::var("WHATSLITE_MESSAGE_SELECT").as_deref() == Ok("1"),
            drag_file_hover: std::env::var("WHATSLITE_DRAG_FILE")
                .ok()
                .and_then(|value| value.parse::<usize>().ok()),
            gif_query: if std::env::var("WHATSLITE_GIF_NO_RESULTS").as_deref() == Ok("1") {
                1
            } else {
                0
            },
            media_zoom: 100,
            media_starred: HashSet::new(),
            media_deleted: HashSet::new(),
            media_selected: if std::env::var("WHATSLITE_MEDIA_SELECT").as_deref() == Ok("1") {
                [0usize, 1].into_iter().collect()
            } else {
                HashSet::new()
            },
            media_select_mode: std::env::var("WHATSLITE_MEDIA_SELECT").as_deref() == Ok("1"),
            text_scale: std::env::var("WHATSLITE_TEXT_SCALE")
                .ok()
                .and_then(|value| value.parse::<f32>().ok())
                .unwrap_or(1.0),
            rtl: std::env::var("WHATSLITE_RTL").as_deref() == Ok("1"),
            reduced_motion,
            system_theme,
            os_theme_dark,
            profile_photo: match std::env::var("WHATSLITE_PROFILE_PHOTO")
                .unwrap_or_default()
                .as_str()
            {
                "cropping" => ProfilePhotoState::Cropping,
                "uploaded" => ProfilePhotoState::Uploaded,
                "saved" => ProfilePhotoState::Saved,
                _ => ProfilePhotoState::Empty,
            },
            profile_photo_zoom: 100,
            shortcut_overrides: vec![None; SHORTCUTS.len()],
            status_items: match std::env::var("WHATSLITE_STATUS_ITEMS")
                .ok()
                .and_then(|value| value.parse::<usize>().ok())
            {
                Some(count) => (0..count.min(STATUS_ITEM_LIMIT)).collect(),
                None => vec![0, 2, 3, 1],
            },
            status_drag: std::env::var("WHATSLITE_STATUS_DRAG")
                .ok()
                .and_then(|value| value.parse::<usize>().ok()),
            disappearing: 0,
            file_selected: std::env::var("WHATSLITE_FILE_SELECTED").as_deref() == Ok("1"),
            channel_following: true,
            member_requests: vec!["Dewi Lestari".into(), "Budi Santoso".into(), "Rina".into()],
            community_admin: std::env::var("WHATSLITE_COMMUNITY_MEMBER").as_deref() != Ok("1"),
            staged_attachments: match std::env::var("WHATSLITE_STAGED")
                .ok()
                .and_then(|value| value.parse::<usize>().ok())
                .unwrap_or(0)
            {
                0 => Vec::new(),
                1 => vec![StagedAttachment {
                    kind: StagedKind::Image,
                    label: "IMG_2241.jpg".into(),
                }],
                _ => vec![
                    StagedAttachment {
                        kind: StagedKind::Image,
                        label: "IMG_2241.jpg".into(),
                    },
                    StagedAttachment {
                        kind: StagedKind::File,
                        label: "Invoice-2026-09.pdf".into(),
                    },
                    StagedAttachment {
                        kind: StagedKind::Contact,
                        label: "Rizky Pratama".into(),
                    },
                ],
            },
            status_font: 0,
            status_font_size: 1,
            status_privacy: 1,
            ai_draft_english: false,
            ai_state: match std::env::var("WHATSLITE_AI").unwrap_or_default().as_str() {
                "suggested" => AiState::Suggested,
                "streaming" => AiState::Streaming,
                "error" => AiState::Error,
                _ => AiState::Idle,
            },
            selected_contact,
            selected_community,
            header_hovered: std::env::var("WHATSLITE_HEADER_HOVER").as_deref() == Ok("1"),
            compact: false,
            reply_to: None,
            editing: None,
            pinned: vec![3],
            transcript_open: false,
            voice_speed: 0,
            searching,
            search_hit: 0,
            chat_search_filter: match std::env::var("WHATSLITE_SEARCH_FILTER")
                .unwrap_or_default()
                .as_str()
            {
                "media" => ChatSearchFilter::Media,
                "links" => ChatSearchFilter::Links,
                "docs" => ChatSearchFilter::Docs,
                _ => ChatSearchFilter::All,
            },
            network_state: match std::env::var("WHATSLITE_NETWORK")
                .unwrap_or_default()
                .as_str()
            {
                "connecting" => NetworkState::Connecting,
                "offline" => NetworkState::Offline,
                "reconnecting" => NetworkState::Reconnecting,
                _ => NetworkState::Online,
            },
            outbox: std::env::var("WHATSLITE_OUTBOX")
                .ok()
                .and_then(|value| value.parse::<usize>().ok())
                .unwrap_or(0),
            media_cache_mb: std::env::var("WHATSLITE_MEDIA_CACHE")
                .ok()
                .and_then(|value| value.parse::<usize>().ok())
                .unwrap_or(612),
            recording: std::env::var("WHATSLITE_RECORDING").as_deref() == Ok("1"),
            recording_seconds: if std::env::var("WHATSLITE_RECORDING").as_deref() == Ok("1") {
                3
            } else {
                0
            },
            toggles,
            qr_state: match std::env::var("WHATSLITE_QR").unwrap_or_default().as_str() {
                "loading" => QrState::Loading,
                "expired" => QrState::Expired,
                "success" => QrState::Success,
                _ => QrState::Active,
            },
            list_state: match std::env::var("WHATSLITE_LIST_STATE")
                .unwrap_or_default()
                .as_str()
            {
                "loading" => ListState::Loading,
                "failed" => ListState::Failed,
                _ => ListState::Ready,
            },
            conversation_scroll: gpui_kit::ScrollHandle::new(),
        };
        cx.set_reduce_motion(reduced_motion);
        app
    }

    fn ts(&self, size: f32) -> gpui_kit::Pixels {
        px(size * self.text_scale)
    }

    fn shortcut_keys(&self, index: usize) -> Vec<String> {
        let default = SHORTCUTS.get(index).map(|(_, keys)| *keys).unwrap_or(&[]);
        let custom = self
            .shortcut_overrides
            .get(index)
            .and_then(|value| value.as_deref());
        shortcut_labels(default, custom)
    }

    fn toast(&mut self, message: impl Into<String>, cx: &mut Context<Self>) {
        let id = self.toast_seq;
        self.toast_seq += 1;
        self.toasts.push(Toast {
            id,
            message: message.into(),
        });
        if self.toasts.len() > 3 {
            self.toasts.remove(0);
        }
        cx.spawn(async move |this, cx| {
            smol::Timer::after(Duration::from_millis(2400)).await;
            if let Some(this) = this.upgrade() {
                this.update(cx, |app, cx| {
                    app.toasts.retain(|toast| toast.id != id);
                    cx.notify();
                });
            }
        })
        .detach();
        cx.notify();
    }

    fn clickable(
        &self,
        cx: &mut Context<Self>,
        el: gpui_kit::Stateful<gpui_kit::Div>,
        action: Action,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let tab_index = if self.overlay.is_some() { -1 } else { 0 };
        self.clickable_with_tab_index(cx, el, action, tab_index)
    }

    fn clickable_modal(
        &self,
        cx: &mut Context<Self>,
        el: gpui_kit::Stateful<gpui_kit::Div>,
        action: Action,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        self.clickable_with_tab_index(cx, el, action, 0)
    }

    fn clickable_with_tab_index(
        &self,
        cx: &mut Context<Self>,
        el: gpui_kit::Stateful<gpui_kit::Div>,
        action: Action,
        tab_index: isize,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let click = action.clone();
        let key = action;
        let ring = cx.theme().ring;
        el.role(Role::Button)
            .tab_index(tab_index)
            .focus_visible(move |style| style.border_2().border_color(ring))
            .on_click(cx.listener(move |this, _, window, cx| {
                this.apply(click.clone(), window, cx);
            }))
            .on_key_down(cx.listener(move |this, event: &KeyDownEvent, window, cx| {
                if event.keystroke.key == "enter" || event.keystroke.key == "space" {
                    this.apply(key.clone(), window, cx);
                }
            }))
    }

    fn apply(&mut self, action: Action, window: &mut Window, cx: &mut Context<Self>) {
        match action {
            Action::Navigate(route) => {
                self.route = route;
                self.overlay = None;
                self.info_panel = None;
                self.status_viewer = None;
                self.selected_media = None;
                if route != Route::Chats {
                    self.searching = false;
                }
            }
            Action::SelectChat(index) => {
                if let Some(chat) = self.chats.get_mut(index) {
                    chat.unread = 0;
                }
                self.selected_chat = index;
                self.pinned = match index {
                    0 => vec![3],
                    1 => vec![1],
                    _ => Vec::new(),
                };
                self.route = Route::Chats;
                self.info_panel = None;
                self.overlay = None;
                self.status_viewer = None;
                self.reply_to = None;
                self.editing = None;
            }
            Action::SetFilter(filter) => self.chat_filter = filter,
            Action::SetTheme(mode) => {
                self.system_theme = false;
                self.theme = mode;
                crate::theme::apply_theme(cx, self.theme);
            }
            Action::SetSystemTheme => {
                self.system_theme = true;
                self.theme = if self.os_theme_dark {
                    ThemeMode::Dark
                } else {
                    ThemeMode::Light
                };
                crate::theme::apply_theme(cx, self.theme);
            }
            Action::SetOsTheme(dark) => {
                self.os_theme_dark = dark;
                if self.system_theme {
                    self.theme = if dark {
                        ThemeMode::Dark
                    } else {
                        ThemeMode::Light
                    };
                    crate::theme::apply_theme(cx, self.theme);
                }
            }
            Action::SetShell(shell) => {
                self.shell = shell;
                self.toast(
                    match shell {
                        ShellStyle::Windows => "Windows shell preview",
                        ShellStyle::MacOS => "macOS Liquid Glass shell preview",
                    },
                    cx,
                );
            }
            Action::OpenInfo(panel) => {
                self.info_panel = Some(panel);
                self.overlay = None;
            }
            Action::OpenContactProfile(name) => {
                self.overlay = Some(Overlay::ContactProfile(name));
            }
            Action::ToggleHeaderMenu => {
                self.overlay = if matches!(self.overlay, Some(Overlay::HeaderMenu)) {
                    None
                } else {
                    Some(Overlay::HeaderMenu)
                };
            }
            Action::OpenChatListMenu => self.overlay = Some(Overlay::ChatListMenu),
            Action::OpenUseCaseDemo => self.overlay = Some(Overlay::UseCaseDemo),
            Action::StartDemo(step) => {
                self.route = Route::Chats;
                self.selected_chat = 1;
                self.info_panel = None;
                self.searching = false;
                self.overlay = None;
                match step {
                    0 => {
                        self.overlay = None;
                        self.toast("Group chat · hover the title or click a sender name", cx);
                    }
                    1 => self.overlay = Some(Overlay::ContactProfile("Rina".into())),
                    2 => self.overlay = Some(Overlay::HeaderMenu),
                    3 => self.overlay = Some(Overlay::AttachmentMenu),
                    4 => self.overlay = Some(Overlay::ContextMenu(4)),
                    5 => {
                        self.overlay = None;
                        self.searching = true;
                    }
                    6 => self.info_panel = Some(InfoPanel::Group),
                    7 => self.overlay = Some(Overlay::NewGroup),
                    8 => self.overlay = Some(Overlay::ConfirmDelete(4)),
                    9 => self.overlay = Some(Overlay::EditContact(0)),
                    10 => self.overlay = Some(Overlay::ChatListMenu),
                    _ => {
                        self.theme = if self.theme == ThemeMode::Dark {
                            ThemeMode::Light
                        } else {
                            ThemeMode::Dark
                        };
                        crate::theme::apply_theme(cx, self.theme);
                    }
                }
            }
            Action::CloseInfo => self.info_panel = None,
            Action::CloseOverlay => {
                if self.overlay.take().is_none() {
                    if self.info_panel.take().is_none() {
                        self.searching = false;
                    }
                }
            }
            Action::CloseOverlayAndToast(message) => {
                self.overlay = None;
                self.toast(message, cx);
            }
            Action::CloseSubpageAndToast(message) => {
                self.settings_subpage = None;
                self.toast(message, cx);
            }
            Action::Toast(message) => self.toast(message, cx),
            Action::ShowComingSoon(message) => self.toast(message, cx),
            Action::Send => self.send_message(window, cx),
            Action::OpenAttachments => self.overlay = Some(Overlay::AttachmentMenu),
            Action::OpenContextMenu(index) => self.overlay = Some(Overlay::ContextMenu(index)),
            Action::Reply(index) => {
                self.reply_to = Some(index);
                self.overlay = None;
                self.composer
                    .update(cx, |state, cx| state.focus(window, cx));
            }
            Action::Edit(index) => {
                let text = self
                    .messages
                    .get(self.selected_chat)
                    .and_then(|messages| messages.get(index))
                    .and_then(Message::text)
                    .unwrap_or_default()
                    .to_owned();
                self.editing = Some(index);
                self.reply_to = None;
                self.overlay = None;
                self.composer
                    .update(cx, |state, cx| state.set_value(text, window, cx));
                self.composer
                    .update(cx, |state, cx| state.focus(window, cx));
            }
            Action::StarMessage(index) => {
                if let Some(message) = self
                    .messages
                    .get_mut(self.selected_chat)
                    .and_then(|messages| messages.get_mut(index))
                {
                    message.starred = !message.starred;
                    let starred = message.starred;
                    self.overlay = None;
                    self.toast(
                        if starred {
                            "Message starred"
                        } else {
                            "Removed from starred"
                        },
                        cx,
                    );
                }
            }
            Action::OpenPinnedMessages => self.overlay = Some(Overlay::PinnedMessages),
            Action::OpenStarredChat(chat_name) => {
                if let Some(index) = self.chats.iter().position(|chat| chat.name == chat_name) {
                    self.selected_chat = index;
                    self.route = Route::Chats;
                    self.overlay = None;
                    self.toast(format!("Opened {chat_name}"), cx);
                } else {
                    self.toast("Original chat is not in this fixture", cx);
                }
            }
            Action::UnarchiveAll => {
                self.archived.clear();
                self.toast("All chats unarchived · local mock", cx);
            }
            Action::JumpToMessage(index) => {
                self.overlay = None;
                self.conversation_scroll.scroll_to_item(index);
                self.toast(format!("Jumped to message {}", index + 1), cx);
            }
            Action::Delete(index) => self.overlay = Some(Overlay::ConfirmDelete(index)),
            Action::ConfirmDelete(index) => {
                if let Some(message) = self
                    .messages
                    .get_mut(self.selected_chat)
                    .and_then(|messages| messages.get_mut(index))
                {
                    message.body = MessageBody::Deleted;
                    message.reply = None;
                    message.reactions.clear();
                }
                self.overlay = None;
                self.toast("Message deleted locally", cx);
            }
            Action::React(index, emoji) => {
                if let Some(message) = self
                    .messages
                    .get_mut(self.selected_chat)
                    .and_then(|messages| messages.get_mut(index))
                {
                    if let Some(reaction) = message.reactions.iter_mut().find(|r| r.emoji == emoji)
                    {
                        reaction.count += 1;
                    } else {
                        message.reactions.push(Reaction { emoji, count: 1 });
                    }
                }
                self.overlay = None;
            }
            Action::StartCall(index, video) => {
                self.call_active = Some(index);
                self.call_video = video;
                self.call_muted = false;
                self.call_state = CallState::Ringing;
                self.route = Route::Calls;
                self.overlay = None;
                cx.spawn(async move |this, cx| {
                    smol::Timer::after(Duration::from_millis(1200)).await;
                    if let Some(this) = this.upgrade() {
                        this.update(cx, |app, cx| {
                            if app.call_active.is_some() {
                                app.call_state = CallState::Connected;
                                cx.notify();
                            }
                        });
                    }
                })
                .detach();
            }
            Action::EndCall => {
                self.call_active = None;
                self.call_state = CallState::EndedByThem;
                self.toast("Call ended · local mock", cx);
            }
            Action::SetCallState(state) => {
                self.call_active = Some(0);
                self.call_video = false;
                self.call_state = state;
                self.route = Route::Calls;
                self.overlay = None;
            }
            Action::PinCallParticipant(index) => {
                self.call_pinned = if self.call_pinned == Some(index) {
                    None
                } else {
                    Some(index)
                };
            }
            Action::ToggleMute => self.call_muted = !self.call_muted,
            Action::ToggleVideo => self.call_video = !self.call_video,
            Action::OpenStatus(index) => {
                if index == 0 {
                    self.overlay = Some(Overlay::StatusComposer);
                } else {
                    self.status_viewer = Some(index);
                    self.route = Route::Updates;
                }
            }
            Action::OpenStatusComposer => {
                self.route = Route::Updates;
                self.overlay = Some(Overlay::StatusComposer);
            }
            Action::CloseStatus => self.status_viewer = None,
            Action::ToggleStatusPause => self.status_paused = !self.status_paused,
            Action::SetMediaTab(tab) => self.media_tab = tab,
            Action::OpenMedia(index) => {
                self.selected_media = Some(index);
                self.overlay = Some(Overlay::Lightbox(index));
            }
            Action::CloseMedia => {
                self.selected_media = None;
                self.overlay = None;
            }
            Action::SelectMediaChat(chat) => {
                self.selected_media_chat = if chat.is_empty() { None } else { Some(chat) };
                self.media_tab = MediaTab::Media;
                self.route = Route::Media;
            }
            Action::MediaPrev | Action::MediaNext => {
                let visible = self
                    .media
                    .iter()
                    .filter(|item| !self.media_deleted.contains(&item.id))
                    .map(|item| item.id)
                    .collect::<Vec<_>>();
                if !visible.is_empty() {
                    let current = self
                        .selected_media
                        .and_then(|id| visible.iter().position(|item| *item == id))
                        .unwrap_or(0);
                    let next = match action {
                        Action::MediaNext => (current + 1) % visible.len(),
                        _ => (current + visible.len() - 1) % visible.len(),
                    };
                    let id = visible[next];
                    self.selected_media = Some(id);
                    self.overlay = Some(Overlay::Lightbox(id));
                }
            }
            Action::ZoomIn => self.media_zoom = (self.media_zoom + 25).min(150),
            Action::ZoomOut => self.media_zoom = self.media_zoom.saturating_sub(25).max(50),
            Action::ToggleMediaSelectMode => {
                self.media_select_mode = !self.media_select_mode;
                self.media_selected.clear();
            }
            Action::ToggleMediaSelection(index) => {
                if !self.media_selected.remove(&index) {
                    self.media_selected.insert(index);
                }
            }
            Action::MediaStar(index) => {
                if !self.media_starred.remove(&index) {
                    self.media_starred.insert(index);
                    self.toast("Starred · saved to Starred messages", cx);
                } else {
                    self.toast("Removed from starred", cx);
                }
            }
            Action::MediaDownload(index) => {
                let name = self
                    .media
                    .get(index)
                    .map(|item| item.name.clone())
                    .unwrap_or_default();
                self.toast(format!("Saved to Downloads · {name}"), cx);
            }
            Action::MediaForward(index) => {
                let name = self
                    .media
                    .get(index)
                    .map(|item| item.name.clone())
                    .unwrap_or_default();
                self.toast(format!("Forwarded · {name}"), cx);
            }
            Action::MediaDelete(index) => {
                self.media_deleted.insert(index);
                self.media_selected.remove(&index);
                if self.selected_media == Some(index) {
                    self.selected_media = None;
                    self.overlay = None;
                }
                self.toast("Deleted from this chat · local only", cx);
            }
            Action::MediaStarSelected => {
                for index in self.media_selected.clone() {
                    self.media_starred.insert(index);
                }
                self.toast(format!("Starred {} items", self.media_selected.len()), cx);
            }
            Action::MediaForwardSelected => {
                self.toast(format!("Forwarded {} items", self.media_selected.len()), cx);
            }
            Action::MediaDeleteSelected => {
                let count = self.media_selected.len();
                for index in self.media_selected.clone() {
                    self.media_deleted.insert(index);
                }
                self.media_selected.clear();
                self.toast(format!("Deleted {count} items · local only"), cx);
            }
            Action::SetTextScale(scale) => {
                self.text_scale = scale;
                crate::theme::apply_theme_scaled(cx, self.theme, scale);
            }
            Action::SetRtl(rtl) => self.rtl = rtl,
            Action::SetReducedMotion(enabled) => {
                self.reduced_motion = enabled;
                if enabled {
                    self.toggles.insert("reduce_motion");
                } else {
                    self.toggles.remove("reduce_motion");
                }
                cx.set_reduce_motion(enabled);
            }
            Action::SettingsPage(page) => {
                self.route = Route::Settings;
                self.settings_page = page;
                self.settings_subpage = None;
                self.overlay = None;
            }
            Action::OpenSubpage(page) => {
                self.route = Route::Settings;
                self.settings_subpage = Some(page);
                self.overlay = None;
            }
            Action::CloseSubpage => self.settings_subpage = None,
            Action::ToggleSetting(key) => {
                if !self.toggles.remove(key) {
                    self.toggles.insert(key);
                }
                if key == "reduce_motion" {
                    cx.set_reduce_motion(self.toggles.contains(key));
                }
                if key == "system_theme" {
                    self.system_theme = self.toggles.contains(key);
                    if self.system_theme {
                        self.theme = if self.os_theme_dark {
                            ThemeMode::Dark
                        } else {
                            ThemeMode::Light
                        };
                        crate::theme::apply_theme(cx, self.theme);
                    }
                }
            }
            Action::NewChat => self.overlay = Some(Overlay::NewChat),
            Action::NewGroup => self.overlay = Some(Overlay::NewGroup),
            Action::CreateGroup => {
                let index = self.chats.len();
                let mut chat = self
                    .chats
                    .get(1)
                    .cloned()
                    .unwrap_or_else(|| self.chats[0].clone());
                chat.id = index;
                chat.name = "New group".into();
                chat.initials = "NG".into();
                chat.preview = "You created this group".into();
                chat.time = "Now".into();
                chat.unread = 0;
                chat.is_group = true;
                chat.archived = false;
                chat.favorite = false;
                chat.pinned = false;
                chat.draft = None;
                chat.typing = None;
                chat.verified = false;
                chat.business = false;
                self.chats.push(chat);
                self.messages.push(Vec::new());
                self.selected_chat = index;
                self.overlay = None;
                self.route = Route::Chats;
                self.toast("Group created · local mock", cx);
            }
            Action::NewBroadcast => {
                self.broadcast_step = 1;
                self.overlay = Some(Overlay::NewBroadcast);
            }
            Action::NewCommunity => self.overlay = Some(Overlay::NewCommunity),
            Action::CreateCommunity => {
                let index = self.communities.len();
                self.communities.push(Community {
                    id: index,
                    name: "Kain Nusantara".into(),
                    initials: "KN".into(),
                    members: "1 member".into(),
                    groups: "1 group".into(),
                    description: "Textiles and studios".into(),
                    joined: true,
                    unread: 0,
                });
                self.selected_community = Some(index);
                self.overlay = None;
                self.route = Route::Communities;
                self.toast("Community created · local mock", cx);
            }
            Action::ClearDraft => {
                if let Some(chat) = self.chats.get_mut(self.selected_chat) {
                    chat.draft = None;
                }
            }
            Action::DismissMissedCall => self.missed_banner = false,
            Action::UnblockChat(index) => {
                self.blocked_chats.remove(&index);
                self.toast("Contact unblocked · local mock", cx);
            }
            Action::FlushOutbox => {
                self.outbox = 0;
                for message in self.messages.iter_mut().flatten() {
                    if message.delivery == Delivery::Sending {
                        message.delivery = Delivery::Sent;
                    }
                }
                self.toast("Outbox flushed · local mock", cx);
            }
            Action::ClearMediaCache => {
                self.media_cache_mb = 0;
                self.toast("Media cache cleared · local mock", cx);
            }
            Action::SetNetworkState(state) => {
                self.network_state = state;
                self.toast(
                    match state {
                        NetworkState::Online => "Back online",
                        NetworkState::Connecting => "Connecting…",
                        NetworkState::Offline => "You are offline — messages will queue",
                        NetworkState::Reconnecting => "Reconnecting to WhatsApp…",
                    },
                    cx,
                );
            }
            Action::SelectBroadcast(index) => {
                self.selected_broadcast = Some(index);
                self.route = Route::Broadcast;
                self.overlay = None;
            }
            Action::EditContact(index) => self.overlay = Some(Overlay::EditContact(index)),
            Action::OpenAccounts => self.overlay = Some(Overlay::Accounts),
            Action::SwitchAccount(index) => {
                for (account_index, account) in self.accounts.iter_mut().enumerate() {
                    account.active = account_index == index;
                }
                self.active_account = index;
                self.overlay = None;
                if let Some(account) = self.accounts.get(index) {
                    self.toast(
                        format!("Switched to {} · {}", account.name, account.label),
                        cx,
                    );
                }
            }
            Action::AddAccount => self.overlay = Some(Overlay::AddAccount),
            Action::BroadcastNextStep => {
                self.broadcast_step = 2;
                self.overlay = Some(Overlay::NewBroadcast);
            }
            Action::BroadcastBack => {
                self.broadcast_step = 1;
                self.overlay = Some(Overlay::NewBroadcast);
            }
            Action::BroadcastCreate => {
                let count = self.broadcast_recipients.len();
                if count == 0 {
                    self.toast("Select at least one recipient", cx);
                } else {
                    let id = self.broadcasts.len();
                    self.broadcasts.push(Broadcast {
                        id,
                        name: "Resellers".into(),
                        recipients: format!("{count} recipients"),
                        time: "Now".into(),
                        preview: "New broadcast list created locally.".into(),
                    });
                    self.broadcast_step = 1;
                    self.overlay = None;
                    self.route = Route::Broadcast;
                    self.selected_broadcast = Some(id);
                    self.toast(format!("Broadcast list created · {count} recipients"), cx);
                }
            }
            Action::ToggleBroadcastRecipient(index) => {
                if !self.broadcast_recipients.remove(&index) {
                    self.broadcast_recipients.insert(index);
                }
            }
            Action::OpenSystemState(state) => self.overlay = Some(Overlay::SystemState(state)),
            Action::ToggleAiTranslation => {
                self.ai_draft_english = !self.ai_draft_english;
                self.toast(
                    if self.ai_draft_english {
                        "Draft translated to English"
                    } else {
                        "Showing original draft"
                    },
                    cx,
                );
            }
            Action::InsertAiDraft => {
                let draft = "Siap, aku kirim deck final jam tiga ya. Kalau ada revisi kecil, tulis di kolom komentar.";
                self.composer
                    .update(cx, |state, cx| state.set_value(draft, window, cx));
                self.overlay = None;
                self.route = Route::Chats;
                self.toast("Draft inserted into the composer", cx);
            }
            Action::AiSuggest => {
                self.ai_state = AiState::Suggested;
                self.route = Route::Ai;
                self.overlay = None;
            }
            Action::AiRegenerate | Action::AiRetry => {
                self.ai_state = AiState::Streaming;
                self.route = Route::Ai;
                self.overlay = None;
                cx.spawn(async move |this, cx| {
                    smol::Timer::after(Duration::from_millis(1100)).await;
                    if let Some(this) = this.upgrade() {
                        this.update(cx, |app, cx| {
                            if app.ai_state == AiState::Streaming {
                                app.ai_state = AiState::Suggested;
                                cx.notify();
                            }
                        });
                    }
                })
                .detach();
            }
            Action::AiPrompt => {
                self.ai_state = AiState::Streaming;
                self.route = Route::Ai;
                self.overlay = None;
                cx.spawn(async move |this, cx| {
                    smol::Timer::after(Duration::from_millis(1100)).await;
                    if let Some(this) = this.upgrade() {
                        this.update(cx, |app, cx| {
                            if app.ai_state == AiState::Streaming {
                                app.ai_state = AiState::Error;
                                cx.notify();
                            }
                        });
                    }
                })
                .detach();
            }
            Action::AiCancel | Action::AiDiscard => {
                self.ai_state = AiState::Idle;
                self.overlay = None;
            }
            Action::InsertAiSuggestion => {
                let draft = "Siap, aku kirim deck final jam tiga ya. Kalau ada revisi kecil, tulis di kolom komentar.";
                self.composer
                    .update(cx, |state, cx| state.set_value(draft, window, cx));
                self.ai_state = AiState::Idle;
                self.route = Route::Chats;
                self.toast("Suggested reply inserted into the composer", cx);
            }
            Action::OpenShortcutEditor(index) => {
                self.overlay = Some(Overlay::ShortcutEditor(index))
            }
            Action::ApplyShortcut(index, keys) => {
                if let Some(slot) = self.shortcut_overrides.get_mut(index) {
                    *slot = Some(keys.clone());
                }
                self.overlay = None;
                self.toast(format!("Shortcut updated · {keys}"), cx);
            }
            Action::ResetShortcuts => {
                for slot in &mut self.shortcut_overrides {
                    *slot = None;
                }
                self.toast("Shortcuts reset to defaults", cx);
            }
            Action::ToggleStatusItem(media) => {
                if !toggle_status_item(&mut self.status_items, media) {
                    self.toast("Status limit reached · 30 items maximum", cx);
                }
            }
            Action::AddStatusItem => {
                if self.status_items.len() >= STATUS_ITEM_LIMIT {
                    self.toast("Status limit reached · remove one to add another", cx);
                } else {
                    let next = (0..12)
                        .find(|index| !self.status_items.contains(index))
                        .unwrap_or(self.status_items.len() % 12);
                    self.status_items.push(next);
                }
            }
            Action::RemoveStatusItem(index) => {
                if index < self.status_items.len() {
                    self.status_items.remove(index);
                }
            }
            Action::SendStatus => {
                let count = self.status_items.len();
                if count > 0 {
                    self.overlay = None;
                    self.toast(
                        format!("Status posted · {count} items · expires in 24 hours"),
                        cx,
                    );
                }
            }
            Action::OpenStatusFont => self.overlay = Some(Overlay::StatusFont),
            Action::SetStatusFont(index) => self.status_font = index.min(2),
            Action::SetStatusFontSize(index) => self.status_font_size = index.min(2),
            Action::OpenStatusPrivacy => self.overlay = Some(Overlay::StatusPrivacy),
            Action::SetStatusPrivacy(index) => self.status_privacy = index.min(3),
            Action::StageAttachment(kind, label) => {
                if self.staged_attachments.len() < 5 {
                    self.staged_attachments
                        .push(StagedAttachment { kind, label });
                }
                self.overlay = None;
                self.toast("Attachment staged · not sent yet", cx);
            }
            Action::RemoveStagedAttachment(index) => {
                if index < self.staged_attachments.len() {
                    self.staged_attachments.remove(index);
                }
            }
            Action::SendStaged => {
                let staged = std::mem::take(&mut self.staged_attachments);
                let text = self.composer.read(cx).value().trim().to_owned();
                let chat = self.selected_chat;
                if !text.is_empty() {
                    let id = self.messages[chat].len();
                    self.messages[chat].push(Message {
                        id,
                        sender: Sender::Me,
                        sender_name: None,
                        body: MessageBody::Text(text),
                        time: "now".into(),
                        delivery: Delivery::Sent,
                        edited: false,
                        reply: None,
                        reactions: Vec::new(),
                        starred: false,
                        countdown: None,
                        forwarded: 0,
                    });
                }
                for attachment in staged {
                    let id = self.messages[chat].len();
                    let body = match attachment.kind {
                        StagedKind::Image => MessageBody::Image {
                            caption: Some(attachment.label.clone()),
                            state: MediaState::Ready,
                            ratio: 1.0,
                        },
                        StagedKind::File => MessageBody::Document {
                            name: attachment.label.clone(),
                            meta: "PDF · 1.2 MB".into(),
                        },
                        StagedKind::Contact => MessageBody::ContactCard {
                            name: attachment.label.clone(),
                            phone: "+62 812 3456 7890".into(),
                        },
                    };
                    self.messages[chat].push(Message {
                        id,
                        sender: Sender::Me,
                        sender_name: None,
                        body,
                        time: "now".into(),
                        delivery: Delivery::Sent,
                        edited: false,
                        reply: None,
                        reactions: Vec::new(),
                        starred: false,
                        countdown: None,
                        forwarded: 0,
                    });
                }
                self.composer
                    .update(cx, |state, cx| state.set_value("", window, cx));
                self.toast("Attachment sent · local mock", cx);
            }
            Action::OpenPollComposer => self.overlay = Some(Overlay::PollComposer),
            Action::OpenEventComposer => self.overlay = Some(Overlay::EventComposer),
            Action::CreatePoll => {
                let chat = self.selected_chat;
                let id = self.messages[chat].len();
                self.messages[chat].push(Message {
                    id,
                    sender: Sender::Me,
                    sender_name: None,
                    body: MessageBody::Poll {
                        question: "Warna mana yang paling cocok?".into(),
                        options: vec![
                            ("Linen natural".into(), 0),
                            ("Hitam arang".into(), 0),
                            ("Coral".into(), 0),
                        ],
                        state: crate::model::PollState::Open,
                        multiple: false,
                    },
                    time: "now".into(),
                    delivery: Delivery::Sent,
                    edited: false,
                    reply: None,
                    reactions: Vec::new(),
                    starred: false,
                    countdown: None,
                    forwarded: 0,
                });
                self.overlay = None;
                self.toast("Poll created · local mock", cx);
            }
            Action::OpenDisappearingPicker => self.overlay = Some(Overlay::DisappearingPicker),
            Action::SetDisappearing(index) => {
                self.disappearing = index.min(3);
                self.overlay = None;
                self.toast(
                    format!(
                        "Disappearing messages · {}",
                        ["24 hours", "7 days", "90 days", "Off"][self.disappearing]
                    ),
                    cx,
                );
            }
            Action::OpenStickerPicker => self.overlay = Some(Overlay::StickerPicker),
            Action::SendSticker(symbol) => {
                let chat = self.selected_chat;
                let id = self.messages[chat].len();
                self.messages[chat].push(Message {
                    id,
                    sender: Sender::Me,
                    sender_name: None,
                    body: MessageBody::Sticker {
                        symbol,
                        pack_missing: false,
                    },
                    time: "now".into(),
                    delivery: Delivery::Sent,
                    edited: false,
                    reply: None,
                    reactions: Vec::new(),
                    starred: false,
                    countdown: None,
                    forwarded: 0,
                });
                self.overlay = None;
                self.toast("Sticker sent · local mock", cx);
            }
            Action::OpenFileDialog => {
                self.file_selected = false;
                self.overlay = Some(Overlay::FileDialog);
            }
            Action::SelectFile => self.file_selected = true,
            Action::OpenGroupMembers => self.overlay = Some(Overlay::GroupMembers),
            Action::OpenClearChat => self.overlay = Some(Overlay::ClearChat),
            Action::ConfirmClearChat => {
                self.messages[self.selected_chat].clear();
                self.overlay = None;
                self.toast("Chat cleared for you · local mock", cx);
            }
            Action::OpenExportChat => self.overlay = Some(Overlay::ExportChat),
            Action::OpenSecurityCode => self.overlay = Some(Overlay::SecurityCode),
            Action::OpenChannelInfo(name) => self.overlay = Some(Overlay::ChannelInfo(name)),
            Action::OpenCommunityMembers => self.overlay = Some(Overlay::CommunityMembers),
            Action::OpenMemberApprovals => self.overlay = Some(Overlay::MemberApprovals),
            Action::ToggleChannelFollow => {
                self.channel_following = !self.channel_following;
                self.toast(
                    if self.channel_following {
                        "Following channel"
                    } else {
                        "Channel unfollowed"
                    },
                    cx,
                );
            }
            Action::OpenStatusViewers => self.overlay = Some(Overlay::StatusViewers),
            Action::OpenForward(index) => self.overlay = Some(Overlay::Forward(index)),
            Action::ForwardTo(message_index, chat_index) => {
                if let Some(message) = self
                    .messages
                    .get(self.selected_chat)
                    .and_then(|messages| messages.get(message_index))
                    .cloned()
                {
                    if let Some(chat) = self.chats.get(chat_index) {
                        let chat_name = chat.name.clone();
                        let id = self.messages[chat_index].len();
                        let mut forwarded = message;
                        forwarded.id = id;
                        forwarded.sender = Sender::Me;
                        forwarded.sender_name = None;
                        forwarded.time = "now".into();
                        forwarded.delivery = Delivery::Sent;
                        forwarded.edited = false;
                        forwarded.reply = None;
                        forwarded.reactions.clear();
                        forwarded.forwarded += 1;
                        self.messages[chat_index].push(forwarded);
                        self.overlay = None;
                        self.toast(format!("Forwarded to {chat_name}"), cx);
                    }
                }
            }
            Action::OpenPollVotes => self.overlay = Some(Overlay::PollVotes),
            Action::OpenReactions(index) => self.overlay = Some(Overlay::Reactions(index)),
            Action::EnterMessageSelection(index) => {
                self.message_select_mode = true;
                self.selected_messages.insert(index);
                self.overlay = None;
            }
            Action::ToggleMessageSelection(index) => {
                if !self.selected_messages.remove(&index) {
                    self.selected_messages.insert(index);
                }
            }
            Action::ExitMessageSelection => {
                self.message_select_mode = false;
                self.selected_messages.clear();
            }
            Action::SelectAllMessages => {
                if let Some(messages) = self.messages.get(self.selected_chat) {
                    self.selected_messages = (0..messages.len()).collect();
                }
            }
            Action::SelectionStar => {
                let selected = self.selected_messages.clone();
                for index in &selected {
                    if let Some(message) = self
                        .messages
                        .get_mut(self.selected_chat)
                        .and_then(|messages| messages.get_mut(*index))
                    {
                        message.starred = true;
                    }
                }
                self.toast(format!("Starred {} messages", selected.len()), cx);
            }
            Action::SelectionDelete => {
                let selected = self.selected_messages.clone();
                for index in &selected {
                    if let Some(message) = self
                        .messages
                        .get_mut(self.selected_chat)
                        .and_then(|messages| messages.get_mut(*index))
                    {
                        message.body = MessageBody::Deleted;
                        message.reply = None;
                        message.reactions.clear();
                    }
                }
                self.toast(
                    format!("Deleted {} messages · local mock", selected.len()),
                    cx,
                );
                self.message_select_mode = false;
                self.selected_messages.clear();
            }
            Action::SelectionForward => {
                if let Some(index) = self.selected_messages.iter().min().copied() {
                    self.overlay = Some(Overlay::Forward(index));
                }
            }
            Action::SelectionCopy => self.toast(
                format!("Copied {} messages", self.selected_messages.len()),
                cx,
            ),
            Action::OpenGifPicker => self.overlay = Some(Overlay::GifPicker),
            Action::SetGifQuery(index) => self.gif_query = index.min(1),
            Action::SendGif(caption) => {
                let chat = self.selected_chat;
                let id = self.messages[chat].len();
                self.messages[chat].push(Message {
                    id,
                    sender: Sender::Me,
                    sender_name: None,
                    body: MessageBody::Gif {
                        caption: Some(caption),
                        failed: false,
                    },
                    time: "now".into(),
                    delivery: Delivery::Sent,
                    edited: false,
                    reply: None,
                    reactions: Vec::new(),
                    starred: false,
                    countdown: None,
                    forwarded: 0,
                });
                self.overlay = None;
                self.toast("GIF sent · local mock", cx);
            }
            Action::SelectionInfo => {
                if self.selected_messages.len() == 1 {
                    self.toast("Message info opened · local mock", cx);
                } else {
                    self.toast("Select one message to see info", cx);
                }
            }
            Action::AcceptCall => {
                self.call_incoming = false;
                self.call_state = CallState::Connected;
                self.toast("Call accepted · local mock", cx);
            }
            Action::OpenLockedUnlock => self.overlay = Some(Overlay::LockedUnlock),
            Action::SetGroupCallSize(size) => {
                self.group_call_size = size.clamp(2, 6);
                self.call_active = Some(1);
                self.call_video = false;
                self.call_state = CallState::Connected;
                self.route = Route::Calls;
                self.overlay = None;
            }
            Action::SendAnnouncement => self.toast("Announcement sent · local mock", cx),
            Action::DeclineCall => {
                self.call_active = None;
                self.call_incoming = false;
                self.call_state = CallState::Declined;
                self.toast("Call declined · local mock", cx);
            }
            Action::ApproveMember(index) => {
                if index < self.member_requests.len() {
                    let name = self.member_requests.remove(index);
                    self.toast(format!("Approved {name} · local mock"), cx);
                }
            }
            Action::AttachSelectedFile => {
                self.staged_attachments.push(StagedAttachment {
                    kind: StagedKind::File,
                    label: "Invoice-2026-09.pdf".into(),
                });
                self.file_selected = false;
                self.overlay = None;
                self.toast("File attached · not sent yet", cx);
            }
            Action::CreateEvent => {
                let chat = self.selected_chat;
                let id = self.messages[chat].len();
                self.messages[chat].push(Message {
                    id,
                    sender: Sender::Me,
                    sender_name: None,
                    body: MessageBody::Event {
                        title: "Review koleksi September".into(),
                        time: "Fri, 26 Sep · 15:00".into(),
                        location: "Studio Kemang".into(),
                        state: crate::model::EventState::Invited,
                    },
                    time: "now".into(),
                    delivery: Delivery::Sent,
                    edited: false,
                    reply: None,
                    reactions: Vec::new(),
                    starred: false,
                    countdown: None,
                    forwarded: 0,
                });
                self.overlay = None;
                self.toast("Event created · local mock", cx);
            }
            Action::ProfilePhotoUpload => self.profile_photo = ProfilePhotoState::Cropping,
            Action::ProfilePhotoCropCancel => self.profile_photo = ProfilePhotoState::Empty,
            Action::ProfilePhotoCropConfirm => self.profile_photo = ProfilePhotoState::Uploaded,
            Action::ProfilePhotoSave => {
                self.profile_photo = ProfilePhotoState::Saved;
                self.toast("Profile photo saved", cx);
            }
            Action::ProfilePhotoRemove => {
                self.profile_photo = ProfilePhotoState::Empty;
                self.toast("Profile photo removed", cx);
            }
            Action::ProfilePhotoZoomIn => {
                self.profile_photo_zoom = (self.profile_photo_zoom + 10).min(200)
            }
            Action::ProfilePhotoZoomOut => {
                self.profile_photo_zoom = self.profile_photo_zoom.saturating_sub(10).max(50)
            }
            Action::SelectContact(index) => {
                self.selected_contact = Some(index);
                self.route = Route::Contacts;
            }
            Action::SelectCommunity(index) => {
                self.selected_community = Some(index);
                self.route = Route::Communities;
            }
            Action::CycleQr => {
                self.qr_state = match self.qr_state {
                    QrState::Loading => QrState::Active,
                    QrState::Active => QrState::Expired,
                    QrState::Expired => QrState::Success,
                    QrState::Success => QrState::Loading,
                };
            }
            Action::ToggleSearch => {
                self.searching = !self.searching;
                self.search_hit = 0;
                if self.searching {
                    self.chat_search
                        .update(cx, |state, cx| state.focus(window, cx));
                }
            }
            Action::SetChatSearchFilter(filter) => {
                self.chat_search_filter = filter;
                self.search_hit = 0;
                let query = self.chat_search.read(cx).value().to_lowercase();
                let first = self.messages.get(self.selected_chat).and_then(|messages| {
                    messages
                        .iter()
                        .position(|message| message_matches_search(message, &query, filter))
                });
                if let Some(first) = first {
                    self.conversation_scroll.scroll_to_item(first);
                }
            }
            Action::SearchNext | Action::SearchPrev => {
                let query = self.chat_search.read(cx).value().to_lowercase();
                let matches: Vec<usize> = self
                    .messages
                    .get(self.selected_chat)
                    .map(|messages| {
                        messages
                            .iter()
                            .enumerate()
                            .filter(|(_, message)| {
                                message_matches_search(message, &query, self.chat_search_filter)
                            })
                            .map(|(index, _)| index)
                            .collect()
                    })
                    .unwrap_or_default();
                if !matches.is_empty() {
                    let step = matches.len();
                    let current = self.search_hit.min(step - 1);
                    let next = match action {
                        Action::SearchNext => (current + 1) % step,
                        _ => (current + step - 1) % step,
                    };
                    self.search_hit = next;
                    self.conversation_scroll.scroll_to_item(matches[next]);
                }
            }
            Action::ScrollToBottom => {
                let last = self
                    .messages
                    .get(self.selected_chat)
                    .map(|messages| messages.len().saturating_sub(1))
                    .unwrap_or(0);
                self.conversation_scroll.scroll_to_item(last);
            }
            Action::ToggleTranscript => self.transcript_open = !self.transcript_open,
            Action::ToggleRecording => {
                if self.recording {
                    self.recording = false;
                    let duration = format!("0:{:02}", self.recording_seconds.max(1));
                    let id = self.messages[self.selected_chat].len();
                    let time = format!("{:02}:{:02}", 9, 20 + self.recording_seconds.min(39));
                    self.messages[self.selected_chat].push(Message {
                        id,
                        sender: Sender::Me,
                        sender_name: None,
                        body: MessageBody::Voice {
                            duration,
                            transcript: None,
                        },
                        time,
                        delivery: Delivery::Sent,
                        edited: false,
                        reply: None,
                        reactions: Vec::new(),
                        starred: false,
                        countdown: None,
                        forwarded: 0,
                    });
                    self.recording_seconds = 0;
                    self.toast("Voice message sent · local mock", cx);
                } else {
                    self.recording = true;
                    self.recording_seconds = 0;
                    cx.spawn(async move |this, cx| {
                        loop {
                            smol::Timer::after(Duration::from_secs(1)).await;
                            let Some(this) = this.upgrade() else { break };
                            let mut stop = false;
                            this.update(cx, |app, cx| {
                                if app.recording {
                                    app.recording_seconds += 1;
                                    cx.notify();
                                } else {
                                    stop = true;
                                }
                            });
                            if stop {
                                break;
                            }
                        }
                    })
                    .detach();
                }
            }
            Action::CycleVoiceSpeed => self.voice_speed = (self.voice_speed + 1) % 3,
            Action::AppendEmoji(emoji) => {
                let current = self.composer.read(cx).value().to_string();
                let next = if current.is_empty() {
                    emoji
                } else {
                    format!("{current} {emoji}")
                };
                self.composer
                    .update(cx, |state, cx| state.set_value(next, window, cx));
                self.overlay = None;
            }
            Action::OpenEmoji => self.overlay = Some(Overlay::EmojiPicker),
            Action::SetListState(state) => self.list_state = state,
        }
        cx.notify();
    }

    fn send_message(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let text = self.composer.read(cx).value().to_string();
        if text.trim().is_empty() {
            return;
        }
        let chat = self.selected_chat;
        if let Some(edit_index) = self.editing.take() {
            if let Some(message) = self
                .messages
                .get_mut(chat)
                .and_then(|messages| messages.get_mut(edit_index))
            {
                message.body = MessageBody::Text(text);
                message.edited = true;
                message.time = "now".into();
            }
            self.composer
                .update(cx, |state, cx| state.set_value("", window, cx));
            cx.notify();
            return;
        }

        let reply = self.reply_to.take().and_then(|index| {
            self.messages
                .get(chat)
                .and_then(|messages| messages.get(index))
                .map(|message| model::ReplyPreview {
                    sender: "You".into(),
                    text: message_preview(message).chars().take(120).collect(),
                    deleted: matches!(message.body, MessageBody::Deleted),
                    message_id: Some(index),
                })
        });

        let index = self.messages[chat].len();
        let delivery = if self.network_state == NetworkState::Online {
            Delivery::Sent
        } else {
            Delivery::Sending
        };
        self.messages[chat].push(Message {
            id: index,
            sender: Sender::Me,
            sender_name: None,
            body: MessageBody::Text(text),
            time: "now".into(),
            delivery,
            edited: false,
            reply,
            reactions: Vec::new(),
            starred: false,
            countdown: None,
            forwarded: 0,
        });
        self.composer
            .update(cx, |state, cx| state.set_value("", window, cx));
        if self.network_state != NetworkState::Online {
            self.outbox += 1;
        }
        if self.network_state == NetworkState::Offline {
            self.toast("No internet — message queued in the outbox", cx);
        }
        if let Some(chat) = self.chats.get_mut(chat) {
            chat.draft = None;
        }
        cx.notify();
    }

    fn selected_chat(&self) -> Option<&ChatModel> {
        self.chats.get(self.selected_chat)
    }

    fn filtered_chats(&self, query: &str) -> Vec<&ChatModel> {
        let query = query.trim().to_lowercase();
        self.chats
            .iter()
            .filter(|chat| !chat.archived)
            .filter(|chat| {
                query.is_empty()
                    || chat.name.to_lowercase().contains(&query)
                    || chat.preview.to_lowercase().contains(&query)
            })
            .filter(|chat| match self.chat_filter {
                ChatFilter::All => true,
                ChatFilter::Unread => chat.is_unread(),
                ChatFilter::Favorites => chat.favorite,
                ChatFilter::Groups => chat.is_group,
            })
            .collect()
    }

    fn render_root(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let p = palette(self.theme);
        let width = window.viewport_size().width;
        let compact = width < px(COMPACT_BREAKPOINT);
        self.compact = compact;
        let status_view = self
            .status_viewer
            .and_then(|index| self.statuses.get(index).cloned());
        let body = if let Some(status) = &status_view {
            self.render_status_viewer(cx, p, status).into_any_element()
        } else if self.route == Route::Onboarding {
            self.render_onboarding(cx, p).into_any_element()
        } else {
            self.render_shell(cx, p, compact, width).into_any_element()
        };

        div()
            .id("app-root")
            .size_full()
            .relative()
            .flex()
            .flex_col()
            .bg(p.bg_conversation)
            .text_color(p.text_primary)
            .font_family("Inter")
            .when(status_view.is_none(), |el| {
                el.child(self.render_titlebar(cx, p))
            })
            .child(body)
            .when_some(self.overlay.clone(), |el, overlay| {
                el.child(self.render_overlay(overlay, cx, p))
            })
            .when(!self.toasts.is_empty(), |el| {
                el.child(self.render_toasts(p))
            })
            .on_key_down(cx.listener(|this, event: &KeyDownEvent, window, cx| {
                if event.keystroke.key == "escape" {
                    this.apply(Action::CloseOverlay, window, cx);
                } else if event.keystroke.modifiers.platform && event.keystroke.key == "k" {
                    this.route = Route::Chats;
                    this.search.update(cx, |state, cx| state.focus(window, cx));
                    cx.notify();
                } else if event.keystroke.modifiers.platform && event.keystroke.key == "n" {
                    this.overlay = Some(Overlay::NewChat);
                    cx.notify();
                }
            }))
    }

    fn render_titlebar(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        if self.shell == ShellStyle::MacOS {
            return div().h(px(0.));
        }
        let brand = div()
            .flex()
            .items_center()
            .gap(px(8.))
            .child(app_mark(p))
            .child(
                div()
                    .text_size(px(12.))
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(p.text_primary)
                    .child("whatslite"),
            );

        let left = brand;

        div()
            .h(px(TITLEBAR_HEIGHT))
            .flex_shrink_0()
            .flex()
            .items_center()
            .justify_between()
            .px(px(12.))
            .bg(p.bg_sidebar)
            .border_b_1()
            .border_color(p.divider)
            .child(left)
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(2.))
                    .child(window_control(
                        cx,
                        IconName::Minus,
                        "Minimize",
                        p,
                        |window| window.minimize_window(),
                    ))
                    .child(window_control(
                        cx,
                        IconName::Square,
                        "Maximize",
                        p,
                        |window| window.zoom_window(),
                    ))
                    .child(window_control(cx, IconName::X, "Close", p, |window| {
                        window.remove_window()
                    })),
            )
    }

    fn render_shell(
        &mut self,
        cx: &mut Context<Self>,
        p: Palette,
        compact: bool,
        width: gpui_kit::Pixels,
    ) -> gpui_kit::Div {
        let sidebar = if self.shell == ShellStyle::MacOS {
            let collapsed = self.collapsed || compact;
            let width = if collapsed { RAIL_WIDTH } else { 192. };
            div()
                .w(px(width + 16.))
                .h_full()
                .flex_shrink_0()
                .p(px(8.))
                .child(self.render_sidebar(cx, p, compact))
                .into_any_element()
        } else {
            self.render_sidebar(cx, p, compact).into_any_element()
        };
        let content = match self.route {
            Route::Chats => {
                let list = self.render_chat_list(cx, p);
                let conversation = self.render_conversation(cx, p);
                let mut row = div().flex_1().flex().min_w(px(0.)).child(list);
                if let Some(panel) = self.info_panel {
                    if width >= px(PANEL_INLINE_BREAKPOINT) {
                        row = row
                            .child(conversation)
                            .child(self.render_info_panel(panel, cx, p));
                    } else {
                        row = row.child(conversation);
                    }
                } else {
                    row = row.child(conversation);
                }
                row.into_any_element()
            }
            Route::Settings => {
                let nav = self.render_settings_nav(cx, p);
                let detail = self.render_settings_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(nav)
                    .child(detail)
                    .into_any_element()
            }
            Route::Calls => {
                let list = self.render_calls_list(cx, p);
                let detail = self.render_calls_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Updates => {
                let list = self.render_updates_list(cx, p);
                let detail = self.render_updates_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Communities => {
                let list = self.render_communities_list(cx, p);
                let detail = self.render_communities_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Media => {
                let list = self.render_media_list(cx, p);
                let detail = self.render_media_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Ai => {
                let list = self.render_ai_list(cx, p);
                let detail = self.render_ai_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Contacts => {
                let list = self.render_contacts_list(cx, p);
                let detail = self.render_contacts_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Archived => {
                let list = self.render_archived_list(cx, p);
                let detail = self.render_brand_empty(cx, p, "Archived");
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Starred => {
                let list = self.render_starred_list(cx, p);
                let detail = self.render_brand_empty(cx, p, "Starred");
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Locked => {
                let list = self.render_locked_list(cx, p);
                let detail = self.render_locked_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Broadcast => {
                let list = self.render_broadcast_list(cx, p);
                let detail = self.render_broadcast_detail(cx, p);
                div()
                    .flex_1()
                    .flex()
                    .min_w(px(0.))
                    .child(list)
                    .child(detail)
                    .into_any_element()
            }
            Route::Onboarding => div().flex_1().into_any_element(),
        };

        let mut shell = div().flex_1().flex().min_h(px(0.));
        if self.rtl {
            shell = shell.flex_row_reverse();
        }
        shell = shell.child(sidebar).child(content);
        if let Some(panel) = self.info_panel {
            if self.route == Route::Chats && width < px(PANEL_INLINE_BREAKPOINT) {
                shell = shell.child(self.render_info_overlay(panel, cx, p));
            }
        }
        shell
    }

    fn render_sidebar(&self, cx: &mut Context<Self>, p: Palette, compact: bool) -> gpui_kit::Div {
        let collapsed = self.collapsed || compact;
        let width = if collapsed { RAIL_WIDTH } else { SIDEBAR_WIDTH };
        let account_row = if self.accounts.len() > 1 {
            let active = self.accounts.get(self.active_account).cloned();
            let other = self
                .accounts
                .iter()
                .find(|account| !account.active)
                .cloned();
            let mut row = div()
                .id("account-switcher")
                .h(px(48.))
                .flex()
                .items_center()
                .gap(px(8.))
                .px(px(8.))
                .rounded(px(RADIUS_LG))
                .hover(move |style| style.bg(p.hover()))
                .child(
                    div()
                        .flex()
                        .items_center()
                        .when_some(active, |el, account| {
                            el.child(ui::avatar(account.initials, 28., false, p))
                        })
                        .when_some(other, |el, account| {
                            el.child(div().ml(px(-8.)).child(ui::avatar(
                                account.initials,
                                28.,
                                false,
                                p,
                            )))
                        }),
                )
                .when(!collapsed, |el| {
                    el.child(
                        div()
                            .flex_1()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(p.text_primary)
                                    .child("2 accounts"),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_secondary)
                                    .child("Personal · Studio"),
                            ),
                    )
                    .child(ui::icon(
                        IconName::ChevronDown,
                        14.0,
                        p.text_secondary,
                    ))
                });
            row = self.clickable(cx, row, Action::OpenAccounts);
            Some(row)
        } else {
            None
        };
        let mut nav =
            |icon, label, route, badge| self.nav_item(cx, p, icon, label, route, badge, collapsed);

        let mut sidebar = div()
            .w(px(width))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .gap(px(2.))
            .p(px(8.))
            .pt(px(16.))
            .bg(p.bg_sidebar)
            .border_r_1()
            .border_color(p.divider);

        if self.shell == ShellStyle::MacOS {
            sidebar = sidebar
                .rounded(px(RADIUS_XL))
                .border_1()
                .border_color(p.border)
                .shadow_sm();
            if !collapsed {
                sidebar = sidebar.child(
                    div()
                        .h(px(24.))
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .px(px(4.))
                        .child(traffic_light(p.traffic_red))
                        .child(traffic_light(p.traffic_yellow))
                        .child(traffic_light(p.traffic_green)),
                );
            }
        }

        sidebar = sidebar
            .child(nav(IconName::MessageSquare, "Chats", Route::Chats, None))
            .child(nav(IconName::CircleDashed, "Updates", Route::Updates, None))
            .child(nav(IconName::Phone, "Calls", Route::Calls, Some(8)))
            .child(nav(
                IconName::Users,
                "Communities",
                Route::Communities,
                None,
            ))
            .child(nav(IconName::Image, "Media", Route::Media, None))
            .child(nav(IconName::Sparkles, "AI", Route::Ai, None));

        if !collapsed {
            sidebar = sidebar.child(ui::section_label("More", p));
        } else {
            sidebar = sidebar.child(div().h(px(8.)));
        }

        sidebar = sidebar
            .child(nav(IconName::Lock, "Locked", Route::Locked, None))
            .child(nav(IconName::Archive, "Archived", Route::Archived, Some(1)))
            .child(nav(IconName::Star, "Starred", Route::Starred, None))
            .child(div().flex_1())
            .when_some(account_row, |el, row| el.child(row))
            .child(nav(IconName::Settings, "Settings", Route::Settings, None))
            .child(
                div()
                    .h(px(48.))
                    .flex()
                    .items_center()
                    .gap(px(10.))
                    .px(px(8.))
                    .rounded(px(RADIUS_LG))
                    .when(collapsed, |el| el.justify_center().px(px(0.)))
                    .child(ui::avatar("Y", 32., false, p))
                    .when(!collapsed, |el| {
                        el.child(
                            div()
                                .flex_1()
                                .text_size(px(13.))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(p.text_primary)
                                .child("You"),
                        )
                        .child(ui::icon(
                            IconName::Pencil,
                            15.0,
                            p.text_secondary,
                        ))
                    }),
            );

        sidebar
    }

    fn nav_item(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        icon_name: IconName,
        label: &'static str,
        route: Route,
        badge: Option<u32>,
        collapsed: bool,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let selected = self.route == route;
        let icon_color = if selected { p.accent } else { p.text_secondary };
        let row = div()
            .id(format!("nav-{label}"))
            .role(Role::Button)
            .tab_index(0)
            .aria_label(label)
            .cursor_pointer()
            .h(px(40.))
            .w_full()
            .flex()
            .items_center()
            .gap(px(12.))
            .rounded(px(RADIUS_LG))
            .bg(if selected {
                p.bg_selected
            } else {
                p.bg_sidebar
            })
            .text_color(p.text_primary)
            .text_size(px(13.))
            .font_weight(FontWeight::MEDIUM)
            .hover(move |style| style.bg(p.hover()))
            .when(collapsed, |el| el.justify_center().px(px(0.)))
            .when(!collapsed, |el| el.px(px(12.)))
            .child(ui::icon(icon_name, 20.0, icon_color))
            .when(!collapsed, |el| el.child(label))
            .when_some(badge, |el, count| {
                el.child(div().flex_1()).child(ui::badge(count, p))
            });
        self.clickable(cx, row, Action::Navigate(route))
    }

    fn list_body<I: IntoElement + 'static>(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        rows: Vec<I>,
        empty_title: &'static str,
        empty_body: &'static str,
    ) -> gpui_kit::AnyElement {
        match self.list_state {
            ListState::Loading => div()
                .flex_1()
                .children((0..5).map(|_| ui::skeleton_row(p).into_any_element()))
                .into_any_element(),
            ListState::Failed => div()
                .flex_1()
                .flex()
                .items_center()
                .justify_center()
                .child(ui::error_state_compact(
                    "Could not load this list",
                    "Check your connection and try again.",
                    Some(self.clickable(
                        cx,
                        ui::primary_button("retry-list", "Retry", p),
                        Action::SetListState(ListState::Ready),
                    )),
                    p,
                ))
                .into_any_element(),
            ListState::Ready if rows.is_empty() => div()
                .flex_1()
                .child(ui::empty_state_compact(empty_title, empty_body, None, p))
                .into_any_element(),
            ListState::Ready => div()
                .flex_1()
                .overflow_y_scrollbar()
                .children(rows)
                .into_any_element(),
        }
    }

    fn render_chat_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let query = self.search.read(cx).value().to_string();
        let chats = self.filtered_chats(&query);
        let rows: Vec<gpui_kit::AnyElement> = chats
            .iter()
            .map(|chat| self.render_chat_row(cx, p, chat).into_any_element())
            .collect();
        let body = match self.list_state {
            ListState::Loading => div()
                .flex_1()
                .children((0..5).map(|_| ui::skeleton_row(p).into_any_element()))
                .into_any_element(),
            ListState::Failed => div()
                .flex_1()
                .flex()
                .items_center()
                .justify_center()
                .child(ui::error_state_compact(
                    "Could not load chats",
                    "Check your connection and try again.",
                    Some(self.clickable(
                        cx,
                        ui::primary_button("retry-chats", "Retry", p),
                        Action::SetListState(ListState::Ready),
                    )),
                    p,
                ))
                .into_any_element(),
            ListState::Ready if rows.is_empty() => div()
                .flex_1()
                .child(ui::empty_state_compact(
                    "No chats found",
                    "Try a different spelling, or search for a message instead.",
                    None,
                    p,
                ))
                .into_any_element(),
            ListState::Ready => div()
                .flex_1()
                .overflow_y_scrollbar()
                .children(rows)
                .into_any_element(),
        };

        let filters = [
            ("All", ChatFilter::All),
            ("Unread", ChatFilter::Unread),
            ("Favorites", ChatFilter::Favorites),
            ("Groups", ChatFilter::Groups),
        ];

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(12.))
                    .child(self.clickable(
                        cx,
                        ui::icon_button(
                            "chat-list-more",
                            IconName::Ellipsis,
                            18.0,
                            p.text_secondary,
                            p.hover(),
                            "Chat list menu",
                        ),
                        Action::OpenChatListMenu,
                    ))
                    .child(
                        div()
                            .text_size(px(17.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Chats"),
                    )
                    .child(
                        self.clickable(
                            cx,
                            div()
                                .id("new-chat")
                                .role(Role::Button)
                                .tab_index(0)
                                .aria_label("New chat")
                                .cursor_pointer()
                                .size(px(32.))
                                .flex()
                                .items_center()
                                .justify_center()
                                .rounded_full()
                                .bg(p.accent)
                                .child(ui::icon(IconName::Plus, 18.0, p.text_inverse)),
                            Action::NewChat,
                        ),
                    ),
            )
            .child(
                div().px(px(12.)).py(px(5.)).child(
                    div()
                        .h(px(32.))
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .rounded_full()
                        .bg(p.bg_search)
                        .px(px(10.))
                        .child(ui::icon(IconName::Search, 16.0, p.text_secondary))
                        .child(
                            div().flex_1().min_w(px(0.)).child(
                                Input::new(&self.search)
                                    .id("chat-search")
                                    .appearance(false)
                                    .bordered(false)
                                    .cleanable(true)
                                    .aria_label("Search chats"),
                            ),
                        ),
                ),
            )
            .child(
                div()
                    .h(px(44.))
                    .flex()
                    .items_center()
                    .gap(px(6.))
                    .px(px(10.))
                    .children(filters.into_iter().map(|(label, filter)| {
                        let chip = ui::chip(label, self.chat_filter == filter, p);
                        self.clickable(cx, chip, Action::SetFilter(filter))
                    })),
            )
            .child(body)
    }

    fn render_chat_row(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        chat: &ChatModel,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let chat_id = chat.id;
        let selected = chat.id == self.selected_chat;
        let preview = chat.preview_text();
        let muted = chat.muted;
        let name_color = p.text_primary;
        let preview_color = if chat.unread > 0 && !muted {
            p.text_primary
        } else {
            p.text_secondary
        };

        let row = div()
            .id(format!("chat-row-{}", chat.id))
            .role(Role::Button)
            .tab_index(0)
            .aria_label(format!(
                "{} · {} · {}{}",
                chat.name,
                chat.time,
                preview,
                if chat.unread > 0 { " · unread" } else { "" }
            ))
            .cursor_pointer()
            .h(px(CHAT_ROW_HEIGHT))
            .w_full()
            .flex()
            .items_center()
            .gap(px(12.))
            .px(px(12.))
            .border_b_1()
            .border_color(p.divider)
            .bg(if self.drag_file_hover == Some(chat_id) {
                p.accent_soft
            } else if selected {
                p.bg_row_selected
            } else {
                p.bg_list
            })
            .when(self.drag_file_hover == Some(chat_id), |el| {
                el.border_1().border_color(p.accent)
            })
            .hover(move |style| style.bg(p.bg_row_selected))
            .child(ui::avatar(chat.initials.clone(), 44., chat.online, p))
            .child(
                div()
                    .flex_1()
                    .min_w(px(0.))
                    .flex()
                    .flex_col()
                    .gap(px(2.))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .justify_between()
                            .gap(px(8.))
                            .child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(5.))
                                    .min_w(px(0.))
                                    .child(
                                        div()
                                            .text_size(self.ts(TEXT_BODY))
                                            .font_weight(FontWeight::SEMIBOLD)
                                            .text_color(name_color)
                                            .overflow_hidden()
                                            .whitespace_nowrap()
                                            .text_ellipsis()
                                            .child(chat.name.clone()),
                                    )
                                    .when(chat.pinned, |el| {
                                        el.child(ui::icon(IconName::Pin, 12.0, p.text_secondary))
                                    })
                                    .when(muted, |el| {
                                        el.child(ui::icon(
                                            IconName::BellOff,
                                            12.0,
                                            p.text_secondary,
                                        ))
                                    })
                                    .when(chat.verified, |el| {
                                        el.child(ui::icon(
                                            IconName::BadgeCheck,
                                            14.0,
                                            p.attach_blue,
                                        ))
                                    })
                                    .when(chat.business, |el| {
                                        el.child(ui::icon(IconName::Briefcase, 12.0, p.attach_blue))
                                    }),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_secondary)
                                    .child(chat.time.clone()),
                            ),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .justify_between()
                            .gap(px(8.))
                            .child(
                                div()
                                    .flex_1()
                                    .min_w(px(0.))
                                    .text_size(self.ts(13.))
                                    .text_color(preview_color)
                                    .overflow_hidden()
                                    .whitespace_nowrap()
                                    .text_ellipsis()
                                    .when(chat.draft.is_some(), |el| el.text_color(p.accent))
                                    .child(preview),
                            )
                            .when(chat.typing.is_some(), |el| {
                                el.child(
                                    div()
                                        .size(px(6.))
                                        .flex_shrink_0()
                                        .rounded_full()
                                        .bg(p.accent)
                                        .with_animation(
                                            format!("typing-dot-{}", chat.id),
                                            Animation::new(Duration::from_millis(900)).repeat(),
                                            |el, delta| el.opacity(0.35 + 0.65 * delta),
                                        ),
                                )
                            })
                            .when(chat.unread > 0 && !muted, |el| {
                                el.child(ui::badge(chat.unread, p))
                            })
                            .when(chat.unread > 0 && muted, |el| el.child(ui::dot_badge(p))),
                    ),
            );

        let row = row.on_mouse_down(
            MouseButton::Right,
            cx.listener(move |this, _, window, cx| {
                this.apply(Action::OpenContextMenu(chat_id), window, cx);
            }),
        );
        let drop_app = cx.entity();
        let target = chat_id;
        let row = row.on_drop::<String>(move |label, _window, cx| {
            let label = label.clone();
            drop_app.update(cx, |this, cx| {
                this.selected_chat = target;
                this.staged_attachments.push(StagedAttachment {
                    kind: StagedKind::File,
                    label: label.clone(),
                });
                this.drag_file_hover = None;
                let name = this
                    .chats
                    .get(target)
                    .map(|chat| chat.name.clone())
                    .unwrap_or_default();
                this.toast(format!("File staged for {name} · {label}"), cx);
            });
        });
        self.clickable(cx, row, Action::SelectChat(chat_id))
    }

    fn render_conversation(&mut self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let Some(chat) = self.selected_chat().cloned() else {
            return div()
                .flex_1()
                .h_full()
                .flex()
                .items_center()
                .justify_center()
                .bg(p.bg_conversation)
                .child(ui::empty_state(
                    "Choose a conversation",
                    "Select a sample chat to view messages.",
                    None,
                    p,
                ));
        };

        let header = self.render_conversation_header(cx, p, &chat);
        let messages = self
            .messages
            .get(self.selected_chat)
            .cloned()
            .unwrap_or_default();
        let message_views = self.render_messages(cx, p, &chat, &messages);

        let search_query = self.chat_search.read(cx).value().to_lowercase();
        let active_search = self.searching
            && (!search_query.trim().is_empty()
                || self.chat_search_filter != ChatSearchFilter::All);
        if active_search {
            if let Some(first) = messages.iter().position(|message| {
                message_matches_search(message, &search_query, self.chat_search_filter)
            }) {
                self.conversation_scroll.scroll_to_item(first);
            }
        } else {
            self.conversation_scroll
                .scroll_to_item(messages.len().saturating_sub(1));
        }
        let mut stage = div()
            .flex_1()
            .relative()
            .overflow_hidden()
            .bg(p.bg_conversation)
            .child(ui::wallpaper(self.theme == ThemeMode::Dark));

        match self.list_state {
            ListState::Loading => {
                stage = stage.child(ui::skeleton_conversation(p));
            }
            ListState::Failed => {
                stage = stage.child(
                    div()
                        .size_full()
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(
                            div()
                                .rounded(px(RADIUS_LG))
                                .bg(p.bg_list)
                                .border_1()
                                .border_color(p.border)
                                .shadow_sm()
                                .child(ui::error_state_compact(
                                    "Could not load messages",
                                    "Check your connection and try again.",
                                    Some(self.clickable(
                                        cx,
                                        ui::primary_button("retry-conversation", "Retry", p),
                                        Action::SetListState(ListState::Ready),
                                    )),
                                    p,
                                )),
                        ),
                );
            }
            ListState::Ready => {
                stage = stage.child(
                    div()
                        .id("conversation-scroll")
                        .size_full()
                        .overflow_y_scroll()
                        .track_scroll(&self.conversation_scroll)
                        .flex()
                        .flex_col()
                        .justify_end()
                        .px(px(16.))
                        .py(px(12.))
                        .children(message_views),
                );
                stage = stage.child(
                    self.clickable(
                        cx,
                        div()
                            .id("scroll-bottom")
                            .absolute()
                            .bottom(px(12.))
                            .right(px(16.))
                            .size(px(34.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .shadow_sm()
                            .child(ui::icon(IconName::ChevronDown, 18.0, p.text_secondary)),
                        Action::ScrollToBottom,
                    ),
                );
            }
        }

        let ready = self.list_state == ListState::Ready;
        let composer = ready.then(|| self.render_composer(cx, p));
        let search_filters = if ready && self.searching {
            Some(
                div()
                    .h(px(36.))
                    .flex_shrink_0()
                    .flex()
                    .items_center()
                    .gap(px(6.))
                    .px(px(16.))
                    .bg(p.bg_header)
                    .border_b_1()
                    .border_color(p.divider)
                    .children(
                        [
                            (ChatSearchFilter::All, "All"),
                            (ChatSearchFilter::Media, "Media"),
                            (ChatSearchFilter::Links, "Links"),
                            (ChatSearchFilter::Docs, "Docs"),
                        ]
                        .into_iter()
                        .map(|(filter, label)| {
                            let selected = self.chat_search_filter == filter;
                            self.clickable(
                                cx,
                                div()
                                    .id(format!("search-filter-{label}"))
                                    .role(Role::Button)
                                    .tab_index(0)
                                    .aria_label(label)
                                    .cursor_pointer()
                                    .h(px(24.))
                                    .px(px(10.))
                                    .flex()
                                    .items_center()
                                    .rounded_full()
                                    .bg(if selected { p.accent_soft } else { p.bg_search })
                                    .text_size(px(TEXT_CAPTION))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(if selected {
                                        p.unread_text
                                    } else {
                                        p.text_secondary
                                    })
                                    .child(label),
                                Action::SetChatSearchFilter(filter),
                            )
                        }),
                    ),
            )
        } else {
            None
        };

        let network_banner = if self.network_state != NetworkState::Online {
            let message = match self.network_state {
                NetworkState::Connecting => "Connecting to WhatsApp…".to_owned(),
                NetworkState::Offline => {
                    if self.outbox > 0 {
                        format!(
                            "No internet — {} messages queued · will send when you reconnect.",
                            self.outbox
                        )
                    } else {
                        "No internet — messages will send when you reconnect.".to_owned()
                    }
                }
                NetworkState::Reconnecting => "Connection lost 3s ago, retrying…".to_owned(),
                NetworkState::Online => String::new(),
            };
            Some(
                div()
                    .h(px(32.))
                    .flex_shrink_0()
                    .flex()
                    .items_center()
                    .justify_center()
                    .gap(px(8.))
                    .bg(p.bg_notice_encrypt)
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_primary)
                    .child(ui::icon(IconName::WifiOff, 14.0, p.attach_orange))
                    .child(message),
            )
        } else {
            None
        };

        div()
            .flex_1()
            .min_w(px(0.))
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_conversation)
            .child(header)
            .when_some(search_filters, |el, filters| el.child(filters))
            .when(ready && !self.pinned.is_empty(), |el| {
                el.child(self.render_pinned_bar(cx, p))
            })
            .child(stage)
            .when_some(network_banner, |el, banner| el.child(banner))
            .when(ready && self.message_select_mode, |el| {
                el.child(self.render_selection_bar(cx, p))
            })
            .when_some(composer, |el, composer| el.child(composer))
    }

    fn render_conversation_header(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        chat: &ChatModel,
    ) -> gpui_kit::Div {
        let status = if chat.is_group {
            "Rina, Bayu, Sari"
        } else if chat.online {
            "online"
        } else {
            "last seen recently"
        };

        let title_area = if self.searching {
            div()
                .id("conversation-search-area")
                .flex_1()
                .min_w(px(0.))
                .h(px(34.))
                .flex()
                .items_center()
                .gap(px(8.))
                .rounded_full()
                .bg(p.bg_search)
                .px(px(10.))
                .child(ui::icon(IconName::Search, 16.0, p.text_secondary))
                .child(
                    div().flex_1().child(
                        Input::new(&self.chat_search)
                            .id("in-chat-search")
                            .appearance(false)
                            .bordered(false)
                            .cleanable(true)
                            .aria_label("Search in conversation"),
                    ),
                )
                .child({
                    let query = self.chat_search.read(cx).value().to_lowercase();
                    let total = self
                        .messages
                        .get(self.selected_chat)
                        .map(|messages| {
                            messages
                                .iter()
                                .filter(|message| {
                                    message_matches_search(message, &query, self.chat_search_filter)
                                })
                                .count()
                        })
                        .unwrap_or(0);
                    let active = !query.trim().is_empty()
                        || self.chat_search_filter != ChatSearchFilter::All;
                    let label = if !active {
                        "Search".to_owned()
                    } else if total == 0 {
                        "0 results".to_owned()
                    } else {
                        format!("{} of {}", self.search_hit.min(total - 1) + 1, total)
                    };
                    div()
                        .flex()
                        .items_center()
                        .gap(px(4.))
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_secondary)
                        .child(label)
                        .child(self.clickable(
                            cx,
                            ui::icon_button_small(
                                "search-prev",
                                IconName::ChevronUp,
                                14.0,
                                p.text_secondary,
                                p.hover(),
                                "Previous hit",
                            ),
                            Action::SearchPrev,
                        ))
                        .child(self.clickable(
                            cx,
                            ui::icon_button_small(
                                "search-next",
                                IconName::ChevronDown,
                                14.0,
                                p.text_secondary,
                                p.hover(),
                                "Next hit",
                            ),
                            Action::SearchNext,
                        ))
                })
                .child(
                    self.clickable(
                        cx,
                        div()
                            .id("close-search")
                            .role(Role::Button)
                            .tab_index(0)
                            .aria_label("Close search")
                            .cursor_pointer()
                            .child(ui::icon(IconName::X, 16.0, p.text_secondary)),
                        Action::ToggleSearch,
                    ),
                )
        } else {
            let menu_open = matches!(self.overlay, Some(Overlay::HeaderMenu));
            let show_chevron = self.header_hovered || menu_open;
            let identity = div()
                .id("conversation-identity")
                .flex()
                .items_center()
                .gap(px(8.))
                .px(px(6.))
                .py(px(4.))
                .rounded(px(RADIUS_MD))
                .cursor_pointer()
                .hover(move |style| style.bg(p.hover()))
                .on_hover(cx.listener(|this, hovered: &bool, _window, cx| {
                    if this.header_hovered != *hovered {
                        this.header_hovered = *hovered;
                        cx.notify();
                    }
                }))
                .child(ui::avatar(chat.initials.clone(), 40., chat.online, p))
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(1.))
                        .min_w(px(0.))
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .gap(px(6.))
                                .child(
                                    div()
                                        .text_size(px(TEXT_BODY))
                                        .font_weight(FontWeight::SEMIBOLD)
                                        .text_color(p.text_primary)
                                        .overflow_hidden()
                                        .whitespace_nowrap()
                                        .child(chat.name.clone()),
                                )
                                .child(
                                    div()
                                        .flex_shrink_0()
                                        .opacity(if show_chevron { 1. } else { 0. })
                                        .child(ui::icon(
                                            IconName::ChevronDown,
                                            16.0,
                                            p.text_secondary,
                                        )),
                                ),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(status),
                        ),
                );
            self.clickable(cx, identity, Action::ToggleHeaderMenu)
        };

        div()
            .h(px(CONVERSATION_HEADER_HEIGHT))
            .flex_shrink_0()
            .flex()
            .items_center()
            .justify_between()
            .gap(px(12.))
            .px(px(16.))
            .bg(p.bg_header)
            .border_b_1()
            .border_color(p.divider)
            .child(title_area)
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(2.))
                    .child(self.clickable(
                        cx,
                        ui::icon_button(
                            "conversation-video",
                            IconName::Video,
                            20.0,
                            p.text_primary,
                            p.hover(),
                            "Video call",
                        ),
                        Action::StartCall(self.selected_chat, true),
                    ))
                    .child(self.clickable(
                        cx,
                        ui::icon_button(
                            "conversation-call",
                            IconName::Phone,
                            19.0,
                            p.text_primary,
                            p.hover(),
                            "Voice call",
                        ),
                        Action::StartCall(self.selected_chat, false),
                    ))
                    .child(self.clickable(
                        cx,
                        ui::icon_button(
                            "conversation-search",
                            IconName::Search,
                            19.0,
                            p.text_primary,
                            p.hover(),
                            "Search in chat",
                        ),
                        Action::ToggleSearch,
                    ))
                    .child(self.clickable(
                        cx,
                        ui::icon_button(
                            "conversation-info",
                            IconName::Ellipsis,
                            19.0,
                            p.text_primary,
                            p.hover(),
                            "Chat menu",
                        ),
                        if chat.is_group {
                            Action::OpenInfo(InfoPanel::Group)
                        } else {
                            Action::OpenInfo(InfoPanel::Contact)
                        },
                    )),
            )
    }

    fn render_messages(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        chat: &ChatModel,
        messages: &[Message],
    ) -> Vec<gpui_kit::AnyElement> {
        let mut views = Vec::new();
        for (index, message) in messages.iter().enumerate() {
            let system = matches!(message.sender, Sender::System)
                || matches!(message.body, MessageBody::Date(_));
            if system {
                views.push(
                    self.render_system_message(cx, p, message)
                        .into_any_element(),
                );
                continue;
            }

            let prev_same = index > 0
                && same_sender(&messages[index - 1], message)
                && !matches!(messages[index - 1].sender, Sender::System)
                && !matches!(messages[index - 1].body, MessageBody::Date(_));
            let next_same = index + 1 < messages.len()
                && same_sender(&messages[index + 1], message)
                && !matches!(messages[index + 1].sender, Sender::System)
                && !matches!(messages[index + 1].body, MessageBody::Date(_));
            let first = !prev_same;
            let last = !next_same;
            let gap = if prev_same { 2. } else { 6. };
            views.push(
                self.render_bubble(cx, p, chat, message, index, first, last, gap)
                    .into_any_element(),
            );
        }
        views
    }

    fn render_system_message(
        &self,
        _cx: &mut Context<Self>,
        p: Palette,
        message: &Message,
    ) -> gpui_kit::Div {
        match &message.body {
            MessageBody::Date(date) => div().w_full().flex().justify_center().my(px(4.)).child(
                div()
                    .px(px(12.))
                    .py(px(5.))
                    .rounded_full()
                    .bg(p.bg_list)
                    .text_size(px(TEXT_SMALL))
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(p.text_secondary)
                    .child(date.clone()),
            ),
            MessageBody::Notice(Notice::Encrypted) => div().w_full().flex().justify_center().my(px(6.)).child(
                div()
                    .max_w(px(600.))
                    .flex()
                    .items_start()
                    .gap(px(10.))
                    .px(px(16.))
                    .py(px(8.))
                    .rounded(px(RADIUS_MD))
                    .bg(p.bg_notice_encrypt)
                    .text_size(px(12.))
                    .text_color(p.text_primary)
                    .child(ui::icon(IconName::Lock, 16.0, p.text_secondary))
                    .child("Messages and calls are end-to-end encrypted. Only people in this chat can read, listen to, or share them."),
            ),
            MessageBody::Notice(Notice::Disappearing) => div().w_full().flex().justify_center().my(px(6.)).child(
                div()
                    .max_w(px(600.))
                    .flex()
                    .items_start()
                    .gap(px(10.))
                    .px(px(16.))
                    .py(px(8.))
                    .rounded(px(RADIUS_MD))
                    .bg(p.bg_notice_system)
                    .text_size(px(12.))
                    .text_color(p.text_primary)
                    .child(ui::icon(IconName::Timer, 16.0, p.text_secondary))
                    .child("You turned on disappearing messages. New messages will disappear 24 hours after they are sent, except when kept."),
            ),
            MessageBody::System(text) => div().w_full().flex().justify_center().my(px(6.)).child(
                div()
                    .px(px(12.))
                    .py(px(6.))
                    .rounded_full()
                    .bg(p.bg_notice_system)
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child(text.clone()),
            ),
            MessageBody::Unread(text) => div()
                .w_full()
                .flex()
                .items_center()
                .gap(px(8.))
                .my(px(6.))
                .child(div().flex_1().h(px(1.)).bg(p.accent))
                .child(
                    div()
                        .px(px(8.))
                        .py(px(2.))
                        .rounded_full()
                        .bg(p.accent_soft)
                        .text_size(px(TEXT_CAPTION))
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(p.unread_text)
                        .child(text.clone()),
                )
                .child(div().flex_1().h(px(1.)).bg(p.accent)),
            _ => div(),
        }
    }

    fn render_bubble(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        chat: &ChatModel,
        message: &Message,
        index: usize,
        first: bool,
        last: bool,
        gap: f32,
    ) -> gpui_kit::Div {
        let outgoing = message.outgoing();
        let show_identity = !outgoing && chat.is_group && first;
        let selected = self.selected_messages.contains(&index);
        let query = self.chat_search.read(cx).value().to_lowercase();
        let highlight = self.searching
            && !query.trim().is_empty()
            && message_matches_search(message, &query, self.chat_search_filter);
        let bare = matches!(
            message.body,
            MessageBody::Emoji(_) | MessageBody::Sticker { .. }
        );
        let bubble_color = if bare {
            p.bubble_in.alpha(0.0)
        } else if outgoing {
            p.bubble_out
        } else {
            p.bubble_in
        };
        let max_width = 420.0;

        let mut body = div()
            .flex()
            .flex_col()
            .gap(px(4.))
            .max_w(px(max_width))
            .px(px(10.))
            .py(px(6.))
            .rounded(px(RADIUS_BUBBLE))
            .bg(bubble_color)
            .text_color(p.text_primary)
            .text_size(self.ts(TEXT_BODY))
            .when(highlight, |el| el.border_1().border_color(p.accent))
            .when(selected, |el| el.border_2().border_color(p.accent));

        if last {
            if outgoing {
                body = body.rounded_br(px(4.));
            } else {
                body = body.rounded_bl(px(4.));
            }
        }

        if let Some(reply) = &message.reply {
            let reply_block = div()
                .id(format!("reply-quote-{}", message.id))
                .mb(px(4.))
                .pl(px(8.))
                .border_l_2()
                .border_color(p.accent)
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.accent)
                        .child(reply.sender.clone()),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .overflow_hidden()
                        .max_h(px(36.))
                        .child(if reply.deleted {
                            "This message was deleted".to_owned()
                        } else {
                            reply.text.clone()
                        }),
                );
            body = body.child(if let Some(target) = reply.message_id {
                self.clickable(cx, reply_block, Action::JumpToMessage(target))
            } else {
                reply_block
            });
        }

        if message.forwarded > 0 {
            body = body.child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(4.))
                    .text_size(px(TEXT_CAPTION))
                    .text_color(p.text_secondary)
                    .child(ui::icon(IconName::Forward, 11.0, p.text_secondary))
                    .child(if message.forwarded > 5 {
                        "Forwarded many times".to_owned()
                    } else {
                        "Forwarded".to_owned()
                    }),
            );
        }

        let content = self.render_message_body(cx, p, message);
        body = body.child(content);
        body = body.child(
            div()
                .flex()
                .items_center()
                .justify_end()
                .gap(px(4.))
                .text_size(px(TEXT_CAPTION))
                .text_color(p.text_secondary)
                .when(message.starred, |el| {
                    el.child(ui::icon(IconName::Star, 11.0, p.attach_orange))
                })
                .when_some(message.countdown.clone(), |el, countdown| {
                    el.child(ui::icon(IconName::Timer, 11.0, p.text_secondary))
                        .child(countdown)
                })
                .child(message.time.clone())
                .when(message.edited, |el| el.child("· edited"))
                .when(outgoing, |el| el.child(delivery_icon(message.delivery, p))),
        );

        let mut row = div()
            .w_full()
            .flex()
            .items_start()
            .gap(px(6.))
            .mt(px(gap))
            .when(outgoing, |el| {
                if self.rtl {
                    el.justify_start()
                } else {
                    el.justify_end()
                }
            })
            .when(!outgoing, |el| {
                if self.rtl {
                    el.justify_end()
                } else {
                    el.justify_start()
                }
            });

        if show_identity {
            let sender = message.sender_name.clone().unwrap_or_default();
            row = row.child(
                self.clickable(
                    cx,
                    div()
                        .id(format!("sender-avatar-{}", message.id))
                        .flex()
                        .flex_col()
                        .items_center()
                        .gap(px(2.))
                        .cursor_pointer()
                        .child(ui::avatar(
                            initials_for(message.sender_name.as_deref()),
                            28.,
                            false,
                            p,
                        )),
                    Action::OpenContactProfile(sender.clone()),
                ),
            );
        }

        let mut column = div()
            .flex()
            .flex_col()
            .items_start()
            .gap(px(2.))
            .max_w(px(max_width));

        if show_identity {
            let sender = message.sender_name.clone().unwrap_or_default();
            column = column.child(
                self.clickable(
                    cx,
                    div()
                        .id(format!("sender-name-{}", message.id))
                        .cursor_pointer()
                        .text_size(px(TEXT_SMALL))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(sender_color(message.sender_name.as_deref(), p))
                        .child(sender.clone()),
                    Action::OpenContactProfile(sender),
                ),
            );
        }

        let bubble_action = if self.message_select_mode {
            Action::ToggleMessageSelection(index)
        } else {
            Action::OpenContextMenu(index)
        };
        let bubble = self.clickable(
            cx,
            body.id(format!("message-{}", message.id))
                .role(Role::Button)
                .tab_index(0)
                .aria_label(format!(
                    "{} · {} · {}",
                    message.time,
                    message.sender_name.clone().unwrap_or_else(|| "You".into()),
                    message_preview(message)
                )),
            bubble_action,
        );
        column = column.child(bubble);

        if !message.reactions.is_empty() {
            let visible = message.reactions.iter().take(6);
            let extra = message.reactions.len().saturating_sub(6);
            column = column.child(
                div()
                    .mt(px(-6.))
                    .ml(px(8.))
                    .flex()
                    .items_center()
                    .gap(px(4.))
                    .children(visible.map(|reaction| {
                        div()
                            .px(px(6.))
                            .py(px(2.))
                            .flex()
                            .items_center()
                            .gap(px(2.))
                            .rounded_full()
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .text_size(px(TEXT_CAPTION))
                            .child(ui::icon(
                                reaction_icon(&reaction.emoji),
                                11.0,
                                p.text_primary,
                            ))
                            .when(reaction.count > 1, |el| {
                                el.child(reaction.count.to_string())
                            })
                    }))
                    .when(extra > 0, |el| {
                        el.child(
                            self.clickable(
                                cx,
                                div()
                                    .id("reaction-overflow")
                                    .px(px(6.))
                                    .py(px(2.))
                                    .rounded_full()
                                    .bg(p.accent_soft)
                                    .text_size(px(TEXT_CAPTION))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(p.unread_text)
                                    .child(format!("+{extra}")),
                                Action::OpenReactions(index),
                            ),
                        )
                    }),
            );
        }

        row.child(column)
    }

    fn render_message_body(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        message: &Message,
    ) -> gpui_kit::AnyElement {
        match &message.body {
            MessageBody::Text(text) => div().child(text.clone()).into_any_element(),
            MessageBody::Rich(runs) => div()
                .flex()
                .flex_wrap()
                .items_center()
                .children(runs.iter().map(|run| {
                    let mut span = div().child(run.text.clone());
                    span = match run.style {
                        crate::model::TextRunStyle::Normal => span,
                        crate::model::TextRunStyle::Bold => span.font_weight(FontWeight::BOLD),
                        crate::model::TextRunStyle::Italic => span.italic(),
                        crate::model::TextRunStyle::Underline => span.underline(),
                        crate::model::TextRunStyle::Strike => span.line_through(),
                        crate::model::TextRunStyle::Code => span
                            .font_family("monospace")
                            .bg(p.bg_search)
                            .px(px(4.))
                            .rounded(px(RADIUS_SM)),
                        crate::model::TextRunStyle::Link => span.text_color(p.accent).underline(),
                    };
                    span
                }))
                .into_any_element(),
            MessageBody::Emoji(emoji) => div()
                .text_size(px(48.))
                .child(emoji.clone())
                .into_any_element(),
            MessageBody::Deleted => div()
                .italic()
                .text_color(p.text_secondary)
                .child("This message was deleted")
                .into_any_element(),
            MessageBody::Image {
                caption,
                state,
                ratio,
            } => {
                let height = 160.0 * ratio.clamp(0.75, 1.1);
                let mut image = div()
                    .w(px(288.))
                    .h(px(height))
                    .rounded(px(RADIUS_LG))
                    .bg(p.attach_orange)
                    .opacity(0.62)
                    .flex()
                    .items_center()
                    .justify_center()
                    .child(ui::icon(IconName::Image, 42.0, p.text_inverse));
                if *state != MediaState::Ready {
                    image = image.child(
                        div()
                            .absolute()
                            .bottom(px(8.))
                            .left(px(8.))
                            .px(px(8.))
                            .py(px(4.))
                            .rounded_full()
                            .bg(p.text_primary)
                            .text_color(p.text_inverse)
                            .text_size(px(TEXT_CAPTION))
                            .child(media_state_label(*state)),
                    );
                }
                div()
                    .flex()
                    .flex_col()
                    .gap(px(4.))
                    .child(image)
                    .when_some(caption.clone(), |el, caption| {
                        el.child(div().text_size(px(13.)).child(caption))
                    })
                    .into_any_element()
            }
            MessageBody::Video {
                caption,
                duration,
                state,
            } => div()
                .flex()
                .flex_col()
                .gap(px(4.))
                .child(
                    div()
                        .relative()
                        .w(px(268.))
                        .h(px(170.))
                        .rounded(px(RADIUS_LG))
                        .bg(p.attach_blue)
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(ui::icon(IconName::Play, 38.0, p.text_inverse))
                        .child(
                            div()
                                .absolute()
                                .bottom(px(8.))
                                .left(px(8.))
                                .px(px(6.))
                                .py(px(2.))
                                .rounded_md()
                                .bg(p.text_primary)
                                .text_color(p.text_inverse)
                                .text_size(px(TEXT_CAPTION))
                                .child(duration.clone()),
                        )
                        .when(*state != MediaState::Ready, |el| {
                            el.child(
                                div()
                                    .absolute()
                                    .top(px(8.))
                                    .left(px(8.))
                                    .px(px(6.))
                                    .py(px(2.))
                                    .rounded_md()
                                    .bg(p.text_primary)
                                    .text_color(p.text_inverse)
                                    .text_size(px(TEXT_CAPTION))
                                    .child(media_state_label(*state)),
                            )
                        }),
                )
                .when_some(caption.clone(), |el, caption| {
                    el.child(div().text_size(px(13.)).child(caption))
                })
                .into_any_element(),
            MessageBody::Voice {
                duration,
                transcript,
            } => div()
                .flex()
                .flex_col()
                .gap(px(4.))
                .child(
                    div()
                        .w(px(260.))
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .child(
                            div()
                                .size(px(36.))
                                .flex()
                                .items_center()
                                .justify_center()
                                .rounded_full()
                                .bg(p.accent)
                                .child(ui::icon(IconName::Play, 16.0, p.text_inverse)),
                        )
                        .child(div().flex_1().flex().items_center().gap(px(2.)).children(
                            (0..24).map(|index| {
                                let height = 6.0 + ((index * 7 + 3) % 18) as f32;
                                div()
                                    .w(px(2.))
                                    .h(px(height))
                                    .rounded_full()
                                    .bg(p.text_secondary)
                            }),
                        ))
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(duration.clone()),
                        )
                        .child(
                            self.clickable(
                                cx,
                                div()
                                    .id("voice-speed")
                                    .text_size(px(TEXT_CAPTION))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(p.accent)
                                    .child(["1×", "1.5×", "2×"][self.voice_speed]),
                                Action::CycleVoiceSpeed,
                            ),
                        ),
                )
                .when_some(transcript.clone(), |el, transcript| {
                    el.child(
                        self.clickable(
                            cx,
                            div()
                                .id("transcript-toggle")
                                .flex()
                                .items_center()
                                .gap(px(3.))
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(ui::icon(
                                    if self.transcript_open {
                                        IconName::ChevronDown
                                    } else {
                                        IconName::ChevronRight
                                    },
                                    12.0,
                                    p.text_secondary,
                                ))
                                .child("Transcript"),
                            Action::ToggleTranscript,
                        ),
                    )
                    .when(self.transcript_open, |el| {
                        el.child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child(transcript),
                        )
                    })
                })
                .into_any_element(),
            MessageBody::Document { name, meta } => div()
                .w(px(280.))
                .flex()
                .items_center()
                .gap(px(10.))
                .child(
                    div()
                        .size(px(38.))
                        .flex()
                        .items_center()
                        .justify_center()
                        .rounded(px(RADIUS_MD))
                        .bg(p.attach_red)
                        .child(ui::icon(IconName::FileText, 20.0, p.text_inverse)),
                )
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(2.))
                        .child(
                            div()
                                .text_size(px(13.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .child(name.clone()),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(meta.clone()),
                        ),
                )
                .into_any_element(),
            MessageBody::Link {
                title,
                domain,
                description,
            } => div()
                .w(px(300.))
                .flex()
                .flex_col()
                .gap(px(6.))
                .child(
                    div()
                        .w_full()
                        .h(px(120.))
                        .rounded(px(RADIUS_MD))
                        .bg(p.attach_blue)
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(ui::icon(IconName::Globe, 36.0, p.text_inverse)),
                )
                .child(
                    div()
                        .text_size(px(13.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .child(title.clone()),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child(description.clone()),
                )
                .child(
                    div()
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_secondary)
                        .child(domain.clone()),
                )
                .into_any_element(),
            MessageBody::Album { count, caption } => {
                let tile = |index: usize, width: f32, height: f32| {
                    div()
                        .w(px(width))
                        .h(px(height))
                        .rounded(px(RADIUS_MD))
                        .bg(if index % 2 == 0 {
                            p.attach_orange
                        } else {
                            p.attach_blue
                        })
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(ui::icon(IconName::Image, 24.0, p.text_inverse))
                };
                let media = match count {
                    1 => tile(0, 300., 180.),
                    2 => div()
                        .flex()
                        .gap(px(3.))
                        .child(tile(0, 148., 120.))
                        .child(tile(1, 148., 120.)),
                    3 => div().flex().gap(px(3.)).child(tile(0, 180., 240.)).child(
                        div()
                            .flex()
                            .flex_col()
                            .gap(px(3.))
                            .child(tile(1, 116., 118.))
                            .child(tile(2, 116., 118.)),
                    ),
                    _ => div()
                        .grid()
                        .grid_cols(2)
                        .gap(px(3.))
                        .children((0..4).map(|index| {
                            tile(index, 148., 90.).when(index == 3 && *count > 4, |el| {
                                el.child(
                                    div()
                                        .text_size(px(16.))
                                        .font_weight(FontWeight::BOLD)
                                        .text_color(p.text_inverse)
                                        .child(format!("+{}", count - 4)),
                                )
                            })
                        })),
                };
                div()
                    .flex()
                    .flex_col()
                    .gap(px(4.))
                    .child(media)
                    .when_some(caption.clone(), |el, caption| {
                        el.child(div().text_size(px(13.)).child(caption))
                    })
                    .into_any_element()
            }
            MessageBody::Poll {
                question,
                options,
                state,
                multiple,
            } => {
                let total: u32 = options.iter().map(|(_, count)| *count).sum();
                let (state_label, state_color) = match state {
                    crate::model::PollState::Open => ("Tap to vote", p.accent),
                    crate::model::PollState::Voted => ("Voted", p.attach_green),
                    crate::model::PollState::Closed => ("Final results", p.text_secondary),
                };
                div()
                    .w(px(280.))
                    .flex()
                    .flex_col()
                    .gap(px(8.))
                    .child(
                        div()
                            .text_size(px(13.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .child(question.clone()),
                    )
                    .when(*multiple, |el| {
                        el.child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child("Select all that apply"),
                        )
                    })
                    .children(options.iter().enumerate().map(|(index, (label, count))| {
                        let pct = if total == 0 {
                            0.0
                        } else {
                            (*count as f32 / total as f32) * 240.0
                        };
                        let selected = *state == crate::model::PollState::Voted && index == 0;
                        div()
                            .flex()
                            .flex_col()
                            .gap(px(2.))
                            .child(
                                div()
                                    .flex()
                                    .items_center()
                                    .justify_between()
                                    .gap(px(6.))
                                    .text_size(px(12.))
                                    .child(
                                        div()
                                            .flex()
                                            .items_center()
                                            .gap(px(4.))
                                            .when(selected, |el| {
                                                el.child(ui::icon(
                                                    IconName::CircleCheck,
                                                    12.0,
                                                    p.attach_green,
                                                ))
                                            })
                                            .child(label.clone()),
                                    )
                                    .child(count.to_string()),
                            )
                            .child(
                                div()
                                    .h(px(5.))
                                    .w_full()
                                    .rounded_full()
                                    .bg(p.bg_search)
                                    .child(
                                        div().h(px(5.)).w(px(pct)).rounded_full().bg(if selected {
                                            p.attach_green
                                        } else {
                                            p.accent
                                        }),
                                    ),
                            )
                    }))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .justify_between()
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(state_color)
                                    .child(state_label.to_owned()),
                            )
                            .when(*state == crate::model::PollState::Voted, |el| {
                                el.child(
                                    self.clickable(
                                        cx,
                                        div()
                                            .id("poll-view-votes")
                                            .text_size(px(TEXT_CAPTION))
                                            .text_color(p.accent)
                                            .child("View votes"),
                                        Action::OpenPollVotes,
                                    ),
                                )
                            }),
                    )
                    .into_any_element()
            }
            MessageBody::Event {
                title,
                time,
                location,
                state,
            } => {
                let (label, color) = match state {
                    crate::model::EventState::Invited => ("Going?", p.accent),
                    crate::model::EventState::Going => ("Going", p.attach_green),
                    crate::model::EventState::Declined => ("Declined", p.text_secondary),
                    crate::model::EventState::Updated => ("Updated · Going", p.attach_blue),
                    crate::model::EventState::Cancelled => ("Cancelled", p.danger()),
                };
                div()
                    .w(px(280.))
                    .flex()
                    .flex_col()
                    .gap(px(5.))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .child(ui::icon(IconName::Calendar, 18.0, p.attach_red))
                            .child(
                                div()
                                    .text_size(px(13.))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .child(title.clone()),
                            ),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(time.clone()),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(location.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(6.))
                            .child(
                                div()
                                    .px(px(8.))
                                    .py(px(2.))
                                    .rounded_full()
                                    .bg(if *state == crate::model::EventState::Cancelled {
                                        p.attach_red
                                    } else {
                                        p.accent_soft
                                    })
                                    .text_size(px(TEXT_CAPTION))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(if *state == crate::model::EventState::Cancelled {
                                        p.text_inverse
                                    } else {
                                        color
                                    })
                                    .child(label.to_owned()),
                            )
                            .when(*state == crate::model::EventState::Invited, |el| {
                                el.child(
                                    div()
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.accent)
                                        .child("Accept · Decline"),
                                )
                            }),
                    )
                    .into_any_element()
            }
            MessageBody::ContactCard { name, phone } => div()
                .w(px(260.))
                .flex()
                .items_center()
                .gap(px(10.))
                .child(ui::avatar(initials_for(Some(name)), 36., false, p))
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .child(
                            div()
                                .text_size(px(13.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .child(name.clone()),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(phone.clone()),
                        ),
                )
                .child(ui::chip("View", false, p))
                .into_any_element(),
            MessageBody::Location { name, address } => div()
                .w(px(280.))
                .flex()
                .flex_col()
                .gap(px(4.))
                .child(
                    div()
                        .h(px(120.))
                        .w_full()
                        .rounded(px(RADIUS_MD))
                        .bg(p.attach_green)
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(ui::icon(IconName::MapPin, 36.0, p.text_inverse)),
                )
                .child(
                    div()
                        .text_size(px(13.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .child(name.clone()),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child(address.clone()),
                )
                .into_any_element(),
            MessageBody::Sticker {
                symbol: _,
                pack_missing,
            } => {
                if *pack_missing {
                    div()
                        .w(px(180.))
                        .h(px(120.))
                        .flex()
                        .flex_col()
                        .items_center()
                        .justify_center()
                        .gap(px(6.))
                        .rounded(px(RADIUS_MD))
                        .bg(p.bg_search)
                        .border_1()
                        .border_color(p.border)
                        .child(ui::icon(IconName::Sparkles, 28.0, p.text_secondary))
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child("Sticker pack unavailable"),
                        )
                        .child(ui::chip("Add pack", false, p))
                        .into_any_element()
                } else {
                    div()
                        .flex()
                        .flex_col()
                        .items_center()
                        .gap(px(4.))
                        .child(ui::icon(IconName::Sparkles, 72.0, p.attach_green))
                        .child(
                            div()
                                .text_size(px(10.))
                                .text_color(p.text_secondary)
                                .child("Sticker"),
                        )
                        .into_any_element()
                }
            }
            MessageBody::Mention { text, mention } => div()
                .w(px(260.))
                .flex()
                .flex_col()
                .gap(px(4.))
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(4.))
                        .text_size(px(13.))
                        .child(text.clone())
                        .child(
                            div()
                                .px(px(5.))
                                .py(px(1.))
                                .rounded(px(RADIUS_SM))
                                .bg(p.accent_soft)
                                .text_size(px(TEXT_CAPTION))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(p.accent)
                                .child(format!("@{mention}")),
                        ),
                )
                .child(
                    div()
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_secondary)
                        .child("Mention · tap to open profile"),
                )
                .into_any_element(),
            MessageBody::ViewOnce { caption, state } => {
                let (icon_name, label) = match state {
                    crate::model::ViewOnceState::Sealed => {
                        (IconName::Eye, "Tap to open · cannot be forwarded")
                    }
                    crate::model::ViewOnceState::Opening => (IconName::LoaderCircle, "Opening…"),
                    crate::model::ViewOnceState::Opened => (IconName::EyeOff, "Opened"),
                };
                div()
                    .w(px(240.))
                    .flex()
                    .flex_col()
                    .gap(px(4.))
                    .child(
                        div()
                            .h(px(120.))
                            .w_full()
                            .rounded(px(RADIUS_MD))
                            .bg(p.attach_orange)
                            .opacity(0.62)
                            .flex()
                            .items_center()
                            .justify_center()
                            .child(ui::icon(icon_name, 36.0, p.text_inverse)),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(label),
                    )
                    .when_some(caption.clone(), |el, caption| {
                        el.child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(caption),
                        )
                    })
                    .into_any_element()
            }
            MessageBody::LiveLocation {
                name,
                address,
                remaining,
                state,
            } => {
                let (label, color, action) = match state {
                    crate::model::LiveLocationState::Sharing => {
                        (format!("Sharing for {remaining}"), p.attach_green, "Stop")
                    }
                    crate::model::LiveLocationState::Paused => {
                        (format!("Paused · {remaining}"), p.attach_orange, "Resume")
                    }
                    crate::model::LiveLocationState::Ended => {
                        (format!("Ended · {remaining}"), p.text_secondary, "View")
                    }
                };
                div()
                    .w(px(280.))
                    .flex()
                    .flex_col()
                    .gap(px(4.))
                    .child(
                        div()
                            .h(px(120.))
                            .w_full()
                            .rounded(px(RADIUS_MD))
                            .bg(if *state == crate::model::LiveLocationState::Ended {
                                p.bg_search
                            } else {
                                p.attach_green
                            })
                            .opacity(if *state == crate::model::LiveLocationState::Paused {
                                0.7
                            } else {
                                1.0
                            })
                            .flex()
                            .items_center()
                            .justify_center()
                            .child(ui::icon(IconName::MapPin, 36.0, p.text_inverse)),
                    )
                    .child(
                        div()
                            .text_size(px(13.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .child(name.clone()),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(address.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .justify_between()
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(color)
                                    .child(label),
                            )
                            .child(ui::chip(action, false, p)),
                    )
                    .into_any_element()
            }
            MessageBody::Call {
                missed,
                video,
                duration,
            } => div()
                .w(px(280.))
                .flex()
                .items_center()
                .gap(px(10.))
                .child(
                    div()
                        .size(px(38.))
                        .flex()
                        .items_center()
                        .justify_center()
                        .rounded_full()
                        .bg(if *missed { p.danger() } else { p.attach_green })
                        .child(ui::icon(
                            if *video {
                                IconName::Video
                            } else {
                                IconName::Phone
                            },
                            18.0,
                            p.text_inverse,
                        )),
                )
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .child(
                            div()
                                .text_size(px(13.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .child(if *missed {
                                    if *video {
                                        "Missed video call".to_owned()
                                    } else {
                                        "Missed voice call".to_owned()
                                    }
                                } else {
                                    format!(
                                        "{} · {}",
                                        if *video { "Video call" } else { "Voice call" },
                                        duration.clone().unwrap_or_else(|| "1 min".into())
                                    )
                                }),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child("Tap to call back"),
                        ),
                )
                .into_any_element(),
            MessageBody::Upload {
                name,
                progress,
                failed,
            } => div()
                .w(px(280.))
                .flex()
                .flex_col()
                .gap(px(4.))
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .child(ui::icon(
                            IconName::FileText,
                            18.0,
                            if *failed { p.danger() } else { p.accent },
                        ))
                        .child(
                            div()
                                .text_size(px(13.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .child(name.clone()),
                        ),
                )
                .child(
                    div()
                        .h(px(6.))
                        .w_full()
                        .rounded_full()
                        .bg(p.bg_search)
                        .child(
                            div()
                                .h(px(6.))
                                .w(px((*progress as f32 * 2.4).min(264.)))
                                .rounded_full()
                                .bg(if *failed { p.danger() } else { p.accent }),
                        ),
                )
                .child(
                    div()
                        .text_size(px(TEXT_CAPTION))
                        .text_color(if *failed {
                            p.danger()
                        } else {
                            p.text_secondary
                        })
                        .child(if *failed {
                            "Upload failed · retry available".to_owned()
                        } else {
                            format!("Uploading · {progress}%")
                        }),
                )
                .into_any_element(),
            MessageBody::Gif { caption, failed } => div()
                .w(px(240.))
                .flex()
                .flex_col()
                .gap(px(4.))
                .child(
                    div()
                        .h(px(130.))
                        .w_full()
                        .rounded(px(RADIUS_MD))
                        .bg(if *failed { p.bg_search } else { p.attach_blue })
                        .border_1()
                        .border_color(if *failed { p.danger() } else { p.attach_blue })
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(
                            div()
                                .flex()
                                .flex_col()
                                .items_center()
                                .gap(px(4.))
                                .text_size(px(if *failed { 12. } else { 24. }))
                                .font_weight(FontWeight::BOLD)
                                .text_color(if *failed {
                                    p.text_secondary
                                } else {
                                    p.text_inverse
                                })
                                .child(if *failed { "GIF failed to load" } else { "GIF" })
                                .when(*failed, |el| el.child(ui::chip("Retry", false, p))),
                        ),
                )
                .when_some(caption.clone(), |el, caption| {
                    el.child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(caption),
                    )
                })
                .into_any_element(),
            MessageBody::Date(_)
            | MessageBody::Notice(_)
            | MessageBody::System(_)
            | MessageBody::Unread(_) => div().into_any_element(),
        }
    }

    fn render_pinned_bar(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let label = if self.pinned.len() == 1 {
            "1 pinned message".to_owned()
        } else {
            format!("{} pinned", self.pinned.len())
        };
        self.clickable(
            cx,
            div()
                .id("pinned-bar")
                .h(px(32.))
                .flex()
                .items_center()
                .gap(px(8.))
                .px(px(16.))
                .bg(p.bg_notice_encrypt)
                .border_b_1()
                .border_color(p.divider)
                .text_size(px(TEXT_SMALL))
                .text_color(p.text_primary)
                .child(ui::icon(IconName::Pin, 14.0, p.text_secondary))
                .child(label)
                .child(div().flex_1())
                .child(ui::icon(IconName::ChevronRight, 14.0, p.text_secondary)),
            Action::OpenPinnedMessages,
        )
    }

    fn render_selection_bar(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let count = self.selected_messages.len();
        let all_text = self.selected_messages.iter().all(|index| {
            self.messages
                .get(self.selected_chat)
                .and_then(|messages| messages.get(*index))
                .map(|message| matches!(message.body, MessageBody::Text(_) | MessageBody::Rich(_)))
                .unwrap_or(false)
        });
        div()
            .h(px(48.))
            .flex_shrink_0()
            .flex()
            .items_center()
            .gap(px(8.))
            .px(px(16.))
            .bg(p.accent_soft)
            .border_t_1()
            .border_color(p.divider)
            .child(
                div()
                    .text_size(px(TEXT_SMALL))
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(p.unread_text)
                    .child(format!("{count} selected")),
            )
            .child(self.clickable(
                cx,
                ui::secondary_button("select-all-messages", "Select all", p),
                Action::SelectAllMessages,
            ))
            .when(all_text, |el| {
                el.child(self.clickable(
                    cx,
                    ui::secondary_button("selection-copy", "Copy", p),
                    Action::SelectionCopy,
                ))
            })
            .child(self.clickable(
                cx,
                ui::secondary_button("selection-forward", "Forward", p),
                Action::SelectionForward,
            ))
            .child(self.clickable(
                cx,
                ui::secondary_button("selection-star", "Star", p),
                Action::SelectionStar,
            ))
            .when(count == 1, |el| {
                el.child(self.clickable(
                    cx,
                    ui::secondary_button("selection-info", "Info", p),
                    Action::SelectionInfo,
                ))
            })
            .child(self.clickable(
                cx,
                ui::danger_button("selection-delete", "Delete", p),
                Action::SelectionDelete,
            ))
            .child(div().flex_1())
            .child(self.clickable(
                cx,
                ui::icon_button_small(
                    "selection-close",
                    IconName::X,
                    16.0,
                    p.text_secondary,
                    p.hover(),
                    "Exit selection",
                ),
                Action::ExitMessageSelection,
            ))
    }

    fn render_composer(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        if self.blocked_chats.contains(&self.selected_chat) {
            let name = self
                .chats
                .get(self.selected_chat)
                .map(|chat| chat.name.clone())
                .unwrap_or_else(|| "this contact".into());
            return div()
                .id("composer-blocked")
                .flex_shrink_0()
                .flex()
                .items_center()
                .gap(px(10.))
                .px(px(16.))
                .py(px(12.))
                .bg(p.bg_composer)
                .border_t_1()
                .border_color(p.divider)
                .child(ui::icon(IconName::Lock, 16.0, p.text_secondary))
                .child(
                    div()
                        .flex_1()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child(format!(
                            "You blocked {name}. Unblock to send messages or make calls."
                        )),
                )
                .child(self.clickable(
                    cx,
                    ui::secondary_button("unblock-chat", "Unblock", p),
                    Action::UnblockChat(self.selected_chat),
                ));
        }

        let has_text = !self.composer.read(cx).value().trim().is_empty();
        let can_send = has_text || !self.staged_attachments.is_empty();
        let mut wrap = div()
            .id("composer-wrap")
            .flex_shrink_0()
            .flex()
            .flex_col()
            .bg(p.bg_composer)
            .border_t_1()
            .border_color(p.divider)
            .on_key_down(cx.listener(|this, event: &KeyDownEvent, window, cx| {
                if event.keystroke.key == "enter" && !event.keystroke.modifiers.shift {
                    this.apply(Action::Send, window, cx);
                }
            }));

        if let Some(draft) = self.selected_chat().and_then(|chat| chat.draft.clone()) {
            wrap = wrap.child(
                div()
                    .h(px(28.))
                    .flex()
                    .items_center()
                    .gap(px(6.))
                    .px(px(16.))
                    .bg(p.accent_soft)
                    .text_size(px(TEXT_CAPTION))
                    .text_color(p.unread_text)
                    .child(ui::icon(IconName::Pencil, 12.0, p.accent))
                    .child("Draft restored")
                    .child(
                        div()
                            .flex_1()
                            .overflow_hidden()
                            .whitespace_nowrap()
                            .text_ellipsis()
                            .child(draft),
                    )
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "clear-draft",
                            IconName::X,
                            12.0,
                            p.text_secondary,
                            p.hover(),
                            "Discard draft",
                        ),
                        Action::ClearDraft,
                    )),
            );
        }

        if !self.staged_attachments.is_empty() {
            wrap = wrap.child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(6.))
                    .px(px(16.))
                    .py(px(6.))
                    .bg(p.bg_composer)
                    .border_b_1()
                    .border_color(p.divider)
                    .children(self.staged_attachments.iter().enumerate().map(
                        |(index, attachment)| {
                            let icon_name = match attachment.kind {
                                StagedKind::Image => IconName::Image,
                                StagedKind::File => IconName::FileText,
                                StagedKind::Contact => IconName::User,
                            };
                            let drag_app = cx.entity();
                            let palette = p;
                            let drag_label = attachment.label.clone();
                            let constructor =
                                move |label: &String,
                                      _offset,
                                      _window: &mut Window,
                                      cx: &mut gpui_kit::App| {
                                    let label = label.clone();
                                    drag_app.update(cx, |this, cx| {
                                        this.drag_file_hover = Some(this.selected_chat);
                                        cx.notify();
                                    });
                                    cx.new(|_| FileDragPreview { label, palette })
                                };
                            div()
                                .id(format!("staged-{index}"))
                                .h(px(40.))
                                .flex()
                                .items_center()
                                .gap(px(6.))
                                .px(px(8.))
                                .rounded(px(RADIUS_MD))
                                .bg(p.bg_list)
                                .border_1()
                                .border_color(p.border)
                                .child(ui::icon(icon_name, 14.0, p.accent))
                                .child(
                                    div()
                                        .max_w(px(120.))
                                        .overflow_hidden()
                                        .whitespace_nowrap()
                                        .text_ellipsis()
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.text_primary)
                                        .child(attachment.label.clone()),
                                )
                                .child(self.clickable(
                                    cx,
                                    ui::icon_button_small(
                                        format!("remove-staged-{index}"),
                                        IconName::X,
                                        10.0,
                                        p.text_secondary,
                                        p.hover(),
                                        "Remove attachment",
                                    ),
                                    Action::RemoveStagedAttachment(index),
                                ))
                                .on_drag(drag_label, constructor)
                        },
                    ))
                    .child(div().flex_1())
                    .child(
                        div()
                            .text_size(px(TEXT_CAPTION))
                            .text_color(p.text_secondary)
                            .child(format!("{} staged", self.staged_attachments.len())),
                    ),
            );
        }

        if self.recording {
            wrap = wrap.child(
                div()
                    .h(px(40.))
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .px(px(16.))
                    .bg(p.bg_notice_encrypt)
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_primary)
                    .child(
                        div()
                            .size(px(8.))
                            .rounded_full()
                            .bg(p.danger())
                            .with_animation(
                                "recording-dot",
                                Animation::new(Duration::from_millis(800)).repeat(),
                                |el, delta| el.opacity(0.3 + 0.7 * delta),
                            ),
                    )
                    .child(format!("Recording · 0:{:02}", self.recording_seconds))
                    .child(div().flex_1())
                    .child(self.clickable(
                        cx,
                        ui::secondary_button("cancel-recording", "Cancel", p),
                        Action::ToggleRecording,
                    ))
                    .child(self.clickable(
                        cx,
                        ui::primary_button("send-recording", "Send voice note", p),
                        Action::ToggleRecording,
                    )),
            );
        }

        if let Some(index) = self.call_active {
            if let Some(call) = self.calls.get(index) {
                wrap = wrap.child(
                    div()
                        .h(px(48.))
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .px(px(16.))
                        .py(px(6.))
                        .bg(p.bg_list)
                        .border_b_1()
                        .border_color(p.divider)
                        .child(ui::avatar(call.initials.clone(), 32., false, p))
                        .child(
                            div()
                                .flex()
                                .flex_col()
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::SEMIBOLD)
                                        .text_color(p.text_primary)
                                        .child(call.name.clone()),
                                )
                                .child(
                                    div()
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.text_secondary)
                                        .child("00:42 · end-to-end encrypted"),
                                ),
                        )
                        .child(div().flex_1())
                        .child(self.clickable(
                            cx,
                            ui::icon_button_small(
                                "call-bar-mute",
                                if self.call_muted {
                                    IconName::MicOff
                                } else {
                                    IconName::Mic
                                },
                                16.0,
                                p.text_primary,
                                p.hover(),
                                "Mute call",
                            ),
                            Action::ToggleMute,
                        ))
                        .child(
                            self.clickable(
                                cx,
                                ui::icon_button_small(
                                    "call-bar-end",
                                    IconName::PhoneOff,
                                    16.0,
                                    p.text_inverse,
                                    p.danger(),
                                    "End call",
                                )
                                .bg(p.danger()),
                                Action::EndCall,
                            ),
                        ),
                );
            }
        }

        if let Some(index) = self.reply_to {
            if let Some(message) = self
                .messages
                .get(self.selected_chat)
                .and_then(|messages| messages.get(index))
            {
                wrap = wrap.child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .px(px(16.))
                        .py(px(6.))
                        .border_b_1()
                        .border_color(p.divider)
                        .child(
                            div()
                                .flex_1()
                                .pl(px(8.))
                                .border_l_2()
                                .border_color(p.accent)
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::SEMIBOLD)
                                        .text_color(p.accent)
                                        .child(
                                            message
                                                .sender_name
                                                .clone()
                                                .unwrap_or_else(|| "You".into()),
                                        ),
                                )
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .text_color(p.text_secondary)
                                        .child(message.text().unwrap_or("Attachment").to_owned()),
                                ),
                        )
                        .child(self.clickable(
                            cx,
                            ui::icon_button_small(
                                "cancel-reply",
                                IconName::X,
                                16.0,
                                p.text_secondary,
                                p.hover(),
                                "Cancel reply",
                            ),
                            Action::Reply(index),
                        )),
                );
            }
        }

        if let Some(index) = self.editing {
            wrap = wrap.child(
                div()
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .py(px(6.))
                    .border_b_1()
                    .border_color(p.divider)
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.accent)
                            .child("Editing message"),
                    )
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "cancel-edit",
                            IconName::X,
                            16.0,
                            p.text_secondary,
                            p.hover(),
                            "Cancel edit",
                        ),
                        Action::Toast("Edit cancelled".into()),
                    )),
            );
            let _ = index;
        }

        wrap.child(
            div()
                .min_h(px(COMPOSER_HEIGHT))
                .flex()
                .items_end()
                .gap(px(8.))
                .px(px(16.))
                .py(px(8.))
                .child(
                    self.clickable(
                        cx,
                        div()
                            .id("attach")
                            .role(Role::Button)
                            .tab_index(0)
                            .aria_label("Attach")
                            .cursor_pointer()
                            .size(px(38.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .hover(move |style| style.bg(p.hover()))
                            .child(ui::icon(IconName::Plus, 20.0, p.text_primary)),
                        Action::OpenAttachments,
                    ),
                )
                .child(
                    div()
                        .flex_1()
                        .min_w(px(0.))
                        .min_h(px(40.))
                        .max_h(px(108.))
                        .flex()
                        .items_center()
                        .rounded(px(RADIUS_XL))
                        .bg(p.bg_list)
                        .border_1()
                        .border_color(p.border)
                        .px(px(12.))
                        .py(px(8.))
                        .child(
                            Textarea::new(&self.composer)
                                .appearance(false)
                                .bordered(false)
                                .aria_label("Message"),
                        ),
                )
                .child(self.clickable(
                    cx,
                    ui::icon_button(
                        "emoji",
                        IconName::CircleDot,
                        20.0,
                        p.text_secondary,
                        p.hover(),
                        "Emoji",
                    ),
                    Action::OpenEmoji,
                ))
                .child(
                    self.clickable(
                        cx,
                        div()
                            .id("send")
                            .role(Role::Button)
                            .tab_index(0)
                            .aria_label(if has_text {
                                "Send message"
                            } else {
                                "Record voice message"
                            })
                            .cursor_pointer()
                            .size(px(40.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.accent)
                            .child(ui::icon(
                                if can_send {
                                    IconName::Send
                                } else {
                                    IconName::Mic
                                },
                                18.0,
                                p.text_inverse,
                            )),
                        if !can_send {
                            Action::ToggleRecording
                        } else if self.staged_attachments.is_empty() {
                            Action::Send
                        } else {
                            Action::SendStaged
                        },
                    ),
                ),
        )
    }

    // ------------------------------------------------------------------
    // Secondary surfaces. These are filled out in the following sections.
    // ------------------------------------------------------------------

    fn render_onboarding(&mut self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let qr = render_qr(self.qr_state, p);
        let (title, body) = match self.qr_state {
            QrState::Loading => (
                "Requesting a fresh code…",
                "First paint while the code is requested.",
            ),
            QrState::Active => (
                "Link with phone number",
                "Open WhatsLite on your phone, tap Menu → Linked devices, then point the camera at this code.",
            ),
            QrState::Expired => (
                "Code expired",
                "Codes last 60 seconds by design. Refresh to scan again.",
            ),
            QrState::Success => ("Syncing your messages", "3.240 of 8.112 · about 2 minutes"),
        };

        div()
            .flex_1()
            .flex()
            .items_center()
            .justify_center()
            .bg(p.bg_conversation)
            .child(
                div()
                    .w(px(640.))
                    .flex()
                    .flex_col()
                    .items_center()
                    .gap(px(18.))
                    .p(px(40.))
                    .rounded(px(RADIUS_XL))
                    .bg(p.bg_list)
                    .border_1()
                    .border_color(p.border)
                    .child(app_mark_large(p))
                    .child(
                        div()
                            .text_size(px(28.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_primary)
                            .child(title),
                    )
                    .child(
                        div()
                            .max_w(px(480.))
                            .text_center()
                            .text_size(px(TEXT_BODY))
                            .text_color(p.text_secondary)
                            .child(body),
                    )
                    .when(self.qr_state != QrState::Success, |el| el.child(qr))
                    .when(self.qr_state == QrState::Success, |el| {
                        el.child(
                            div()
                                .w(px(420.))
                                .h(px(6.))
                                .rounded_full()
                                .bg(p.bg_search)
                                .child(div().h(px(6.)).w(px(168.)).rounded_full().bg(p.accent)),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child(
                                    "Keep your phone connected while WhatsLite finishes syncing.",
                                ),
                        )
                    })
                    .child(
                        self.clickable(
                            cx,
                            div()
                                .id("qr-refresh")
                                .h(px(36.))
                                .px(px(16.))
                                .flex()
                                .items_center()
                                .gap(px(8.))
                                .rounded_full()
                                .bg(p.bg_search)
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_primary)
                                .child(ui::icon(IconName::RefreshCw, 15.0, p.text_secondary))
                                .child(match self.qr_state {
                                    QrState::Active => "Refresh code · expires in 0:48",
                                    QrState::Loading => "Requesting a fresh code…",
                                    QrState::Expired => "Refresh code",
                                    QrState::Success => "Open WhatsLite",
                                }),
                            if self.qr_state == QrState::Success {
                                Action::Navigate(Route::Chats)
                            } else {
                                Action::CycleQr
                            },
                        ),
                    ),
            )
    }

    fn render_calls_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let rows = self
            .calls
            .iter()
            .map(|call| {
                let row = div()
                    .id(format!("call-row-{}", call.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(format!("{} · {}", call.name, call.time))
                    .cursor_pointer()
                    .h(px(76.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .border_b_1()
                    .border_color(p.divider)
                    .hover(move |style| style.bg(p.hover()))
                    .child(ui::avatar(call.initials.clone(), 44., false, p))
                    .child(
                        div()
                            .flex_1()
                            .flex()
                            .flex_col()
                            .gap(px(3.))
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(call.name.clone()),
                            )
                            .child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(5.))
                                    .child(ui::call_direction_icon(call.missed, call.incoming, p))
                                    .child(
                                        div()
                                            .text_size(px(TEXT_SMALL))
                                            .text_color(p.text_secondary)
                                            .child(call_label(call)),
                                    ),
                            ),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .items_end()
                            .gap(px(6.))
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_secondary)
                                    .child(call.time.clone()),
                            )
                            .child(ui::icon(IconName::Phone, 15.0, p.accent)),
                    );
                self.clickable(cx, row, Action::StartCall(call.id, call.video))
            })
            .collect::<Vec<_>>();

        let banner = if self.missed_banner {
            self.calls
                .iter()
                .find(|call| call.missed)
                .cloned()
                .map(|call| {
                    let id = call.id;
                    let video = call.video;
                    let name = call.name.clone();
                    div()
                        .id("missed-call-banner")
                        .h(px(32.))
                        .flex()
                        .items_center()
                        .gap(px(6.))
                        .px(px(12.))
                        .bg(p.bg_notice_encrypt)
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_primary)
                        .child(ui::icon(IconName::PhoneMissed, 13.0, p.danger()))
                        .child(format!("Missed call from {name}"))
                        .child(div().flex_1())
                        .child(self.clickable(
                            cx,
                            ui::chip("Call back", false, p),
                            Action::StartCall(id, video),
                        ))
                        .child(self.clickable(
                            cx,
                            ui::icon_button_small(
                                "dismiss-missed",
                                IconName::X,
                                12.0,
                                p.text_secondary,
                                p.hover(),
                                "Dismiss missed call",
                            ),
                            Action::DismissMissedCall,
                        ))
                })
        } else {
            None
        };

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(section_header(
                cx,
                p,
                "Calls",
                Some(IconName::Plus),
                Action::StartCall(0, false),
            ))
            .child(
                div()
                    .h(px(36.))
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .px(px(16.))
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child(ui::icon(IconName::Lock, 14.0, p.text_secondary))
                    .child("Your calls are end-to-end encrypted"),
            )
            .when_some(banner, |el, banner| el.child(banner))
            .child(self.list_body(
                cx,
                p,
                rows,
                "No calls yet",
                "Start a voice or video call from any chat.",
            ))
    }

    fn render_calls_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        if let Some(index) = self.call_active {
            if let Some(call) = self.calls.get(index) {
                let media = if call.group {
                    let size = self.group_call_size.clamp(2, 6);
                    div()
                        .grid()
                        .grid_cols(if size <= 4 { 2 } else { 3 })
                        .gap(px(8.))
                        .w(px(if size <= 4 { 420. } else { 560. }))
                        .children((0..size).map(|index| {
                            let names = ["Rina", "Bayu", "Sari", "You", "Dewi", "Budi"];
                            let initials = ["RI", "BA", "SA", "Y", "DL", "BS"];
                            let pinned = self.call_pinned == Some(index);
                            let speaking = index == 0 && self.call_state == CallState::Connected;
                            div()
                                .relative()
                                .h(px(130.))
                                .rounded(px(RADIUS_LG))
                                .bg(if pinned || speaking {
                                    p.accent_soft
                                } else {
                                    p.bg_list
                                })
                                .border_2()
                                .border_color(if pinned {
                                    p.attach_orange
                                } else if speaking {
                                    p.accent
                                } else {
                                    p.border
                                })
                                .flex()
                                .flex_col()
                                .items_center()
                                .justify_center()
                                .gap(px(6.))
                                .child(ui::avatar(initials[index], 52., false, p))
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(p.text_primary)
                                        .child(names[index]),
                                )
                                .when(speaking, |el| el.child(ui::chip("Speaking", true, p)))
                                .when(pinned, |el| el.child(ui::chip("Pinned", false, p)))
                                .child(
                                    self.clickable(
                                        cx,
                                        div()
                                            .id(format!("pin-participant-{index}"))
                                            .absolute()
                                            .top(px(6.))
                                            .right(px(6.))
                                            .size(px(24.))
                                            .flex()
                                            .items_center()
                                            .justify_center()
                                            .rounded_full()
                                            .bg(if pinned { p.attach_orange } else { p.bg_list })
                                            .child(ui::icon(
                                                IconName::Pin,
                                                12.0,
                                                if pinned {
                                                    p.text_inverse
                                                } else {
                                                    p.text_secondary
                                                },
                                            )),
                                        Action::PinCallParticipant(index),
                                    ),
                                )
                        }))
                        .into_any_element()
                } else if self.call_video {
                    div()
                        .relative()
                        .w(px(520.))
                        .h(px(320.))
                        .rounded(px(RADIUS_LG))
                        .bg(p.attach_blue)
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(ui::icon(IconName::Video, 64.0, p.text_inverse))
                        .child(
                            div()
                                .absolute()
                                .bottom(px(12.))
                                .right(px(12.))
                                .w(px(96.))
                                .h(px(128.))
                                .rounded(px(RADIUS_MD))
                                .bg(p.attach_orange)
                                .border_2()
                                .border_color(p.bg_list)
                                .flex()
                                .items_center()
                                .justify_center()
                                .child(ui::icon(IconName::User, 28.0, p.text_inverse)),
                        )
                        .into_any_element()
                } else {
                    ui::avatar(call.initials.clone(), 112., true, p).into_any_element()
                };

                let state_label = match self.call_state {
                    CallState::Dialing => "Dialing…",
                    CallState::Ringing => "Ringing…",
                    CallState::Connecting => "Connecting…",
                    CallState::Connected => "00:42 · end-to-end encrypted",
                    CallState::Reconnecting => "Reconnecting…",
                    CallState::Busy => "Busy · try again later",
                    CallState::NoAnswer => "No answer",
                    CallState::EndedByThem => "Call ended by them",
                    CallState::Declined => "Declined by you",
                };
                let call_kind = if call.group {
                    if self.call_video {
                        "Group video call"
                    } else {
                        "Group voice call"
                    }
                } else if self.call_video {
                    "Video call"
                } else {
                    "Voice call"
                };

                return div()
                    .flex_1()
                    .h_full()
                    .flex()
                    .flex_col()
                    .items_center()
                    .justify_center()
                    .gap(px(16.))
                    .bg(p.bg_conversation)
                    .child(media)
                    .child(
                        div()
                            .text_size(px(24.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child(call.name.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .items_center()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .text_color(p.text_secondary)
                                    .child(call_kind),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .text_color(p.text_secondary)
                                    .child(state_label),
                            )
                            .when(self.call_state == CallState::Reconnecting, |el| {
                                el.child(
                                    div()
                                        .mt(px(4.))
                                        .px(px(8.))
                                        .py(px(3.))
                                        .rounded_full()
                                        .bg(p.bg_notice_encrypt)
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.attach_orange)
                                        .child("Poor connection · audio may drop"),
                                )
                            }),
                    )
                    .when(
                        self.call_incoming && self.call_state == CallState::Ringing,
                        |el| {
                            el.child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(12.))
                                    .child(
                                        self.clickable(
                                            cx,
                                            div()
                                                .id("decline-call")
                                                .role(Role::Button)
                                                .tab_index(0)
                                                .aria_label("Decline call")
                                                .cursor_pointer()
                                                .size(px(52.))
                                                .flex()
                                                .items_center()
                                                .justify_center()
                                                .rounded_full()
                                                .bg(p.danger())
                                                .child(ui::icon(
                                                    IconName::PhoneOff,
                                                    22.0,
                                                    p.text_inverse,
                                                )),
                                            Action::DeclineCall,
                                        ),
                                    )
                                    .child(
                                        self.clickable(
                                            cx,
                                            div()
                                                .id("accept-call")
                                                .role(Role::Button)
                                                .tab_index(0)
                                                .aria_label("Accept call")
                                                .cursor_pointer()
                                                .size(px(52.))
                                                .flex()
                                                .items_center()
                                                .justify_center()
                                                .rounded_full()
                                                .bg(p.accent)
                                                .child(ui::icon(
                                                    IconName::Phone,
                                                    22.0,
                                                    p.text_inverse,
                                                )),
                                            Action::AcceptCall,
                                        ),
                                    ),
                            )
                        },
                    )
                    .when(
                        !(self.call_incoming && self.call_state == CallState::Ringing),
                        |el| {
                            el.child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(12.))
                                    .child(self.clickable(
                                        cx,
                                        call_control(
                                            "call-mute",
                                            if self.call_muted {
                                                IconName::MicOff
                                            } else {
                                                IconName::Mic
                                            },
                                            self.call_muted,
                                            p,
                                        ),
                                        Action::ToggleMute,
                                    ))
                                    .child(self.clickable(
                                        cx,
                                        call_control(
                                            "call-video",
                                            if self.call_video {
                                                IconName::Video
                                            } else {
                                                IconName::VideoOff
                                            },
                                            self.call_video,
                                            p,
                                        ),
                                        Action::ToggleVideo,
                                    ))
                                    .child(
                                        self.clickable(
                                            cx,
                                            div()
                                                .id("call-end")
                                                .role(Role::Button)
                                                .tab_index(0)
                                                .aria_label("End call")
                                                .cursor_pointer()
                                                .size(px(52.))
                                                .flex()
                                                .items_center()
                                                .justify_center()
                                                .rounded_full()
                                                .bg(p.danger())
                                                .child(ui::icon(
                                                    IconName::PhoneOff,
                                                    22.0,
                                                    p.text_inverse,
                                                )),
                                            Action::EndCall,
                                        ),
                                    ),
                            )
                        },
                    );
            }
        }

        div()
            .flex_1()
            .h_full()
            .flex()
            .items_center()
            .justify_center()
            .bg(p.bg_conversation)
            .child(ui::empty_state(
                "No active call",
                "Pick a contact from the list or start a voice call. The call window, states and controls are local previews.",
                Some(self.clickable(
                    cx,
                    ui::primary_button("start-call", "Start a voice call", p),
                    Action::StartCall(0, false),
                )),
                p,
            ))
    }

    fn render_updates_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let rows = self
            .statuses
            .iter()
            .map(|status| {
                let row = div()
                    .id(format!("status-row-{}", status.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(format!("{} · {}", status.name, status.time))
                    .cursor_pointer()
                    .h(px(72.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .hover(move |style| style.bg(p.hover()))
                    .child(if status.id == 0 {
                        ui::avatar("Y", 44., false, p)
                    } else {
                        ui::avatar_with_ring(status.initials.clone(), 44., status.seen, p)
                    })
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap(px(3.))
                            .child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(5.))
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(status.name.clone())
                                    .when(status.muted, |el| {
                                        el.child(ui::icon(
                                            IconName::BellOff,
                                            12.0,
                                            p.text_secondary,
                                        ))
                                    }),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(status.time.clone()),
                            ),
                    );
                self.clickable(cx, row, Action::OpenStatus(status.id))
            })
            .collect::<Vec<_>>();

        let mut channel = |name: &'static str, preview: &'static str| {
            self.clickable(
                cx,
                div()
                    .id(format!("channel-{name}"))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .py(px(8.))
                    .hover(move |style| style.bg(p.hover()))
                    .child(
                        div()
                            .size(px(38.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.accent_soft)
                            .child(ui::icon(IconName::Megaphone, 18.0, p.accent)),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(name),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(preview),
                            ),
                    ),
                Action::OpenChannelInfo(name.to_owned()),
            )
        };

        let mut body_rows: Vec<gpui_kit::AnyElement> = Vec::new();
        body_rows.push(ui::section_label("Recent updates", p).into_any_element());
        body_rows.extend(rows.into_iter().map(|row| row.into_any_element()));
        body_rows.push(ui::section_label("CHANNELS", p).into_any_element());
        body_rows.push(channel("Design Weekly", "Calm interfaces, issue 14").into_any_element());
        body_rows.push(channel("Studio Linen", "Notes on weaving · 2 new").into_any_element());

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Updates"),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(2.))
                            .child(self.clickable(
                                cx,
                                ui::icon_button(
                                    "updates-camera",
                                    IconName::Camera,
                                    18.0,
                                    p.text_primary,
                                    p.hover(),
                                    "Post a status",
                                ),
                                Action::OpenStatusComposer,
                            ))
                            .child(self.clickable(
                                cx,
                                ui::icon_button(
                                    "updates-pencil",
                                    IconName::Pencil,
                                    18.0,
                                    p.text_primary,
                                    p.hover(),
                                    "Text status",
                                ),
                                Action::OpenStatusComposer,
                            )),
                    ),
            )
            .child(self.list_body(
                cx,
                p,
                body_rows,
                "No updates yet",
                "Post a status or join a channel.",
            ))
    }

    fn render_updates_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        if let Some(index) = self.status_viewer {
            if let Some(status) = self.statuses.get(index) {
                return self.render_status_viewer(cx, p, status);
            }
        }

        div()
            .flex_1()
            .h_full()
            .flex()
            .items_center()
            .justify_center()
            .bg(p.bg_conversation)
            .child(ui::empty_state(
                "Post your first status",
                "Photos, videos and text blocks share one sequence. Post a status to see the viewer, filmstrip and seen-by states.",
                Some(self.clickable(
                    cx,
                    ui::primary_button("post-status", "Post a status", p),
                    Action::OpenStatusComposer,
                )),
                p,
            ))
    }

    fn render_status_viewer(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        status: &StatusUpdate,
    ) -> gpui_kit::Div {
        div()
            .flex_1()
            .relative()
            .flex()
            .flex_col()
            .bg(p.text_primary)
            .child(div().flex().items_center().gap(px(6.)).p(px(16.)).children(
                (0..status.items.max(1)).map(|index| {
                    div().flex_1().h(px(3.)).rounded_full().bg(if index == 0 {
                        p.accent
                    } else {
                        p.text_secondary
                    })
                }),
            ))
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(10.))
                    .px(px(16.))
                    .child(ui::avatar(status.initials.clone(), 36., false, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_inverse)
                                    .child(status.name.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_inverse)
                                    .child(status.time.clone()),
                            ),
                    )
                    .child(div().flex_1())
                    .child(
                        div()
                            .text_size(px(TEXT_CAPTION))
                            .text_color(p.text_inverse)
                            .child("2 of 5"),
                    )
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "status-pause",
                            if self.status_paused {
                                IconName::Play
                            } else {
                                IconName::Pause
                            },
                            18.0,
                            p.text_inverse,
                            p.text_primary,
                            "Pause status",
                        ),
                        Action::ToggleStatusPause,
                    ))
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "close-status",
                            IconName::X,
                            18.0,
                            p.text_inverse,
                            p.text_primary,
                            "Close status",
                        ),
                        Action::CloseStatus,
                    )),
            )
            .child(
                div().flex_1().flex().items_center().justify_center().child(
                    div()
                        .w(px(420.))
                        .h(px(560.))
                        .rounded(px(RADIUS_XL))
                        .bg(if status.accent % 2 == 0 {
                            p.attach_orange
                        } else {
                            p.attach_blue
                        })
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(
                            div()
                                .max_w(px(320.))
                                .text_center()
                                .text_size(px(20.))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(p.text_inverse)
                                .child(status.caption.clone()),
                        ),
                ),
            )
            .child(
                div()
                    .px(px(16.))
                    .pb(px(8.))
                    .flex()
                    .items_center()
                    .justify_center()
                    .gap(px(6.))
                    .children((0..5).map(|index| {
                        div()
                            .w(px(52.))
                            .h(px(38.))
                            .rounded(px(RADIUS_SM))
                            .bg(if index == 1 {
                                p.attach_orange
                            } else {
                                p.attach_blue
                            })
                            .opacity(if index == 1 { 1.0 } else { 0.55 })
                    })),
            )
            .child(
                self.clickable(
                    cx,
                    div()
                        .id("status-viewers")
                        .px(px(16.))
                        .pb(px(4.))
                        .flex()
                        .items_center()
                        .gap(px(6.))
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_inverse)
                        .child(ui::icon(IconName::Eye, 14.0, p.text_inverse))
                        .child("Seen by 12")
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child("· tap to see viewers"),
                        ),
                    Action::OpenStatusViewers,
                ),
            )
            .child(
                div()
                    .p(px(16.))
                    .flex()
                    .items_center()
                    .gap(px(10.))
                    .child(
                        div()
                            .flex_1()
                            .h(px(40.))
                            .rounded_full()
                            .border_1()
                            .border_color(p.text_secondary)
                            .px(px(16.))
                            .flex()
                            .items_center()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Reply to status…"),
                    )
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "status-send",
                            IconName::Send,
                            18.0,
                            p.text_inverse,
                            p.text_primary,
                            "Reply to status",
                        ),
                        Action::Toast("Reply sent to Rizky · local mock".into()),
                    )),
            )
    }

    fn render_communities_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let rows = self
            .communities
            .iter()
            .map(|community| {
                let row = div()
                    .id(format!("community-row-{}", community.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(community.name.clone())
                    .cursor_pointer()
                    .h(px(76.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .border_b_1()
                    .border_color(p.divider)
                    .hover(move |style| style.bg(p.hover()))
                    .child(ui::avatar(community.initials.clone(), 44., false, p))
                    .child(
                        div()
                            .flex_1()
                            .flex()
                            .flex_col()
                            .gap(px(3.))
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(community.name.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(format!("{} · {}", community.groups, community.members)),
                            ),
                    )
                    .when(community.unread > 0, |el| {
                        el.child(ui::badge(community.unread, p))
                    })
                    .when(!community.joined, |el| el.child(ui::chip("View", false, p)));
                self.clickable(cx, row, Action::SelectCommunity(community.id))
            })
            .collect::<Vec<_>>();

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(section_header(cx, p, "Communities", Some(IconName::Plus), Action::Toast("Coming soon — creating a community is not wired up.".into())))
            .child(
                div()
                    .px(px(16.))
                    .pb(px(8.))
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child("Communities let admins reach everyone, while members talk in smaller groups."),
            )
            .child(ui::section_label("YOUR COMMUNITIES", p))
            .child(self.list_body(
                cx,
                p,
                rows,
                "No communities yet",
                "Create a community to bring related groups together.",
            ))
    }

    fn render_communities_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let Some(index) = self.selected_community else {
            return div()
                .flex_1()
                .h_full()
                .flex()
                .items_center()
                .justify_center()
                .bg(p.bg_conversation)
                .child(ui::empty_state(
                    "Start a community",
                    "Communities let admins reach everyone while members talk in smaller groups. Create one locally to preview the flow.",
                    Some(self.clickable(
                        cx,
                        ui::primary_button("new-community-empty", "New community", p),
                        Action::NewCommunity,
                    )),
                    p,
                ));
        };
        let Some(community) = self.communities.get(index) else {
            return div().flex_1().bg(p.bg_conversation);
        };

        div()
            .flex_1()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_conversation)
            .child(
                div()
                    .h(px(CONVERSATION_HEADER_HEIGHT))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .bg(p.bg_header)
                    .border_b_1()
                    .border_color(p.divider)
                    .child(ui::avatar(community.initials.clone(), 40., false, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(community.name.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_secondary)
                                    .child(format!("{} · only admins can post", community.members)),
                            ),
                    )
                    .child(div().flex_1())
                    .child(self.clickable(
                        cx,
                        ui::icon_button(
                            "community-info",
                            IconName::Ellipsis,
                            19.0,
                            p.text_primary,
                            p.hover(),
                            "Community info",
                        ),
                        Action::OpenInfo(InfoPanel::Community),
                    )),
            )
            .child(
                div()
                    .px(px(16.))
                    .py(px(8.))
                    .bg(p.accent_soft)
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.unread_text)
                    .child("Only admins can post to Announcements."),
            )
            .child(
                div()
                    .flex_1()
                    .overflow_y_scrollbar()
                    .p(px(16.))
                    .flex()
                    .flex_col()
                    .gap(px(8.))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(10.))
                            .p(px(12.))
                            .rounded(px(RADIUS_LG))
                            .bg(p.bg_list)
                            .child(ui::icon(IconName::Megaphone, 20.0, p.accent))
                            .child(
                                div()
                                    .flex()
                                    .flex_col()
                                    .child(
                                        div()
                                            .text_size(px(TEXT_BODY))
                                            .font_weight(FontWeight::SEMIBOLD)
                                            .text_color(p.text_primary)
                                            .child("Announcements"),
                                    )
                                    .child(
                                        div()
                                            .text_size(px(TEXT_SMALL))
                                            .text_color(p.text_secondary)
                                            .child("Rina: Restock Jumat pagi"),
                                    ),
                            ),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap(px(6.))
                            .child(ui::section_label(
                                format!("GROUPS · {}", community.groups),
                                p,
                            ))
                            .children(
                                [
                                    ("Produksi", "12 members"),
                                    ("Desain Motif", "8 members"),
                                    ("Penjualan", "24 members"),
                                    ("Arsip Foto", "6 members"),
                                ]
                                .into_iter()
                                .map(|(name, members)| {
                                    div()
                                        .flex()
                                        .items_center()
                                        .gap(px(10.))
                                        .p(px(10.))
                                        .rounded(px(RADIUS_LG))
                                        .bg(p.bg_list)
                                        .child(ui::avatar(initials_for(Some(name)), 34., false, p))
                                        .child(
                                            div()
                                                .flex()
                                                .flex_col()
                                                .child(
                                                    div()
                                                        .text_size(px(13.))
                                                        .font_weight(FontWeight::MEDIUM)
                                                        .text_color(p.text_primary)
                                                        .child(name),
                                                )
                                                .child(
                                                    div()
                                                        .text_size(px(TEXT_CAPTION))
                                                        .text_color(p.text_secondary)
                                                        .child(members),
                                                ),
                                        )
                                }),
                            ),
                    ),
            )
            .child(
                div()
                    .h(px(COMPOSER_HEIGHT))
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .px(px(16.))
                    .bg(p.bg_composer)
                    .border_t_1()
                    .border_color(p.divider)
                    .child(
                        div()
                            .flex_1()
                            .h(px(40.))
                            .rounded(px(RADIUS_XL))
                            .bg(if self.community_admin {
                                p.bg_list
                            } else {
                                p.bg_search
                            })
                            .border_1()
                            .border_color(p.border)
                            .px(px(12.))
                            .flex()
                            .items_center()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(if self.community_admin {
                                "Announce to 1.2k members"
                            } else {
                                "Only admins can post to Announcements"
                            }),
                    )
                    .when(self.community_admin, |el| {
                        el.child(
                            self.clickable(
                                cx,
                                ui::icon_button(
                                    "send-announcement",
                                    IconName::Send,
                                    18.0,
                                    p.text_inverse,
                                    p.accent,
                                    "Send announcement",
                                )
                                .bg(p.accent),
                                Action::SendAnnouncement,
                            ),
                        )
                    }),
            )
    }

    fn render_media_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let all_row = self.clickable(
            cx,
            div()
                .id("media-all")
                .h(px(64.))
                .flex()
                .items_center()
                .gap(px(12.))
                .px(px(16.))
                .bg(if self.selected_media_chat.is_none() {
                    p.bg_row_selected
                } else {
                    p.bg_list
                })
                .hover(move |style| style.bg(p.bg_row_selected))
                .child(
                    div()
                        .size(px(38.))
                        .flex()
                        .items_center()
                        .justify_center()
                        .rounded(px(RADIUS_MD))
                        .bg(p.accent_soft)
                        .text_size(px(TEXT_SMALL))
                        .font_weight(FontWeight::BOLD)
                        .text_color(p.accent)
                        .child("AM"),
                )
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .child(
                            div()
                                .text_size(px(TEXT_BODY))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("All media"),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("128 files"),
                        ),
                ),
            Action::SelectMediaChat(String::new()),
        );
        let mut rows = self
            .chats
            .iter()
            .take(5)
            .map(|chat| {
                let count = self
                    .media
                    .iter()
                    .filter(|item| item.chat == chat.name)
                    .count();
                let row = div()
                    .id(format!("media-chat-{}", chat.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(chat.name.clone())
                    .cursor_pointer()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .hover(move |style| style.bg(p.hover()))
                    .child(ui::avatar(chat.initials.clone(), 38., false, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(chat.name.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(format!("{count} files")),
                            ),
                    );
                self.clickable(cx, row, Action::SelectMediaChat(chat.name.clone()))
            })
            .collect::<Vec<_>>();
        rows.insert(0, all_row);

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(section_header(
                cx,
                p,
                "Media",
                Some(IconName::Ellipsis),
                Action::Toast("Coming soon — media settings are not wired up.".into()),
            ))
            .child(
                div()
                    .mx(px(16.))
                    .mb(px(10.))
                    .h(px(34.))
                    .flex()
                    .rounded_full()
                    .bg(p.bg_search)
                    .p(px(3.))
                    .child(
                        div()
                            .flex_1()
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.bg_list)
                            .text_size(px(TEXT_SMALL))
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(p.text_primary)
                            .child("All media"),
                    )
                    .child(
                        div()
                            .flex_1()
                            .flex()
                            .items_center()
                            .justify_center()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Chats"),
                    ),
            )
            .child(self.list_body(
                cx,
                p,
                rows,
                "No media chats yet",
                "Photos, videos and documents from your chats appear here.",
            ))
    }

    fn render_media_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let tabs = [
            (MediaTab::Media, "Media"),
            (MediaTab::Links, "Links"),
            (MediaTab::Docs, "Docs"),
        ];
        let items = self
            .media
            .iter()
            .filter(|item| !self.media_deleted.contains(&item.id))
            .filter(|item| {
                self.selected_media_chat
                    .as_ref()
                    .is_none_or(|chat| &item.chat == chat)
            })
            .filter(|item| match self.media_tab {
                MediaTab::Media => matches!(item.kind, MediaKind::Image | MediaKind::Video),
                MediaTab::Links => matches!(item.kind, MediaKind::Link),
                MediaTab::Docs => matches!(item.kind, MediaKind::Document),
            })
            .collect::<Vec<_>>();

        let grid = items
            .iter()
            .map(|item| {
                let selected = self.media_selected.contains(&item.id);
                let starred = self.media_starred.contains(&item.id);
                let tile = div()
                    .id(format!("media-tile-{}", item.id))
                    .relative()
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(item.name.clone())
                    .cursor_pointer()
                    .child(ui::media_tile(
                        item.name.clone(),
                        item.duration.clone().unwrap_or_else(|| item.meta.clone()),
                        media_icon(item),
                        item.accent,
                        p,
                    ))
                    .when(starred, |el| {
                        el.child(
                            div()
                                .absolute()
                                .top(px(6.))
                                .left(px(6.))
                                .size(px(20.))
                                .flex()
                                .items_center()
                                .justify_center()
                                .rounded_full()
                                .bg(p.bg_list)
                                .child(ui::icon(IconName::Star, 12.0, p.attach_orange)),
                        )
                    })
                    .when(self.media_select_mode, |el| {
                        el.child(
                            div()
                                .absolute()
                                .top(px(6.))
                                .right(px(6.))
                                .size(px(20.))
                                .flex()
                                .items_center()
                                .justify_center()
                                .rounded_full()
                                .bg(if selected { p.accent } else { p.bg_list })
                                .border_1()
                                .border_color(if selected { p.accent } else { p.border })
                                .when(selected, |el| {
                                    el.child(ui::icon(IconName::Check, 12.0, p.text_inverse))
                                }),
                        )
                    });
                let action = if self.media_select_mode {
                    Action::ToggleMediaSelection(item.id)
                } else {
                    Action::OpenMedia(item.id)
                };
                self.clickable(cx, tile, action)
            })
            .collect::<Vec<_>>();

        div()
            .flex_1()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .child(
                div()
                    .h(px(72.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(24.))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(20.))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(self.selected_media_chat.clone().unwrap_or_else(|| {
                                        match self.media_tab {
                                            MediaTab::Media => "All media".to_owned(),
                                            MediaTab::Links => "Links".to_owned(),
                                            MediaTab::Docs => "Docs".to_owned(),
                                        }
                                    })),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(match self.list_state {
                                        ListState::Loading => "Loading media…".to_owned(),
                                        ListState::Failed => "Could not load media".to_owned(),
                                        ListState::Ready => match self.media_tab {
                                            MediaTab::Media => {
                                                "128 photos and videos across 6 chats".to_owned()
                                            }
                                            MediaTab::Links => {
                                                "Links shared in the chat, newest first".to_owned()
                                            }
                                            MediaTab::Docs => {
                                                "Documents shared in the chat, newest first"
                                                    .to_owned()
                                            }
                                        },
                                    }),
                            ),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(4.))
                            .children(tabs.into_iter().map(|(tab, label)| {
                                let selected = self.media_tab == tab;
                                self.clickable(
                                    cx,
                                    div()
                                        .id(format!("media-tab-{label}"))
                                        .role(Role::Button)
                                        .tab_index(0)
                                        .aria_label(label)
                                        .cursor_pointer()
                                        .h(px(32.))
                                        .px(px(14.))
                                        .flex()
                                        .items_center()
                                        .rounded_full()
                                        .bg(if selected { p.accent_soft } else { p.bg_list })
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(if selected {
                                            p.unread_text
                                        } else {
                                            p.text_secondary
                                        })
                                        .child(label),
                                    Action::SetMediaTab(tab),
                                )
                            }))
                            .child(
                                self.clickable(
                                    cx,
                                    div()
                                        .id("media-select-mode")
                                        .role(Role::Button)
                                        .tab_index(0)
                                        .aria_label("Select media")
                                        .cursor_pointer()
                                        .h(px(32.))
                                        .px(px(12.))
                                        .flex()
                                        .items_center()
                                        .rounded_full()
                                        .bg(if self.media_select_mode {
                                            p.accent
                                        } else {
                                            p.bg_search
                                        })
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(if self.media_select_mode {
                                            p.text_inverse
                                        } else {
                                            p.text_secondary
                                        })
                                        .child(if self.media_select_mode {
                                            "Done"
                                        } else {
                                            "Select"
                                        }),
                                    Action::ToggleMediaSelectMode,
                                ),
                            ),
                    ),
            )
            .when(
                self.media_select_mode && !self.media_selected.is_empty(),
                |el| {
                    el.child(
                        div()
                            .mx(px(24.))
                            .mb(px(10.))
                            .px(px(12.))
                            .py(px(8.))
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .rounded(px(RADIUS_MD))
                            .bg(p.bg_search)
                            .child(
                                div()
                                    .flex_1()
                                    .text_size(px(TEXT_SMALL))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(p.text_primary)
                                    .child(format!("{} selected", self.media_selected.len())),
                            )
                            .child(self.clickable(
                                cx,
                                ui::secondary_button("media-star-selected", "Star", p),
                                Action::MediaStarSelected,
                            ))
                            .child(self.clickable(
                                cx,
                                ui::secondary_button("media-forward-selected", "Forward", p),
                                Action::MediaForwardSelected,
                            ))
                            .child(self.clickable(
                                cx,
                                ui::danger_button("media-delete-selected", "Delete", p),
                                Action::MediaDeleteSelected,
                            )),
                    )
                },
            )
            .child(match self.list_state {
                ListState::Loading => div()
                    .flex_1()
                    .px(px(24.))
                    .pb(px(24.))
                    .child(div().grid().grid_cols(4).gap(px(10.)).children(
                        (0..8).map(|_| div().h(px(140.)).rounded(px(RADIUS_LG)).bg(p.bg_search)),
                    ))
                    .into_any_element(),
                ListState::Failed => div()
                    .flex_1()
                    .flex()
                    .items_center()
                    .justify_center()
                    .child(ui::error_state_compact(
                        "Could not load media",
                        "Check your connection and try again.",
                        Some(self.clickable(
                            cx,
                            ui::primary_button("retry-media", "Retry", p),
                            Action::SetListState(ListState::Ready),
                        )),
                        p,
                    ))
                    .into_any_element(),
                ListState::Ready if grid.is_empty() => div()
                    .flex_1()
                    .child(ui::empty_state(
                        match self.media_tab {
                            MediaTab::Media => "No media yet",
                            MediaTab::Links => "No links yet",
                            MediaTab::Docs => "No documents yet",
                        },
                        match self.media_tab {
                            MediaTab::Media => {
                                "Photos and videos shared in the chat appear here, newest first."
                            }
                            MediaTab::Links => {
                                "Links shared in the chat appear here, newest first."
                            }
                            MediaTab::Docs => {
                                "Documents shared in the chat appear here, newest first."
                            }
                        },
                        None,
                        p,
                    ))
                    .into_any_element(),
                ListState::Ready => div()
                    .flex_1()
                    .overflow_y_scrollbar()
                    .px(px(24.))
                    .pb(px(24.))
                    .child(div().grid().grid_cols(4).gap(px(10.)).children(grid))
                    .into_any_element(),
            })
    }

    fn render_ai_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let suggestions = [
            ("Summarise unread", "Ringkas 24 pesan terakhir"),
            ("Draft a reply", "Tulis balasan singkat"),
            ("Translate", "Terjemahkan ke Bahasa Inggris"),
            ("Find decisions", "Cari kesepakatan di chat ini"),
        ];
        let rows = suggestions
            .into_iter()
            .map(|(title, subtitle)| {
                let row = div()
                    .id(format!("ai-{title}"))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(title)
                    .cursor_pointer()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .border_b_1()
                    .border_color(p.divider)
                    .hover(move |style| style.bg(p.hover()))
                    .child(
                        div()
                            .size(px(32.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.accent_soft)
                            .child(ui::icon(IconName::Sparkles, 16.0, p.accent)),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(title),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(subtitle),
                            ),
                    );
                let action = if title == "Draft a reply" {
                    Action::AiSuggest
                } else {
                    Action::ShowComingSoon(
                        "Coming soon — the assistant endpoint is not part of this build.",
                    )
                };
                self.clickable(cx, row, action)
            })
            .collect::<Vec<_>>();

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("AI"),
                    )
                    .child(ui::chip("Preview", true, p)),
            )
            .child(
                div()
                    .px(px(16.))
                    .pb(px(10.))
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child(
                        "Answers use only the chat you have open. Nothing is sent anywhere else.",
                    ),
            )
            .child(self.list_body(
                cx,
                p,
                rows,
                "No AI drafts yet",
                "Open a chat and ask AI to draft a reply.",
            ))
    }

    fn render_ai_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let scope = div()
            .mx(px(16.))
            .my(px(8.))
            .px(px(12.))
            .py(px(8.))
            .flex()
            .items_center()
            .gap(px(8.))
            .rounded(px(RADIUS_MD))
            .bg(p.bg_notice_encrypt)
            .text_size(px(TEXT_CAPTION))
            .text_color(p.text_secondary)
            .child(ui::icon(IconName::Lock, 13.0, p.text_secondary))
            .child("Answers use only the chat you have open. Nothing is sent anywhere else.");

        let card = |body: gpui_kit::Div| {
            div()
                .w(px(540.))
                .max_w_full()
                .flex()
                .flex_col()
                .gap(px(12.))
                .p(px(20.))
                .rounded(px(RADIUS_XL))
                .bg(p.bg_list)
                .border_1()
                .border_color(p.border)
                .child(
                    div()
                        .flex()
                        .items_center()
                        .justify_between()
                        .child(
                            div()
                                .text_size(px(16.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Suggested reply"),
                        )
                        .child(ui::chip("DRAFT REPLY", true, p)),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Replying to Rizky Pratama · Tim Produk"),
                )
                .child(body)
        };

        let content = match self.ai_state {
            AiState::Idle => div()
                .flex_1()
                .flex()
                .items_center()
                .justify_center()
                .child(ui::empty_state(
                    "Ask for a suggested reply",
                    "The assistant reads only this conversation and drafts a reply you can edit before sending.",
                    Some(self.clickable(
                        cx,
                        ui::primary_button("ai-suggest", "Draft a reply", p),
                        Action::AiSuggest,
                    )),
                    p,
                ))
                .into_any_element(),
            AiState::Suggested => div()
                .flex_1()
                .flex()
                .items_center()
                .justify_center()
                .p(px(24.))
                .child(card(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(12.))
                        .child(
                            div()
                                .p(px(14.))
                                .rounded(px(RADIUS_LG))
                                .bg(p.bg_search)
                                .text_size(px(TEXT_BODY))
                                .text_color(p.text_primary)
                                .child("Siap, aku kirim deck final jam tiga ya. Kalau ada revisi kecil, tulis di kolom komentar."),
                        )
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .justify_end()
                                .gap(px(8.))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("ai-discard-card", "Discard", p),
                                    Action::AiDiscard,
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("ai-regenerate", "Regenerate", p),
                                    Action::AiRegenerate,
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::primary_button("ai-insert-card", "Insert", p),
                                    Action::InsertAiSuggestion,
                                )),
                        ),
                ))
                .into_any_element(),
            AiState::Streaming => {
                let line = |index: usize| {
                    div()
                        .h(px(12.))
                        .w(px(if index == 2 { 260. } else { 420. }))
                        .rounded_full()
                        .bg(p.bg_search)
                        .with_animation(
                            format!("ai-stream-{index}"),
                            Animation::new(Duration::from_millis(900)).repeat(),
                            |el, delta| el.opacity(0.35 + 0.45 * delta),
                        )
                };
                div()
                    .flex_1()
                    .flex()
                    .items_center()
                    .justify_center()
                    .p(px(24.))
                    .child(card(
                        div()
                            .flex()
                            .flex_col()
                            .gap(px(10.))
                            .child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(8.))
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(
                                        div()
                                            .size(px(8.))
                                            .rounded_full()
                                            .bg(p.accent)
                                            .with_animation(
                                                "ai-streaming-dot",
                                                Animation::new(Duration::from_millis(700)).repeat(),
                                                |el, delta| el.opacity(0.3 + 0.7 * delta),
                                            ),
                                    )
                                    .child("Writing a reply…"),
                            )
                            .children((0..3).map(line))
                            .child(
                                div()
                                    .flex()
                                    .justify_end()
                                    .child(self.clickable(
                                        cx,
                                        ui::secondary_button("ai-cancel-stream", "Cancel", p),
                                        Action::AiCancel,
                                    )),
                            ),
                    ))
                    .into_any_element()
            }
            AiState::Error => div()
                .flex_1()
                .flex()
                .items_center()
                .justify_center()
                .p(px(24.))
                .child(card(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(12.))
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .gap(px(10.))
                                .child(ui::icon(IconName::CircleAlert, 22.0, p.danger()))
                                .child(
                                    div()
                                        .text_size(px(TEXT_BODY))
                                        .font_weight(FontWeight::SEMIBOLD)
                                        .text_color(p.text_primary)
                                        .child("Could not draft a reply"),
                                ),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("The assistant timed out. Your chat was not sent anywhere. Try again or keep writing yourself."),
                        )
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .justify_end()
                                .gap(px(8.))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("ai-dismiss-error", "Dismiss", p),
                                    Action::AiDiscard,
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::primary_button("ai-retry", "Retry", p),
                                    Action::AiRetry,
                                )),
                        ),
                ))
                .into_any_element(),
        };

        div()
            .flex_1()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_conversation)
            .child(scope)
            .child(content)
            .child(
                div()
                    .p(px(16.))
                    .flex()
                    .items_center()
                    .gap(px(10.))
                    .child(
                        div()
                            .flex_1()
                            .h(px(44.))
                            .rounded_full()
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .px(px(16.))
                            .flex()
                            .items_center()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Ask about this conversation"),
                    )
                    .child(
                        self.clickable(
                            cx,
                            ui::icon_button(
                                "ai-send",
                                IconName::Send,
                                18.0,
                                p.text_inverse,
                                p.accent,
                                "Send prompt",
                            )
                            .bg(p.accent),
                            Action::AiPrompt,
                        ),
                    ),
            )
    }

    fn render_contacts_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let rows = self
            .contacts
            .iter()
            .map(|contact| {
                let row = div()
                    .id(format!("contact-row-{}", contact.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(contact.name.clone())
                    .cursor_pointer()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .hover(move |style| style.bg(p.hover()))
                    .child(ui::avatar(contact.initials.clone(), 40., contact.online, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(contact.name.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(contact.about.clone()),
                            ),
                    );
                self.clickable(cx, row, Action::SelectContact(contact.id))
            })
            .collect::<Vec<_>>();

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(section_header(
                cx,
                p,
                "Contacts",
                Some(IconName::Plus),
                Action::Toast("Add contact sheet opened · local mock".into()),
            ))
            .child(ui::section_label("ALL CONTACTS", p))
            .child(self.list_body(
                cx,
                p,
                rows,
                "No contacts yet",
                "Add a contact to start a conversation.",
            ))
    }

    fn render_contacts_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let Some(index) = self.selected_contact else {
            return div()
                .flex_1()
                .h_full()
                .flex()
                .items_center()
                .justify_center()
                .bg(p.bg_conversation)
                .child(ui::empty_state(
                    "Select a contact",
                    "Choose a contact to preview their profile and actions.",
                    None,
                    p,
                ));
        };
        let Some(contact) = self.contacts.get(index) else {
            return div().flex_1().bg(p.bg_conversation);
        };

        div()
            .flex_1()
            .h_full()
            .flex()
            .items_center()
            .justify_center()
            .bg(p.bg_conversation)
            .child(
                div()
                    .w(px(520.))
                    .flex()
                    .flex_col()
                    .items_center()
                    .gap(px(16.))
                    .p(px(32.))
                    .rounded(px(RADIUS_XL))
                    .bg(p.bg_list)
                    .border_1()
                    .border_color(p.border)
                    .child(ui::avatar(
                        contact.initials.clone(),
                        112.,
                        contact.online,
                        p,
                    ))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .child(
                                div()
                                    .text_size(px(24.))
                                    .font_weight(FontWeight::BOLD)
                                    .text_color(p.text_primary)
                                    .child(contact.name.clone()),
                            )
                            .when(contact.favorite, |el| {
                                el.child(ui::icon(IconName::Star, 16.0, p.attach_orange))
                            })
                            .when(contact.blocked, |el| el.child(ui::soon_pill(p))),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .text_color(p.text_secondary)
                            .child(format!("{} · {}", contact.phone, contact.about)),
                    )
                    .child(
                        div()
                            .flex()
                            .gap(px(8.))
                            .child(self.clickable(
                                cx,
                                ui::primary_button("contact-message", "Message", p),
                                Action::SelectChat(0),
                            ))
                            .child(self.clickable(
                                cx,
                                ui::secondary_button("contact-video", "Video", p),
                                Action::StartCall(0, true),
                            ))
                            .child(self.clickable(
                                cx,
                                ui::secondary_button("contact-voice", "Voice", p),
                                Action::StartCall(0, false),
                            ))
                            .child(self.clickable(
                                cx,
                                ui::secondary_button("contact-edit", "Edit", p),
                                Action::EditContact(contact.id),
                            )),
                    )
                    .child(ui::divider(p))
                    .child(self.clickable(
                        cx,
                        ui::settings_row("Mute notifications", "Silence this contact", None, p),
                        Action::CloseOverlayAndToast("Muted for 8 hours · local demo".into()),
                    ))
                    .child(self.clickable(
                        cx,
                        ui::settings_row(
                            "Disappearing messages",
                            "Default timer for new chats",
                            Some(
                                ["24 hours", "7 days", "90 days", "Off"][self.disappearing].into(),
                            ),
                            p,
                        ),
                        Action::OpenDisappearingPicker,
                    ))
                    .child(self.clickable(
                        cx,
                        ui::settings_row(
                            "Chat lock",
                            "Hide this chat behind device authentication",
                            None,
                            p,
                        ),
                        Action::CloseOverlayAndToast("Chat lock is coming soon.".into()),
                    ))
                    .child(self.clickable(
                        cx,
                        ui::settings_row(
                            "Encryption",
                            "Messages and calls are end-to-end encrypted",
                            None,
                            p,
                        ),
                        Action::OpenSecurityCode,
                    ))
                    .child(
                        self.clickable(
                            cx,
                            div()
                                .id("block-contact-detail")
                                .w_full()
                                .py(px(12.))
                                .text_size(px(TEXT_BODY))
                                .text_color(p.danger())
                                .child("Block contact"),
                            Action::OpenSubpage(SettingsSubpage::BlockedContactPicker),
                        ),
                    ),
            )
    }

    fn render_archived_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let rows = self
            .archived
            .iter()
            .map(|chat| {
                let row = div()
                    .id(format!("archived-row-{}", chat.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(chat.name.clone())
                    .cursor_pointer()
                    .h(px(72.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .border_b_1()
                    .border_color(p.divider)
                    .hover(move |style| style.bg(p.hover()))
                    .child(ui::avatar(chat.initials.clone(), 44., false, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(chat.name.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(chat.preview.clone()),
                            ),
                    )
                    .child(div().flex_1())
                    .child(
                        div()
                            .text_size(px(TEXT_CAPTION))
                            .text_color(p.text_secondary)
                            .child(chat.time.clone()),
                    );
                self.clickable(cx, row, Action::SelectChat(chat.id))
            })
            .collect::<Vec<_>>();

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Archived"),
                    )
                    .child(ui::chip("4 chats", false, p)),
            )
            .child(
                div()
                    .px(px(16.))
                    .pb(px(10.))
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child("Archived chats stay here and stay muted. New messages keep them archived unless you switch that off."),
            )
            .child(self.list_body(
                cx,
                p,
                rows,
                "No archived chats",
                "Archived chats stay here, out of the way.",
            ))
            .child(
                div().p(px(16.)).child(self.clickable(
                    cx,
                    ui::secondary_button("unarchive-all", "Unarchive all", p),
                    Action::UnarchiveAll,
                )),
            )
    }

    fn render_starred_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let rows = self
            .starred
            .iter()
            .map(|starred| {
                let row = div()
                    .id(format!("starred-row-{}", starred.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(starred.preview.clone())
                    .cursor_pointer()
                    .h(px(76.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .border_b_1()
                    .border_color(p.divider)
                    .hover(move |style| style.bg(p.hover()))
                    .child(ui::avatar(
                        initials_for(Some(&starred.sender)),
                        44.,
                        false,
                        p,
                    ))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(starred.chat.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(starred.preview.clone()),
                            ),
                    )
                    .child(div().flex_1())
                    .child(
                        div()
                            .text_size(px(TEXT_CAPTION))
                            .text_color(p.text_secondary)
                            .child(starred.date.clone()),
                    );
                self.clickable(cx, row, Action::OpenStarredChat(starred.chat.clone()))
            })
            .collect::<Vec<_>>();

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Starred messages"),
                    )
                    .child(ui::chip("09", false, p)),
            )
            .child(
                div()
                    .px(px(16.))
                    .pb(px(10.))
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child("Messages you star stay on this device and stay searchable when the original chat is archived."),
            )
            .child(self.list_body(
                cx,
                p,
                rows,
                "No starred messages",
                "Star a message to keep it close.",
            ))
    }

    fn render_locked_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let body = match self.list_state {
            ListState::Loading => div()
                .flex_1()
                .children((0..4).map(|_| ui::skeleton_row(p).into_any_element()))
                .into_any_element(),
            ListState::Failed => div()
                .flex_1()
                .flex()
                .items_center()
                .justify_center()
                .child(ui::error_state_compact(
                    "Could not load locked chats",
                    "Check your connection and try again.",
                    Some(self.clickable(
                        cx,
                        ui::primary_button("retry-locked", "Retry", p),
                        Action::SetListState(ListState::Ready),
                    )),
                    p,
                ))
                .into_any_element(),
            ListState::Ready => div()
                .flex_1()
                .flex()
                .items_center()
                .justify_center()
                .child(ui::empty_state_compact(
                    "No locked chats yet",
                    "Locked chats are hidden from the list and only open after your device confirms it is you.",
                    Some(self.clickable(
                        cx,
                        ui::primary_button("setup-lock", "Set up", p),
                        Action::OpenLockedUnlock,
                    )),
                    p,
                ))
                .into_any_element(),
        };

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(section_header(
                cx,
                p,
                "Locked chats",
                Some(IconName::Ellipsis),
                Action::Toast("Locked chat settings opened · local mock".into()),
            ))
            .child(body)
    }

    fn render_locked_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        div()
            .flex_1()
            .h_full()
            .flex()
            .items_center()
            .justify_center()
            .bg(p.bg_conversation)
            .child(ui::empty_state(
                "Locked chats arrive with local authentication",
                "The list, the empty state and the unlock flow are all designed. Until the OS keychain integration exists, nothing here can actually lock a chat.",
                Some(self.clickable(
                    cx,
                    ui::primary_button("setup-biometric", "Set up biometric unlock", p),
                    Action::OpenLockedUnlock,
                )),
                p,
            ))
    }

    fn render_broadcast_list(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let rows = self
            .broadcasts
            .iter()
            .map(|broadcast| {
                let selected = self.selected_broadcast == Some(broadcast.id);
                let row = div()
                    .id(format!("broadcast-row-{}", broadcast.id))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(broadcast.name.clone())
                    .cursor_pointer()
                    .h(px(76.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .border_b_1()
                    .border_color(p.divider)
                    .bg(if selected {
                        p.bg_row_selected
                    } else {
                        p.bg_list
                    })
                    .hover(move |style| style.bg(p.bg_row_selected))
                    .child(
                        div()
                            .size(px(44.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.accent_soft)
                            .child(ui::icon(IconName::Megaphone, 20.0, p.accent)),
                    )
                    .child(
                        div()
                            .flex_1()
                            .min_w(px(0.))
                            .flex()
                            .flex_col()
                            .gap(px(3.))
                            .child(
                                div()
                                    .flex()
                                    .justify_between()
                                    .child(
                                        div()
                                            .text_size(px(TEXT_BODY))
                                            .font_weight(FontWeight::SEMIBOLD)
                                            .text_color(p.text_primary)
                                            .child(broadcast.name.clone()),
                                    )
                                    .child(
                                        div()
                                            .text_size(px(TEXT_CAPTION))
                                            .text_color(p.text_secondary)
                                            .child(broadcast.time.clone()),
                                    ),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(format!(
                                        "{} · {}",
                                        broadcast.recipients, broadcast.preview
                                    )),
                            ),
                    );
                self.clickable(cx, row, Action::SelectBroadcast(broadcast.id))
            })
            .collect::<Vec<_>>();

        div()
            .w(px(LIST_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Broadcast"),
                    )
                    .child(
                        self.clickable(
                            cx,
                            ui::icon_button(
                                "new-broadcast",
                                IconName::Plus,
                                18.0,
                                p.text_inverse,
                                p.accent,
                                "New broadcast list",
                            )
                            .bg(p.accent),
                            Action::NewBroadcast,
                        ),
                    ),
            )
            .child(self.list_body(
                cx,
                p,
                rows,
                "No broadcast lists",
                "Create a list to message many contacts at once.",
            ))
    }

    fn render_broadcast_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let Some(index) = self.selected_broadcast else {
            return div()
                .flex_1()
                .h_full()
                .flex()
                .items_center()
                .justify_center()
                .bg(p.bg_conversation)
                .child(ui::empty_state(
                    "No broadcast lists yet",
                    "Create a list, pick recipients, and send one message to everyone.",
                    Some(self.clickable(
                        cx,
                        ui::primary_button("new-broadcast-empty", "New list", p),
                        Action::NewBroadcast,
                    )),
                    p,
                ));
        };
        let Some(broadcast) = self.broadcasts.get(index) else {
            return div().flex_1().bg(p.bg_conversation);
        };

        div()
            .flex_1()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_conversation)
            .child(
                div()
                    .h(px(CONVERSATION_HEADER_HEIGHT))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .bg(p.bg_header)
                    .border_b_1()
                    .border_color(p.divider)
                    .child(
                        div()
                            .size(px(40.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.accent_soft)
                            .child(ui::icon(IconName::Megaphone, 20.0, p.accent)),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(broadcast.name.clone()),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_secondary)
                                    .child(format!("Broadcast · {}", broadcast.recipients)),
                            ),
                    )
                    .child(div().flex_1())
                    .child(self.clickable(
                        cx,
                        ui::icon_button(
                            "broadcast-info",
                            IconName::Ellipsis,
                            19.0,
                            p.text_primary,
                            p.hover(),
                            "Broadcast info",
                        ),
                        Action::Toast("Broadcast settings opened · local mock".into()),
                    )),
            )
            .child(
                div()
                    .px(px(16.))
                    .py(px(8.))
                    .bg(p.accent_soft)
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.unread_text)
                    .child("Recipients get a normal message; replies land in normal chats."),
            )
            .child(
                div()
                    .flex_1()
                    .overflow_y_scrollbar()
                    .p(px(16.))
                    .flex()
                    .flex_col()
                    .justify_end()
                    .child(
                        div()
                            .self_end()
                            .max_w(px(420.))
                            .px(px(10.))
                            .py(px(6.))
                            .rounded(px(RADIUS_BUBBLE))
                            .rounded_br(px(4.))
                            .bg(p.bubble_out)
                            .text_size(px(TEXT_BODY))
                            .text_color(p.text_primary)
                            .child(broadcast.preview.clone())
                            .child(
                                div()
                                    .mt(px(2.))
                                    .flex()
                                    .justify_end()
                                    .gap(px(4.))
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_secondary)
                                    .child(broadcast.time.clone())
                                    .child(ui::icon(IconName::CheckCheck, 13.0, p.tick_read)),
                            ),
                    ),
            )
            .child(
                div()
                    .min_h(px(COMPOSER_HEIGHT))
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .px(px(16.))
                    .py(px(8.))
                    .bg(p.bg_composer)
                    .border_t_1()
                    .border_color(p.divider)
                    .child(
                        div()
                            .flex_1()
                            .h(px(40.))
                            .rounded(px(RADIUS_XL))
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .px(px(12.))
                            .flex()
                            .items_center()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(format!("Broadcast · {}", broadcast.recipients)),
                    )
                    .child(
                        self.clickable(
                            cx,
                            ui::icon_button(
                                "broadcast-send",
                                IconName::Send,
                                18.0,
                                p.text_inverse,
                                p.accent,
                                "Send broadcast",
                            )
                            .bg(p.accent),
                            Action::Toast(format!("Broadcast sent to {}", broadcast.recipients)),
                        ),
                    ),
            )
    }

    fn render_brand_empty(
        &self,
        _cx: &mut Context<Self>,
        p: Palette,
        _label: &str,
    ) -> gpui_kit::Div {
        div()
            .flex_1()
            .h_full()
            .flex()
            .items_center()
            .justify_center()
            .bg(p.bg_conversation)
            .child(
                div()
                    .flex()
                    .flex_col()
                    .items_center()
                    .gap(px(10.))
                    .child(app_mark_large(p))
                    .child(
                        div()
                            .text_size(px(22.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_primary)
                            .child("WhatsLite for Windows"),
                    )
                    .child(
                        div()
                            .max_w(px(420.))
                            .text_center()
                            .text_size(px(TEXT_BODY))
                            .text_color(p.text_secondary)
                            .child("Send and receive messages without keeping your phone online. Use WhatsLite on up to four linked devices at the same time."),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(6.))
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(ui::icon(IconName::Lock, 14.0, p.text_secondary))
                            .child("End-to-end encrypted"),
                    ),
            )
    }

    fn render_settings_nav(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let items = SettingsPage::ALL
            .into_iter()
            .map(|page| {
                let selected = self.settings_page == page;
                let row = div()
                    .id(format!("settings-nav-{:?}", page))
                    .role(Role::Button)
                    .tab_index(0)
                    .aria_label(page.title())
                    .cursor_pointer()
                    .h(px(40.))
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(12.))
                    .rounded(px(RADIUS_LG))
                    .bg(if selected { p.bg_selected } else { p.bg_list })
                    .text_size(px(13.))
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(if selected {
                        p.text_primary
                    } else {
                        p.text_secondary
                    })
                    .hover(move |style| style.bg(p.hover()))
                    .child(ui::icon(
                        page.icon(),
                        18.0,
                        if selected { p.accent } else { p.text_secondary },
                    ))
                    .child(page.title());
                self.clickable(cx, row, Action::SettingsPage(page))
            })
            .collect::<Vec<_>>();

        div()
            .w(px(SETTINGS_NAV_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_r_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .px(px(16.))
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "settings-back",
                            IconName::ArrowLeft,
                            18.0,
                            p.text_primary,
                            p.hover(),
                            "Back to chats",
                        ),
                        Action::Navigate(Route::Chats),
                    ))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Settings"),
                    ),
            )
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(12.))
                    .px(px(16.))
                    .py(px(10.))
                    .child(ui::avatar("Y", 56., false, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child("You"),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child("Available"),
                            ),
                    ),
            )
            .child(
                div()
                    .flex_1()
                    .overflow_y_scrollbar()
                    .p(px(8.))
                    .flex()
                    .flex_col()
                    .gap(px(2.))
                    .children(items),
            )
    }

    fn render_settings_detail(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        if let Some(subpage) = self.settings_subpage {
            return self.render_settings_subpage(subpage, cx, p);
        }
        let title = self.settings_page.title();
        let content = match self.settings_page {
            SettingsPage::Account => self.settings_account(cx, p),
            SettingsPage::Privacy => self.settings_privacy(cx, p),
            SettingsPage::Chats => self.settings_chats(cx, p),
            SettingsPage::Notifications => self.settings_notifications(cx, p),
            SettingsPage::Storage => self.settings_storage(cx, p),
            SettingsPage::Help => self.settings_help(cx, p),
        };

        div()
            .flex_1()
            .h_full()
            .w_full()
            .min_w(px(0.))
            .flex()
            .flex_col()
            .bg(p.bg_conversation)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(24.))
                    .child(
                        div()
                            .text_size(px(22.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_primary)
                            .child(title),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(6.))
                            .child(ui::live_pill(p))
                            .child(ui::soon_pill(p)),
                    ),
            )
            .child(
                div()
                    .flex_1()
                    .overflow_y_scrollbar()
                    .px(px(24.))
                    .pb(px(24.))
                    .child(
                        div()
                            .rounded(px(RADIUS_XL))
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .child(content),
                    ),
            )
    }

    fn settings_account(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        div()
            .flex()
            .flex_col()
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(16.))
                    .p(px(20.))
                    .child(ui::avatar("Y", 88., false, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(20.))
                                    .font_weight(FontWeight::BOLD)
                                    .text_color(p.text_primary)
                                    .child("You"),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .text_color(p.text_secondary)
                                    .child("Available · +62 812 3456 7890"),
                            ),
                    ),
            )
            .child(ui::section_label("PROFILE", p))
            .child(self.clickable(
                cx,
                ui::settings_row("Name", "Shown to your contacts", Some("You".into()), p),
                Action::Toast("Name editor opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("About", "Up to 139 characters", Some("Available".into()), p),
                Action::Toast("About editor opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Profile photo", "Change or remove", None, p),
                Action::OpenSubpage(SettingsSubpage::ProfilePhoto),
            ))
            .child(ui::section_label("CONTACT", p))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Phone number",
                    "Used for sign-in and pairing",
                    Some("+62 812 3456 7890".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::ChangeNumber),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Email",
                    "Optional recovery address",
                    Some("Not set".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::RecoveryEmail),
            ))
            .child(ui::section_label("SECURITY", p))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Two-step verification",
                    "6-digit PIN + recovery email",
                    Some("Off".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::TwoStep),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Linked devices",
                    "Other machines with access",
                    Some("2 devices".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::LinkedDevices),
            ))
            .child(ui::section_label("ACCOUNT", p))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Request account info",
                    "Download a copy of your data",
                    None,
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::AccountInfo),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Delete account",
                    "Removes the account and its data",
                    None,
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::DeleteAccount),
            ))
    }

    fn settings_privacy(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        div()
            .flex()
            .flex_col()
            .child(ui::section_label("VISIBILITY", p))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Last seen and online",
                    "Everyone in your contacts",
                    Some("Contacts".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::Visibility),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Profile photo",
                    "Everyone except blocked",
                    Some("Everyone".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::Visibility),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "About",
                    "Everyone except blocked",
                    Some("Everyone".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::Visibility),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Status",
                    "Everyone except blocked",
                    Some("Contacts".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::Visibility),
            ))
            .child(ui::section_label("MESSAGING", p))
            .child(self.toggle_row(
                cx,
                p,
                "read_receipts",
                "Read receipts",
                "Turns off both blue ticks in both directions",
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Disappearing messages",
                    "Default timer for new chats",
                    Some(["24 hours", "7 days", "90 days", "Off"][self.disappearing].into()),
                    p,
                ),
                Action::OpenDisappearingPicker,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Blocked contacts",
                    "Blocked people cannot call or message you",
                    Some("3".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::BlockedContacts),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Block a contact", "Search a name or number", None, p),
                Action::OpenSubpage(SettingsSubpage::BlockedContactPicker),
            ))
            .child(ui::section_label("APP LOCK", p))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Unlock with Windows Hello",
                    "Locks the app when idle",
                    Some("Off".into()),
                    p,
                ),
                Action::OpenLockedUnlock,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Auto-lock after",
                    "Applies while WhatsLite is in the background",
                    Some("15 minutes".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::AutoLock),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Advanced", "IP protection in calls, relayed calls", None, p),
                Action::Toast("Advanced privacy opened · local mock".into()),
            ))
    }

    fn settings_chats(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let themes = [
            (Some(ThemeMode::Light), "Light"),
            (Some(ThemeMode::Dark), "Dark"),
            (None, "System"),
        ];
        div()
            .flex()
            .flex_col()
            .child(
                div()
                    .p(px(20.))
                    .flex()
                    .flex_col()
                    .gap(px(8.))
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(p.text_primary)
                            .child("Theme"),
                    )
                    .child(
                        div()
                            .h(px(36.))
                            .flex()
                            .rounded_full()
                            .bg(p.bg_search)
                            .p(px(3.))
                            .children(themes.into_iter().map(|(mode, label)| {
                                let selected = match mode {
                                    Some(mode) => !self.system_theme && self.theme == mode,
                                    None => self.system_theme,
                                };
                                let action = match mode {
                                    Some(mode) => Action::SetTheme(mode),
                                    None => Action::SetSystemTheme,
                                };
                                self.clickable(
                                    cx,
                                    div()
                                        .id(format!("theme-{label}"))
                                        .flex_1()
                                        .flex()
                                        .items_center()
                                        .justify_center()
                                        .rounded_full()
                                        .bg(if selected { p.accent } else { p.bg_search })
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(if selected { p.text_inverse } else { p.text_secondary })
                                        .child(label),
                                    action,
                                )
                            })),
                    ),
            )
            .child(
                div()
                    .p(px(20.))
                    .flex()
                    .flex_col()
                    .gap(px(8.))
                    .border_t_1()
                    .border_color(p.divider)
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .justify_between()
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(p.text_primary)
                                    .child("Live OS theme"),
                            )
                            .child(ui::chip(if self.system_theme { "Following OS" } else { "Manual" }, self.system_theme, p)),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("When System is selected, WhatsLite follows the OS appearance as it changes."),
                    )
                    .when(self.system_theme, |el| {
                        el.child(
                            div()
                                .flex()
                                .items_center()
                                .gap(px(6.))
                                .child(self.clickable(
                                    cx,
                                    ui::chip("OS light", !self.os_theme_dark, p),
                                    Action::SetOsTheme(false),
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::chip("OS dark", self.os_theme_dark, p),
                                    Action::SetOsTheme(true),
                                )),
                        )
                    }),
            )
            .child(
                div()
                    .p(px(20.))
                    .flex()
                    .flex_col()
                    .gap(px(8.))
                    .border_t_1()
                    .border_color(p.divider)
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(p.text_primary)
                            .child("Window shell"),
                    )
                    .child(
                        div()
                            .h(px(36.))
                            .flex()
                            .rounded_full()
                            .bg(p.bg_search)
                            .p(px(3.))
                            .children([(ShellStyle::Windows, "Windows"), (ShellStyle::MacOS, "macOS")].into_iter().map(|(shell, label)| {
                                let selected = self.shell == shell;
                                self.clickable(
                                    cx,
                                    div()
                                        .id(format!("shell-{label}"))
                                        .flex_1()
                                        .flex()
                                        .items_center()
                                        .justify_center()
                                        .rounded_full()
                                        .bg(if selected { p.accent } else { p.bg_search })
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(if selected { p.text_inverse } else { p.text_secondary })
                                        .child(label),
                                    Action::SetShell(shell),
                                )
                            })),
                    ),
            )
            .child(ui::section_label("APPEARANCE", p))
            .child(self.clickable(
                cx,
                ui::settings_row("Chat wallpaper", "Doodle, solid colour or your own image", Some("Doodle".into()), p),
                Action::OpenSubpage(SettingsSubpage::Wallpaper),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Chat font size", "Applies to message text only", Some("Default (14)".into()), p),
                Action::OpenSubpage(SettingsSubpage::FontSize),
            ))
            .child(ui::section_label("ACCESSIBILITY", p))
            .child(self.toggle_row(cx, p, "reduce_motion", "Reduce motion", "Transitions become instant fades"))
            .child(ui::section_label("WINDOW", p))
            .child(self.toggle_row(cx, p, "autostart", "Start WhatsLite on login", "Launch minimised to the tray"))
            .child(self.toggle_row(cx, p, "always_on_top", "Keep window on top", "Pin the WhatsLite window above other apps"))
            .child(self.toggle_row(cx, p, "close_to_tray", "Close to tray", "Closing the window keeps WhatsLite running"))
            .child(self.toggle_row(cx, p, "spellcheck", "Spellcheck", "Underline misspelled words in the composer"))
            .child(ui::section_label("COMPOSER", p))
            .child(self.toggle_row(cx, p, "enter_sends", "Enter sends a message", "Otherwise Enter adds a new line"))
            .child(self.toggle_row(cx, p, "media_preview", "Show media in chat list", "Preview photos instead of “Photo”"))
            .child(ui::section_label("ARCHIVE", p))
            .child(self.toggle_row(cx, p, "keep_archived", "Keep chats archived", "New messages do not unarchive a chat"))
            .child(self.toggle_row(cx, p, "mute_archived", "Mute archived chats", "Notifications stay off while archived"))
    }

    fn settings_notifications(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        div()
            .flex()
            .flex_col()
            .child(ui::section_label("MESSAGING", p))
            .child(self.toggle_row(
                cx,
                p,
                "notifications",
                "Desktop notifications",
                "Banners while WhatsLite is running",
            ))
            .child(self.toggle_row(
                cx,
                p,
                "taskbar_badge",
                "Taskbar badge",
                "Show the total unread count on the app icon",
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Notification sound",
                    "Plays the default tone",
                    Some("Default".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::NotificationSound),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Message preview",
                    "Show sender, message text or nothing",
                    Some("Sender and message".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::NotificationPreview),
            ))
            .child(self.toggle_row(
                cx,
                p,
                "reaction_notifications",
                "Reaction notifications",
                "Notify when someone reacts to your message",
            ))
            .child(ui::section_label("DO NOT DISTURB", p))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Quiet hours",
                    "Silence banners between the times below",
                    Some("22:00 – 07:00".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::QuietHours),
            ))
            .child(self.toggle_row(
                cx,
                p,
                "call_exceptions",
                "Allow calls during quiet hours",
                "Applies every day",
            ))
            .child(ui::section_label("EXCEPTIONS", p))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Per-chat overrides",
                    "2 chats always notify, 4 muted",
                    None,
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::NotificationOverrides),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Top chats",
                    "Always notify for these conversations",
                    Some("6 chats".into()),
                    p,
                ),
                Action::OpenSubpage(SettingsSubpage::NotificationOverrides),
            ))
    }

    fn settings_storage(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        div()
            .flex()
            .flex_col()
            .child(
                div()
                    .p(px(20.))
                    .flex()
                    .flex_col()
                    .gap(px(10.))
                    .child(
                        div()
                            .flex()
                            .justify_between()
                            .text_size(px(TEXT_BODY))
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(p.text_primary)
                            .child("Storage used")
                            .child("1.8 GB of 20 GB"),
                    )
                    .child(
                        div()
                            .h(px(6.))
                            .w_full()
                            .rounded_full()
                            .bg(p.bg_search)
                            .child(div().h(px(6.)).w(px(160.)).rounded_full().bg(p.accent)),
                    ),
            )
            .child(ui::section_label("BY TYPE", p))
            .child(self.clickable(
                cx,
                ui::settings_row("Photos", "Images in chats", Some("1.2 GB".into()), p),
                Action::Toast("Storage manager opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Video", "Videos in chats", Some("0.5 GB".into()), p),
                Action::Toast("Storage manager opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Docs", "Documents in chats", Some("0.1 GB".into()), p),
                Action::Toast("Storage manager opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Storage per chat", "Manage 612 MB in Tim Produk", None, p),
                Action::OpenSubpage(SettingsSubpage::StoragePerChat),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Media cache",
                    "Cached photos, videos and documents",
                    Some(format!("{} MB", self.media_cache_mb)),
                    p,
                ),
                Action::ClearMediaCache,
            ))
            .child(ui::section_label("AUTO-DOWNLOAD", p))
            .child(self.toggle_row(
                cx,
                p,
                "auto_download",
                "Auto-download media",
                "Wi-Fi only in this mock",
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Photos", "Download automatically", Some("Wi-Fi".into()), p),
                Action::Toast("Auto-download rules opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Audio", "Download automatically", Some("Wi-Fi".into()), p),
                Action::Toast("Auto-download rules opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Documents", "Download automatically", Some("Off".into()), p),
                Action::Toast("Auto-download rules opened · local mock".into()),
            ))
    }

    fn settings_help(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        div()
            .flex()
            .flex_col()
            .child(ui::section_label("SUPPORT", p))
            .child(self.clickable(
                cx,
                ui::settings_row("Help centre", "Answers about pairing, notifications and storage", None, p),
                Action::OpenSubpage(SettingsSubpage::HelpCentre),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Contact us", "Send a description and optional logs", None, p),
                Action::OpenSubpage(SettingsSubpage::ContactUs),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Report a problem", "Attach a screenshot from the last session", None, p),
                Action::OpenSubpage(SettingsSubpage::ReportProblem),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Language", "App language and date formats", Some("English".into()), p),
                Action::OpenSubpage(SettingsSubpage::Language),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Proxy", "Connect through a proxy server", Some("Off".into()), p),
                Action::OpenSubpage(SettingsSubpage::Proxy),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Update channel", "Stable · check for updates manually", Some("Stable".into()), p),
                Action::Toast("Update channel opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Print thread", "Save the current conversation as PDF", None, p),
                Action::Toast("Print dialog opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("System status", "Permissions, tray, update, error and drag & drop", None, p),
                Action::OpenSubpage(SettingsSubpage::SystemStatus),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Call states", "Dialing, ringing, busy, no answer and more", None, p),
                Action::OpenSubpage(SettingsSubpage::CallStates),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Network states", "Connecting, offline and reconnecting", None, p),
                Action::OpenSubpage(SettingsSubpage::NetworkStates),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Primitives", "Checkbox, radio, progress, keycap and avatar group", None, p),
                Action::OpenSubpage(SettingsSubpage::Primitives),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Keyboard shortcuts", "Search, compose, navigate and close layers", None, p),
                Action::OpenSubpage(SettingsSubpage::Shortcuts),
            ))
            .child(ui::section_label("ABOUT", p))
            .child(self.clickable(
                cx,
                ui::settings_row("WhatsLite for Windows", "1.0.0 (build 42)", None, p),
                Action::OpenSubpage(SettingsSubpage::About),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Licences", "Inter — SIL Open Font License 1.1 · Material Symbols Rounded — Apache License 2.0", None, p),
                Action::OpenSubpage(SettingsSubpage::About),
            ))
            .child(
                div()
                    .p(px(20.))
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child("WhatsLite is an independent project. Not affiliated with WhatsApp or Meta."),
            )
    }

    fn render_settings_subpage(
        &self,
        page: SettingsSubpage,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        let body = match page {
            SettingsSubpage::LinkedDevices => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("2 devices are using this account. Logging out removes the device's access to WhatsLite."),
                )
                .child(ui::settings_row_field("WhatsLite for Windows", "This device · Active now", None, p))
                .child(ui::settings_row_field("iPhone 15", "Last active 2 minutes ago", None, p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("logout-devices", "Log out of other devices", p),
                        Action::OpenSubpage(SettingsSubpage::LinkedDeviceLogout),
                    )),
                ),
            SettingsSubpage::Visibility => div()
                .flex()
                .flex_col()
                .child(ui::section_label("WHO CAN SEE MY PERSONAL INFO", p))
                .child(self.subpage_option(cx, p, "Everyone", false, Action::Toast("Visibility · Everyone".into())))
                .child(self.subpage_option(cx, p, "My contacts", true, Action::Toast("Visibility · My contacts".into())))
                .child(self.subpage_option(cx, p, "My contacts except…", false, Action::Toast("Visibility · My contacts except".into())))
                .child(self.subpage_option(cx, p, "Nobody", false, Action::Toast("Visibility · Nobody".into())))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-visibility", "Save", p),
                        Action::ShowComingSoon("Saved locally — visibility is a mock preference."),
                    )),
                ),
            SettingsSubpage::ChangeNumber => div()
                .flex()
                .flex_col()
                .child(ui::settings_row_field("Current number", "Used for sign-in and pairing", Some("+62 812 3456 7890".into()), p))
                .child(ui::settings_row_field("New number", "We will send a 6-digit code", Some("Not set".into()), p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("change-number", "Next", p),
                        Action::OpenSubpage(SettingsSubpage::ChangeNumberOtp),
                    )),
                ),
            SettingsSubpage::RecoveryEmail => div()
                .flex()
                .flex_col()
                .child(ui::settings_row_field("Recovery email", "Optional address for account recovery", Some("Not set".into()), p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("add-email", "Add email", p),
                        Action::OpenSubpage(SettingsSubpage::RecoveryEmailCode),
                    )),
                ),
            SettingsSubpage::ProfilePhoto => {
                let avatar = ui::avatar("Y", 120., false, p);
                let body = match self.profile_photo {
                    ProfilePhotoState::Empty => div()
                        .flex()
                        .flex_col()
                        .items_center()
                        .gap(px(14.))
                        .p(px(24.))
                        .child(avatar)
                        .child(
                            div()
                                .text_size(px(TEXT_BODY))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("No photo yet"),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("A photo helps people recognise you. You can change it at any time."),
                        )
                        .child(
                            div()
                                .flex()
                                .gap(px(8.))
                                .child(self.clickable(
                                    cx,
                                    ui::primary_button("upload-photo", "Upload photo", p),
                                    Action::ProfilePhotoUpload,
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("remove-photo", "Remove", p),
                                    Action::ProfilePhotoRemove,
                                )),
                        )
                        .into_any_element(),
                    ProfilePhotoState::Cropping => div()
                        .flex()
                        .flex_col()
                        .gap(px(14.))
                        .p(px(24.))
                        .child(
                            div()
                                .text_size(px(18.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Crop photo"),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("The OS crop dialog is mocked here. In the real build the system handles cropping, then WhatsLite uploads the result."),
                        )
                        .child(
                            div()
                                .w_full()
                                .flex()
                                .justify_center()
                                .child(
                                    div()
                                        .relative()
                                        .w(px(320.))
                                        .h(px(320.))
                                        .rounded(px(RADIUS_LG))
                                        .bg(p.attach_orange)
                                        .flex()
                                        .items_center()
                                        .justify_center()
                                        .child(ui::icon(IconName::Image, 72.0, p.text_inverse))
                                        .child(
                                            div()
                                                .absolute()
                                                .top(px(24.))
                                                .left(px(24.))
                                                .right(px(24.))
                                                .bottom(px(24.))
                                                .border_2()
                                                .border_color(p.text_inverse),
                                        )
                                        .child(
                                            div()
                                                .absolute()
                                                .top(px(24.))
                                                .left(px(24.))
                                                .right(px(24.))
                                                .h(px(1.))
                                                .bg(p.text_inverse)
                                                .opacity(0.6),
                                        )
                                        .child(
                                            div()
                                                .absolute()
                                                .top(px(24.))
                                                .bottom(px(24.))
                                                .left(px(24.))
                                                .w(px(1.))
                                                .bg(p.text_inverse)
                                                .opacity(0.6),
                                        ),
                                ),
                        )
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .justify_center()
                                .gap(px(10.))
                                .child(self.clickable(
                                    cx,
                                    ui::icon_button_small(
                                        "photo-zoom-out",
                                        IconName::Minus,
                                        14.0,
                                        p.text_primary,
                                        p.hover(),
                                        "Zoom out",
                                    ),
                                    Action::ProfilePhotoZoomOut,
                                ))
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(p.text_primary)
                                        .child(format!("{}%", self.profile_photo_zoom)),
                                )
                                .child(self.clickable(
                                    cx,
                                    ui::icon_button_small(
                                        "photo-zoom-in",
                                        IconName::Plus,
                                        14.0,
                                        p.text_primary,
                                        p.hover(),
                                        "Zoom in",
                                    ),
                                    Action::ProfilePhotoZoomIn,
                                )),
                        )
                        .child(
                            div()
                                .flex()
                                .justify_end()
                                .gap(px(8.))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("cancel-crop", "Cancel", p),
                                    Action::ProfilePhotoCropCancel,
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::primary_button("use-photo", "Use photo", p),
                                    Action::ProfilePhotoCropConfirm,
                                )),
                        )
                        .into_any_element(),
                    ProfilePhotoState::Uploaded => div()
                        .flex()
                        .flex_col()
                        .items_center()
                        .gap(px(14.))
                        .p(px(24.))
                        .child(avatar)
                        .child(ui::chip("Uploaded", true, p))
                        .child(
                            div()
                                .text_size(px(TEXT_BODY))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Photo uploaded"),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("Save to keep this photo, or remove it and choose another."),
                        )
                        .child(
                            div()
                                .flex()
                                .gap(px(8.))
                                .child(self.clickable(
                                    cx,
                                    ui::primary_button("save-photo", "Save", p),
                                    Action::ProfilePhotoSave,
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("discard-photo", "Remove", p),
                                    Action::ProfilePhotoRemove,
                                )),
                        )
                        .into_any_element(),
                    ProfilePhotoState::Saved => div()
                        .flex()
                        .flex_col()
                        .items_center()
                        .gap(px(14.))
                        .p(px(24.))
                        .child(
                            div()
                                .relative()
                                .child(avatar)
                                .child(
                                    div()
                                        .absolute()
                                        .bottom(px(2.))
                                        .right(px(2.))
                                        .size(px(28.))
                                        .flex()
                                        .items_center()
                                        .justify_center()
                                        .rounded_full()
                                        .bg(p.accent)
                                        .border_2()
                                        .border_color(p.bg_list)
                                        .child(ui::icon(IconName::Check, 14.0, p.text_inverse)),
                                ),
                        )
                        .child(ui::chip("Saved", true, p))
                        .child(
                            div()
                                .text_size(px(TEXT_BODY))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Profile photo saved"),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("People you chat with can now see this photo."),
                        )
                        .child(
                            div()
                                .flex()
                                .gap(px(8.))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("change-photo", "Change photo", p),
                                    Action::ProfilePhotoUpload,
                                ))
                                .child(self.clickable(
                                    cx,
                                    ui::secondary_button("delete-photo", "Remove", p),
                                    Action::ProfilePhotoRemove,
                                )),
                        )
                        .into_any_element(),
                };
                div().flex().flex_col().child(body)
            }
            SettingsSubpage::AutoLock => div()
                .flex()
                .flex_col()
                .child(ui::section_label("LOCK AFTER", p))
                .child(self.subpage_option(cx, p, "Immediately", false, Action::Toast("Auto-lock · Immediately".into())))
                .child(self.subpage_option(cx, p, "After 1 minute", false, Action::Toast("Auto-lock · 1 minute".into())))
                .child(self.subpage_option(cx, p, "After 15 minutes", true, Action::Toast("Auto-lock · 15 minutes".into())))
                .child(self.subpage_option(cx, p, "After 1 hour", false, Action::Toast("Auto-lock · 1 hour".into())))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-autolock", "Save", p),
                        Action::Toast("Auto-lock saved · local mock".into()),
                    )),
                ),
            SettingsSubpage::NotificationOverrides => div()
                .flex()
                .flex_col()
                .child(ui::section_label("PER-CHAT OVERRIDES", p))
                .child(self.toggle_row(cx, p, "override-tim", "Tim Produk", "Always notify"))
                .child(self.toggle_row(cx, p, "override-maya", "Maya", "Muted"))
                .child(self.toggle_row(cx, p, "override-jakarta", "Jakarta Crew", "Muted"))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-overrides", "Save", p),
                        Action::ShowComingSoon("Saved locally — notification overrides are a mock."),
                    )),
                ),
            SettingsSubpage::BlockedContacts => div()
                .flex()
                .flex_col()
                .child(ui::settings_row_field("Promo Grosir", "Blocked", None, p))
                .child(ui::settings_row_field("Unknown +62 811 0000 111", "Blocked", None, p))
                .child(ui::settings_row_field("Unknown +62 812 9999 0000", "Blocked", None, p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("unblock-all", "Unblock all", p),
                        Action::Toast("All contacts unblocked · local mock".into()),
                    )),
                ),
            SettingsSubpage::Wallpaper => div()
                .flex()
                .flex_col()
                .child(ui::section_label("WALLPAPER", p))
                .child(self.subpage_option(cx, p, "Doodle", true, Action::Toast("Wallpaper · Doodle".into())))
                .child(self.subpage_option(cx, p, "Solid colour", false, Action::Toast("Wallpaper · Solid colour".into())))
                .child(self.subpage_option(cx, p, "Upload image", false, Action::Toast("Wallpaper · Upload image".into())))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-wallpaper", "Save", p),
                        Action::ShowComingSoon("Saved locally — wallpaper is a mock preference."),
                    )),
                ),
            SettingsSubpage::FontSize => div()
                .flex()
                .flex_col()
                .child(ui::section_label("MESSAGE TEXT SIZE", p))
                .child(self.subpage_option(cx, p, "Small (12)", self.text_scale < 1.0, Action::SetTextScale(0.9)))
                .child(self.subpage_option(cx, p, "Default (14)", (self.text_scale - 1.0).abs() < f32::EPSILON, Action::SetTextScale(1.0)))
                .child(self.subpage_option(cx, p, "Large (16)", self.text_scale > 1.0, Action::SetTextScale(1.15)))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-font", "Save", p),
                        Action::ShowComingSoon("Saved locally — font size is a mock preference."),
                    )),
                ),
            SettingsSubpage::TwoStep => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Choose a six-digit PIN. You will enter it when registering WhatsLite on a new device."),
                )
                .child(ui::settings_row_field("PIN", "6-digit PIN + recovery email", Some("Not set".into()), p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("turn-on-2fa", "Turn on", p),
                        Action::Toast("Two-step verification enabled · local mock".into()),
                    )),
                ),
            SettingsSubpage::AccountInfo => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Download a copy of your account information and media. The report is prepared locally in this mock."),
                )
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("request-info", "Request account info", p),
                        Action::Toast("Account report queued · local mock".into()),
                    )),
                ),
            SettingsSubpage::DeleteAccount => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Deleting your account removes your messages, media and contacts from this device. This action cannot be undone."),
                )
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::danger_button("delete-account", "Delete account", p),
                        Action::Toast("Account deletion is disabled in this mock".into()),
                    )),
                ),
            SettingsSubpage::HelpCentre => div()
                .flex()
                .flex_col()
                .child(self.clickable(
                    cx,
                    ui::settings_row("Design reference", "Foundations, wallpaper, components, handoff and audit", None, p),
                    Action::OpenSubpage(SettingsSubpage::DesignReference),
                ))
                .child(ui::settings_row("How do I link a device?", "Pairing, QR codes and phone numbers", None, p))
                .child(ui::settings_row("How do I restore a backup?", "Local backups and media", None, p))
                .child(ui::settings_row_field("How do I change my theme?", "Appearance and dark mode", None, p))
                .child(
                    div().p(px(20.)).text_size(px(TEXT_SMALL)).text_color(p.text_secondary)
                        .child("These answers are local placeholders in this build."),
                ),
            SettingsSubpage::ContactUs => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Send a description of the problem. Do not include your password or private keys."),
                )
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("send-contact", "Send message", p),
                        Action::Toast("Message queued for support · local mock".into()),
                    )),
                ),
            SettingsSubpage::ReportProblem => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Attach a screenshot from the last session and describe what went wrong."),
                )
                .child(ui::settings_row_field("Screenshot", "Optional attachment", Some("Not attached".into()), p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("send-report", "Send report", p),
                        Action::Toast("Problem report queued · local mock".into()),
                    )),
                ),
            SettingsSubpage::LinkedDeviceLogout => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("iPhone 15 will lose access to WhatsLite. You can link it again at any time."),
                )
                .child(ui::settings_row_field("iPhone 15", "Last active 2 minutes ago", None, p))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .p(px(20.))
                        .child(self.clickable(
                            cx,
                            ui::secondary_button("cancel-device-logout", "Cancel", p),
                            Action::CloseSubpage,
                        ))
                        .child(self.clickable(
                            cx,
                            ui::danger_button("confirm-device-logout", "Log out", p),
                            Action::CloseSubpageAndToast("iPhone 15 logged out · local mock".into()),
                        )),
                ),
            SettingsSubpage::ChangeNumberOtp => {
                let boxes = (0..6).map(|_| {
                    div()
                        .size(px(38.))
                        .flex()
                        .items_center()
                        .justify_center()
                        .rounded(px(RADIUS_MD))
                        .border_1()
                        .border_color(p.border)
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("•")
                });
                div()
                    .flex()
                    .flex_col()
                    .gap(px(14.))
                    .p(px(20.))
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Enter the 6-digit code sent to +62 812 3456 7890."),
                    )
                    .child(div().flex().gap(px(8.)).children(boxes))
                    .child(self.clickable(
                        cx,
                        ui::primary_button("verify-number", "Verify", p),
                        Action::CloseSubpageAndToast("Phone number updated · local mock".into()),
                    ))
            }
            SettingsSubpage::RecoveryEmailCode => div()
                .flex()
                .flex_col()
                .gap(px(12.))
                .p(px(20.))
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("We sent a 6-digit code to your recovery email. Enter it to finish setting up recovery."),
                )
                .child(ui::settings_row_field("Email", "Recovery address", Some("Not set".into()), p))
                .child(ui::settings_row_field("Code", "6-digit code", Some("••••••".into()), p))
                .child(self.clickable(
                    cx,
                    ui::primary_button("verify-email", "Verify email", p),
                    Action::CloseSubpageAndToast("Recovery email verified · local mock".into()),
                )),
            SettingsSubpage::NotificationSound => div()
                .flex()
                .flex_col()
                .child(ui::section_label("SOUND", p))
                .child(self.subpage_option(cx, p, "Default", true, Action::Toast("Sound · Default".into())))
                .child(self.subpage_option(cx, p, "Chime", false, Action::Toast("Sound · Chime".into())))
                .child(self.subpage_option(cx, p, "Pop", false, Action::Toast("Sound · Pop".into())))
                .child(self.subpage_option(cx, p, "None", false, Action::Toast("Sound · None".into())))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-sound", "Save", p),
                        Action::ShowComingSoon("Saved locally — notification sound is a mock."),
                    )),
                ),
            SettingsSubpage::NotificationPreview => div()
                .flex()
                .flex_col()
                .child(ui::section_label("PREVIEW", p))
                .child(self.subpage_option(cx, p, "Sender and message", true, Action::Toast("Preview · Sender and message".into())))
                .child(self.subpage_option(cx, p, "Sender only", false, Action::Toast("Preview · Sender only".into())))
                .child(self.subpage_option(cx, p, "No preview", false, Action::Toast("Preview · No preview".into())))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-preview", "Save", p),
                        Action::ShowComingSoon("Saved locally — notification preview is a mock."),
                    )),
                ),
            SettingsSubpage::QuietHours => div()
                .flex()
                .flex_col()
                .child(ui::settings_row_field("From", "Quiet hours start", Some("22:00".into()), p))
                .child(ui::settings_row_field("To", "Quiet hours end", Some("07:00".into()), p))
                .child(ui::settings_row_field("Days", "Applies every day", Some("Every day".into()), p))
                .child(ui::settings_row_field("Allow calls", "Calls can still ring during quiet hours", Some("Off".into()), p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-quiet", "Save", p),
                        Action::ShowComingSoon("Saved locally — quiet hours are a mock."),
                    )),
                ),
            SettingsSubpage::BlockedContactPicker => div()
                .flex()
                .flex_col()
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Blocked people cannot call you or send messages. They are never told."),
                )
                .child(ui::settings_row_field("Promo Grosir", "Blocked", None, p))
                .child(ui::settings_row_field("Unknown +62 811 0000 111", "Blocked", None, p))
                .child(ui::settings_row_field("Unknown +62 812 9999 0000", "Blocked", None, p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("block-contact", "Block a contact", p),
                        Action::CloseSubpageAndToast("Contact blocked · local mock".into()),
                    )),
                ),
            SettingsSubpage::StoragePerChat => div()
                .flex()
                .flex_col()
                .child(ui::settings_row_field("Tim Produk", "612 MB", None, p))
                .child(ui::settings_row_field("Maya", "295 MB", None, p))
                .child(ui::settings_row_field("Jakarta Crew", "488 MB", None, p))
                .child(ui::settings_row_field("Dewi Lestari", "142 MB", None, p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("manage-storage", "Manage storage", p),
                        Action::Toast("Storage manager opened · local mock".into()),
                    )),
                ),
            SettingsSubpage::Language => div()
                .flex()
                .flex_col()
                .child(ui::section_label("APP LANGUAGE", p))
                .child(self.subpage_option(cx, p, "English", true, Action::Toast("Language · English".into())))
                .child(self.subpage_option(cx, p, "Bahasa Indonesia", false, Action::Toast("Language · Bahasa Indonesia".into())))
                .child(ui::section_label("TEXT DIRECTION", p))
                .child(self.subpage_option(cx, p, "Left-to-right", !self.rtl, Action::SetRtl(false)))
                .child(self.subpage_option(cx, p, "Right-to-left", self.rtl, Action::SetRtl(true)))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-language", "Save", p),
                        Action::ShowComingSoon("Saved locally — language is a mock preference."),
                    )),
                ),
            SettingsSubpage::Proxy => div()
                .flex()
                .flex_col()
                .child(ui::settings_row_field("Use proxy", "Connect through a proxy server", Some("Off".into()), p))
                .child(ui::settings_row_field("Host", "Proxy hostname", Some("Not set".into()), p))
                .child(ui::settings_row_field("Port", "Proxy port", Some("Not set".into()), p))
                .child(
                    div().p(px(20.)).child(self.clickable(
                        cx,
                        ui::primary_button("save-proxy", "Save", p),
                        Action::ShowComingSoon("Saved locally — proxy settings are a mock."),
                    )),
                ),
            SettingsSubpage::NetworkStates => div()
                .flex()
                .flex_col()
                .child(ui::section_label("NETWORK", p))
                .child(self.subpage_option(cx, p, "Online", self.network_state == NetworkState::Online, Action::SetNetworkState(NetworkState::Online)))
                .child(self.subpage_option(cx, p, "Connecting", self.network_state == NetworkState::Connecting, Action::SetNetworkState(NetworkState::Connecting)))
                .child(self.subpage_option(cx, p, "Offline", self.network_state == NetworkState::Offline, Action::SetNetworkState(NetworkState::Offline)))
                .child(self.subpage_option(cx, p, "Reconnecting", self.network_state == NetworkState::Reconnecting, Action::SetNetworkState(NetworkState::Reconnecting)))
                .child(ui::section_label("OUTBOX", p))
                .child(self.clickable(
                    cx,
                    ui::settings_row(
                        "Queued messages",
                        "Messages waiting for a connection",
                        Some(format!("{} queued", self.outbox)),
                        p,
                    ),
                    Action::FlushOutbox,
                ))
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Only Offline blocks sending; Connecting and Reconnecting queue silently. Flush marks queued messages as sent."),
                ),
            SettingsSubpage::Primitives => {
                let card = |title: &'static str, element: gpui_kit::AnyElement| {
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(8.))
                        .p(px(14.))
                        .rounded(px(RADIUS_LG))
                        .bg(p.bg_list)
                        .border_1()
                        .border_color(p.border)
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_secondary)
                                .child(title),
                        )
                        .child(element)
                };
                div()
                    .grid()
                    .grid_cols(2)
                    .gap(px(8.))
                    .p(px(16.))
                    .child(card(
                        "CHECKBOX",
                        Checkbox::new("primitive-checkbox")
                            .label("Checkbox")
                            .checked(true)
                            .into_any_element(),
                    ))
                    .child(card(
                        "RADIO",
                        Radio::new("primitive-radio")
                            .label("Radio")
                            .checked(true)
                            .into_any_element(),
                    ))
                    .child(card(
                        "PROGRESS",
                        Progress::new("primitive-progress").value(0.62).into_any_element(),
                    ))
                    .child(card(
                        "KEYCAP",
                        div()
                            .flex()
                            .items_center()
                            .gap(px(4.))
                            .child(Kbd::new(Keystroke::parse("ctrl-k").expect("valid keystroke")))
                            .child(Kbd::new(Keystroke::parse("enter").expect("valid keystroke")))
                            .into_any_element(),
                    ))
                    .child(card(
                        "AVATAR GROUP",
                        AvatarGroup::new()
                            .children([
                                Avatar::new().name("Rizky Pratama"),
                                Avatar::new().name("Maya Putri"),
                                Avatar::new().name("Dewi Lestari"),
                            ])
                            .into_any_element(),
                    ))
                    .child(card(
                        "BADGE",
                        div()
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .child(Badge::new().count(8))
                            .child(Badge::new().dot())
                            .into_any_element(),
                    ))
                    .child(card(
                        "TOOLTIP",
                        Button::new("primitive-tooltip")
                            .label("Hover me")
                            .tooltip("Tooltip from GPUI Kit")
                            .into_any_element(),
                    ))
                    .child(card(
                        "SELECT",
                        div()
                            .h(px(36.))
                            .flex()
                            .items_center()
                            .justify_between()
                            .px(px(10.))
                            .rounded(px(RADIUS_MD))
                            .bg(p.bg_search)
                            .border_1()
                            .border_color(p.border)
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_primary)
                            .child("Default (14)")
                            .child(ui::icon(IconName::ChevronDown, 14.0, p.text_secondary))
                            .into_any_element(),
                    ))
            }
            SettingsSubpage::CallStates => {
                let states = [
                    ("Dialing", CallState::Dialing),
                    ("Ringing", CallState::Ringing),
                    ("Connecting", CallState::Connecting),
                    ("Connected", CallState::Connected),
                    ("Reconnecting", CallState::Reconnecting),
                    ("Busy", CallState::Busy),
                    ("No answer", CallState::NoAnswer),
                    ("Ended by them", CallState::EndedByThem),
                    ("Declined by you", CallState::Declined),
                ];
                div()
                    .flex()
                    .flex_col()
                    .p(px(16.))
                    .gap(px(6.))
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Preview every call state with the same call surface."),
                    )
                    .child(ui::section_label("GROUP CALL SIZE", p))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(6.))
                            .px(px(12.))
                            .pb(px(8.))
                            .children([2usize, 3, 4, 6].into_iter().map(|size| {
                                self.clickable(
                                    cx,
                                    ui::chip(
                                        if size == 6 { "6+".to_owned() } else { size.to_string() },
                                        self.group_call_size == size,
                                        p,
                                    ),
                                    Action::SetGroupCallSize(size),
                                )
                            })),
                    )
                    .children(states.into_iter().map(|(label, state)| {
                        self.clickable(
                            cx,
                            div()
                                .id(format!("call-state-{label}"))
                                .flex()
                                .items_center()
                                .justify_between()
                                .px(px(12.))
                                .py(px(10.))
                                .rounded(px(RADIUS_MD))
                                .bg(p.bg_list)
                                .border_1()
                                .border_color(p.border)
                                .hover(move |style| style.bg(p.hover()))
                                .child(
                                    div()
                                        .text_size(px(TEXT_BODY))
                                        .text_color(p.text_primary)
                                        .child(label),
                                )
                                .child(ui::icon(IconName::Phone, 16.0, p.accent)),
                            Action::SetCallState(state),
                        )
                    }))
            }
            SettingsSubpage::SystemStatus => {
                let mut card = |id: &'static str, icon_name: IconName, title: &'static str, body: &'static str, state: SystemState| {
                    self.clickable(
                        cx,
                        div()
                            .id(id)
                            .flex()
                            .flex_col()
                            .gap(px(5.))
                            .p(px(14.))
                            .rounded(px(RADIUS_LG))
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .hover(move |style| style.bg(p.hover()))
                            .child(ui::icon(icon_name, 22.0, p.accent))
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_primary)
                                    .child(title),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child(body),
                            ),
                        Action::OpenSystemState(state),
                    )
                };
                div()
                    .grid()
                    .grid_cols(2)
                    .gap(px(8.))
                    .p(px(16.))
                    .child(card("state-os", IconName::Bell, "OS notification", "Windows toast with sender and preview", SystemState::OsNotification))
                    .child(card("state-permission", IconName::CameraOff, "Permission denied", "Camera and microphone blocked", SystemState::Permission))
                    .child(card("state-tray", IconName::Monitor, "Tray menu", "Open, mute and quit from the tray", SystemState::Tray))
                    .child(card("state-update", IconName::RefreshCw, "Update available", "WhatsLite 1.1 is ready to install", SystemState::Update))
                    .child(card("state-error", IconName::CircleAlert, "Error screen", "Something went wrong · restart or report", SystemState::Error))
                    .child(card("state-drop", IconName::FolderOpen, "Drag & drop", "Drop to send to Rizky Pratama", SystemState::DragDrop))
                    .child(card("state-paste", IconName::Clipboard, "Paste image", "Ctrl+V stages the image as an attachment", SystemState::Paste))
                    .child(card("state-toast", IconName::Info, "Toast stack", "Up to three toasts, newest on top", SystemState::ToastStack))
            }
            SettingsSubpage::Shortcuts => {
                let key = |label: String| {
                    div()
                        .px(px(6.))
                        .py(px(2.))
                        .rounded(px(RADIUS_SM))
                        .bg(p.bg_search)
                        .text_size(px(TEXT_CAPTION))
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(p.text_primary)
                        .child(label)
                };
                let rows = SHORTCUTS
                    .iter()
                    .enumerate()
                    .map(|(index, (label, _))| {
                        let keys = self.shortcut_keys(index);
                        let row = div()
                            .id(format!("shortcut-row-{index}"))
                            .flex()
                            .items_center()
                            .justify_between()
                            .gap(px(12.))
                            .px(px(20.))
                            .py(px(12.))
                            .border_b_1()
                            .border_color(p.divider)
                            .hover(move |style| style.bg(p.hover()))
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .text_color(p.text_primary)
                                    .child(label.to_owned()),
                            )
                            .child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(8.))
                                    .child(
                                        div()
                                            .flex()
                                            .items_center()
                                            .gap(px(4.))
                                            .children(keys.into_iter().map(key)),
                                    )
                                    .child(ui::icon(IconName::Pencil, 14.0, p.text_secondary)),
                            );
                        self.clickable(cx, row, Action::OpenShortcutEditor(index))
                    })
                    .collect::<Vec<_>>();
                div()
                    .flex()
                    .flex_col()
                    .child(
                        div()
                            .px(px(20.))
                            .py(px(12.))
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Select a shortcut to edit it. Esc always closes the top layer and cannot be removed."),
                    )
                    .children(rows)
                    .child(
                        div()
                            .p(px(20.))
                            .flex()
                            .items_center()
                            .justify_between()
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child("Focus rings and tab order are provided by GPUI Kit."),
                            )
                            .child(self.clickable(
                                cx,
                                ui::secondary_button("reset-shortcuts", "Reset all", p),
                                Action::ResetShortcuts,
                            )),
                    )
            }
            SettingsSubpage::DesignReference => {
                let tokens: [(&str, gpui_kit::Rgba); 28] = [
                    ("bg_sidebar", p.bg_sidebar),
                    ("bg_list", p.bg_list),
                    ("bg_conversation", p.bg_conversation),
                    ("bg_header", p.bg_header),
                    ("bg_search", p.bg_search),
                    ("bg_selected", p.bg_selected),
                    ("bg_row_selected", p.bg_row_selected),
                    ("bg_composer", p.bg_composer),
                    ("bg_notice_encrypt", p.bg_notice_encrypt),
                    ("bg_notice_system", p.bg_notice_system),
                    ("bubble_in", p.bubble_in),
                    ("bubble_out", p.bubble_out),
                    ("text_primary", p.text_primary),
                    ("text_secondary", p.text_secondary),
                    ("text_inverse", p.text_inverse),
                    ("accent", p.accent),
                    ("accent_soft", p.accent_soft),
                    ("unread_text", p.unread_text),
                    ("tick_read", p.tick_read),
                    ("border", p.border),
                    ("divider", p.divider),
                    ("attach_blue", p.attach_blue),
                    ("attach_orange", p.attach_orange),
                    ("attach_red", p.attach_red),
                    ("attach_green", p.attach_green),
                    ("traffic_red", p.traffic_red),
                    ("traffic_yellow", p.traffic_yellow),
                    ("traffic_green", p.traffic_green),
                ];
                let swatch = |name: &'static str, color: gpui_kit::Rgba| {
                    div()
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .min_w(px(0.))
                        .child(div().size(px(22.)).flex_shrink_0().rounded(px(RADIUS_SM)).bg(color).border_1().border_color(p.border))
                        .child(
                            div()
                                .min_w(px(0.))
                                .overflow_hidden()
                                .whitespace_nowrap()
                                .text_ellipsis()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(name),
                        )
                };
                let type_row = |label: &'static str, size: f32, sample: &'static str| {
                    div()
                        .w_full()
                        .flex()
                        .items_center()
                        .px(px(20.))
                        .py(px(8.))
                        .border_b_1()
                        .border_color(p.divider)
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .gap(px(10.))
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .text_color(p.text_secondary)
                                        .child(label),
                                )
                                .child(
                                    div()
                                        .px(px(8.))
                                        .py(px(2.))
                                        .rounded(px(RADIUS_SM))
                                        .bg(p.bg_search)
                                        .text_size(px(size))
                                        .text_color(p.text_primary)
                                        .child(sample),
                                ),
                        )
                };
                div()
                    .flex()
                    .flex_col()
                    .w_full()
                    .min_w(px(0.))
                    .child(
                        div()
                            .w_full()
                            .p(px(20.))
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("The 39-page handoff package is documentation, not WhatsApp features. This page mirrors its essentials, including the page 38 doodle wallpaper proof, so the build can be checked in-app."),
                    )
                    .child(ui::section_label("FOUNDATIONS · 01 COLOR TOKENS", p))
                    .child(
                        div()
                            .w_full()
                            .grid()
                            .grid_cols(2)
                            .gap(px(8.))
                            .p(px(20.))
                            .children(tokens.into_iter().map(|(name, color)| swatch(name, color))),
                    )
                    .child(ui::section_label("FOUNDATIONS · 02 TYPOGRAPHY", p))
                    .child(type_row("Caption · 11", 11., "Aa"))
                    .child(type_row("Small · 12", 12., "Aa"))
                    .child(type_row("Body · 14", 14., "Aa"))
                    .child(type_row("Heading · 20", 20., "Aa"))
                    .child(ui::section_label("FOUNDATIONS · 03 SPACING & RADIUS", p))
                    .child(
                        div()
                            .flex()
                            .flex_wrap()
                            .gap(px(6.))
                            .p(px(20.))
                            .children([2, 4, 6, 8, 10, 12, 16, 20, 24, 32, 40, 48].into_iter().map(|space| {
                                ui::chip(format!("{space}"), false, p)
                            }))
                            .children([4, 8, 10, 12, 18, 999].into_iter().map(|radius| {
                                ui::chip(format!("r{radius}"), false, p)
                            })),
                    )
                    .child(ui::section_label("FOUNDATIONS · 04 BORDERS & ELEVATION", p))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(10.))
                            .px(px(20.))
                            .py(px(10.))
                            .child(
                                div()
                                    .size(px(12.))
                                    .rounded(px(3.))
                                    .bg(p.border)
                                    .border_1()
                                    .border_color(p.border),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child("Border · active theme"),
                            )
                            .child(
                                div()
                                    .size(px(12.))
                                    .rounded_full()
                                    .bg(p.accent),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child("Accent · active theme"),
                            ),
                    )
                    .child(ui::settings_row_field(
                        "Hairline",
                        "1px inner · active theme border",
                        Some("1px inner".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "Focus ring",
                        "2px accent · 2px offset",
                        Some("Keyboard only".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "Bubble",
                        "0 1 2 / 8%",
                        Some("8%".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "Popover",
                        "0 8 24 / 12%",
                        Some("12%".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "Glass",
                        "macOS shell · translucent fill · 1px white 50% edge",
                        Some("macOS".into()),
                        p,
                    ))
                    .child(ui::section_label("FOUNDATIONS · 05 ICONOGRAPHY", p))
                    .child(ui::settings_row_field(
                        "Material Symbols Rounded",
                        "Outlined default · filled selected",
                        Some("Apache 2.0".into()),
                        p,
                    ))
                    .child(
                        div()
                            .flex()
                            .flex_wrap()
                            .gap(px(6.))
                            .px(px(20.))
                            .py(px(10.))
                            .children([
                                "16 · inline meta",
                                "18 · row actions",
                                "20 · navigation",
                                "24 · headers + composer",
                                "32 · empty states",
                            ]
                            .into_iter()
                            .map(|label| ui::chip(label, false, p))),
                    )
                    .child(ui::section_label("FOUNDATIONS · 06 LOGO", p))
                    .child(ui::settings_row_field(
                        "Mark",
                        "messages-square · Lucide · ISC",
                        Some("ISC".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "App icon",
                        "Squircle radius 28% of tile · clear space one stroke width",
                        Some("28%".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "Primary lockup",
                        "Mark + Inter 700 wordmark · gap = half mark width",
                        Some("Inter 700".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "Monochrome",
                        "One ink colour · never on the accent",
                        Some("1 ink".into()),
                        p,
                    ))
                    .child(
                        div()
                            .flex()
                            .flex_wrap()
                            .gap(px(6.))
                            .px(px(20.))
                            .py(px(10.))
                            .children(
                                ["64 px", "40 px", "28 px", "20 px", "16 px · tile only"]
                                    .into_iter()
                                    .map(|label| ui::chip(label, false, p)),
                            ),
                    )
                    .child(ui::section_label("FOUNDATIONS · 07 WALLPAPER DOODLE", p))
                    .child(
                        div()
                            .px(px(20.))
                            .py(px(12.))
                            .flex()
                            .items_center()
                            .gap(px(14.))
                            .child(
                                div()
                                    .relative()
                                    .w(px(168.))
                                    .h(px(96.))
                                    .flex_shrink_0()
                                    .rounded(px(RADIUS_SM))
                                    .overflow_hidden()
                                    .border_1()
                                    .border_color(p.border)
                                    .bg(p.bg_conversation)
                                    .child(ui::wallpaper(self.theme == ThemeMode::Dark)),
                            )
                            .child(
                                div()
                                    .flex_1()
                                    .flex()
                                    .flex_col()
                                    .gap(px(4.))
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child("1024 × 1024 seamless doodle SVG")
                                    .child("800 pt repeat · light #C9B9A3 60% · dark #D8CCBC 20%")
                                    .child("300 objects · 846 small icons · 520 dots · gaps ≥18 px")
                                    .child("Runtime: build-time alpha mask, tinted from the same SVG source"),
                            ),
                    )
                    .child(ui::section_label("FOUNDATIONS · 08 MOTION", p))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .justify_between()
                            .px(px(20.))
                            .py(px(10.))
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child("hover 120 · micro 160 · panel 200 · overlay 240 ms"),
                            )
                            .child(
                                div()
                                    .size(px(10.))
                                    .rounded_full()
                                    .bg(p.accent)
                                    .with_animation(
                                        "design-motion-dot",
                                        Animation::new(Duration::from_millis(1200)).repeat(),
                                        |el, delta| el.opacity(0.25 + 0.75 * delta),
                                    ),
                            ),
                    )
                    .child(self.toggle_row(cx, p, "reduce_motion", "Reduce motion", "Transitions become instant fades"))
                    .child(ui::section_label("COMPONENTS", p))
                    .child(self.clickable(
                        cx,
                        ui::settings_row("Primitives", "Checkbox, radio, progress, keycap, avatar group, badge, tooltip and select", None, p),
                        Action::OpenSubpage(SettingsSubpage::Primitives),
                    ))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(10.))
                            .px(px(20.))
                            .py(px(12.))
                            .child(Checkbox::new("design-checkbox").label("Checkbox").checked(true))
                            .child(Radio::new("design-radio").label("Radio").checked(true))
                            .child(Progress::new("design-progress").value(0.62))
                            .child(Kbd::new(Keystroke::parse("ctrl-k").expect("valid keystroke")))
                            .child(Badge::new().count(8)),
                    )
                    .child(ui::section_label("HANDOFF", p))
                    .child(ui::settings_row_field("State matrix", "175 rows across 31 pages · pages 31–37 are tables", Some("175".into()), p))
                    .child(ui::settings_row_field("List rule", "Every list ships empty, loading, failed and ready", Some("5 rows".into()), p))
                    .child(ui::settings_row_field("Focus ring", "2px accent, 2px offset, keyboard only", Some("D5.5".into()), p))
                    .child(ui::settings_row_field("Text scaling", "125% and 150% without clipping", Some("D5.12".into()), p))
                    .child(ui::settings_row_field("RTL", "Mirror layout, keep numbers LTR inside RTL text", Some("D5.1".into()), p))
                    .child(ui::section_label("AUDIT", p))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .px(px(20.))
                            .py(px(10.))
                            .child(ui::chip("P1 · 26", false, p))
                            .child(ui::chip("P2 · 62", false, p))
                            .child(ui::chip("P3 · 50", false, p)),
                    )
                    .child(
                        div()
                            .px(px(20.))
                            .pb(px(20.))
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Closed in this build: accounts, sync, network states, call states, status sequence, profile photo crop, shortcut editor, AI suggested reply/streaming/error, reduced motion, list states and the system-state surfaces."),
                    )
            }
            SettingsSubpage::About => div()
                .flex()
                .flex_col()
                .child(self.clickable(
                cx,
                ui::settings_row("WhatsLite for Windows", "1.0.0 (build 42)", None, p),
                Action::OpenSubpage(SettingsSubpage::About),
            ))
                .child(self.clickable(
                cx,
                ui::settings_row("Licences", "Inter — SIL Open Font License 1.1 · Material Symbols Rounded — Apache License 2.0", None, p),
                Action::OpenSubpage(SettingsSubpage::About),
            ))
                .child(
                    div()
                        .p(px(20.))
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("WhatsLite is an independent project. Not affiliated with WhatsApp or Meta."),
                ),
        };

        div()
            .flex_1()
            .h_full()
            .w_full()
            .min_w(px(0.))
            .flex()
            .flex_col()
            .bg(p.bg_conversation)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .px(px(24.))
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "subpage-back",
                            IconName::ArrowLeft,
                            18.0,
                            p.text_primary,
                            p.hover(),
                            "Back to settings",
                        ),
                        Action::CloseSubpage,
                    ))
                    .child(
                        div()
                            .text_size(px(TEXT_HEADING))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_primary)
                            .child(page.title()),
                    ),
            )
            .child(
                div()
                    .flex_1()
                    .w_full()
                    .min_w(px(0.))
                    .overflow_y_scrollbar()
                    .px(px(24.))
                    .pb(px(24.))
                    .child(
                        div()
                            .w_full()
                            .min_w(px(0.))
                            .rounded(px(RADIUS_XL))
                            .bg(p.bg_list)
                            .border_1()
                            .border_color(p.border)
                            .child(body),
                    ),
            )
    }

    fn subpage_option(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        label: &'static str,
        selected: bool,
        action: Action,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let row = div()
            .id(format!("option-{label}"))
            .flex()
            .items_center()
            .justify_between()
            .px(px(20.))
            .py(px(14.))
            .border_b_1()
            .border_color(p.divider)
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(10.))
                    .child(ui::icon(
                        if selected {
                            IconName::CircleCheck
                        } else {
                            IconName::Circle
                        },
                        18.0,
                        if selected { p.accent } else { p.text_secondary },
                    ))
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .text_color(p.text_primary)
                            .child(label),
                    ),
            )
            .when(selected, |el| {
                el.child(ui::icon(IconName::Check, 16.0, p.accent))
            });
        self.clickable(cx, row, action)
    }

    fn toggle_row(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        key: &'static str,
        title: &'static str,
        description: &'static str,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let enabled = self.toggles.contains(key);
        let row = div()
            .id(format!("toggle-{key}"))
            .flex()
            .items_center()
            .justify_between()
            .gap(px(16.))
            .px(px(20.))
            .py(px(12.))
            .border_b_1()
            .border_color(p.divider)
            .child(
                div()
                    .flex()
                    .flex_col()
                    .gap(px(2.))
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(p.text_primary)
                            .child(title),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(description),
                    ),
            )
            .child(
                div()
                    .w(px(42.))
                    .h(px(24.))
                    .rounded_full()
                    .bg(if enabled { p.accent } else { p.bg_search })
                    .p(px(2.))
                    .flex()
                    .when(enabled, |el| el.justify_end())
                    .child(div().size(px(20.)).rounded_full().bg(p.bg_list)),
            );
        let action = if key == "reduce_motion" {
            Action::SetReducedMotion(!enabled)
        } else {
            Action::ToggleSetting(key)
        };
        self.clickable(cx, row, action)
    }

    fn render_info_panel(
        &self,
        panel: InfoPanel,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        let chat = self
            .selected_chat()
            .cloned()
            .unwrap_or_else(|| model::chats()[0].clone());
        let content = match panel {
            InfoPanel::Contact => self.info_contact(cx, p, &chat),
            InfoPanel::Group => self.info_group(cx, p, &chat),
            InfoPanel::Community => self.info_community(cx, p),
        };
        div()
            .w(px(INFO_PANEL_WIDTH))
            .flex_shrink_0()
            .h_full()
            .flex()
            .flex_col()
            .bg(p.bg_list)
            .border_l_1()
            .border_color(p.divider)
            .child(
                div()
                    .h(px(58.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .border_b_1()
                    .border_color(p.divider)
                    .child(
                        div()
                            .text_size(px(17.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child(match panel {
                                InfoPanel::Contact => "Contact info",
                                InfoPanel::Group => "Group info",
                                InfoPanel::Community => "Community info",
                            }),
                    )
                    .child(self.clickable(
                        cx,
                        ui::icon_button_small(
                            "close-info",
                            IconName::X,
                            18.0,
                            p.text_secondary,
                            p.hover(),
                            "Close info",
                        ),
                        Action::CloseInfo,
                    )),
            )
            .child(div().flex_1().overflow_y_scrollbar().child(content))
    }

    fn render_info_overlay(
        &self,
        panel: InfoPanel,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        div()
            .absolute()
            .top_0()
            .right_0()
            .bottom_0()
            .left_0()
            .flex()
            .justify_end()
            .child(
                div()
                    .id("info-overlay-scrim")
                    .absolute()
                    .top_0()
                    .right_0()
                    .bottom_0()
                    .left_0()
                    .bg(p.text_primary)
                    .opacity(0.18)
                    .on_click(cx.listener(|this, _, window, cx| {
                        this.apply(Action::CloseInfo, window, cx);
                    })),
            )
            .child(self.render_info_panel(panel, cx, p))
    }

    fn info_contact(&self, cx: &mut Context<Self>, p: Palette, chat: &ChatModel) -> gpui_kit::Div {
        let contact = self
            .contacts
            .iter()
            .find(|contact| contact.name == chat.name)
            .cloned()
            .unwrap_or(Contact {
                id: 0,
                name: chat.name.clone(),
                initials: chat.initials.clone(),
                about: "Available".into(),
                phone: "+62 812 3456 7890".into(),
                online: chat.online,
                blocked: false,
                favorite: true,
            });
        div()
            .flex()
            .flex_col()
            .child(
                div()
                    .flex()
                    .flex_col()
                    .items_center()
                    .gap(px(6.))
                    .p(px(20.))
                    .child(ui::avatar(contact.initials.clone(), 88., contact.online, p))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_primary)
                            .child(contact.name.clone()),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(format!("{} · online", contact.phone)),
                    ),
            )
            .child(
                div()
                    .flex()
                    .gap(px(8.))
                    .px(px(16.))
                    .pb(px(16.))
                    .child(self.clickable(
                        cx,
                        info_action("info-message", IconName::MessageSquare, "Message", p),
                        Action::SelectChat(self.selected_chat),
                    ))
                    .child(self.clickable(
                        cx,
                        info_action("info-video", IconName::Video, "Video", p),
                        Action::StartCall(self.selected_chat, true),
                    ))
                    .child(self.clickable(
                        cx,
                        info_action("info-voice", IconName::Phone, "Voice", p),
                        Action::StartCall(self.selected_chat, false),
                    )),
            )
            .child(ui::section_label("MEDIA, LINKS AND DOCS", p))
            .child(
                div()
                    .grid()
                    .grid_cols(3)
                    .gap(px(6.))
                    .px(px(16.))
                    .pb(px(12.))
                    .children((0..3).map(|index| {
                        div()
                            .h(px(84.))
                            .rounded(px(RADIUS_MD))
                            .bg(if index % 2 == 0 {
                                p.attach_orange
                            } else {
                                p.attach_blue
                            })
                            .flex()
                            .items_center()
                            .justify_center()
                            .child(ui::icon(IconName::Image, 22.0, p.text_inverse))
                    })),
            )
            .child(self.clickable(
                cx,
                ui::settings_row("Mute notifications", "Silence this chat", None, p),
                Action::CloseOverlayAndToast("Muted for 8 hours · local demo".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Disappearing messages", "24 hours", None, p),
                Action::OpenDisappearingPicker,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Chat lock", "Hide this chat", None, p),
                Action::CloseOverlayAndToast("Chat lock is coming soon.".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Starred messages", "6 messages", None, p),
                Action::Navigate(Route::Starred),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Encryption",
                    "Messages and calls are end-to-end encrypted",
                    None,
                    p,
                ),
                Action::OpenSecurityCode,
            ))
            .child(
                div().px(px(16.)).py(px(10.)).child(
                    self.clickable(
                        cx,
                        div()
                            .id("block-contact")
                            .text_size(px(TEXT_BODY))
                            .text_color(p.danger())
                            .child("Block contact"),
                        Action::Toast("Contact blocked · local mock".into()),
                    ),
                ),
            )
    }

    fn group_member_row(
        &self,
        cx: &mut Context<Self>,
        p: Palette,
        name: &'static str,
        role: &'static str,
    ) -> gpui_kit::Stateful<gpui_kit::Div> {
        let action = if name == "You" {
            Action::CloseOverlayAndToast("Opening your profile…".into())
        } else {
            Action::OpenContactProfile(name.to_owned())
        };
        self.clickable(
            cx,
            div()
                .id(format!("group-member-{name}"))
                .flex()
                .items_center()
                .gap(px(10.))
                .px(px(16.))
                .py(px(8.))
                .cursor_pointer()
                .hover(move |style| style.bg(p.hover()))
                .child(ui::avatar(initials_for(Some(name)), 34., false, p))
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(1.))
                        .child(
                            div()
                                .text_size(px(TEXT_BODY))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(p.text_primary)
                                .child(name),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_CAPTION))
                                .text_color(p.text_secondary)
                                .child(role),
                        ),
                )
                .child(div().flex_1()),
            action,
        )
    }

    fn info_group(&self, cx: &mut Context<Self>, p: Palette, chat: &ChatModel) -> gpui_kit::Div {
        div()
            .flex()
            .flex_col()
            .child(
                div()
                    .flex()
                    .flex_col()
                    .items_center()
                    .gap(px(6.))
                    .p(px(20.))
                    .child(ui::avatar(chat.initials.clone(), 88., false, p))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_primary)
                            .child(chat.name.clone()),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child("Group · 4 members"),
                    ),
            )
            .child(
                div()
                    .flex()
                    .gap(px(8.))
                    .px(px(16.))
                    .pb(px(16.))
                    .child(self.clickable(
                        cx,
                        info_action("group-add", IconName::UserPlus, "Add member", p),
                        Action::OpenGroupMembers,
                    ))
                    .child(self.clickable(
                        cx,
                        info_action("group-invite", IconName::Link, "Invite", p),
                        Action::Toast("Invite link copied · local mock".into()),
                    ))
                    .child(self.clickable(
                        cx,
                        info_action("group-video", IconName::Video, "Video", p),
                        Action::StartCall(self.selected_chat, true),
                    )),
            )
            .child(ui::section_label("4 MEMBERS", p))
            .child(self.group_member_row(cx, p, "You", "Admin"))
            .child(self.group_member_row(cx, p, "Rina", "Admin"))
            .child(self.group_member_row(cx, p, "Bayu", "Member"))
            .child(self.group_member_row(cx, p, "Sari", "Member"))
            .child(ui::section_label("MEDIA, LINKS AND DOCS", p))
            .child(
                div()
                    .grid()
                    .grid_cols(3)
                    .gap(px(6.))
                    .px(px(16.))
                    .pb(px(12.))
                    .children((0..3).map(|index| {
                        div()
                            .h(px(84.))
                            .rounded(px(RADIUS_MD))
                            .bg(if index % 2 == 0 {
                                p.attach_orange
                            } else {
                                p.attach_blue
                            })
                            .flex()
                            .items_center()
                            .justify_center()
                            .child(ui::icon(IconName::Image, 22.0, p.text_inverse))
                    })),
            )
            .child(self.clickable(
                cx,
                ui::settings_row("Approve new members", "Only admins can add people", None, p),
                Action::OpenMemberApprovals,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Disappearing messages", "24 hours", None, p),
                Action::OpenDisappearingPicker,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Mute notifications", "Silence this group", None, p),
                Action::CloseOverlayAndToast("Muted for 8 hours · local demo".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Encryption",
                    "Messages and calls are end-to-end encrypted",
                    None,
                    p,
                ),
                Action::OpenSecurityCode,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Clear chat", "Messages are removed for you only", None, p),
                Action::OpenClearChat,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Export chat", "Text only or with media", None, p),
                Action::OpenExportChat,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Security code", "Scan to verify with Rizky", None, p),
                Action::OpenSecurityCode,
            ))
            .child(
                div().px(px(16.)).py(px(10.)).child(
                    self.clickable(
                        cx,
                        div()
                            .id("exit-group")
                            .text_size(px(TEXT_BODY))
                            .text_color(p.danger())
                            .child("Exit group"),
                        Action::Toast("You left the group · local mock".into()),
                    ),
                ),
            )
    }

    fn info_community(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let community = self
            .communities
            .get(self.selected_community.unwrap_or(0))
            .cloned()
            .unwrap_or_else(|| model::communities()[0].clone());
        div()
            .flex()
            .flex_col()
            .child(
                div()
                    .flex()
                    .flex_col()
                    .items_center()
                    .gap(px(6.))
                    .p(px(20.))
                    .child(ui::avatar(community.initials.clone(), 88., false, p))
                    .child(
                        div()
                            .text_size(px(20.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_primary)
                            .child(community.name.clone()),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(format!("{} · {}", community.members, community.groups)),
                    ),
            )
            .child(
                div()
                    .px(px(16.))
                    .pb(px(16.))
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child(community.description.clone()),
            )
            .child(self.clickable(
                cx,
                ui::settings_row("Announcements", "Only admins can post", None, p),
                Action::CloseOverlayAndToast("Announcements · local demo".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Groups", community.groups.clone(), None, p),
                Action::CloseOverlayAndToast("Community groups · local demo".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Members", community.members.clone(), None, p),
                Action::OpenCommunityMembers,
            ))
            .child(self.clickable(
                cx,
                ui::settings_row("Mute notifications", "Silence this community", None, p),
                Action::CloseOverlayAndToast("Muted community for 8 hours · local demo".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Edit community info",
                    "Name, description and photo",
                    None,
                    p,
                ),
                Action::Toast("Community info editor opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Community settings",
                    "Approvals, announcements and groups",
                    None,
                    p,
                ),
                Action::Toast("Community settings opened · local mock".into()),
            ))
            .child(self.clickable(
                cx,
                ui::settings_row(
                    "Approve new members",
                    "Requests waiting for an admin",
                    Some(self.member_requests.len().to_string()),
                    p,
                ),
                Action::OpenMemberApprovals,
            ))
            .child(
                div().px(px(16.)).py(px(10.)).child(
                    self.clickable(
                        cx,
                        div()
                            .id("exit-community")
                            .text_size(px(TEXT_BODY))
                            .text_color(p.danger())
                            .child("Exit community"),
                        Action::Toast("You left the community · local mock".into()),
                    ),
                ),
            )
    }

    fn status_composer(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let tile = |index: usize, selected: bool| {
            div()
                .id(format!("status-media-{index}"))
                .relative()
                .h(px(96.))
                .rounded(px(RADIUS_MD))
                .bg(match index % 3 {
                    0 => p.attach_orange,
                    1 => p.attach_blue,
                    _ => p.attach_green,
                })
                .border_2()
                .border_color(if selected { p.accent } else { p.bg_list })
                .flex()
                .items_center()
                .justify_center()
                .child(ui::icon(IconName::Image, 26.0, p.text_inverse))
                .when(selected, |el| {
                    el.child(
                        div()
                            .absolute()
                            .top(px(6.))
                            .left(px(6.))
                            .size(px(20.))
                            .flex()
                            .items_center()
                            .justify_center()
                            .rounded_full()
                            .bg(p.accent)
                            .text_size(px(10.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_inverse)
                            .child(
                                self.status_items
                                    .iter()
                                    .position(|item| *item == index)
                                    .map(|position| (position + 1).to_string())
                                    .unwrap_or_default(),
                            ),
                    )
                })
        };

        let mut grid = div().grid().grid_cols(4).gap(px(8.));
        for index in 0..12 {
            let is_selected = self.status_items.contains(&index);
            grid = grid.child(self.clickable_modal(
                cx,
                tile(index, is_selected),
                Action::ToggleStatusItem(index),
            ));
        }

        let count = self.status_items.len();
        let item_word = if count == 1 { "item" } else { "items" };
        let can_send = count > 0;
        let send = if can_send {
            self.clickable_modal(
                cx,
                ui::primary_button("send-status", format!("Send {count} {item_word}"), p),
                Action::SendStatus,
            )
        } else {
            div()
                .id("send-status-disabled")
                .h(px(36.))
                .px(px(14.))
                .flex()
                .items_center()
                .rounded(px(RADIUS_MD))
                .bg(p.bg_search)
                .text_size(px(TEXT_SMALL))
                .text_color(p.text_secondary)
                .child("Send")
        };

        let slots = self
            .status_items
            .iter()
            .enumerate()
            .map(|(position, media)| {
                let remove = self.clickable_modal(
                    cx,
                    div()
                        .id(format!("remove-status-{position}"))
                        .absolute()
                        .top(px(1.))
                        .right(px(1.))
                        .size(px(14.))
                        .flex()
                        .items_center()
                        .justify_center()
                        .rounded_full()
                        .bg(p.bg_list)
                        .opacity(0.9)
                        .child(ui::icon(IconName::X, 9.0, p.text_primary)),
                    Action::RemoveStatusItem(position),
                );
                let drag_app = cx.entity();
                let drop_app = cx.entity();
                let palette = p;
                let accent = *media;
                let target = position;
                let constructor = move |data: &StatusDrag,
                                        _offset,
                                        _window: &mut Window,
                                        cx: &mut gpui_kit::App| {
                    let from = data.from;
                    drag_app.update(cx, |this, cx| {
                        this.status_drag = Some(from);
                        cx.notify();
                    });
                    cx.new(|_| StatusDragPreview { palette, accent })
                };
                let lifted = self.status_drag == Some(position);
                div()
                    .id(format!("status-slot-{position}"))
                    .relative()
                    .w(px(56.))
                    .h(px(44.))
                    .flex_shrink_0()
                    .rounded(px(RADIUS_SM))
                    .bg(match media % 3 {
                        0 => p.attach_orange,
                        1 => p.attach_blue,
                        _ => p.attach_green,
                    })
                    .border_2()
                    .border_color(if lifted { p.accent } else { p.bg_list })
                    .when(lifted, |el| el.opacity(0.55).shadow_sm())
                    .flex()
                    .items_center()
                    .justify_center()
                    .child(ui::icon(IconName::Image, 16.0, p.text_inverse))
                    .child(
                        div()
                            .absolute()
                            .bottom(px(1.))
                            .left(px(3.))
                            .text_size(px(9.))
                            .font_weight(FontWeight::BOLD)
                            .text_color(p.text_inverse)
                            .child((position + 1).to_string()),
                    )
                    .child(remove)
                    .on_drag(StatusDrag { from: position }, constructor)
                    .on_drop::<StatusDrag>(move |data, _window, cx| {
                        let from = data.from;
                        drop_app.update(cx, |this, cx| {
                            reorder_items(&mut this.status_items, from, target);
                            this.status_drag = None;
                            cx.notify();
                        });
                    })
            })
            .collect::<Vec<_>>();

        let add = if count >= STATUS_ITEM_LIMIT {
            div()
                .id("add-status-item-disabled")
                .h(px(36.))
                .flex()
                .items_center()
                .justify_center()
                .rounded(px(RADIUS_MD))
                .bg(p.bg_search)
                .text_size(px(TEXT_SMALL))
                .text_color(p.text_secondary)
                .child("Add another item")
        } else {
            self.clickable_modal(
                cx,
                ui::secondary_button("add-status-item", "Add another item", p),
                Action::AddStatusItem,
            )
        };

        modal_frame(
            p,
            div()
                .w(px(1100.))
                .h(px(700.))
                .flex()
                .flex_col()
                .rounded(px(RADIUS_XL))
                .bg(p.bg_list)
                .border_1()
                .border_color(p.border)
                .child(
                    div()
                        .h(px(56.))
                        .flex()
                        .items_center()
                        .gap(px(12.))
                        .px(px(20.))
                        .border_b_1()
                        .border_color(p.divider)
                        .child(self.clickable_modal(
                            cx,
                            ui::icon_button_small("status-composer-close", IconName::X, 18.0, p.text_primary, p.hover(), "Close"),
                            Action::CloseOverlay,
                        ))
                        .child(
                            div()
                                .text_size(px(17.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("New status"),
                        )
                        .child(div().flex_1())
                        .child(ui::chip("Contacts", false, p))
                        .child(send),
                )
                .child(
                    div()
                        .flex_1()
                        .flex()
                        .min_h(px(0.))
                        .child(
                            div()
                                .flex_1()
                                .flex()
                                .flex_col()
                                .gap(px(10.))
                                .p(px(20.))
                                .child(
                                    div()
                                        .flex()
                                        .items_center()
                                        .justify_between()
                                        .child(
                                            div()
                                                .text_size(px(14.))
                                                .font_weight(FontWeight::SEMIBOLD)
                                                .text_color(p.text_primary)
                                                .child("Camera roll · September"),
                                        )
                                        .child(
                                            div()
                                                .text_size(px(TEXT_SMALL))
                                                .font_weight(FontWeight::MEDIUM)
                                                .text_color(if count >= STATUS_ITEM_LIMIT { p.attach_orange } else { p.accent })
                                                .child(format!("{count} of {STATUS_ITEM_LIMIT} selected")),
                                        ),
                                )
                                .child(grid),
                        )
                        .child(
                            div()
                                .w(px(420.))
                                .flex()
                                .flex_col()
                                .gap(px(10.))
                                .p(px(20.))
                                .border_l_1()
                                .border_color(p.divider)
                                .child(
                                    div()
                                        .flex()
                                        .items_center()
                                        .justify_between()
                                        .child(
                                            div()
                                                .text_size(px(14.))
                                                .font_weight(FontWeight::SEMIBOLD)
                                                .text_color(p.text_primary)
                                                .child(format!("Sequence · {count} {item_word}")),
                                        )
                                        .child(
                                            div()
                                                .text_size(px(TEXT_CAPTION))
                                                .text_color(p.text_secondary)
                                                .child("Drag to reorder"),
                                        ),
                                )
                                .child(
                                    div()
                                        .flex()
                                        .gap(px(6.))
                                        .overflow_x_scrollbar()
                                        .children(slots),
                                )
                                .when(count >= STATUS_ITEM_LIMIT, |el| {
                                    el.child(
                                        div()
                                            .text_size(px(TEXT_CAPTION))
                                            .text_color(p.attach_orange)
                                            .child("30 item limit reached — remove one to add another."),
                                    )
                                })
                                .when(count == 0, |el| {
                                    el.child(
                                        div()
                                            .text_size(px(TEXT_CAPTION))
                                            .text_color(p.text_secondary)
                                            .child("Pick at least one item from the camera roll."),
                                    )
                                })
                                .child(add)
                                .child(
                                    div()
                                        .flex()
                                        .justify_between()
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.text_secondary)
                                        .child("Caption · item 2")
                                        .child("48/700"),
                                )
                                .child(
                                    div()
                                        .h(px(100.))
                                        .rounded(px(RADIUS_MD))
                                        .border_1()
                                        .border_color(p.accent)
                                        .p(px(10.))
                                        .text_size(px(TEXT_SMALL))
                                        .text_color(p.text_primary)
                                        .child("Motif anyaman baru — tiga warna, semua natural dye."),
                                )
                                .child(self.clickable_modal(
                                    cx,
                                    ui::settings_row(
                                        "Text font",
                                        "For text-only status",
                                        Some(format!(
                                            "{} {} Bold",
                                            ["Inter", "Georgia", "Mono"][self.status_font],
                                            [18, 22, 26][self.status_font_size]
                                        )),
                                        p,
                                    ),
                                    Action::OpenStatusFont,
                                ))
                                .child(self.clickable_modal(
                                    cx,
                                    ui::settings_row(
                                        "Privacy",
                                        "Who can see this status",
                                        Some(
                                            ["Everyone", "My contacts", "My contacts except…", "Only share with…"]
                                                [self.status_privacy]
                                                .into(),
                                        ),
                                        p,
                                    ),
                                    Action::OpenStatusPrivacy,
                                ))
                                .child(ui::settings_row_field("Expires", "How long the status stays", Some("24 hours".into()), p))
                                .child(ui::settings_row_field("Also share to", "Extra destinations", Some("My status".into()), p))
                                .child(
                                    div()
                                        .p(px(10.))
                                        .rounded(px(RADIUS_MD))
                                        .bg(p.accent_soft)
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.unread_text)
                                        .child("Each item expires together — one status, one 24-hour clock."),
                                ),
                        ),
                ),
        )
    }

    fn status_privacy_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let options = [
            ("Everyone", "Anyone who has your number"),
            ("My contacts", "People you have saved"),
            ("My contacts except…", "Hide from specific people"),
            ("Only share with…", "Pick people to include"),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Who can see my status"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Changes here apply to this status only. Defaults live in Settings › Privacy."),
                )
                .children(options.into_iter().enumerate().map(|(index, (label, description))| {
                    self.clickable_modal(
                        cx,
                        ui::settings_row(
                            label,
                            description,
                            Some(if self.status_privacy == index { "Selected".into() } else { String::new() }),
                            p,
                        ),
                        Action::SetStatusPrivacy(index),
                    )
                }))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("save-status-privacy", "Done", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn group_members_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let members = [
            ("You", "Admin"),
            ("Rina", "Admin"),
            ("Bayu", "Member"),
            ("Sari", "Member"),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Group members"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("4 members · 2 admins. Search, promote, demote or remove people."),
                )
                .child(ui::settings_row_field(
                    "Search",
                    "Name or phone number",
                    Some("Search members".into()),
                    p,
                ))
                .children(members.into_iter().map(|(name, role)| {
                    div()
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .px(px(10.))
                        .py(px(8.))
                        .rounded(px(RADIUS_MD))
                        .hover(move |style| style.bg(p.hover()))
                        .child(
                            self.clickable_modal(
                                cx,
                                div()
                                    .id(format!("open-profile-{name}"))
                                    .flex_1()
                                    .flex()
                                    .items_center()
                                    .gap(px(10.))
                                    .cursor_pointer()
                                    .child(ui::avatar(initials_for(Some(name)), 34., false, p))
                                    .child(
                                        div()
                                            .flex()
                                            .flex_col()
                                            .child(
                                                div()
                                                    .text_size(px(TEXT_BODY))
                                                    .font_weight(FontWeight::MEDIUM)
                                                    .text_color(p.text_primary)
                                                    .child(name),
                                            )
                                            .child(
                                                div()
                                                    .text_size(px(TEXT_CAPTION))
                                                    .text_color(p.text_secondary)
                                                    .child(role),
                                            ),
                                    ),
                                if name == "You" {
                                    Action::CloseOverlayAndToast("Opening your profile…".into())
                                } else {
                                    Action::OpenContactProfile(name.to_owned())
                                },
                            ),
                        )
                        .child(ui::chip(
                            if role == "Admin" { "Demote" } else { "Promote" },
                            false,
                            p,
                        ))
                        .when(name != "You", |el| {
                            el.child(self.clickable_modal(
                                cx,
                                ui::icon_button_small(
                                    "remove-member",
                                    IconName::X,
                                    12.0,
                                    p.text_secondary,
                                    p.hover(),
                                    "Remove member",
                                ),
                                Action::Toast(format!("Removed {name} · local mock")),
                            ))
                        })
                }))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("add-group-member", "Add member", p),
                            Action::Toast("Add member sheet opened · local mock".into()),
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("close-group-members", "Done", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn use_case_demo_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let scenarios = [
            (
                "Group chat · identity",
                "Hover the title for the chevron, click a sender name for their profile.",
            ),
            (
                "Contact profile",
                "Click Rina in the group. Saved name, number, username rule and actions.",
            ),
            (
                "Header dropdown",
                "Open the chat menu beside the title: info, search, media, mute, clear.",
            ),
            (
                "Attachment menu",
                "Every attach action opens a surface; sending is deferred.",
            ),
            (
                "Message actions",
                "Reply, react, edit, forward, star, select and delete are reachable.",
            ),
            (
                "Search in chat",
                "Search field with hit count, previous and next controls.",
            ),
            (
                "Group info",
                "Members open profiles. Add member opens the members sheet.",
            ),
            (
                "Create group",
                "The full create flow is clickable; the new group stays local until the backend exists.",
            ),
            (
                "Delete message",
                "Confirmation is wired; the change stays local until the backend exists.",
            ),
            (
                "Edit contact",
                "The form opens and validates; saving stays local until the backend exists.",
            ),
            (
                "Chat list menu",
                "The ellipsis in the Chats header opens create, find and organise.",
            ),
            (
                "Theme swap",
                "Toggle light and dark without changing the layout.",
            ),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .w(px(520.))
                .gap(px(10.))
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .child(ui::icon(IconName::Sparkles, 20.0, p.accent))
                        .child(
                            div()
                                .text_size(px(18.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Use case demo"),
                        ),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Every control is connected. Add, delete and edit open their surfaces; commits stay local until the real backend is wired."),
                )
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .children(scenarios.into_iter().enumerate().map(|(index, (title, description))| {
                    self.clickable_modal(
                        cx,
                        div()
                            .id(format!("demo-step-{index}"))
                            .flex()
                            .items_center()
                            .gap(px(10.))
                            .px(px(10.))
                            .py(px(6.))
                            .rounded(px(RADIUS_MD))
                            .cursor_pointer()
                            .hover(move |style| style.bg(p.hover()))
                            .child(
                                div()
                                    .size(px(24.))
                                    .flex()
                                    .items_center()
                                    .justify_center()
                                    .rounded_full()
                                    .bg(p.accent_soft)
                                    .text_size(px(TEXT_CAPTION))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.unread_text)
                                    .child(format!("{}", index + 1)),
                            )
                            .child(
                                div()
                                    .flex_1()
                                    .flex()
                                    .flex_col()
                                    .gap(px(1.))
                                    .child(
                                        div()
                                            .text_size(px(TEXT_BODY))
                                            .font_weight(FontWeight::MEDIUM)
                                            .text_color(p.text_primary)
                                            .child(title),
                                    )
                                    .child(
                                        div()
                                            .text_size(px(TEXT_CAPTION))
                                            .text_color(p.text_secondary)
                                            .child(description),
                                    ),
                            )
                            .child(ui::icon(IconName::ChevronRight, 15.0, p.text_secondary)),
                        Action::StartDemo(index),
                    )
                }))
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("demo-close", "Close", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("demo-start", "Open group chat", p),
                            Action::StartDemo(0),
                        )),
                ),
        )
    }

    fn clear_chat_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Clear this chat?"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Messages are removed for you only. The other person keeps their copy. This cannot be undone."),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-clear-chat", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::danger_button("confirm-clear-chat", "Clear chat", p),
                            Action::ConfirmClearChat,
                        )),
                ),
        )
    }

    fn export_chat_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Export chat"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Export includes the messages in this chat. Media can be included or left out."),
                )
                .child(self.clickable_modal(
                    cx,
                    ui::settings_row("Without media", "Text only · smaller file", Some("TXT".into()), p),
                    Action::CloseOverlayAndToast("Chat exported without media · local mock".into()),
                ))
                .child(self.clickable_modal(
                    cx,
                    ui::settings_row("With media", "Text, photos, videos and documents", Some("ZIP".into()), p),
                    Action::CloseOverlayAndToast("Chat exported with media · local mock".into()),
                ))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-export", "Cancel", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn security_code_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let cells = (0..64).map(|index| {
            let dark = (index * 7 + index / 3) % 5 < 2;
            div()
                .size(px(7.))
                .bg(if dark { p.text_primary } else { p.bg_list })
        });
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Security code"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Scan this code with Rizky to verify the end-to-end encryption. The code changes when either device is re-linked."),
                )
                .child(
                    div()
                        .flex()
                        .justify_center()
                        .child(
                            div()
                                .grid()
                                .grid_cols(8)
                                .gap(px(2.))
                                .p(px(10.))
                                .rounded(px(RADIUS_MD))
                                .bg(p.bg_list)
                                .border_1()
                                .border_color(p.border)
                                .children(cells),
                        ),
                )
                .child(
                    div()
                        .text_size(px(TEXT_BODY))
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(p.text_primary)
                        .text_center()
                        .child("4821 9930 7712 6640"),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("copy-security-code", "Copy code", p),
                            Action::Toast("Security code copied · local mock".into()),
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("scan-security-code", "Scan code", p),
                            Action::CloseOverlayAndToast("Scanner opened · local mock".into()),
                        )),
                ),
        )
    }

    fn channel_info_sheet(
        &self,
        name: String,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .child(
                            div()
                                .size(px(48.))
                                .flex()
                                .items_center()
                                .justify_center()
                                .rounded_full()
                                .bg(p.accent_soft)
                                .child(ui::icon(IconName::Megaphone, 22.0, p.accent)),
                        )
                        .child(
                            div()
                                .flex()
                                .flex_col()
                                .child(
                                    div()
                                        .flex()
                                        .items_center()
                                        .gap(px(5.))
                                        .text_size(px(18.))
                                        .font_weight(FontWeight::SEMIBOLD)
                                        .text_color(p.text_primary)
                                        .child(name)
                                        .child(ui::icon(IconName::BadgeCheck, 16.0, p.attach_blue)),
                                )
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .text_color(p.text_secondary)
                                        .child("Channel · 12.4k followers"),
                                ),
                        ),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Calm interfaces, issue 14. Notes on design systems, typography and the small details that make desktop apps feel right."),
                )
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("follow-channel", if self.channel_following { "Following" } else { "Follow" }, p),
                            Action::ToggleChannelFollow,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("report-channel", "Report", p),
                            Action::Toast("Channel report queued · local mock".into()),
                        )),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("close-channel", "Close", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn community_members_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let members = [
            ("You", "Admin"),
            ("Rina", "Admin"),
            ("Bayu", "Member"),
            ("Sari", "Member"),
            ("Dewi", "Member"),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Community members"),
                )
                .child(ui::settings_row_field(
                    "Search",
                    "Name or phone number",
                    Some("Search members".into()),
                    p,
                ))
                .children(members.into_iter().map(|(name, role)| {
                    ui::settings_row_field(
                        name,
                        role,
                        Some(if role == "Admin" {
                            "Admin".into()
                        } else {
                            String::new()
                        }),
                        p,
                    )
                }))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("community-approvals", "Approvals", p),
                            Action::OpenMemberApprovals,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("close-community-members", "Done", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn member_approvals_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Approve new members"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Requests waiting for an admin. Approving adds the person to the announcement group."),
                )
                .when(self.member_requests.is_empty(), |el| {
                    el.child(ui::empty_state_compact(
                        "No pending requests",
                        "New requests will appear here.",
                        None,
                        p,
                    ))
                })
                .children(self.member_requests.iter().enumerate().map(|(index, name)| {
                    div()
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .px(px(10.))
                        .py(px(8.))
                        .child(ui::avatar(initials_for(Some(name)), 34., false, p))
                        .child(
                            div()
                                .flex_1()
                                .text_size(px(TEXT_BODY))
                                .text_color(p.text_primary)
                                .child(name.clone()),
                        )
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("decline-member", "Decline", p),
                            Action::Toast(format!("Declined {name} · local mock")),
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("approve-member", "Approve", p),
                            Action::ApproveMember(index),
                        ))
                }))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("close-approvals", "Done", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn status_viewers_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let viewers = [
            ("Rizky Pratama", "Seen 2 min ago"),
            ("Maya", "Seen 4 min ago"),
            ("Rina", "Seen 7 min ago"),
            ("Bayu", "Seen 9 min ago"),
            ("Sari", "Seen 12 min ago"),
            ("Dewi Lestari", "Seen 14 min ago"),
            ("Jakarta Crew", "Seen 18 min ago"),
            ("Budi Santoso", "Seen 22 min ago"),
            ("Promo Grosir", "Seen 31 min ago"),
            ("Bank BCA", "Seen 44 min ago"),
            ("Studio Linen", "Seen 1 hour ago"),
            ("Kain Nusantara", "Seen 2 hours ago"),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .h(px(620.))
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Seen by 12"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Viewers are ordered by when they saw this status."),
                )
                .child(
                    div()
                        .flex_1()
                        .overflow_y_scrollbar()
                        .children(viewers.into_iter().map(|(name, time)| {
                            div()
                                .flex()
                                .items_center()
                                .gap(px(10.))
                                .px(px(8.))
                                .py(px(6.))
                                .child(ui::avatar(initials_for(Some(name)), 34., false, p))
                                .child(
                                    div()
                                        .flex_1()
                                        .flex()
                                        .flex_col()
                                        .child(
                                            div()
                                                .text_size(px(TEXT_BODY))
                                                .text_color(p.text_primary)
                                                .child(name),
                                        )
                                        .child(
                                            div()
                                                .text_size(px(TEXT_CAPTION))
                                                .text_color(p.text_secondary)
                                                .child(time),
                                        ),
                                )
                                .child(ui::icon(IconName::Eye, 14.0, p.text_secondary))
                        })),
                )
                .child(div().flex().justify_end().child(self.clickable_modal(
                    cx,
                    ui::primary_button("close-viewers", "Done", p),
                    Action::CloseOverlay,
                ))),
        )
    }

    fn forward_sheet(
        &self,
        message_index: usize,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Forward to"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Pick one chat. Forwarding keeps the original message unchanged."),
                )
                .children(self.chats.iter().enumerate().map(|(chat_index, chat)| {
                    self.clickable_modal(
                        cx,
                        div()
                            .id(format!("forward-chat-{chat_index}"))
                            .flex()
                            .items_center()
                            .gap(px(10.))
                            .px(px(8.))
                            .py(px(7.))
                            .rounded(px(RADIUS_MD))
                            .hover(move |style| style.bg(p.hover()))
                            .child(ui::avatar(chat.initials.clone(), 34., false, p))
                            .child(
                                div()
                                    .flex_1()
                                    .text_size(px(TEXT_BODY))
                                    .text_color(p.text_primary)
                                    .child(chat.name.clone()),
                            )
                            .child(ui::icon(IconName::Forward, 14.0, p.text_secondary)),
                        Action::ForwardTo(message_index, chat_index),
                    )
                }))
                .child(div().flex().justify_end().child(self.clickable_modal(
                    cx,
                    ui::secondary_button("cancel-forward", "Cancel", p),
                    Action::CloseOverlay,
                ))),
        )
    }

    fn poll_votes_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let votes = [
            ("Rizky Pratama", "Linen natural"),
            ("Maya", "Linen natural"),
            ("Rina", "Hitam arang"),
            ("Bayu", "Linen natural"),
            ("Sari", "Coral"),
            ("Dewi Lestari", "Linen natural"),
            ("Budi Santoso", "Hitam arang"),
            ("Jakarta Crew", "Linen natural"),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Poll votes"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("8 votes · single answer. Names are visible to the group."),
                )
                .children(votes.into_iter().map(|(name, option)| {
                    div()
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .px(px(8.))
                        .py(px(6.))
                        .child(ui::avatar(initials_for(Some(name)), 32., false, p))
                        .child(
                            div()
                                .flex_1()
                                .text_size(px(TEXT_BODY))
                                .text_color(p.text_primary)
                                .child(name),
                        )
                        .child(ui::chip(option, false, p))
                }))
                .child(div().flex().justify_end().child(self.clickable_modal(
                    cx,
                    ui::primary_button("close-poll-votes", "Done", p),
                    Action::CloseOverlay,
                ))),
        )
    }

    fn locked_unlock_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .items_center()
                        .gap(px(10.))
                        .child(
                            div()
                                .size(px(64.))
                                .flex()
                                .items_center()
                                .justify_center()
                                .rounded_full()
                                .bg(p.accent_soft)
                                .child(ui::icon(IconName::Lock, 28.0, p.accent)),
                        )
                        .child(
                            div()
                                .text_size(px(18.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Confirm it is you"),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .text_center()
                                .child("Your device handles the check — WhatsLite never sees the result. Use Windows Hello or your device passcode."),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-unlock", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("confirm-unlock", "Unlock", p),
                            Action::CloseOverlayAndToast("Unlocked with Windows Hello · local mock".into()),
                        )),
                ),
        )
    }

    fn gif_picker_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let gifs = ["Celebrate", "Studio", "Linen", "Wave", "Applause", "Coffee"];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("GIFs"),
                )
                .child(
                    div()
                        .h(px(36.))
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .px(px(10.))
                        .rounded_full()
                        .bg(p.bg_search)
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child(ui::icon(IconName::Search, 14.0, p.text_secondary))
                        .child("Search GIFs"),
                )
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(6.))
                        .child(self.clickable_modal(
                            cx,
                            ui::chip("Trending", self.gif_query == 0, p),
                            Action::SetGifQuery(0),
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::chip("No results", self.gif_query == 1, p),
                            Action::SetGifQuery(1),
                        )),
                )
                .when(self.gif_query == 1, |el| {
                    el.child(ui::empty_state_compact(
                        "No GIFs found",
                        "Try another search or check the spelling.",
                        None,
                        p,
                    ))
                })
                .when(self.gif_query == 0, |el| {
                    el.child(
                        div()
                            .grid()
                            .grid_cols(3)
                            .gap(px(8.))
                            .children(gifs.into_iter().map(|caption| {
                                self.clickable_modal(
                                    cx,
                                    div()
                                        .id(format!("gif-{caption}"))
                                        .h(px(84.))
                                        .rounded(px(RADIUS_MD))
                                        .bg(if caption.len() % 2 == 0 {
                                            p.attach_blue
                                        } else {
                                            p.attach_orange
                                        })
                                        .flex()
                                        .flex_col()
                                        .items_center()
                                        .justify_center()
                                        .gap(px(3.))
                                        .text_size(px(16.))
                                        .font_weight(FontWeight::BOLD)
                                        .text_color(p.text_inverse)
                                        .child("GIF")
                                        .child(
                                            div()
                                                .text_size(px(TEXT_CAPTION))
                                                .font_weight(FontWeight::MEDIUM)
                                                .child(caption),
                                        ),
                                    Action::SendGif(caption.into()),
                                )
                            })),
                    )
                })
                .child(div().flex().justify_end().child(self.clickable_modal(
                    cx,
                    ui::secondary_button("close-gifs", "Close", p),
                    Action::CloseOverlay,
                ))),
        )
    }

    fn reactions_sheet(&self, index: usize, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let message = self
            .messages
            .get(self.selected_chat)
            .and_then(|messages| messages.get(index))
            .cloned();
        let names = [
            "Rizky Pratama",
            "Maya",
            "Rina",
            "Bayu",
            "Sari",
            "Dewi Lestari",
            "Jakarta Crew",
            "Budi Santoso",
        ];
        let reactions = message
            .map(|message| message.reactions)
            .filter(|reactions| !reactions.is_empty())
            .or_else(|| {
                self.messages
                    .get(self.selected_chat)
                    .and_then(|messages| {
                        messages
                            .iter()
                            .find(|message| !message.reactions.is_empty())
                    })
                    .map(|message| message.reactions.clone())
            })
            .unwrap_or_default();
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Reactions"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Everyone who reacted to this message."),
                )
                .children(
                    reactions
                        .into_iter()
                        .enumerate()
                        .map(|(reaction_index, reaction)| {
                            div()
                                .flex()
                                .items_center()
                                .gap(px(10.))
                                .px(px(8.))
                                .py(px(6.))
                                .child(
                                    div()
                                        .size(px(34.))
                                        .flex()
                                        .items_center()
                                        .justify_center()
                                        .rounded_full()
                                        .bg(p.accent_soft)
                                        .child(ui::icon(
                                            reaction_icon(&reaction.emoji),
                                            16.0,
                                            p.accent,
                                        )),
                                )
                                .child(
                                    div()
                                        .flex_1()
                                        .flex()
                                        .flex_col()
                                        .child(
                                            div()
                                                .text_size(px(TEXT_BODY))
                                                .text_color(p.text_primary)
                                                .child(
                                                    names
                                                        .get(reaction_index)
                                                        .copied()
                                                        .unwrap_or("Someone"),
                                                ),
                                        )
                                        .child(
                                            div()
                                                .text_size(px(TEXT_CAPTION))
                                                .text_color(p.text_secondary)
                                                .child(format!(
                                                    "{} · {} reaction{}",
                                                    reaction_label(&reaction.emoji),
                                                    reaction.count,
                                                    if reaction.count == 1 { "" } else { "s" }
                                                )),
                                        ),
                                )
                        }),
                )
                .child(div().flex().justify_end().child(self.clickable_modal(
                    cx,
                    ui::primary_button("close-reactions", "Done", p),
                    Action::CloseOverlay,
                ))),
        )
    }

    fn disappearing_picker_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let options = [
            ("24 hours", "Default for new chats"),
            ("7 days", "A week of history"),
            ("90 days", "Three months of history"),
            ("Off", "Keep messages until deleted"),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Disappearing messages"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("New messages disappear from this chat after the timer. This does not affect existing messages."),
                )
                .children(options.into_iter().enumerate().map(|(index, (label, description))| {
                    self.clickable_modal(
                        cx,
                        ui::settings_row(
                            label,
                            description,
                            Some(if self.disappearing == index { "Selected".into() } else { String::new() }),
                            p,
                        ),
                        Action::SetDisappearing(index),
                    )
                }))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("close-disappearing", "Done", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn sticker_picker_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let stickers = [
            (IconName::Flower, "✿"),
            (IconName::Cloud, "☁"),
            (IconName::Heart, "♥"),
            (IconName::Star, "★"),
            (IconName::Sun, "☀"),
            (IconName::Umbrella, "☂"),
            (IconName::Coffee, "☕"),
            (IconName::Music, "♫"),
            (IconName::Zap, "⚡"),
            (IconName::Plane, "✈"),
            (IconName::Phone, "☎"),
            (IconName::Flame, "♨"),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Stickers"),
                )
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(6.))
                        .child(ui::chip("Recent", true, p))
                        .child(ui::chip("Packs", false, p))
                        .child(ui::chip("Search", false, p)),
                )
                .child(
                    div()
                        .grid()
                        .grid_cols(6)
                        .gap(px(8.))
                        .children(stickers.into_iter().map(|(icon_name, symbol)| {
                            self.clickable_modal(
                                cx,
                                div()
                                    .id(format!("sticker-{symbol}"))
                                    .size(px(52.))
                                    .flex()
                                    .items_center()
                                    .justify_center()
                                    .rounded(px(RADIUS_MD))
                                    .bg(p.bg_search)
                                    .child(ui::icon(icon_name, 24.0, p.accent)),
                                Action::SendSticker(symbol.into()),
                            )
                        })),
                )
                .child(div().flex().justify_end().child(self.clickable_modal(
                    cx,
                    ui::secondary_button("close-stickers", "Close", p),
                    Action::CloseOverlay,
                ))),
        )
    }

    fn file_dialog_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Choose a file"),
                )
                .when(!self.file_selected, |el| {
                    el.child(
                        div()
                            .h(px(140.))
                            .flex()
                            .flex_col()
                            .items_center()
                            .justify_center()
                            .gap(px(8.))
                            .rounded(px(RADIUS_LG))
                            .bg(p.bg_search)
                            .child(ui::icon(IconName::FolderOpen, 34.0, p.text_secondary))
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_secondary)
                                    .child("No file selected"),
                            )
                            .child(self.clickable_modal(
                                cx,
                                ui::secondary_button("choose-file", "Choose file", p),
                                Action::SelectFile,
                            )),
                    )
                })
                .when(self.file_selected, |el| {
                    el.child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(10.))
                            .p(px(12.))
                            .rounded(px(RADIUS_MD))
                            .bg(p.bg_search)
                            .child(ui::icon(IconName::FileText, 22.0, p.attach_blue))
                            .child(
                                div()
                                    .flex()
                                    .flex_col()
                                    .child(
                                        div()
                                            .text_size(px(TEXT_BODY))
                                            .font_weight(FontWeight::MEDIUM)
                                            .text_color(p.text_primary)
                                            .child("Invoice-2026-09.pdf"),
                                    )
                                    .child(
                                        div()
                                            .text_size(px(TEXT_CAPTION))
                                            .text_color(p.text_secondary)
                                            .child("PDF · 1.2 MB"),
                                    ),
                            ),
                    )
                })
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-file", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .when(self.file_selected, |el| {
                            el.child(self.clickable_modal(
                                cx,
                                ui::primary_button("attach-file", "Attach file", p),
                                Action::AttachSelectedFile,
                            ))
                        }),
                ),
        )
    }

    fn poll_composer_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let options = ["Linen natural", "Hitam arang", "Coral"];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Create poll"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Ask one question with 2–12 options. Choose single or multiple answers."),
                )
                .child(ui::settings_row_field("Question", "What do you want to ask?", Some("Warna mana yang paling cocok?".into()), p))
                .children(options.into_iter().enumerate().map(|(index, option)| {
                    ui::settings_row_field(format!("Option {}", index + 1), "Poll option", Some(option.into()), p)
                }))
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .px(px(20.))
                        .py(px(8.))
                        .child(ui::chip("Single answer", true, p))
                        .child(ui::chip("Multiple answers", false, p)),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-poll", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("create-poll", "Create poll", p),
                            Action::CreatePoll,
                        )),
                ),
        )
    }

    fn event_composer_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Create event"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Add a title, date, time and place. Guests can accept or decline."),
                )
                .child(ui::settings_row_field(
                    "Title",
                    "Event name",
                    Some("Review koleksi September".into()),
                    p,
                ))
                .child(ui::settings_row_field(
                    "Date",
                    "Calendar date",
                    Some("Fri, 26 Sep".into()),
                    p,
                ))
                .child(ui::settings_row_field(
                    "Time",
                    "Start time",
                    Some("15:00".into()),
                    p,
                ))
                .child(ui::settings_row_field(
                    "Location",
                    "Where it happens",
                    Some("Studio Kemang".into()),
                    p,
                ))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-event", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("create-event", "Create event", p),
                            Action::CreateEvent,
                        )),
                ),
        )
    }

    fn status_font_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let fonts = ["Inter", "Georgia", "Mono"];
        let sizes = [18, 22, 26];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Text status font"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Applies to text-only status items. Photos and videos keep their own captions."),
                )
                .child(ui::section_label("FONT", p))
                .children(fonts.into_iter().enumerate().map(|(index, name)| {
                    self.clickable_modal(
                        cx,
                        ui::settings_row(
                            name,
                            "Sample: Motif anyaman baru",
                            Some(if self.status_font == index { "Selected".into() } else { String::new() }),
                            p,
                        ),
                        Action::SetStatusFont(index),
                    )
                }))
                .child(ui::section_label("SIZE", p))
                .children(sizes.into_iter().enumerate().map(|(index, size)| {
                    self.clickable_modal(
                        cx,
                        ui::settings_row(
                            format!("{size} px"),
                            "Line height follows the size",
                            Some(if self.status_font_size == index { "Selected".into() } else { String::new() }),
                            p,
                        ),
                        Action::SetStatusFontSize(index),
                    )
                }))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("save-status-font", "Done", p),
                            Action::CloseOverlay,
                        )),
                ),
        )
    }

    fn accounts_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Accounts"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Maximum two, same as WhatsApp Desktop. Both accounts stay online at once."),
                )
                .children(self.accounts.iter().map(|account| {
                    let row = div()
                        .id(format!("account-row-{}", account.id))
                        .flex()
                        .items_center()
                        .gap(px(12.))
                        .p(px(8.))
                        .rounded(px(RADIUS_MD))
                        .hover(move |style| style.bg(p.hover()))
                        .child(ui::avatar(account.initials.clone(), 40., false, p))
                        .child(
                            div()
                                .flex()
                                .flex_col()
                                .child(
                                    div()
                                        .text_size(px(TEXT_BODY))
                                        .font_weight(FontWeight::SEMIBOLD)
                                        .text_color(p.text_primary)
                                        .child(account.name.clone()),
                                )
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .text_color(p.text_secondary)
                                        .child(account.label.clone()),
                                ),
                        )
                        .child(div().flex_1())
                        .when(account.active, |el| {
                            el.child(ui::icon(IconName::CircleCheck, 18.0, p.accent))
                        })
                        .when(!account.active && account.unread > 0, |el| {
                            el.child(ui::badge(account.unread, p))
                        });
                    self.clickable_modal(cx, row, Action::SwitchAccount(account.id))
                }))
                .child(ui::divider(p))
                .child(self.clickable_modal(
                    cx,
                    ui::menu_item("Add account", IconName::UserPlus, p),
                    Action::AddAccount,
                ))
                .child(self.clickable_modal(
                    cx,
                    ui::menu_item("Manage accounts", IconName::Settings, p),
                    Action::OpenAccounts,
                )),
        )
    }

    fn add_account_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Add a second account"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Scan from WhatsApp on the second phone. Both accounts stay online at once."),
                )
                .child(
                    div()
                        .flex()
                        .justify_center()
                        .child(render_qr(QrState::Active, p)),
                )
                .child(
                    div()
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_secondary)
                        .child("Maximum two accounts. Same QR component as the first pairing, second context."),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-add-account", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("add-account-phone", "Use phone number", p),
                            Action::CloseOverlayAndToast("Phone number flow opened · local mock".into()),
                        )),
                ),
        )
    }

    fn ai_draft_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .flex()
                        .items_center()
                        .justify_between()
                        .child(
                            div()
                                .text_size(px(18.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Draft a reply"),
                        )
                        .child(ui::chip("DRAFT REPLY", true, p)),
                )
                .child(
                    div()
                        .p(px(12.))
                        .rounded(px(RADIUS_LG))
                        .bg(p.bg_search)
                        .text_size(px(TEXT_BODY))
                        .text_color(p.text_primary)
                        .child(if self.ai_draft_english {
                            "Sure, I'll send the final deck at three. If there are small revisions, leave them in the comments."
                        } else {
                            "Siap, aku kirim deck final jam tiga ya. Kalau ada revisi kecil, tulis di kolom komentar."
                        }),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Answers use only the chat you have open. Nothing is sent anywhere else."),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("ai-discard", "Discard", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("ai-translate", if self.ai_draft_english { "Original" } else { "Translate" }, p),
                            Action::ToggleAiTranslation,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("ai-insert", "Insert", p),
                            Action::InsertAiDraft,
                        )),
                ),
        )
    }

    fn system_state_sheet(
        &self,
        state: SystemState,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        let body = match state {
            SystemState::OsNotification => div()
                .flex()
                .flex_col()
                .gap(px(10.))
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .child(app_mark(p))
                        .child(
                            div()
                                .flex()
                                .flex_col()
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .font_weight(FontWeight::SEMIBOLD)
                                        .text_color(p.text_primary)
                                        .child("WhatsLite"),
                                )
                                .child(
                                    div()
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.text_secondary)
                                        .child("now"),
                                ),
                        ),
                )
                .child(
                    div()
                        .text_size(px(TEXT_BODY))
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(p.text_primary)
                        .child("Rizky Pratama"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Pagi! Foto produk yang kemarin sudah jadi belum?"),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("toast-dismiss", "Dismiss", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("toast-open", "Open", p),
                            Action::CloseOverlay,
                        )),
                ),
            SystemState::Permission => div()
                .flex()
                .flex_col()
                .gap(px(10.))
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Camera blocked"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Video calls need camera access. Enable it in your system settings and try again."),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("permission-later", "Not now", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("permission-settings", "Open settings", p),
                            Action::CloseOverlayAndToast("System settings opened · camera permission is a mock".into()),
                        )),
                ),
            SystemState::Tray => div()
                .flex()
                .flex_col()
                .gap(px(2.))
                .child(
                    div()
                        .pb(px(6.))
                        .text_size(px(16.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("WhatsLite"),
                )
                .child(self.clickable_modal(cx, ui::menu_item("Open WhatsLite", IconName::Monitor, p), Action::CloseOverlay))
                .child(self.clickable_modal(cx, ui::menu_item("Mute notifications", IconName::BellOff, p), Action::ShowComingSoon("Muted locally.")))
                .child(self.clickable_modal(cx, ui::menu_item("Quit", IconName::LogOut, p), Action::CloseOverlay)),
            SystemState::Update => div()
                .flex()
                .flex_col()
                .gap(px(10.))
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Update available"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("WhatsLite 1.1 is ready. Restart to install. You will stay signed in."),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("update-later", "Later", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("update-restart", "Restart", p),
                            Action::CloseOverlayAndToast("Restarting WhatsLite 1.1…".into()),
                        )),
                ),
            SystemState::Error => div()
                .flex()
                .flex_col()
                .gap(px(10.))
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Something went wrong"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Your messages are safe. Restart to continue, or send a report."),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("error-report", "Send report", p),
                            Action::CloseOverlayAndToast("Crash report queued · local mock".into()),
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("error-restart", "Restart", p),
                            Action::CloseOverlayAndToast("Restarted locally".into()),
                        )),
                ),
            SystemState::DragDrop => div()
                .flex()
                .flex_col()
                .items_center()
                .gap(px(10.))
                .child(
                    div()
                        .w_full()
                        .h(px(160.))
                        .flex()
                        .flex_col()
                        .items_center()
                        .justify_center()
                        .gap(px(8.))
                        .rounded(px(RADIUS_XL))
                        .border_2()
                        .border_color(p.accent)
                        .bg(p.accent_soft)
                        .child(ui::icon(IconName::FolderOpen, 34.0, p.accent))
                        .child(
                            div()
                                .text_size(px(TEXT_BODY))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Drop to send to Rizky Pratama"),
                        ),
                )
                .child(
                    div()
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_secondary)
                        .child("Whole window is the target; Esc cancels."),
                )
                .child(self.clickable_modal(
                    cx,
                    ui::secondary_button("drop-cancel", "Cancel", p),
                    Action::CloseOverlay,
                )),
            SystemState::Paste => div()
                .flex()
                .flex_col()
                .gap(px(10.))
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Paste image"),
                )
                .child(
                    div()
                        .h(px(150.))
                        .rounded(px(RADIUS_MD))
                        .bg(p.attach_blue)
                        .opacity(0.65)
                        .flex()
                        .items_center()
                        .justify_center()
                        .child(ui::icon(IconName::Image, 36.0, p.text_inverse)),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Screenshot 2026-09-19.png · Ctrl+V stages the image as an attachment, never sends it."),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("paste-cancel", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("paste-stage", "Stage attachment", p),
                            Action::CloseOverlayAndToast("Attachment staged in the composer".into()),
                        )),
                ),
            SystemState::ToastStack => div()
                .flex()
                .flex_col()
                .gap(px(8.))
                .child(ui::toast_shell("Copied to clipboard", p))
                .child(ui::toast_shell("Message sent", p))
                .child(ui::toast_shell("Could not send · tap to retry", p))
                .child(
                    div()
                        .text_size(px(TEXT_CAPTION))
                        .text_color(p.text_secondary)
                        .child("Max 3 stacked, newest on top, 8px apart."),
                ),
        };
        modal_frame(p, ui::modal_shell(p).child(body))
    }

    fn shortcut_editor_sheet(
        &self,
        index: usize,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        let (label, default_keys) = SHORTCUTS
            .get(index)
            .copied()
            .unwrap_or(("Shortcut", &["Ctrl", "K"]));
        let current = self.shortcut_keys(index);
        let primary = default_keys.last().copied().unwrap_or("K");
        let choices = [
            ("Default", default_keys.join("+")),
            ("Ctrl + Shift", format!("Ctrl+Shift+{primary}")),
            ("Alt", format!("Alt+{primary}")),
            ("Ctrl + Alt", format!("Ctrl+Alt+{primary}")),
        ];
        let key = |value: String| {
            div()
                .px(px(6.))
                .py(px(2.))
                .rounded(px(RADIUS_SM))
                .bg(p.bg_search)
                .text_size(px(TEXT_CAPTION))
                .font_weight(FontWeight::MEDIUM)
                .text_color(p.text_primary)
                .child(value)
        };
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Edit shortcut"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child(format!("{label} · choose the key combination to use.")),
                )
                .child(
                    div()
                        .flex()
                        .items_center()
                        .justify_between()
                        .p(px(10.))
                        .rounded(px(RADIUS_MD))
                        .bg(p.bg_search)
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("Current"),
                        )
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .gap(px(4.))
                                .children(current.into_iter().map(key)),
                        ),
                )
                .children(choices.into_iter().map(|(title, value)| {
                    let selected = self
                        .shortcut_overrides
                        .get(index)
                        .and_then(|value| value.clone())
                        .as_deref()
                        == Some(value.as_str());
                    let row =
                        div()
                            .id(format!("shortcut-choice-{title}"))
                            .flex()
                            .items_center()
                            .justify_between()
                            .px(px(10.))
                            .py(px(9.))
                            .rounded(px(RADIUS_MD))
                            .hover(move |style| style.bg(p.hover()))
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .text_color(p.text_primary)
                                    .child(title),
                            )
                            .child(
                                div()
                                    .flex()
                                    .items_center()
                                    .gap(px(6.))
                                    .child(div().flex().items_center().gap(px(4.)).children(
                                        value.split('+').map(|part| key(part.to_owned())),
                                    ))
                                    .when(selected, |el| {
                                        el.child(ui::icon(IconName::Check, 14.0, p.accent))
                                    }),
                            );
                    let apply = value.clone();
                    self.clickable_modal(cx, row, Action::ApplyShortcut(index, apply))
                }))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-shortcut-editor", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("reset-shortcut", "Reset to default", p),
                            Action::ResetShortcuts,
                        )),
                ),
        )
    }

    fn render_overlay(
        &self,
        overlay: Overlay,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        match overlay {
            Overlay::AttachmentMenu => self.attachment_menu(cx, p),
            Overlay::HeaderMenu => self.header_menu(cx, p),
            Overlay::ChatListMenu => self.chat_list_menu(cx, p),
            Overlay::ContactProfile(name) => self.contact_profile_sheet(&name, cx, p),
            Overlay::ContextMenu(index) => self.context_menu(index, cx, p),
            Overlay::EmojiPicker => self.emoji_picker(cx, p),
            Overlay::ConfirmDelete(index) => self.confirm_delete(index, cx, p),
            Overlay::PinnedMessages => self.pinned_messages_sheet(cx, p),
            Overlay::NewChat => self.new_chat_sheet(cx, p),
            Overlay::NewGroup => self.new_group_sheet(cx, p),
            Overlay::Lightbox(index) => self.lightbox(index, cx, p),
            Overlay::StatusComposer => self.status_composer(cx, p),
            Overlay::StatusFont => self.status_font_sheet(cx, p),
            Overlay::StatusPrivacy => self.status_privacy_sheet(cx, p),
            Overlay::PollComposer => self.poll_composer_sheet(cx, p),
            Overlay::EventComposer => self.event_composer_sheet(cx, p),
            Overlay::DisappearingPicker => self.disappearing_picker_sheet(cx, p),
            Overlay::StickerPicker => self.sticker_picker_sheet(cx, p),
            Overlay::FileDialog => self.file_dialog_sheet(cx, p),
            Overlay::GroupMembers => self.group_members_sheet(cx, p),
            Overlay::UseCaseDemo => self.use_case_demo_sheet(cx, p),
            Overlay::ClearChat => self.clear_chat_sheet(cx, p),
            Overlay::ExportChat => self.export_chat_sheet(cx, p),
            Overlay::SecurityCode => self.security_code_sheet(cx, p),
            Overlay::ChannelInfo(name) => self.channel_info_sheet(name, cx, p),
            Overlay::CommunityMembers => self.community_members_sheet(cx, p),
            Overlay::MemberApprovals => self.member_approvals_sheet(cx, p),
            Overlay::LockedUnlock => self.locked_unlock_sheet(cx, p),
            Overlay::StatusViewers => self.status_viewers_sheet(cx, p),
            Overlay::Forward(index) => self.forward_sheet(index, cx, p),
            Overlay::PollVotes => self.poll_votes_sheet(cx, p),
            Overlay::Reactions(index) => self.reactions_sheet(index, cx, p),
            Overlay::GifPicker => self.gif_picker_sheet(cx, p),
            Overlay::NewBroadcast => self.new_broadcast_sheet(cx, p),
            Overlay::NewCommunity => self.new_community_sheet(cx, p),
            Overlay::EditContact(index) => self.edit_contact_sheet(index, cx, p),
            Overlay::Accounts => self.accounts_sheet(cx, p),
            Overlay::AddAccount => self.add_account_sheet(cx, p),
            Overlay::AiDraft => self.ai_draft_sheet(cx, p),
            Overlay::ShortcutEditor(index) => self.shortcut_editor_sheet(index, cx, p),
            Overlay::SystemState(state) => self.system_state_sheet(state, cx, p),
        }
    }

    fn attachment_menu(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let items = [
            (
                IconName::FileText,
                "File",
                p.attach_blue,
                Action::OpenFileDialog,
            ),
            (
                IconName::Image,
                "Photos and videos",
                p.attach_blue,
                Action::StageAttachment(StagedKind::Image, "IMG_2241.jpg".into()),
            ),
            (
                IconName::ChartBar,
                "Poll",
                p.attach_orange,
                Action::OpenPollComposer,
            ),
            (
                IconName::Calendar,
                "Event",
                p.attach_red,
                Action::OpenEventComposer,
            ),
            (
                IconName::User,
                "Contact",
                p.attach_orange,
                Action::StageAttachment(StagedKind::Contact, "Rizky Pratama".into()),
            ),
            (
                IconName::Sparkles,
                "Sticker",
                p.attach_green,
                Action::OpenStickerPicker,
            ),
            (IconName::Video, "GIF", p.attach_blue, Action::OpenGifPicker),
        ];
        div()
            .absolute()
            .bottom(px(90.))
            .left(px(if self.collapsed { 380. } else { 516. }))
            .w(px(380.))
            .p(px(16.))
            .rounded(px(RADIUS_XL))
            .bg(p.bg_list)
            .border_1()
            .border_color(p.border)
            .child(
                div()
                    .flex()
                    .flex_wrap()
                    .gap(px(10.))
                    .children(items.into_iter().map(|(icon_name, label, color, action)| {
                        self.clickable_modal(
                            cx,
                            div()
                                .id(format!("attachment-{label}"))
                                .w(px(108.))
                                .flex()
                                .flex_col()
                                .items_center()
                                .gap(px(6.))
                                .p(px(8.))
                                .rounded(px(RADIUS_MD))
                                .hover(move |style| style.bg(p.hover()))
                                .child(
                                    div()
                                        .size(px(44.))
                                        .flex()
                                        .items_center()
                                        .justify_center()
                                        .rounded_full()
                                        .bg(color)
                                        .child(ui::icon(icon_name, 22.0, p.text_inverse)),
                                )
                                .child(
                                    div()
                                        .text_size(px(TEXT_SMALL))
                                        .text_color(p.text_primary)
                                        .child(label),
                                ),
                            action,
                        )
                    })),
            )
    }

    fn chat_list_menu(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let titlebar = if self.shell == ShellStyle::MacOS {
            0.
        } else {
            TITLEBAR_HEIGHT
        };
        let collapsed = self.collapsed || self.compact;
        let sidebar = if self.shell == ShellStyle::MacOS {
            if collapsed { RAIL_WIDTH + 16. } else { 208. }
        } else if collapsed {
            RAIL_WIDTH
        } else {
            SIDEBAR_WIDTH
        };
        let scrim = div()
            .id("chat-list-menu-scrim")
            .absolute()
            .top_0()
            .left_0()
            .right_0()
            .bottom_0()
            .on_click(cx.listener(|this, _, window, cx| {
                this.apply(Action::CloseOverlay, window, cx);
            }));
        let mut item = |label: &'static str, icon_name: IconName, action: Action| {
            self.clickable_modal(cx, ui::menu_item(label, icon_name, p), action)
        };
        let menu = div()
            .absolute()
            .top(px(titlebar + 64. + 2.))
            .left(px(sidebar + 12.))
            .w(px(238.))
            .rounded(px(RADIUS_LG))
            .bg(p.bg_list)
            .border_1()
            .border_color(p.border)
            .shadow_sm()
            .child(
                div()
                    .px(px(12.))
                    .py(px(10.))
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Chats"),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_CAPTION))
                            .text_color(p.text_secondary)
                            .child("Create, find or organise"),
                    ),
            )
            .child(div().h(px(1.)).bg(p.divider))
            .child(
                div()
                    .p(px(6.))
                    .child(item("New group", IconName::Users, Action::NewGroup))
                    .child(item(
                        "New broadcast",
                        IconName::Megaphone,
                        Action::NewBroadcast,
                    ))
                    .child(div().my(px(4.)).h(px(1.)).bg(p.divider))
                    .child(item(
                        "Starred messages",
                        IconName::Star,
                        Action::Navigate(Route::Starred),
                    ))
                    .child(item(
                        "Archived",
                        IconName::Archive,
                        Action::Navigate(Route::Archived),
                    ))
                    .child(item(
                        "Settings",
                        IconName::Settings,
                        Action::Navigate(Route::Settings),
                    ))
                    .child(item(
                        "Use case demo",
                        IconName::Sparkles,
                        Action::OpenUseCaseDemo,
                    )),
            );
        div()
            .absolute()
            .top_0()
            .left_0()
            .right_0()
            .bottom_0()
            .child(scrim)
            .child(menu)
    }

    fn header_menu(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let chat = self
            .selected_chat()
            .cloned()
            .unwrap_or_else(|| model::chats()[0].clone());
        let group = chat.is_group;
        let titlebar = if self.shell == ShellStyle::MacOS {
            0.
        } else {
            TITLEBAR_HEIGHT
        };
        let collapsed = self.collapsed || self.compact;
        let sidebar = if self.shell == ShellStyle::MacOS {
            if collapsed { RAIL_WIDTH + 16. } else { 208. }
        } else if collapsed {
            RAIL_WIDTH
        } else {
            SIDEBAR_WIDTH
        };
        let scrim = div()
            .id("header-menu-scrim")
            .absolute()
            .top_0()
            .left_0()
            .right_0()
            .bottom_0()
            .on_click(cx.listener(|this, _, window, cx| {
                this.apply(Action::CloseOverlay, window, cx);
            }));
        let mut item = |label: &'static str, icon_name: IconName, action: Action| {
            self.clickable_modal(cx, ui::menu_item(label, icon_name, p), action)
        };
        let menu = div()
            .absolute()
            .top(px(titlebar + CONVERSATION_HEADER_HEIGHT + 2.))
            .left(px(sidebar + LIST_WIDTH + 12.))
            .w(px(268.))
            .rounded(px(RADIUS_LG))
            .bg(p.bg_list)
            .border_1()
            .border_color(p.border)
            .shadow_sm()
            .child(
                div()
                    .px(px(12.))
                    .py(px(10.))
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child(chat.name.clone()),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_CAPTION))
                            .text_color(p.text_secondary)
                            .child(if group {
                                "Group options"
                            } else {
                                "Chat options"
                            }),
                    ),
            )
            .child(div().h(px(1.)).bg(p.divider))
            .child(
                div()
                    .p(px(6.))
                    .child(item(
                        if group { "Group info" } else { "View profile" },
                        IconName::User,
                        if group {
                            Action::OpenInfo(InfoPanel::Group)
                        } else {
                            Action::OpenContactProfile(chat.name.clone())
                        },
                    ))
                    .when(group, |el| {
                        el.child(item(
                            "Add members",
                            IconName::UserPlus,
                            Action::OpenGroupMembers,
                        ))
                    })
                    .child(item(
                        "Search in chat",
                        IconName::Search,
                        Action::ToggleSearch,
                    ))
                    .child(item(
                        "Media, links and docs",
                        IconName::Image,
                        Action::Navigate(Route::Media),
                    ))
                    .child(item(
                        "Mute notifications",
                        IconName::BellOff,
                        Action::CloseOverlayAndToast("Muted for 8 hours · local demo".into()),
                    ))
                    .child(item(
                        "Disappearing messages",
                        IconName::Timer,
                        Action::OpenDisappearingPicker,
                    ))
                    .child(div().my(px(4.)).h(px(1.)).bg(p.divider))
                    .child(item("Clear chat", IconName::Trash, Action::OpenClearChat))
                    .child(item(
                        "Use case demo",
                        IconName::Sparkles,
                        Action::OpenUseCaseDemo,
                    ))
                    .when(group, |el| {
                        el.child(item(
                            "Exit group",
                            IconName::LogOut,
                            Action::CloseOverlayAndToast("Exit group · coming soon".into()),
                        ))
                    }),
            );
        div()
            .absolute()
            .top_0()
            .left_0()
            .right_0()
            .bottom_0()
            .child(scrim)
            .child(menu)
    }

    fn contact_profile_sheet(
        &self,
        name: &str,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        let found = self
            .contacts
            .iter()
            .find(|contact| contact.name == name)
            .cloned();
        let saved = found.is_some();
        let contact = found.unwrap_or_else(|| Contact {
            id: 0,
            name: name.to_owned(),
            initials: initials_for(Some(name)),
            about: "Not in your contacts".into(),
            phone: "+62 8•• •••• ••••".into(),
            online: false,
            blocked: false,
            favorite: false,
        });
        let display = if saved {
            contact.name.clone()
        } else {
            contact.phone.clone()
        };
        let save_action =
            Action::CloseOverlayAndToast(format!("Saved {name} to contacts · local demo"));
        let message_action = self
            .chats
            .iter()
            .position(|chat| chat.name == contact.name)
            .map(Action::SelectChat)
            .unwrap_or_else(|| {
                Action::CloseOverlayAndToast(format!("Opening chat with {display}…"))
            });
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .flex()
                        .items_center()
                        .justify_between()
                        .child(
                            div()
                                .text_size(px(17.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Contact profile"),
                        )
                        .child(self.clickable_modal(
                            cx,
                            ui::icon_button_small(
                                "close-contact-profile",
                                IconName::X,
                                16.0,
                                p.text_secondary,
                                p.hover(),
                                "Close profile",
                            ),
                            Action::CloseOverlay,
                        )),
                )
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .items_center()
                        .gap(px(6.))
                        .py(px(4.))
                        .child(ui::avatar(contact.initials.clone(), 88., contact.online, p))
                        .child(
                            div()
                                .text_size(px(20.))
                                .font_weight(FontWeight::BOLD)
                                .text_color(p.text_primary)
                                .child(display.clone()),
                        )
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child(if saved {
                                    contact.about.clone()
                                } else {
                                    "Not in your contacts".into()
                                }),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            info_action("profile-message", IconName::MessageSquare, "Message", p),
                            message_action.clone(),
                        ))
                        .child(self.clickable_modal(
                            cx,
                            info_action("profile-voice", IconName::Phone, "Voice", p),
                            Action::CloseOverlayAndToast(format!("Calling {display}…")),
                        ))
                        .child(self.clickable_modal(
                            cx,
                            info_action("profile-video", IconName::Video, "Video", p),
                            Action::CloseOverlayAndToast(format!("Video calling {display}…")),
                        )),
                )
                .child(self.clickable_modal(
                    cx,
                    ui::settings_row(
                        "Phone",
                        if saved {
                            "From your address book"
                        } else {
                            "This number is not saved"
                        },
                        Some(contact.phone.clone()),
                        p,
                    ),
                    Action::CloseOverlayAndToast(format!("Phone copied · {}", contact.phone)),
                ))
                .child(self.clickable_modal(
                    cx,
                    ui::settings_row(
                        "Username",
                        "Shown only when this person shares it",
                        Some("Not shared".into()),
                        p,
                    ),
                    Action::CloseOverlayAndToast(
                        "Usernames are opt-in; this person has not shared one.".into(),
                    ),
                ))
                .child(if saved {
                    self.clickable_modal(
                        cx,
                        ui::settings_row(
                            "Saved contact",
                            "Name from your address book",
                            Some("Yes".into()),
                            p,
                        ),
                        Action::EditContact(contact.id),
                    )
                } else {
                    self.clickable_modal(
                        cx,
                        ui::settings_row(
                            "Save contact",
                            "Numbers are never saved automatically",
                            Some("Save".into()),
                            p,
                        ),
                        save_action,
                    )
                })
                .child(self.clickable_modal(
                    cx,
                    ui::settings_row(
                        "Add to favourites",
                        "Keep this contact close",
                        Some("Add".into()),
                        p,
                    ),
                    Action::CloseOverlayAndToast(format!(
                        "Added {name} to favourites · local demo"
                    )),
                ))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("profile-close", "Close", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("profile-open-chat", "Message", p),
                            message_action,
                        )),
                ),
        )
    }

    fn context_menu(&self, index: usize, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let mut item = |label: &'static str, icon_name: IconName, action: Action| {
            self.clickable_modal(cx, ui::menu_item(label, icon_name, p), action)
        };
        div().absolute().top(px(250.)).right(px(80.)).child(
            ui::context_menu_shell(p)
                .child(item("Reply", IconName::Reply, Action::Reply(index)))
                .child(item(
                    "React",
                    IconName::Heart,
                    Action::React(index, "♥".into()),
                ))
                .child(item("Edit", IconName::Pencil, Action::Edit(index)))
                .child(item(
                    "Forward",
                    IconName::Forward,
                    Action::OpenForward(index),
                ))
                .child(item(
                    "Copy",
                    IconName::Copy,
                    Action::Toast("Copied to clipboard".into()),
                ))
                .child(item(
                    "Star message",
                    IconName::Star,
                    Action::StarMessage(index),
                ))
                .child(item(
                    "Select",
                    IconName::Check,
                    Action::EnterMessageSelection(index),
                ))
                .child(item("Delete", IconName::Trash, Action::Delete(index))),
        )
    }

    fn emoji_picker(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let emojis = [
            "♥", "☺", "★", "☀", "☁", "✈", "✂", "✎", "✓", "✗", "☎", "☕", "♫", "⚑", "⚡", "❄", "♠",
            "♦", "♣", "☂", "☃", "♨", "☯", "✿",
        ];
        div()
            .absolute()
            .bottom(px(90.))
            .right(px(16.))
            .w(px(300.))
            .p(px(12.))
            .rounded(px(RADIUS_XL))
            .bg(p.bg_list)
            .border_1()
            .border_color(p.border)
            .child(
                div()
                    .grid()
                    .grid_cols(8)
                    .gap(px(4.))
                    .children(emojis.into_iter().map(|emoji| {
                        self.clickable_modal(
                            cx,
                            div()
                                .id(format!("emoji-{emoji}"))
                                .size(px(30.))
                                .flex()
                                .items_center()
                                .justify_center()
                                .rounded(px(RADIUS_SM))
                                .text_size(px(18.))
                                .hover(move |style| style.bg(p.hover()))
                                .child(emoji),
                            Action::AppendEmoji(emoji.into()),
                        )
                    })),
            )
    }

    fn pinned_messages_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let messages = self
            .messages
            .get(self.selected_chat)
            .cloned()
            .unwrap_or_default();
        let pinned = self
            .pinned
            .iter()
            .filter_map(|index| {
                messages
                    .get(*index)
                    .map(|message| (*index, message.clone()))
            })
            .collect::<Vec<_>>();
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("Pinned messages"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Up to three messages can stay pinned in a chat."),
                )
                .children(pinned.into_iter().map(|(index, message)| {
                    let row = div()
                        .id(format!("pinned-message-{index}"))
                        .flex()
                        .items_center()
                        .gap(px(10.))
                        .p(px(10.))
                        .rounded(px(RADIUS_MD))
                        .bg(p.bg_search)
                        .child(ui::icon(IconName::Pin, 14.0, p.text_secondary))
                        .child(
                            div()
                                .flex_1()
                                .min_w(px(0.))
                                .flex()
                                .flex_col()
                                .child(
                                    div()
                                        .text_size(px(TEXT_CAPTION))
                                        .text_color(p.text_secondary)
                                        .child(
                                            message
                                                .sender_name
                                                .clone()
                                                .unwrap_or_else(|| "You".into()),
                                        ),
                                )
                                .child(
                                    div()
                                        .overflow_hidden()
                                        .whitespace_nowrap()
                                        .text_ellipsis()
                                        .text_size(px(TEXT_SMALL))
                                        .text_color(p.text_primary)
                                        .child(message_preview(&message)),
                                ),
                        )
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button(format!("jump-pinned-{index}"), "Jump", p),
                            Action::JumpToMessage(index),
                        ));
                    row
                }))
                .child(div().flex().justify_end().child(self.clickable_modal(
                    cx,
                    ui::primary_button("close-pinned", "Done", p),
                    Action::CloseOverlay,
                ))),
        )
    }

    fn confirm_delete(&self, index: usize, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                    .child(
                        div()
                            .text_size(px(18.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Delete this message?"),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_BODY))
                            .text_color(p.text_secondary)
                            .child("Delete for everyone removes it from both sides. Delete for me keeps it on their device."),
                    )
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap(px(8.))
                            .child(self.clickable_modal(
                                cx,
                                ui::secondary_button("cancel-delete", "Cancel", p),
                                Action::CloseOverlay,
                            ))
                            .child(self.clickable_modal(
                                cx,
                                ui::danger_button("confirm-delete", "Delete for everyone", p),
                                Action::ConfirmDelete(index),
                            )),
                    ),
        )
    }

    fn new_chat_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let actions = [
            ("New group", IconName::Users, Action::NewGroup),
            ("New community", IconName::Users, Action::NewCommunity),
            (
                "Invite via link",
                IconName::Link,
                Action::Toast("Invite link copied · local mock".into()),
            ),
        ];
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("New chat"),
                )
                .child(
                    div()
                        .h(px(40.))
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .rounded_full()
                        .bg(p.bg_search)
                        .px(px(12.))
                        .child(ui::icon(IconName::Search, 16.0, p.text_secondary))
                        .child(
                            div()
                                .text_size(px(TEXT_SMALL))
                                .text_color(p.text_secondary)
                                .child("Search name or number"),
                        ),
                )
                .children(actions.into_iter().map(|(label, icon_name, action)| {
                    self.clickable_modal(cx, ui::menu_item(label, icon_name, p), action)
                })),
        )
    }

    fn new_group_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("New group"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("Pick members, then name the group. This flow is a local preview."),
                )
                .child(div().flex().flex_col().gap(px(4.)).children(
                    self.contacts.iter().take(4).map(|contact| {
                        div()
                            .flex()
                            .items_center()
                            .gap(px(10.))
                            .p(px(6.))
                            .child(ui::avatar(contact.initials.clone(), 32., false, p))
                            .child(
                                div()
                                    .text_size(px(TEXT_BODY))
                                    .text_color(p.text_primary)
                                    .child(contact.name.clone()),
                            )
                    }),
                ))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-group", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("create-group", "Create group", p),
                            Action::CreateGroup,
                        )),
                ),
        )
    }

    fn new_broadcast_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        if self.broadcast_step == 2 {
            return modal_frame(
                p,
                ui::modal_shell(p)
                    .child(
                        div()
                            .text_size(px(18.))
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(p.text_primary)
                            .child("Name the list"),
                    )
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(
                                "Recipients get a normal message; replies land in normal chats.",
                            ),
                    )
                    .child(ui::settings_row_field(
                        "List name",
                        "Up to 25 characters",
                        Some("Resellers".into()),
                        p,
                    ))
                    .child(ui::settings_row_field(
                        "Recipients",
                        "Selected in the previous step",
                        Some(format!("{} recipients", self.broadcast_recipients.len())),
                        p,
                    ))
                    .child(
                        div()
                            .flex()
                            .justify_between()
                            .gap(px(8.))
                            .child(self.clickable_modal(
                                cx,
                                ui::secondary_button("back-broadcast", "Back", p),
                                Action::BroadcastBack,
                            ))
                            .child(self.clickable_modal(
                                cx,
                                ui::primary_button("create-broadcast", "Create broadcast", p),
                                Action::BroadcastCreate,
                            )),
                    ),
            );
        }

        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("New broadcast"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child(format!(
                            "Pick recipients, then name the list. {} selected. Replies will land in normal chats.",
                            self.broadcast_recipients.len()
                        )),
                )
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(4.))
                        .children(self.contacts.iter().map(|contact| {
                            let selected = self.broadcast_recipients.contains(&contact.id);
                            let row = div()
                                .id(format!("broadcast-recipient-{}", contact.id))
                                .flex()
                                .items_center()
                                .gap(px(10.))
                                .p(px(6.))
                                .rounded(px(RADIUS_MD))
                                .hover(move |style| style.bg(p.hover()))
                                .child(ui::avatar(contact.initials.clone(), 32., false, p))
                                .child(
                                    div()
                                        .flex_1()
                                        .text_size(px(TEXT_BODY))
                                        .text_color(p.text_primary)
                                        .child(contact.name.clone()),
                                )
                                .when(selected, |el| el.child(ui::icon(IconName::Check, 16.0, p.accent)));
                            self.clickable_modal(cx, row, Action::ToggleBroadcastRecipient(contact.id))
                        })),
                )
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-broadcast", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("next-broadcast", "Next · name the list", p),
                            Action::BroadcastNextStep,
                        )),
                ),
        )
    }

    fn new_community_sheet(&self, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .text_size(px(18.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(p.text_primary)
                        .child("New community"),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child("A community is a container: one announcement group plus any number of sub-groups."),
                )
                .child(ui::settings_row_field("Community name", "Shown to members", Some("Kain Nusantara".into()), p))
                .child(ui::settings_row_field("Description", "What is this community about?", Some("Textiles and studios".into()), p))
                .child(ui::settings_row_field("Announcement group", "Created automatically, admin-only", Some("Announcements".into()), p))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-community", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("create-community", "Create community", p),
                            Action::CreateCommunity,
                        )),
                ),
        )
    }

    fn edit_contact_sheet(
        &self,
        index: usize,
        cx: &mut Context<Self>,
        p: Palette,
    ) -> gpui_kit::Div {
        let contact = self
            .contacts
            .get(index)
            .cloned()
            .unwrap_or_else(|| model::contacts()[0].clone());
        modal_frame(
            p,
            ui::modal_shell(p)
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(12.))
                        .child(ui::avatar(contact.initials.clone(), 48., contact.online, p))
                        .child(
                            div()
                                .text_size(px(18.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(p.text_primary)
                                .child("Edit contact"),
                        ),
                )
                .child(ui::settings_row_field(
                    "Name",
                    "Shown to your contacts",
                    Some(contact.name.clone()),
                    p,
                ))
                .child(ui::settings_row_field(
                    "Phone",
                    "Used for calls and messages",
                    Some(contact.phone.clone()),
                    p,
                ))
                .child(ui::settings_row_field(
                    "Label",
                    "Optional label",
                    Some("Studio".into()),
                    p,
                ))
                .child(ui::settings_row_field(
                    "Favourite",
                    "Keep this contact close",
                    Some(if contact.favorite { "On" } else { "Off" }.into()),
                    p,
                ))
                .child(
                    div()
                        .flex()
                        .justify_end()
                        .gap(px(8.))
                        .child(self.clickable_modal(
                            cx,
                            ui::secondary_button("cancel-edit-contact", "Cancel", p),
                            Action::CloseOverlay,
                        ))
                        .child(self.clickable_modal(
                            cx,
                            ui::primary_button("save-contact", "Save", p),
                            Action::ShowComingSoon("Saved locally — editing contacts is a mock."),
                        )),
                ),
        )
    }

    fn lightbox(&self, index: usize, cx: &mut Context<Self>, p: Palette) -> gpui_kit::Div {
        let item = self
            .media
            .get(index)
            .cloned()
            .unwrap_or_else(|| model::media_items()[0].clone());
        div()
            .absolute()
            .top_0()
            .left_0()
            .right_0()
            .bottom_0()
            .flex()
            .flex_col()
            .bg(p.text_primary)
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .gap(px(10.))
                    .px(px(16.))
                    .child(ui::avatar("RP", 32., false, p))
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .font_weight(FontWeight::SEMIBOLD)
                                    .text_color(p.text_inverse)
                                    .child("Rizky Pratama"),
                            )
                            .child(
                                div()
                                    .text_size(px(TEXT_CAPTION))
                                    .text_color(p.text_secondary)
                                    .child("19 Sep 2025 · 21:16"),
                            ),
                    )
                    .child(div().flex_1())
                    .child(self.clickable_modal(
                        cx,
                        ui::icon_button(
                            "lightbox-star",
                            IconName::Star,
                            18.0,
                            if self.media_starred.contains(&item.id) {
                                p.attach_orange
                            } else {
                                p.text_inverse
                            },
                            p.text_secondary,
                            "Star",
                        ),
                        Action::MediaStar(item.id),
                    ))
                    .child(self.clickable_modal(
                        cx,
                        ui::icon_button(
                            "lightbox-download",
                            IconName::Download,
                            18.0,
                            p.text_inverse,
                            p.text_secondary,
                            "Download",
                        ),
                        Action::MediaDownload(item.id),
                    ))
                    .child(self.clickable_modal(
                        cx,
                        ui::icon_button(
                            "lightbox-forward",
                            IconName::Forward,
                            18.0,
                            p.text_inverse,
                            p.text_secondary,
                            "Forward",
                        ),
                        Action::MediaForward(item.id),
                    ))
                    .child(self.clickable_modal(
                        cx,
                        ui::icon_button(
                            "lightbox-delete",
                            IconName::Trash,
                            18.0,
                            p.text_inverse,
                            p.text_secondary,
                            "Delete",
                        ),
                        Action::MediaDelete(item.id),
                    ))
                    .child(self.clickable_modal(
                        cx,
                        ui::icon_button(
                            "lightbox-close",
                            IconName::X,
                            18.0,
                            p.text_inverse,
                            p.text_secondary,
                            "Close",
                        ),
                        Action::CloseMedia,
                    )),
            )
            .child(
                div()
                    .flex_1()
                    .relative()
                    .flex()
                    .items_center()
                    .justify_center()
                    .child(
                        div()
                            .w(px(520. * self.media_zoom as f32 / 100.))
                            .h(px(560. * self.media_zoom as f32 / 100.))
                            .rounded(px(RADIUS_LG))
                            .bg(if item.accent % 2 == 0 {
                                p.attach_orange
                            } else {
                                p.attach_blue
                            })
                            .flex()
                            .items_center()
                            .justify_center()
                            .child(ui::icon(IconName::Image, 80.0, p.text_inverse)),
                    )
                    .child(div().absolute().left(px(16.)).child(self.clickable_modal(
                        cx,
                        ui::icon_button(
                            "lightbox-prev",
                            IconName::ChevronLeft,
                            22.0,
                            p.text_inverse,
                            p.text_secondary,
                            "Previous media",
                        ),
                        Action::MediaPrev,
                    )))
                    .child(div().absolute().right(px(16.)).child(self.clickable_modal(
                        cx,
                        ui::icon_button(
                            "lightbox-next",
                            IconName::ChevronRight,
                            22.0,
                            p.text_inverse,
                            p.text_secondary,
                            "Next media",
                        ),
                        Action::MediaNext,
                    ))),
            )
            .child(
                div()
                    .h(px(64.))
                    .flex()
                    .items_center()
                    .justify_center()
                    .gap(px(6.))
                    .children((0..5).map(|thumb| {
                        div()
                            .w(px(56.))
                            .h(px(40.))
                            .rounded(px(RADIUS_SM))
                            .bg(if thumb == 2 {
                                p.attach_orange
                            } else {
                                p.attach_blue
                            })
                            .opacity(if thumb == 2 { 1.0 } else { 0.55 })
                    })),
            )
            .child(
                div()
                    .h(px(56.))
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(16.))
                    .child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_inverse)
                            .child(format!("{} of 24", index + 3)),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(4.))
                            .child(self.clickable_modal(
                                cx,
                                ui::icon_button_small(
                                    "zoom-out",
                                    IconName::Minus,
                                    14.0,
                                    p.text_inverse,
                                    p.text_secondary,
                                    "Zoom out",
                                ),
                                Action::ZoomOut,
                            ))
                            .child(
                                div()
                                    .text_size(px(TEXT_SMALL))
                                    .text_color(p.text_inverse)
                                    .child(format!("{}%", self.media_zoom)),
                            )
                            .child(self.clickable_modal(
                                cx,
                                ui::icon_button_small(
                                    "zoom-in",
                                    IconName::Plus,
                                    14.0,
                                    p.text_inverse,
                                    p.text_secondary,
                                    "Zoom in",
                                ),
                                Action::ZoomIn,
                            )),
                    ),
            )
    }

    fn render_toasts(&self, p: Palette) -> gpui_kit::Div {
        div()
            .absolute()
            .bottom(px(24.))
            .left_0()
            .right_0()
            .flex()
            .flex_col()
            .items_center()
            .gap(px(8.))
            .children(
                self.toasts
                    .iter()
                    .map(|toast| ui::toast_shell(toast.message.clone(), p)),
            )
    }
}

impl Render for App {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        self.render_root(window, cx)
    }
}

// ----------------------------------------------------------------------------
// Small pure helpers
// ----------------------------------------------------------------------------

fn modal_frame(p: Palette, content: gpui_kit::Div) -> gpui_kit::Div {
    div()
        .absolute()
        .top_0()
        .left_0()
        .right_0()
        .bottom_0()
        .flex()
        .items_center()
        .justify_center()
        .child(
            div()
                .absolute()
                .top_0()
                .left_0()
                .right_0()
                .bottom_0()
                .bg(p.text_primary)
                .opacity(0.25),
        )
        .child(content)
}

fn initial_route() -> Route {
    match std::env::var("WHATSLITE_ROUTE")
        .unwrap_or_default()
        .as_str()
    {
        "calls" => Route::Calls,
        "updates" => Route::Updates,
        "communities" => Route::Communities,
        "media" => Route::Media,
        "ai" => Route::Ai,
        "contacts" => Route::Contacts,
        "settings" => Route::Settings,
        "archived" => Route::Archived,
        "starred" => Route::Starred,
        "locked" => Route::Locked,
        "broadcast" => Route::Broadcast,
        "onboarding" => Route::Onboarding,
        _ => Route::Chats,
    }
}

fn app_mark(p: Palette) -> gpui_kit::Div {
    div()
        .size(px(16.))
        .flex()
        .items_center()
        .justify_center()
        .rounded(px(4.))
        .bg(p.accent)
        .child(ui::icon(IconName::MessagesSquare, 12.0, p.text_inverse))
}

fn app_mark_large(p: Palette) -> gpui_kit::Div {
    div()
        .size(px(72.))
        .flex()
        .items_center()
        .justify_center()
        .rounded(px(20.))
        .bg(p.accent)
        .child(ui::icon(IconName::MessagesSquare, 40.0, p.text_inverse))
}

fn traffic_light(color: gpui_kit::Rgba) -> gpui_kit::Div {
    div().size(px(12.)).rounded_full().bg(color)
}

fn window_control(
    cx: &mut Context<App>,
    icon_name: IconName,
    label: &'static str,
    p: Palette,
    action: impl Fn(&mut Window) + 'static,
) -> gpui_kit::Stateful<gpui_kit::Div> {
    div()
        .id(format!("window-{label}"))
        .role(Role::Button)
        .tab_index(0)
        .aria_label(label)
        .cursor_pointer()
        .w(px(46.))
        .h(px(32.))
        .flex()
        .items_center()
        .justify_center()
        .hover(move |style| style.bg(p.hover()))
        .child(ui::icon(icon_name, 12.0, p.text_secondary))
        .on_click(cx.listener(move |_, _, window, _| action(window)))
}

fn section_header(
    cx: &mut Context<App>,
    p: Palette,
    title: &'static str,
    action_icon: Option<IconName>,
    action: Action,
) -> gpui_kit::Div {
    let mut header = div()
        .h(px(64.))
        .flex()
        .items_center()
        .justify_between()
        .px(px(16.))
        .child(
            div()
                .text_size(px(20.))
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(p.text_primary)
                .child(title),
        );
    if let Some(icon_name) = action_icon {
        let button = div()
            .id(format!("section-action-{title}"))
            .role(Role::Button)
            .tab_index(0)
            .aria_label(title)
            .cursor_pointer()
            .size(px(32.))
            .flex()
            .items_center()
            .justify_center()
            .rounded_full()
            .bg(p.accent)
            .child(ui::icon(icon_name, 16.0, p.text_inverse));
        let click = action.clone();
        header = header.child(button.on_click(cx.listener(move |this, _, window, cx| {
            this.apply(click.clone(), window, cx);
        })));
    }
    header
}

fn call_label(call: &CallEntry) -> String {
    let kind = if call.video { "video" } else { "voice" };
    let group = if call.group { "group " } else { "" };
    if call.missed {
        format!("Missed {group}{kind} call")
    } else if call.incoming {
        format!(
            "Incoming · {}",
            call.duration.clone().unwrap_or_else(|| "1 min".into())
        )
    } else {
        format!(
            "Outgoing · {}",
            call.duration.clone().unwrap_or_else(|| "4 min".into())
        )
    }
}

fn call_control(
    id: &'static str,
    icon_name: IconName,
    active: bool,
    p: Palette,
) -> gpui_kit::Stateful<gpui_kit::Div> {
    div()
        .id(id)
        .role(Role::Button)
        .tab_index(0)
        .aria_label(id)
        .cursor_pointer()
        .size(px(52.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_full()
        .bg(if active { p.accent } else { p.bg_list })
        .border_1()
        .border_color(p.border)
        .child(ui::icon(
            icon_name,
            22.0,
            if active {
                p.text_inverse
            } else {
                p.text_primary
            },
        ))
}

fn info_action(
    id: &'static str,
    icon_name: IconName,
    label: &'static str,
    p: Palette,
) -> gpui_kit::Stateful<gpui_kit::Div> {
    div()
        .id(id)
        .role(Role::Button)
        .tab_index(0)
        .aria_label(label)
        .cursor_pointer()
        .flex_1()
        .h(px(52.))
        .flex()
        .flex_col()
        .items_center()
        .justify_center()
        .gap(px(3.))
        .rounded(px(RADIUS_LG))
        .bg(p.bg_list)
        .border_1()
        .border_color(p.border)
        .hover(move |style| style.bg(p.hover()))
        .child(ui::icon(icon_name, 18.0, p.accent))
        .child(
            div()
                .text_size(px(TEXT_CAPTION))
                .text_color(p.text_primary)
                .child(label),
        )
}

fn initials_for(name: Option<&str>) -> String {
    let Some(name) = name else {
        return "?".into();
    };
    let mut parts = name.split_whitespace();
    let first = parts.next().unwrap_or("?");
    let second = parts.next();
    match second {
        Some(second) => format!(
            "{}{}",
            first.chars().next().unwrap_or('?').to_uppercase(),
            second.chars().next().unwrap_or('?').to_uppercase()
        ),
        None => first.chars().take(2).collect::<String>().to_uppercase(),
    }
}

fn same_sender(left: &Message, right: &Message) -> bool {
    left.sender == right.sender && left.sender_name == right.sender_name
}

fn sender_color(name: Option<&str>, p: Palette) -> gpui_kit::Rgba {
    let colors = [
        p.attach_blue,
        p.attach_orange,
        p.attach_green,
        p.attach_red,
        p.accent,
        p.tick_read,
    ];
    let index = name
        .unwrap_or("")
        .bytes()
        .fold(0usize, |acc, byte| acc.wrapping_add(byte as usize));
    colors[index % colors.len()]
}

fn delivery_icon(delivery: Delivery, p: Palette) -> gpui_kit::AnyElement {
    match delivery {
        Delivery::Sending => ui::icon(IconName::Clock, 13.0, p.text_secondary).into_any_element(),
        Delivery::Sent => ui::icon(IconName::Check, 13.0, p.text_secondary).into_any_element(),
        Delivery::Delivered => {
            ui::icon(IconName::CheckCheck, 13.0, p.text_secondary).into_any_element()
        }
        Delivery::Read => ui::icon(IconName::CheckCheck, 13.0, p.tick_read).into_any_element(),
        Delivery::Failed => ui::icon(IconName::CircleAlert, 13.0, p.danger()).into_any_element(),
    }
}

fn media_state_label(state: MediaState) -> &'static str {
    match state {
        MediaState::TapToDownload => "Tap to download",
        MediaState::Downloading => "Downloading",
        MediaState::Failed => "Tap to retry",
        MediaState::Expired => "No longer available",
        MediaState::Ready => "",
    }
}

fn media_icon(item: &MediaItem) -> IconName {
    match item.kind {
        MediaKind::Image => IconName::Image,
        MediaKind::Video => IconName::Video,
        MediaKind::Document => IconName::FileText,
        MediaKind::Link => IconName::Link,
    }
}

fn shortcut_labels(default: &[&str], custom: Option<&str>) -> Vec<String> {
    if let Some(custom) = custom {
        let labels: Vec<String> = custom
            .split('+')
            .map(|part| part.trim().to_owned())
            .filter(|part| !part.is_empty())
            .collect();
        if !labels.is_empty() {
            return labels;
        }
    }
    default.iter().map(|key| (*key).to_owned()).collect()
}

struct FileDragPreview {
    label: String,
    palette: Palette,
}

impl Render for FileDragPreview {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.palette;
        div()
            .flex()
            .items_center()
            .gap(px(6.))
            .px(px(10.))
            .py(px(8.))
            .rounded(px(RADIUS_MD))
            .bg(p.bg_list)
            .border_1()
            .border_color(p.accent)
            .shadow_sm()
            .child(ui::icon(IconName::FileText, 14.0, p.attach_blue))
            .child(
                div()
                    .text_size(px(TEXT_CAPTION))
                    .text_color(p.text_primary)
                    .child(self.label.clone()),
            )
    }
}

#[derive(Clone, Copy, Debug)]
struct StatusDrag {
    from: usize,
}

struct StatusDragPreview {
    palette: Palette,
    accent: usize,
}

impl Render for StatusDragPreview {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.palette;
        div()
            .w(px(56.))
            .h(px(44.))
            .flex()
            .items_center()
            .justify_center()
            .rounded(px(RADIUS_SM))
            .bg(match self.accent % 3 {
                0 => p.attach_orange,
                1 => p.attach_blue,
                _ => p.attach_green,
            })
            .shadow_sm()
            .child(ui::icon(IconName::Image, 16.0, p.text_inverse))
    }
}

fn toggle_status_item(items: &mut Vec<usize>, media: usize) -> bool {
    if let Some(position) = items.iter().position(|item| *item == media) {
        items.remove(position);
        true
    } else if items.len() < STATUS_ITEM_LIMIT {
        items.push(media);
        true
    } else {
        false
    }
}

fn reorder_items<T>(items: &mut Vec<T>, from: usize, to: usize) {
    if from == to || from >= items.len() || to >= items.len() {
        return;
    }
    let item = items.remove(from);
    items.insert(to, item);
}

fn reaction_icon(emoji: &str) -> IconName {
    match emoji {
        "👍" => IconName::ThumbsUp,
        "❤️" | "♥" => IconName::Heart,
        "😂" => IconName::Sparkles,
        "🙏" => IconName::HandHeart,
        "🔥" => IconName::Flame,
        "🎉" => IconName::PartyPopper,
        "👀" => IconName::Eye,
        "✅" => IconName::Check,
        _ => IconName::Star,
    }
}

fn reaction_label(emoji: &str) -> &'static str {
    match emoji {
        "👍" => "Like",
        "❤️" | "♥" => "Love",
        "😂" => "Haha",
        "🙏" => "Thanks",
        "🔥" => "Fire",
        "🎉" => "Celebrate",
        "👀" => "Seen",
        "✅" => "Yes",
        _ => "Reaction",
    }
}

fn message_matches_filter(message: &Message, filter: ChatSearchFilter) -> bool {
    match filter {
        ChatSearchFilter::All => true,
        ChatSearchFilter::Media => matches!(
            message.body,
            MessageBody::Image { .. }
                | MessageBody::Video { .. }
                | MessageBody::Album { .. }
                | MessageBody::Gif { .. }
                | MessageBody::ViewOnce { .. }
        ),
        ChatSearchFilter::Links => matches!(message.body, MessageBody::Link { .. }),
        ChatSearchFilter::Docs => matches!(message.body, MessageBody::Document { .. }),
    }
}

fn message_matches_search(message: &Message, query: &str, filter: ChatSearchFilter) -> bool {
    if !message_matches_filter(message, filter) {
        return false;
    }
    let query = query.trim();
    query.is_empty() || message_preview(message).to_lowercase().contains(query)
}

fn message_preview(message: &Message) -> String {
    match &message.body {
        MessageBody::Text(text) => text.chars().take(60).collect(),
        MessageBody::Rich(runs) => runs
            .iter()
            .map(|run| run.text.as_str())
            .collect::<String>()
            .chars()
            .take(60)
            .collect(),
        MessageBody::Image { .. } => "Photo".into(),
        MessageBody::Video { .. } => "Video".into(),
        MessageBody::Voice { transcript, .. } => {
            transcript.clone().unwrap_or_else(|| "Voice note".into())
        }
        MessageBody::Document { name, .. } => name.clone(),
        MessageBody::Link { title, .. } => title.clone(),
        MessageBody::Album { count, .. } => format!("{count} photos"),
        MessageBody::Emoji(emoji) => emoji.clone(),
        MessageBody::Poll { question, .. } => question.clone(),
        MessageBody::Event { title, .. } => title.clone(),
        MessageBody::ContactCard { name, .. } => name.clone(),
        MessageBody::Location { name, .. } => name.clone(),
        MessageBody::Sticker { symbol, .. } => format!("Sticker {symbol}"),
        MessageBody::Mention { mention, .. } => format!("Mention @{mention}"),
        MessageBody::ViewOnce { .. } => "Photo · view once".into(),
        MessageBody::Gif { failed, .. } => {
            if *failed {
                "GIF failed".into()
            } else {
                "GIF".into()
            }
        }
        MessageBody::Upload { name, .. } => name.clone(),
        MessageBody::LiveLocation { name, .. } => format!("Live location · {name}"),
        MessageBody::Unread(text) => text.clone(),
        MessageBody::Call { missed, video, .. } => {
            if *missed {
                if *video {
                    "Missed video call".into()
                } else {
                    "Missed voice call".into()
                }
            } else {
                "Call".into()
            }
        }
        MessageBody::Deleted => "This message was deleted".into(),
        MessageBody::System(text) => text.clone(),
        MessageBody::Notice(_) => "Notice".into(),
        MessageBody::Date(date) => date.clone(),
    }
}

fn render_qr(state: QrState, p: Palette) -> gpui_kit::Div {
    let pattern: [u8; 21 * 21] = [
        1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0,
        1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 1, 1, 0,
        1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0,
        1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0,
        0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 0, 1, 1, 0, 1, 0, 1, 0,
        1, 1, 0, 1, 0, 1, 1, 0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 1, 0, 0, 0, 0,
        1, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
        0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 1, 1, 1, 1, 1, 1,
        1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0,
        1, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 0, 1, 1, 0, 1,
        1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0,
        0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0,
        1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    ];
    let opacity = match state {
        QrState::Loading => 0.15,
        QrState::Expired => 0.25,
        QrState::Success => 0.9,
        QrState::Active => 1.0,
    };
    let mut grid = div()
        .grid()
        .grid_cols(21)
        .gap(px(1.))
        .p(px(12.))
        .rounded(px(RADIUS_XL))
        .bg(p.bg_list)
        .border_1()
        .border_color(p.border)
        .opacity(opacity);
    for cell in pattern {
        grid =
            grid.child(
                div()
                    .size(px(9.))
                    .bg(if cell == 1 { p.text_primary } else { p.bg_list }),
            );
    }
    div()
        .flex()
        .flex_col()
        .items_center()
        .gap(px(10.))
        .child(grid)
        .when(state == QrState::Loading, |el| {
            el.child(
                div()
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.text_secondary)
                    .child("Requesting a fresh code…"),
            )
        })
        .when(state == QrState::Expired, |el| {
            el.child(
                div()
                    .text_size(px(TEXT_SMALL))
                    .text_color(p.danger())
                    .child("Code expired"),
            )
        })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::theme::{ThemeMode, palette};

    #[test]
    fn initials_are_uppercase_for_people_and_groups() {
        assert_eq!(initials_for(Some("Rizky Pratama")), "RP");
        assert_eq!(initials_for(Some("Produksi")), "PR");
        assert_eq!(initials_for(None), "?");
    }

    #[test]
    fn same_sender_compares_the_group_sender_name() {
        let messages = model::messages_for(1);
        assert!(!same_sender(&messages[1], &messages[2]));
        assert!(same_sender(&messages[1], &messages[5]));
    }

    #[test]
    fn message_preview_covers_every_bubble_family() {
        let text = Message {
            id: 0,
            sender: Sender::Contact,
            sender_name: Some("Rina".into()),
            body: MessageBody::Text("Halo".into()),
            time: "09:00".into(),
            delivery: Delivery::Delivered,
            edited: false,
            reply: None,
            reactions: Vec::new(),
            starred: false,
            countdown: None,
            forwarded: 0,
        };
        assert_eq!(message_preview(&text), "Halo");
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Image {
                    caption: None,
                    state: MediaState::Ready,
                    ratio: 1.0
                },
                ..text.clone()
            }),
            "Photo"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Album {
                    count: 5,
                    caption: None
                },
                ..text.clone()
            }),
            "5 photos"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Deleted,
                ..text.clone()
            }),
            "This message was deleted"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Upload {
                    name: "IMG_2241.jpg".into(),
                    progress: 62,
                    failed: false
                },
                ..text.clone()
            }),
            "IMG_2241.jpg"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::ViewOnce {
                    caption: None,
                    state: crate::model::ViewOnceState::Sealed
                },
                ..text.clone()
            }),
            "Photo · view once"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::LiveLocation {
                    name: "Studio Kain".into(),
                    address: "Jl. Kemang".into(),
                    remaining: "12 more minutes".into(),
                    state: crate::model::LiveLocationState::Sharing
                },
                ..text.clone()
            }),
            "Live location · Studio Kain"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Call {
                    missed: true,
                    video: false,
                    duration: None
                },
                ..text.clone()
            }),
            "Missed voice call"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Mention {
                    text: "Tolong cek".into(),
                    mention: "Rina".into()
                },
                ..text.clone()
            }),
            "Mention @Rina"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Sticker {
                    symbol: "✿".into(),
                    pack_missing: false
                },
                ..text.clone()
            }),
            "Sticker ✿"
        );
        assert_eq!(
            message_preview(&Message {
                body: MessageBody::Gif {
                    caption: None,
                    failed: true
                },
                ..text
            }),
            "GIF failed"
        );
    }

    #[test]
    fn media_state_labels_are_user_facing() {
        assert_eq!(
            media_state_label(MediaState::TapToDownload),
            "Tap to download"
        );
        assert_eq!(media_state_label(MediaState::Downloading), "Downloading");
        assert_eq!(media_state_label(MediaState::Failed), "Tap to retry");
        assert_eq!(
            media_state_label(MediaState::Expired),
            "No longer available"
        );
        assert_eq!(media_state_label(MediaState::Ready), "");
    }

    #[test]
    fn call_labels_cover_incoming_outgoing_and_missed() {
        let missed = model::calls()
            .into_iter()
            .find(|call| call.missed)
            .expect("missed fixture");
        assert_eq!(call_label(&missed), "Missed voice call");
        let incoming = model::calls()
            .into_iter()
            .find(|call| call.incoming && !call.missed)
            .expect("incoming fixture");
        assert!(call_label(&incoming).starts_with("Incoming"));
        let outgoing = model::calls()
            .into_iter()
            .find(|call| !call.incoming && !call.missed)
            .expect("outgoing fixture");
        assert!(call_label(&outgoing).starts_with("Outgoing"));
    }

    #[test]
    fn status_sequence_reorders_without_losing_items() {
        let mut items = vec![0, 2, 3, 1];
        reorder_items(&mut items, 0, 2);
        assert_eq!(items, vec![2, 3, 0, 1]);
        reorder_items(&mut items, 9, 1);
        assert_eq!(items, vec![2, 3, 0, 1]);
        assert_eq!(STATUS_ITEM_LIMIT, 30);
    }

    #[test]
    fn status_sequence_stops_at_thirty_items() {
        let mut items = (0..STATUS_ITEM_LIMIT).collect::<Vec<_>>();
        assert!(!toggle_status_item(&mut items, STATUS_ITEM_LIMIT));
        assert_eq!(items.len(), STATUS_ITEM_LIMIT);
        assert!(toggle_status_item(&mut items, 0));
        assert_eq!(items.len(), STATUS_ITEM_LIMIT - 1);
        assert!(toggle_status_item(&mut items, STATUS_ITEM_LIMIT));
        assert_eq!(items.len(), STATUS_ITEM_LIMIT);
    }

    #[test]
    fn shortcut_labels_use_defaults_and_overrides() {
        assert_eq!(shortcut_labels(&["Ctrl", "K"], None), vec!["Ctrl", "K"]);
        assert_eq!(
            shortcut_labels(&["Ctrl", "K"], Some("Ctrl+Shift+K")),
            vec!["Ctrl", "Shift", "K"]
        );
        assert_eq!(SHORTCUTS.len(), 8);
    }

    #[test]
    fn profile_photo_flow_has_four_states() {
        let states = [
            ProfilePhotoState::Empty,
            ProfilePhotoState::Cropping,
            ProfilePhotoState::Uploaded,
            ProfilePhotoState::Saved,
        ];
        assert_eq!(states.len(), 4);
    }

    #[test]
    fn ai_states_cover_idle_suggested_streaming_and_error() {
        let states = [
            AiState::Idle,
            AiState::Suggested,
            AiState::Streaming,
            AiState::Error,
        ];
        assert_eq!(states.len(), 4);
    }

    #[test]
    fn search_filters_split_media_links_and_docs() {
        let messages = model::messages_for(2);
        let media = messages
            .iter()
            .filter(|message| message_matches_search(message, "", ChatSearchFilter::Media))
            .count();
        let links = messages
            .iter()
            .filter(|message| message_matches_search(message, "", ChatSearchFilter::Links))
            .count();
        let docs = messages
            .iter()
            .filter(|message| message_matches_search(message, "", ChatSearchFilter::Docs))
            .count();
        assert!(media >= 4);
        assert_eq!(links, 1);
        assert_eq!(docs, 1);
        assert_eq!(
            messages
                .iter()
                .filter(|message| message_matches_search(message, "", ChatSearchFilter::All))
                .count(),
            messages.len()
        );
    }

    #[test]
    fn rich_text_preview_concatenates_runs() {
        let message = Message {
            id: 0,
            sender: Sender::Me,
            sender_name: None,
            body: MessageBody::Rich(vec![
                crate::model::TextRun {
                    text: "Deck ".into(),
                    style: crate::model::TextRunStyle::Bold,
                },
                crate::model::TextRun {
                    text: "final".into(),
                    style: crate::model::TextRunStyle::Link,
                },
            ]),
            time: "09:20".into(),
            delivery: Delivery::Sent,
            edited: false,
            reply: None,
            reactions: Vec::new(),
            starred: false,
            countdown: None,
            forwarded: 0,
        };
        assert_eq!(message_preview(&message), "Deck final");
    }

    #[test]
    fn reaction_icons_cover_known_emoji() {
        assert!(matches!(reaction_icon("👍"), IconName::ThumbsUp));
        assert!(matches!(reaction_icon("❤️"), IconName::Heart));
        assert!(matches!(reaction_icon("🔥"), IconName::Flame));
        assert_eq!(reaction_label("👍"), "Like");
        assert_eq!(reaction_label("unknown"), "Reaction");
    }

    #[test]
    fn theme_tokens_match_the_design_package() {
        let light = palette(ThemeMode::Light);
        let dark = palette(ThemeMode::Dark);
        assert_eq!(light.bg_list, gpui_kit::rgb(0xFFFFFF));
        assert_eq!(light.accent, gpui_kit::rgb(0x00AD5A));
        assert_eq!(light.bubble_out, gpui_kit::rgb(0xD9FDD3));
        assert_eq!(dark.bg_conversation, gpui_kit::rgb(0x0B141A));
        assert_eq!(dark.bubble_out, gpui_kit::rgb(0x005C4B));
        assert_eq!(dark.accent, gpui_kit::rgb(0x21C063));
    }

    #[test]
    fn design_handoff_counts_match_the_rendered_reference() {
        let screens: Vec<serde_json::Value> =
            serde_json::from_str(include_str!("../../reference/design/machine/screens.json"))
                .expect("screens.json must parse");
        assert_eq!(screens.len(), 39, "handoff has 39 pages");
        assert!(
            screens
                .iter()
                .any(|page| page["name"].as_str() == Some("38 · Wallpaper Doodle")),
            "page 38 must be registered"
        );

        let components: Vec<serde_json::Value> = serde_json::from_str(include_str!(
            "../../reference/design/machine/components.json"
        ))
        .expect("components.json must parse");
        assert_eq!(components.len(), 54, "handoff has 54 masters");

        let theme: serde_json::Value =
            serde_json::from_str(include_str!("../../reference/design/machine/theme.json"))
                .expect("theme.json must parse");
        assert_eq!(
            theme["theme"]["colors"]
                .as_object()
                .expect("colors object")
                .len(),
            28,
            "theme has 28 color tokens"
        );

        let tokens: serde_json::Value =
            serde_json::from_str(include_str!("../../reference/design/machine/tokens.json"))
                .expect("tokens.json must parse");
        assert_eq!(tokens.as_object().expect("tokens object").len(), 28);

        let theme_colors = theme["theme"]["colors"].as_object().expect("colors");
        let token_map = tokens.as_object().expect("tokens");
        for (name, values) in theme_colors {
            let token = token_map
                .get(&format!("wl-{}", name.replace('_', "-")))
                .unwrap_or_else(|| panic!("tokens.json missing wl-{name}"));
            assert_eq!(
                token["light"].as_str(),
                values["light"].as_str(),
                "light value for {name}"
            );
            assert_eq!(
                token["dark"].as_str(),
                values["dark"].as_str(),
                "dark value for {name}"
            );
        }
    }

    #[test]
    fn design_docs_match_machine_sources() {
        let strings: serde_json::Value =
            serde_json::from_str(include_str!("../../reference/design/machine/strings.json"))
                .expect("strings.json must parse");
        let foundations = strings["01 · Foundations"][0]
            .as_str()
            .expect("foundation wallpaper copy");
        assert!(foundations.contains("800 pt"), "wallpaper repeat is 800 pt");
        assert!(foundations.contains("60%"), "wallpaper light ink is 60%");
        assert!(
            !foundations.contains("550"),
            "stale 550 repeat must be gone"
        );

        let design = include_str!("../../DESIGN.md");
        assert!(design.contains("bubble: \"18px\""), "bubble radius is 18px");
        assert!(design.contains("Inter on Windows"), "Windows font is Inter");
        assert!(
            !design.contains("Segoe UI on Windows"),
            "Segoe UI is not the Windows design font"
        );

        let product = include_str!("../../PRODUCT.md");
        assert!(product.contains("175-row"), "state matrix is 175 rows");
        assert!(
            !product.contains("120 discoverable"),
            "stale 120-state claim must be gone"
        );
        assert!(
            product.contains("page 07 · Handoff"),
            "state matrix points at page 07"
        );
    }
}
