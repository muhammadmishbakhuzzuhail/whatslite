//! Typed recovery over the error chain.

use std::error::Error as StdError;

use crate::http::HttpStatusError;
use crate::request::IqError as ClientIqError;
use wacore::request::IqError as CoreIqError;
use wacore::store::error::StoreError;

/// A rejection the server sent in response to a request.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub struct ServerRejection<'a> {
    pub code: u16,
    pub text: &'a str,
    pub error_type: Option<&'a str>,
    pub backoff: Option<u32>,
}

/// Iterator over an error and everything reachable from its source.
#[derive(Clone)]
pub struct Sources<'a> {
    next: Option<&'a (dyn StdError + 'static)>,
}

impl<'a> Iterator for Sources<'a> {
    type Item = &'a (dyn StdError + 'static);

    fn next(&mut self) -> Option<Self::Item> {
        let current = self.next?;
        self.next = current.source();
        Some(current)
    }
}

/// Read-only questions about an arbitrary error chain.
pub trait ErrorChainExt {
    #[doc(hidden)]
    fn as_dyn_error(&self) -> &(dyn StdError + 'static);

    fn sources(&self) -> Sources<'_> {
        Sources {
            next: Some(self.as_dyn_error()),
        }
    }

    fn server_rejection(&self) -> Option<ServerRejection<'_>> {
        self.sources().find_map(server_rejection_of)
    }

    /// The HTTP status behind this error, if any.
    ///
    /// Reports a status the client refused a *completed* HTTP exchange on —
    /// today the media download and upload paths. `None` means no HTTP exchange
    /// got that far: the request never left, or the failure was ours. That
    /// distinction lets a caller serving media onwards tell "the CDN says this
    /// is gone" from "we broke".
    fn http_status(&self) -> Option<u16> {
        self.sources()
            .find_map(|cause| cause.downcast_ref::<HttpStatusError>())
            .map(|refused| refused.status)
    }

    /// Whether the operation ran out of time waiting for the server.
    ///
    /// Covers a request that got no answer and a connect or handshake step that
    /// never completed.
    fn is_timeout(&self) -> bool {
        self.sources().any(|cause| {
            if let Some(iq) = cause.downcast_ref::<ClientIqError>() {
                return iq.is_timeout();
            }
            if let Some(iq) = cause.downcast_ref::<CoreIqError>() {
                return iq.is_timeout();
            }
            if let Some(connect) = cause.downcast_ref::<crate::client::lifecycle::ConnectError>() {
                return connect.is_timeout();
            }
            cause
                .downcast_ref::<crate::handshake::HandshakeError>()
                .is_some_and(crate::handshake::HandshakeError::is_timeout)
        })
    }

    /// Whether the failure was the transport being gone rather than the
    /// operation being refused.
    ///
    /// Mirrors the judgement the send and receive paths already make when
    /// deciding whether a failure is worth retrying.
    fn is_transport_unavailable(&self) -> bool {
        self.sources().any(|cause| {
            if let Some(client) = cause.downcast_ref::<crate::client::ClientError>() {
                return client.is_transport_unavailable();
            }
            if let Some(iq) = cause.downcast_ref::<ClientIqError>() {
                return iq.is_transport_unavailable();
            }
            if let Some(encrypt) = cause.downcast_ref::<crate::socket::error::EncryptSendError>() {
                return encrypt.is_transport_unavailable();
            }
            cause
                .downcast_ref::<CoreIqError>()
                .is_some_and(CoreIqError::is_transport_unavailable)
        })
    }

    fn store_failure(&self) -> Option<&StoreError> {
        self.sources().find_map(|cause| cause.downcast_ref())
    }
}

fn server_rejection_of<'a>(cause: &'a (dyn StdError + 'static)) -> Option<ServerRejection<'a>> {
    if let Some(CoreIqError::ServerError { code, text }) = cause.downcast_ref::<CoreIqError>() {
        return Some(ServerRejection {
            code: *code,
            text,
            error_type: None,
            backoff: None,
        });
    }
    if let Some(ClientIqError::ServerError { code, text, .. }) =
        cause.downcast_ref::<ClientIqError>()
    {
        return Some(ServerRejection {
            code: *code,
            text,
            error_type: None,
            backoff: None,
        });
    }
    None
}

impl<E: StdError + 'static> ErrorChainExt for E {
    fn as_dyn_error(&self) -> &(dyn StdError + 'static) {
        self
    }
}

impl ErrorChainExt for dyn StdError + 'static {
    fn as_dyn_error(&self) -> &(dyn StdError + 'static) {
        self
    }
}

impl ErrorChainExt for dyn StdError + Send + Sync + 'static {
    fn as_dyn_error(&self) -> &(dyn StdError + 'static) {
        self
    }
}
