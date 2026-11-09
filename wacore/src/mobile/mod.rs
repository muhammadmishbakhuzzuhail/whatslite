//! Runtime-independent foundations for mobile clients.

mod environment;
pub mod primary;

pub use environment::{
    EnvironmentBinding, EnvironmentBound, MobileConfigurationError, MobileEnvironment, MobileTrust,
    RegistrationEndpoint, TcpEndpoint,
};

/// Mobile client platform family.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum MobilePlatform {
    Android,
    AndroidBusiness,
    Ios,
    IosBusiness,
}

/// Mobile profile/session validation errors.
#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum MobileSessionError {
    #[error("invalid mobile profile: {0}")]
    InvalidProfile(&'static str),
    #[error("mobile handshake failed: {0}")]
    Handshake(String),
}
