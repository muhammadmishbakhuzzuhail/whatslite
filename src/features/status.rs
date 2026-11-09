//! Status surface.

#[derive(Debug, Clone, Default)]
pub struct StatusSendOptions {
    pub background_color: Option<u32>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StatusPrivacySetting {
    Contacts,
    AllowList,
    DenyList,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Status;
