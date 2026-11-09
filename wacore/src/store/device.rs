use crate::libsignal::protocol::{IdentityKeyPair, KeyPair};
use once_cell::sync::Lazy;
use serde::{Deserialize, Serialize};
use serde_big_array::BigArray;
use wacore_binary::jid::Jid;
use waproto::buffa::Message as _;
use waproto::whatsapp as wa;

pub mod key_pair_serde {
    use super::KeyPair;
    use crate::libsignal::protocol::{PrivateKey, PublicKey};
    use serde::{self, Deserializer, Serializer};

    pub fn serialize<S>(key_pair: &KeyPair, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        let bytes: Vec<u8> = key_pair
            .private_key
            .serialize()
            .iter()
            .copied()
            .chain(key_pair.public_key.public_key_bytes().iter().copied())
            .collect();
        serializer.serialize_bytes(&bytes)
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<KeyPair, D::Error>
    where
        D: Deserializer<'de>,
    {
        let bytes: &[u8] = serde::Deserialize::deserialize(deserializer)?;
        if bytes.len() != 64 {
            return Err(serde::de::Error::invalid_length(bytes.len(), &"64"));
        }
        let private_key = PrivateKey::deserialize(&bytes[0..32])
            .map_err(|e| serde::de::Error::custom(e.to_string()))?;
        let public_key = PublicKey::from_djb_public_key_bytes(&bytes[32..64])
            .map_err(|e| serde::de::Error::custom(e.to_string()))?;
        Ok(KeyPair::new(public_key, private_key))
    }
}

fn build_base_client_payload(
    app_version: wa::client_payload::user_agent::AppVersion,
) -> wa::ClientPayload {
    let mut user_agent = wa::client_payload::UserAgent::default();
    user_agent.platform = Some(wa::client_payload::user_agent::Platform::WEB);
    user_agent.release_channel = Some(wa::client_payload::user_agent::ReleaseChannel::RELEASE);
    user_agent.app_version = app_version.into();
    user_agent.mcc = Some("000".to_string());
    user_agent.mnc = Some("000".to_string());
    user_agent.os_version = Some("0.1.0".to_string());
    user_agent.manufacturer = Some("".to_string());
    user_agent.device = Some("Desktop".to_string());
    user_agent.os_build_number = Some("0.1.0".to_string());
    user_agent.locale_language_iso6391 = Some("en".to_string());
    user_agent.locale_country_iso31661_alpha2 = Some("en".to_string());

    let mut web_info = wa::client_payload::WebInfo::default();
    web_info.web_sub_platform = Some(wa::client_payload::web_info::WebSubPlatform::WEB_BROWSER);

    let mut payload = wa::ClientPayload::default();
    payload.user_agent = user_agent.into();
    payload.web_info = web_info.into();
    payload.connect_type = Some(wa::client_payload::ConnectType::WIFI_UNKNOWN);
    payload.connect_reason = Some(wa::client_payload::ConnectReason::USER_ACTIVATED);
    payload
}

/// Override for selected `DeviceProps` fields before pairing. `None` fields
/// preserve the current value on the device (oxidezap #586).
#[derive(Debug, Clone, Default)]
pub struct DevicePropsOverride {
    pub os: Option<String>,
    pub version: Option<wa::device_props::AppVersion>,
    pub platform_type: Option<wa::device_props::PlatformType>,
    pub history_sync_config: Option<wa::device_props::HistorySyncConfig>,
}

impl DevicePropsOverride {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_os(mut self, os: impl Into<String>) -> Self {
        self.os = Some(os.into());
        self
    }

    pub fn with_version(mut self, version: wa::device_props::AppVersion) -> Self {
        self.version = Some(version);
        self
    }

    pub fn with_platform_type(mut self, platform_type: wa::device_props::PlatformType) -> Self {
        self.platform_type = Some(platform_type);
        self
    }

    /// Replaces the entire `HistorySyncConfig`. Spread [`default_history_sync_config`]
    /// into the literal to patch only specific fields while keeping sane defaults.
    pub fn with_history_sync_config(
        mut self,
        history_sync_config: wa::device_props::HistorySyncConfig,
    ) -> Self {
        self.history_sync_config = Some(history_sync_config);
        self
    }

    pub fn is_empty(&self) -> bool {
        self.os.is_none()
            && self.version.is_none()
            && self.platform_type.is_none()
            && self.history_sync_config.is_none()
    }
}

/// Default `HistorySyncConfig` aligned with WA Web's static claims
/// (`Payload.js` in `WAWebClientPayload`). Runtime-derived fields like
/// `storage_quota_mb`, `on_demand_ready`, and justknobx-gated flags are left
/// unset so callers can populate them through
/// [`DevicePropsOverride::with_history_sync_config`] without fighting stale
/// hardcoded values (oxidezap #586).
pub fn default_history_sync_config() -> wa::device_props::HistorySyncConfig {
    let mut config = wa::device_props::HistorySyncConfig::default();
    config.full_sync_days_limit = Some(30);
    config.inline_initial_payload_in_e2_ee_msg = Some(true);
    config.support_bot_user_agent_chat_history = Some(true);
    config.support_cag_reactions_and_polls = Some(true);
    config.support_recent_sync_chunk_message_count_tuning = Some(true);
    config.support_hosted_group_msg = Some(true);
    config.support_biz_hosted_msg = Some(true);
    config.support_fbid_bot_chat_history = Some(true);
    config.support_message_association = Some(true);
    config
}

pub static DEVICE_PROPS: Lazy<wa::DeviceProps> = Lazy::new(|| {
    let mut version = wa::device_props::AppVersion::default();
    version.primary = Some(0);
    version.secondary = Some(1);
    version.tertiary = Some(0);

    let mut props = wa::DeviceProps::default();
    props.os = Some("rust".to_string());
    props.version = version.into();
    props.platform_type = Some(wa::device_props::PlatformType::UNKNOWN);
    props.require_full_sync = Some(true);
    props.history_sync_config = default_history_sync_config().into();
    props
});

#[derive(Clone, Serialize, Deserialize)]
pub struct Device {
    pub pn: Option<Jid>,
    pub lid: Option<Jid>,
    pub registration_id: u32,
    #[serde(with = "key_pair_serde")]
    pub noise_key: KeyPair,
    #[serde(with = "key_pair_serde")]
    pub identity_key: KeyPair,
    #[serde(with = "key_pair_serde")]
    pub signed_pre_key: KeyPair,
    pub signed_pre_key_id: u32,
    #[serde(with = "BigArray")]
    pub signed_pre_key_signature: [u8; 64],
    pub adv_secret_key: [u8; 32],
    pub account: Option<wa::ADVSignedDeviceIdentity>,
    pub push_name: String,
    pub app_version_primary: u32,
    pub app_version_secondary: u32,
    pub app_version_tertiary: u32,
    pub app_version_last_fetched_ms: i64,
    #[serde(skip)]
    pub device_props: wa::DeviceProps,
    /// Edge routing info received from server, used for optimized reconnection.
    /// When present, this should be sent as a pre-intro before the Noise handshake.
    #[serde(default)]
    pub edge_routing_info: Option<Vec<u8>>,
    /// Hash from the last props (A/B experiment config) fetch.
    /// Sent on subsequent connects to enable delta updates instead of full fetches.
    #[serde(default)]
    pub props_hash: Option<String>,
    /// Server cert chain cached from the last successful XX (or XX-fallback)
    /// handshake. Enables Noise IK on the next connect by exposing
    /// `leaf.key` as the server's static public key, and lets us reject
    /// stale entries via `not_after` before even attempting IK.
    /// `None` forces XX on the next connect (oxidezap #598).
    #[serde(default)]
    pub server_cert_chain: Option<CachedServerCertChain>,
}

/// Minimal cached form of a Noise certificate. Mirrors the JSON shape WA Web
/// persists in `waNoiseInfo.certificateChainBuffer` (only `key` plus the
/// validity window — signatures and issuer_serial are intentionally dropped).
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct CachedNoiseCert {
    /// 32-byte X25519 public key from `NoiseCertificate.Details.key`.
    pub key: [u8; 32],
    /// Unix epoch seconds. Validation window from `NoiseCertificate.Details`.
    pub not_before: i64,
    pub not_after: i64,
}

/// Cached form of the server's two-cert chain. `leaf.key` is the server
/// static public key consumed by Noise IK; the intermediate is kept solely
/// to mirror WA Web's expiry checks.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct CachedServerCertChain {
    pub intermediate: CachedNoiseCert,
    pub leaf: CachedNoiseCert,
}

impl From<wacore_noise::VerifiedServerCertChain> for CachedServerCertChain {
    fn from(v: wacore_noise::VerifiedServerCertChain) -> Self {
        Self {
            intermediate: CachedNoiseCert {
                key: v.intermediate_key,
                not_before: v.intermediate_not_before,
                not_after: v.intermediate_not_after,
            },
            leaf: CachedNoiseCert {
                key: v.leaf_key,
                not_before: v.leaf_not_before,
                not_after: v.leaf_not_after,
            },
        }
    }
}

impl Default for Device {
    fn default() -> Self {
        Self::new()
    }
}

impl Device {
    pub fn new() -> Self {
        use rand::Rng;
        use rand::RngCore;

        let mut rng = rand::rng();
        let identity_key_pair = IdentityKeyPair::generate(&mut rng);

        let identity_key: KeyPair = KeyPair::new(
            *identity_key_pair.public_key(),
            identity_key_pair.private_key().clone(),
        );
        let signed_pre_key = KeyPair::generate(&mut rng);
        let signature_box = identity_key_pair
            .private_key()
            .calculate_signature(&signed_pre_key.public_key.serialize(), &mut rng)
            .expect("signing with valid Ed25519 key should succeed");
        let signed_pre_key_signature: [u8; 64] = signature_box
            .as_ref()
            .try_into()
            .expect("Ed25519 signature is always 64 bytes");
        let mut adv_secret_key = [0u8; 32];
        rng.fill_bytes(&mut adv_secret_key);

        Self {
            pn: None,
            lid: None,
            registration_id: rng.random_range(1..=2147483647),
            noise_key: KeyPair::generate(&mut rng),
            identity_key,
            signed_pre_key,
            signed_pre_key_id: 1,
            signed_pre_key_signature,
            adv_secret_key,
            account: None,
            push_name: String::new(),
            app_version_primary: 2,
            app_version_secondary: 3000,
            app_version_tertiary: 1031424117,
            app_version_last_fetched_ms: 0,
            device_props: DEVICE_PROPS.clone(),
            edge_routing_info: None,
            props_hash: None,
            server_cert_chain: None,
        }
    }

    /// Returns the default OS string used for device props
    pub fn default_os() -> &'static str {
        "rust"
    }

    /// Returns the default device props version
    pub fn default_device_props_version() -> wa::device_props::AppVersion {
        let mut version = wa::device_props::AppVersion::default();
        version.primary = Some(0);
        version.secondary = Some(1);
        version.tertiary = Some(0);
        version
    }

    pub fn is_ready_for_presence(&self) -> bool {
        self.pn.is_some() && !self.push_name.is_empty()
    }

    /// Mirrors WA Web `WAWebUserPrefsMultiDevice.isRegistered()`:
    /// `!!(m() && getMaybeMeDevicePn())`.
    pub fn is_registered(&self) -> bool {
        self.pn.is_some()
    }

    pub fn set_device_props(&mut self, o: DevicePropsOverride) {
        if let Some(os) = o.os {
            self.device_props.os = Some(os);
        }
        if let Some(version) = o.version {
            self.device_props.version = version.into();
        }
        if let Some(platform_type) = o.platform_type {
            self.device_props.platform_type = Some(platform_type);
        }
        if let Some(history_sync_config) = o.history_sync_config {
            self.device_props.history_sync_config = history_sync_config.into();
        }
    }

    pub fn get_client_payload(&self) -> wa::ClientPayload {
        match &self.pn {
            Some(jid) => self.get_login_payload(jid),
            None => self.get_registration_payload(),
        }
    }

    fn get_login_payload(&self, jid: &Jid) -> wa::ClientPayload {
        let mut app_version = wa::client_payload::user_agent::AppVersion::default();
        app_version.primary = Some(self.app_version_primary);
        app_version.secondary = Some(self.app_version_secondary);
        app_version.tertiary = Some(self.app_version_tertiary);
        let mut payload = build_base_client_payload(app_version);
        payload.username = jid.user.parse::<u64>().ok();
        payload.device = Some(jid.device as u32);
        payload.passive = Some(true);
        payload
    }

    fn get_registration_payload(&self) -> wa::ClientPayload {
        let mut app_version = wa::client_payload::user_agent::AppVersion::default();
        app_version.primary = Some(self.app_version_primary);
        app_version.secondary = Some(self.app_version_secondary);
        app_version.tertiary = Some(self.app_version_tertiary);
        let mut payload = build_base_client_payload(app_version);

        let device_props_bytes = self.device_props.encode_to_vec();

        let version = payload
            .user_agent
            .as_option()
            .expect("payload should have user_agent")
            .app_version
            .as_option()
            .expect("user_agent should have app_version");
        let version_str = format!(
            "{}.{}.{}",
            version.primary.unwrap_or(0),
            version.secondary.unwrap_or(0),
            version.tertiary.unwrap_or(0)
        );
        let build_hash: [u8; 16] = md5::compute(version_str.as_bytes()).into();

        let mut reg_data = wa::client_payload::DevicePairingRegistrationData::default();
        reg_data.e_regid = Some(self.registration_id.to_be_bytes().to_vec());
        reg_data.e_keytype = Some(vec![5]);
        reg_data.e_ident = Some(self.identity_key.public_key.public_key_bytes().to_vec());
        reg_data.e_skey_id = Some(self.signed_pre_key_id.to_be_bytes()[1..].to_vec());
        reg_data.e_skey_val = Some(self.signed_pre_key.public_key.public_key_bytes().to_vec());
        reg_data.e_skey_sig = Some(self.signed_pre_key_signature.to_vec());
        reg_data.build_hash = Some(build_hash.to_vec());
        reg_data.device_props = Some(device_props_bytes);

        payload.device_pairing_data = reg_data.into();
        payload.passive = Some(false);
        payload.pull = Some(false);
        payload
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_registration_id_range() {
        for _ in 0..1000 {
            let device = Device::new();
            assert!(device.registration_id >= 1);
            assert!(device.registration_id <= 2147483647);
        }
    }

    /// Override survives the ClientPayload → bytes → DeviceProps round-trip;
    /// `None` fields preserve the prior value (oxidezap #586).
    #[test]
    fn set_device_props_override_reaches_registration_payload() {
        use waproto::buffa::Message as _;
        let mut device = Device::new();
        assert!(device.pn.is_none());

        device.set_device_props(
            DevicePropsOverride::new()
                .with_os("Android 14")
                .with_platform_type(wa::device_props::PlatformType::Chrome),
        );

        let payload = device.get_client_payload();
        let reg = payload
            .device_pairing_data
            .expect("registration payload must carry device_pairing_data");
        let bytes = reg
            .device_props
            .expect("device_pairing_data must carry device_props");
        let props =
            wa::DeviceProps::decode_from_slice(bytes.as_slice()).expect("device_props must decode");

        assert_eq!(props.os.as_deref(), Some("Android 14"));
        assert_eq!(
            props.platform_type,
            Some(wa::device_props::PlatformType::Chrome)
        );
    }

    /// The static defaults now carry WA Web's HistorySyncConfig claim set.
    #[test]
    fn device_props_default_history_sync_config_matches_wa_web() {
        let cfg = default_history_sync_config();
        assert_eq!(cfg.full_sync_days_limit, Some(30));
        assert_eq!(cfg.inline_initial_payload_in_e2_ee_msg, Some(true));
        assert_eq!(cfg.support_message_association, Some(true));
        assert_eq!(cfg.support_hosted_group_msg, Some(true));
        // storage_quota_mb is runtime-derived: the static must not pin it.
        assert_eq!(cfg.storage_quota_mb, None);
    }
}
