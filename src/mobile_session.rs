//! Mobile registration/login session profiles. Experimental Phase 1 surface;
//! the wire registration flow remains gated by `mobile-core`.

use std::sync::Arc;

use wacore::libsignal::protocol::KeyPair;
use wacore::mobile::MobileEnvironment;
pub use wacore::mobile::{MobilePlatform, MobileSessionError};

/// App-state behavior selected explicitly by a mobile host.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum MobileAppStatePolicy {
    /// Preserve the shared snapshot/bootstrap exchange.
    SnapshotCompatible,
    /// Authoritative local state, no snapshot restoration.
    AuthoritativeZapo,
}

/// Host-provided ephemeral key source for a mobile login attempt.
pub trait MobileEphemeralKeyProvider: wacore::sync_marker::MaybeSendSync + 'static {
    fn generate(&self) -> Result<KeyPair, MobileSessionError>;
}

/// Context for one payload-generation attempt.
pub struct MobileLoginAttempt<'a> {
    pub device: &'a crate::store::Device,
    pub zero_based_attempt: u32,
}

/// Minimal per-attempt login options.
#[derive(Debug, Clone, Default)]
pub struct MobileLoginOptions {
    pub passive: Option<bool>,
    pub session_id: Option<i32>,
    pub connect_attempt_count: Option<u32>,
    pub push_name: Option<String>,
}

pub trait MobileLoginOptionsProvider: wacore::sync_marker::MaybeSendSync + 'static {
    fn options(
        &self,
        context: MobileLoginAttempt<'_>,
        configured: &MobileLoginOptions,
    ) -> Result<MobileLoginOptions, MobileSessionError>;
}

/// Explicit registration-to-login installation identity mapping.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum RegistrationLoginPhoneIdMapping {
    FdidUppercase,
    IndependentHostId,
}

/// Metadata shared by registration and login profiles.
#[derive(Clone, Copy, Debug)]
pub struct RegistrationLoginMetadata<'a> {
    pub platform: MobilePlatform,
    pub app_version: &'a [u32],
    pub manufacturer: &'a str,
}

/// Validate shared registration/login metadata.
pub fn validate_registration_login_profile(
    profile: &MobileLoginProfile,
    metadata: &RegistrationLoginMetadata<'_>,
    _phone_id_mapping: RegistrationLoginPhoneIdMapping,
) -> Result<(), MobileSessionError> {
    if metadata.app_version.is_empty() || profile.version.is_empty() {
        return Err(MobileSessionError::InvalidProfile("missing app version"));
    }
    if matches!(
        metadata.platform,
        MobilePlatform::Android | MobilePlatform::AndroidBusiness
    ) && metadata.manufacturer.is_empty()
    {
        return Err(MobileSessionError::InvalidProfile(
            "Android manufacturer is required",
        ));
    }
    Ok(())
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct MobileLoginProfile {
    pub version: Vec<u32>,
    pub platform: MobilePlatform,
    pub user_agent: String,
}

/// Optional media HTTP profile for mobile sessions.
#[derive(Clone, Debug)]
pub struct MobileMediaHttpProfile {
    pub user_agent: String,
}

/// Immutable connection profile bound to one account and environment.
pub struct RegisteredMobileSession {
    environment: MobileEnvironment,
    username: u64,
    identity: Vec<u8>,
    profile: MobileLoginProfile,
    options: MobileLoginOptions,
    app_state_policy: Option<MobileAppStatePolicy>,
    ephemeral_keys: Option<Arc<dyn MobileEphemeralKeyProvider>>,
    media_http_profile: Option<MobileMediaHttpProfile>,
}

impl RegisteredMobileSession {
    pub fn new(
        environment: MobileEnvironment,
        username: u64,
        identity: Vec<u8>,
        profile: MobileLoginProfile,
        options: MobileLoginOptions,
    ) -> Self {
        Self {
            environment,
            username,
            identity,
            profile,
            options,
            app_state_policy: None,
            ephemeral_keys: None,
            media_http_profile: None,
        }
    }

    pub fn with_app_state_policy(mut self, policy: MobileAppStatePolicy) -> Self {
        self.app_state_policy = Some(policy);
        self
    }

    pub fn with_ephemeral_key_provider(
        mut self,
        provider: Arc<dyn MobileEphemeralKeyProvider>,
    ) -> Self {
        self.ephemeral_keys = Some(provider);
        self
    }

    pub fn with_media_http_profile(mut self, profile: MobileMediaHttpProfile) -> Self {
        self.media_http_profile = Some(profile);
        self
    }

    pub fn environment(&self) -> &MobileEnvironment {
        &self.environment
    }
    pub fn username(&self) -> u64 {
        self.username
    }
    pub fn identity(&self) -> &[u8] {
        &self.identity
    }
    pub fn profile(&self) -> &MobileLoginProfile {
        &self.profile
    }
    pub fn options(&self) -> &MobileLoginOptions {
        &self.options
    }
    pub fn app_state_policy(&self) -> Option<MobileAppStatePolicy> {
        self.app_state_policy
    }
    pub fn media_http_profile(&self) -> Option<&MobileMediaHttpProfile> {
        self.media_http_profile.as_ref()
    }
}

/// Primary-device identity policy selector.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PrimaryDeviceIdentityPolicy {
    Companion,
    Primary,
}

/// Primary self-identity profile selector.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PrimarySelfIdentityProfile {
    Web,
    Android,
    Ios,
}
