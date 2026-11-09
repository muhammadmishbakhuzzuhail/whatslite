//! Typed recovery over the error chain.

use std::error::Error as StdError;

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
