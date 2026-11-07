# Component hierarchy

```text
AppWindow
├── MacWindowHost (native titlebar/window API)
├── NavigationRail
│   ├── NavigationItem × N
│   └── AccountButton
├── RouteOutlet
│   ├── AuthView (welcome | qr | phone | status)
│   ├── ChatRoute
│   │   ├── ChatListPanel (SearchField, Filters, ChatRow × N)
│   │   └── ConversationPanel
│   │       ├── ConversationHeader
│   │       ├── MessageList (MessageBubble, MediaCard, InfoBadge)
│   │       └── Composer (ReplyPreview, AttachmentButton, Input, SendButton)
│   ├── CallsRoute (CallHistory, CallSurface, CallControls)
│   ├── UpdatesRoute (StatusList, ChannelList, CommunityDetail)
│   ├── SettingsRoute (Sections, SettingsDetail)
│   └── MediaRoute (MediaGallery, Viewer)
├── DialogHost (modal / menu / picker / confirm)
└── ToastHost
```

State is owned at the smallest durable parent; domain/session state lives in `AppModel`, transient hover/menu state lives in the feature view. Screens are **states** and not 120 independent GPUI structs.
