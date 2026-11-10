//! Newsletter surface.

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum NewsletterError {
    #[error("newsletter operation failed: {0}")]
    Failed(String),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewsletterMetadata {
    pub jid: String,
    pub name: String,
    pub state: NewsletterState,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewsletterMessage {
    pub id: String,
    pub text: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NewsletterAdminInfo {
    pub admins: Vec<String>,
}

/// A poll option's vote tally on a newsletter message.
///
/// Channel polls are counted server-side, so unlike a DM or group vote (which
/// arrives encrypted) there is nothing to decrypt here: the server hands over
/// the totals directly, keyed by `option_hash`, the SHA-256 of the option name.
#[derive(Debug, Clone, PartialEq, Eq, bon::Builder)]
#[non_exhaustive]
pub struct NewsletterPollVote {
    pub option_hash: [u8; 32],
    pub count: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NewsletterRole {
    Owner,
    Admin,
    Subscriber,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NewsletterState {
    Active,
    Suspended,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Newsletter<'a> {
    pub _marker: std::marker::PhantomData<&'a ()>,
}
