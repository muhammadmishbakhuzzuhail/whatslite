//! Mobile feature facade.

pub use crate::mobile_session::{
    MobileAppStatePolicy, MobileEphemeralKeyProvider, MobileLoginAttempt, MobileLoginOptions,
    MobileLoginOptionsProvider, MobileMediaHttpProfile, MobilePlatform, MobileSessionError,
    PrimaryDeviceIdentityPolicy, PrimarySelfIdentityProfile, RegisteredMobileSession,
    RegistrationLoginPhoneIdMapping, validate_registration_login_profile,
};
