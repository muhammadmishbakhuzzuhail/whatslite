//! Passkey flow state. The WebAuthn ceremony is host-driven; Phase 1 exposes
//! the registration seam and keeps the network steps behind the passkey gate.

use std::sync::Arc;

use tokio::sync::Mutex;

use super::{Assertion, PasskeyAuthenticator, PasskeyError};
use crate::client::Client;

#[derive(Default)]
pub(crate) struct Passkey {
    pub(crate) authenticator: Mutex<Option<Arc<dyn PasskeyAuthenticator>>>,
}

impl Client {
    pub(crate) fn passkey(&self) -> &Passkey {
        &self.passkey
    }

    /// Register the host authenticator used to answer server passkey requests.
    pub async fn set_passkey_authenticator(&self, authenticator: Arc<dyn PasskeyAuthenticator>) {
        *self.passkey().authenticator.lock().await = Some(authenticator);
    }

    /// Send a WebAuthn assertion for the current passkey request.
    pub async fn send_passkey_response(&self, _assertion: Assertion) -> Result<(), PasskeyError> {
        Err(PasskeyError::Flow(
            "passkey response flow is not wired in this build".to_string(),
        ))
    }

    /// Confirm the passkey verification code shown by the primary device.
    pub async fn send_passkey_confirmation(&self) -> Result<(), PasskeyError> {
        Err(PasskeyError::Flow(
            "passkey confirmation flow is not wired in this build".to_string(),
        ))
    }
}
