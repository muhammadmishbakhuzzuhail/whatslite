pub use wacore::net::{HttpClient, HttpRequest, HttpResponse};

/// An HTTP exchange that completed, carrying the status the client refused it
/// on.
///
/// The media paths will attach it as the `source` of their errors (G5/G6), so a
/// consumer recovers the status by type — [`crate::error::ErrorChainExt::http_status`] —
/// instead of parsing a message. Kept apart from an IQ `code`: a CDN refusing a
/// byte range and the chat server refusing a stanza are different layers with
/// different remedies.
#[derive(Debug, Clone, Copy, PartialEq, Eq, thiserror::Error)]
#[error("HTTP status {status}")]
pub struct HttpStatusError {
    /// The status the server answered with.
    pub status: u16,
}

impl HttpStatusError {
    /// Wraps `context` around this status so the message reads as the operation
    /// that failed while the status stays recoverable underneath it.
    #[allow(dead_code)] // Used once the media paths attach statuses (G5/G6).
    pub(crate) fn into_error(self, context: String) -> anyhow::Error {
        anyhow::Error::new(self).context(context)
    }
}
