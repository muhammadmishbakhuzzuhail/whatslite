use crate::companion_reg::CompanionWebClientType;
use crate::libsignal::protocol::{KeyPair, PublicKey};
use aes_gcm::Aes256Gcm;
use aes_gcm::aead::{Aead, KeyInit, Payload};
use base64::Engine as _;
use base64::prelude::*;
use hkdf::Hkdf;
use hmac::{Hmac, Mac};
use rand::TryRngCore;
use sha2::Sha256;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::{Jid, SERVER_JID};
use wacore_binary::node::Node;
use waproto::buffa::Message as _;
use waproto::whatsapp as wa;
use waproto::whatsapp::ADVEncryptionType;

// Prefixes from whatsmeow/pair.go, crucial for signature verification
const ADV_PREFIX_ACCOUNT_SIGNATURE: &[u8] = &[6, 0];
const ADV_PREFIX_DEVICE_SIGNATURE_GENERATE: &[u8] = &[6, 1];
const ADV_HOSTED_PREFIX_ACCOUNT_SIGNATURE: &[u8] = &[6, 5];

// Aliases for HMAC-SHA256
type HmacSha256 = Hmac<Sha256>;

#[derive(Debug)]
pub struct PairCryptoError {
    pub code: u16,
    pub text: &'static str,
    pub source: anyhow::Error,
}

impl std::fmt::Display for PairCryptoError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "pairing crypto failed with code {}: {} (source: {})",
            self.code, self.text, self.source
        )
    }
}

impl std::error::Error for PairCryptoError {}

/// Device state needed for pairing operations
pub struct DeviceState {
    pub identity_key: KeyPair,
    pub noise_key: KeyPair,
    pub adv_secret_key: [u8; 32],
}

/// Core pairing utilities that are platform-independent
pub struct PairUtils;

impl PairUtils {
    /// Constructs the full QR code string from the ref and device keys.
    ///
    /// `<ref>,<noise>,<identity>,<adv>,<client_type>` per WA Web
    /// (`WAWebLinkDeviceQrcode.react`); the 4-field form is rejected since
    /// tulir/whatsmeow#1110.
    pub fn make_qr_data(
        device_state: &DeviceState,
        ref_str: String,
        client_type: CompanionWebClientType,
    ) -> String {
        let noise_b64 =
            BASE64_STANDARD.encode(device_state.noise_key.public_key.public_key_bytes());
        let identity_b64 =
            BASE64_STANDARD.encode(device_state.identity_key.public_key.public_key_bytes());
        let adv_b64 = BASE64_STANDARD.encode(device_state.adv_secret_key);

        format!("{ref_str},{noise_b64},{identity_b64},{adv_b64},{client_type}")
    }

    /// Builds acknowledgment node for a pairing request
    pub fn build_ack_node(request_node: &Node) -> Option<Node> {
        if let (Some(to), Some(id)) = (request_node.attrs.get("from"), request_node.attrs.get("id"))
        {
            Some(
                NodeBuilder::new("iq")
                    .attrs([
                        ("to", to.to_string_value()),
                        ("id", id.to_string_value()),
                        ("type", "result".to_string()),
                    ])
                    .build(),
            )
        } else {
            None
        }
    }

    /// Builds pair error node
    pub fn build_pair_error_node(req_id: &str, code: u16, text: &str) -> Node {
        let error_node = NodeBuilder::new("error")
            .attrs([("code", code.to_string()), ("text", text.to_string())])
            .build();
        NodeBuilder::new("iq")
            .attrs([
                ("to", SERVER_JID.to_string()),
                ("type", "error".to_string()),
                ("id", req_id.to_string()),
            ])
            .children([error_node])
            .build()
    }

    /// Performs the cryptographic operations for pairing
    pub fn do_pair_crypto(
        device_state: &DeviceState,
        device_identity_bytes: &[u8],
    ) -> Result<(Vec<u8>, u32), PairCryptoError> {
        // 1. Unmarshal HMAC container and verify HMAC
        let hmac_container = wa::ADVSignedDeviceIdentityHMAC::decode_from_slice(
            device_identity_bytes,
        )
        .map_err(|e| PairCryptoError {
            code: 500,
            text: "internal-error",
            source: e.into(),
        })?;

        // Determine if this is a hosted account
        let is_hosted_account = hmac_container.account_type == Some(ADVEncryptionType::HOSTED);

        let mut mac =
            <HmacSha256 as Mac>::new_from_slice(&device_state.adv_secret_key).map_err(|e| {
                PairCryptoError {
                    code: 500,
                    text: "internal-error",
                    source: e.into(),
                }
            })?;
        // Get details and hmac as slices, handling potential None values
        let details_bytes = hmac_container
            .details
            .as_deref()
            .ok_or_else(|| PairCryptoError {
                code: 500,
                text: "internal-error",
                source: anyhow::anyhow!("HMAC container missing details"),
            })?;
        let hmac_bytes = hmac_container
            .hmac
            .as_deref()
            .ok_or_else(|| PairCryptoError {
                code: 500,
                text: "internal-error",
                source: anyhow::anyhow!("HMAC container missing hmac"),
            })?;

        if is_hosted_account {
            mac.update(ADV_HOSTED_PREFIX_ACCOUNT_SIGNATURE);
        }
        mac.update(details_bytes);
        // adv_secret is shared with the primary out-of-band (QR string or
        // pair-code DH). HMAC mismatch means the container is forged:
        // account_signature alone is not a backstop, since its key comes
        // from the same untrusted blob (oxidezap #594).
        mac.verify_slice(hmac_bytes).map_err(|_| PairCryptoError {
            code: 401,
            text: "hmac-mismatch",
            source: anyhow::anyhow!("ADV signed-device-identity HMAC verification failed"),
        })?;

        // 2. Unmarshal inner container and verify account signature
        let mut signed_identity = wa::ADVSignedDeviceIdentity::decode_from_slice(details_bytes)
            .map_err(|e| PairCryptoError {
                code: 500,
                text: "internal-error",
                source: e.into(),
            })?;

        let account_sig_key_bytes = signed_identity
            .account_signature_key
            .as_deref()
            .unwrap_or_default();
        let account_sig_bytes = signed_identity
            .account_signature
            .as_deref()
            .unwrap_or_default();
        let inner_details_bytes = signed_identity
            .details
            .as_deref()
            .ok_or_else(|| PairCryptoError {
                code: 500,
                text: "internal-error",
                source: anyhow::anyhow!("ADVSignedDeviceIdentity missing details"),
            })?
            .to_vec();

        // The signed inner ADVDeviceIdentity drives the account-signature
        // prefix: WA Web keys it off the signed deviceType, independently of
        // the outer (HMAC container) accountType. Mixed hosted-account /
        // regular-device chains exist and must verify with the prefixes of
        // their own fields (oxidezap #1440).
        let identity_details = wa::ADVDeviceIdentity::decode_from_slice(&inner_details_bytes)
            .map_err(|e| PairCryptoError {
                code: 500,
                text: "internal-error",
                source: e.into(),
            })?;
        let account_sig_prefix = if identity_details.device_type == Some(ADVEncryptionType::HOSTED)
        {
            ADV_HOSTED_PREFIX_ACCOUNT_SIGNATURE
        } else {
            ADV_PREFIX_ACCOUNT_SIGNATURE
        };

        let msg_to_verify = Self::concat_bytes(&[
            account_sig_prefix,
            &inner_details_bytes,
            device_state.identity_key.public_key.public_key_bytes(),
        ]);

        let account_public_key = PublicKey::from_djb_public_key_bytes(account_sig_key_bytes)
            .map_err(|e| PairCryptoError {
                code: 401,
                text: "invalid-key",
                source: e.into(),
            })?;

        if !account_public_key.verify_signature(&msg_to_verify, account_sig_bytes) {
            return Err(PairCryptoError {
                code: 401,
                text: "signature-mismatch",
                source: anyhow::anyhow!("libsignal signature verification failed"),
            });
        }

        // 3. Generate our device signature.
        // WAWebAdvSignatureApi.generateDeviceSignature always signs a regular
        // companion ([6,1]), never a hosted variant (oxidezap #1440).
        let msg_to_sign = Self::concat_bytes(&[
            ADV_PREFIX_DEVICE_SIGNATURE_GENERATE,
            &inner_details_bytes,
            device_state.identity_key.public_key.public_key_bytes(),
            account_sig_key_bytes,
        ]);
        let device_signature = device_state
            .identity_key
            .private_key
            .calculate_signature(&msg_to_sign, &mut rand::rngs::OsRng.unwrap_err())
            .map_err(|e| PairCryptoError {
                code: 500,
                text: "internal-error",
                source: e.into(),
            })?;
        signed_identity.device_signature = Some(device_signature.to_vec());

        let key_index = identity_details.key_index.ok_or_else(|| PairCryptoError {
            code: 500,
            text: "internal-error",
            source: anyhow::anyhow!("ADVDeviceIdentity missing key_index"),
        })?;

        // 5. Marshal the modified signed_identity to send back
        let self_signed_identity_bytes = signed_identity.encode_to_vec();

        Ok((self_signed_identity_bytes, key_index))
    }

    /// Builds the pair-device-sign response node
    pub fn build_pair_success_response(
        req_id: &str,
        self_signed_identity_bytes: Vec<u8>,
        key_index: u32,
    ) -> Node {
        let response_content = NodeBuilder::new("pair-device-sign")
            .children([NodeBuilder::new("device-identity")
                .attr("key-index", key_index.to_string())
                .bytes(self_signed_identity_bytes)
                .build()])
            .build();
        NodeBuilder::new("iq")
            .attrs([
                ("to", SERVER_JID.to_string()),
                ("id", req_id.to_string()),
                ("type", "result".to_string()),
            ])
            .children([response_content])
            .build()
    }

    /// Parses QR code and extracts crypto keys for pairing
    pub fn parse_qr_code(qr_code: &str) -> Result<(String, [u8; 32], [u8; 32]), anyhow::Error> {
        let parts: Vec<&str> = qr_code.split(',').collect();
        if parts.len() != 4 {
            return Err(anyhow::anyhow!("Invalid QR code format"));
        }
        let pairing_ref = parts[0].to_string();
        let dut_noise_pub_b64 = parts[1];
        let dut_identity_pub_b64 = parts[2];
        // The ADV secret is not used by the phone side.

        let dut_noise_pub_bytes = BASE64_STANDARD
            .decode(dut_noise_pub_b64)
            .map_err(|e| anyhow::anyhow!(e))?;
        let dut_identity_pub_bytes = BASE64_STANDARD
            .decode(dut_identity_pub_b64)
            .map_err(|e| anyhow::anyhow!(e))?;

        let dut_noise_pub: [u8; 32] = dut_noise_pub_bytes
            .try_into()
            .map_err(|_| anyhow::anyhow!("Invalid noise public key length"))?;
        let dut_identity_pub: [u8; 32] = dut_identity_pub_bytes
            .try_into()
            .map_err(|_| anyhow::anyhow!("Invalid identity public key length"))?;

        Ok((pairing_ref, dut_noise_pub, dut_identity_pub))
    }

    /// Prepares pairing message for master device (phone simulation)
    pub fn prepare_master_pairing_message(
        device_state: &DeviceState,
        pairing_ref: &str,
        dut_noise_pub: &[u8; 32],
        dut_identity_pub: &[u8; 32],
        master_ephemeral: KeyPair,
    ) -> Result<Vec<u8>, anyhow::Error> {
        // Perform the cryptographic exchange to create the shared secrets
        let adv_key = &device_state.adv_secret_key;
        let identity_key = &device_state.identity_key;

        let mut mac = <HmacSha256 as Mac>::new_from_slice(adv_key)
            .map_err(|e| anyhow::anyhow!("Failed to init HMAC for master pairing: {e}"))?;
        mac.update(ADV_PREFIX_ACCOUNT_SIGNATURE);
        mac.update(dut_identity_pub);
        mac.update(master_ephemeral.public_key.public_key_bytes());
        let account_signature = mac.finalize().into_bytes();

        let their_public_key = PublicKey::from_djb_public_key_bytes(dut_noise_pub)?;
        let shared_secret = master_ephemeral
            .private_key
            .calculate_agreement(&their_public_key)?;

        let mut final_message = Vec::new();
        final_message.extend_from_slice(&account_signature);
        final_message.extend_from_slice(master_ephemeral.public_key.public_key_bytes());
        final_message.extend_from_slice(identity_key.public_key.public_key_bytes());

        // Encrypt the final message
        let encryption_key = {
            let hk = Hkdf::<Sha256>::new(None, &shared_secret);
            let mut result = vec![0u8; 32];
            hk.expand(b"WA-Ads-Key", &mut result)
                .map_err(|_| anyhow::anyhow!("HKDF expand failed"))?;
            result
        };
        let cipher = Aes256Gcm::new_from_slice(&encryption_key)
            .map_err(|_| anyhow::anyhow!("Invalid key size for AES-GCM"))?;
        #[allow(deprecated)]
        let nonce = aes_gcm::Nonce::from_slice(&[0; 12]);
        let payload = Payload {
            msg: &final_message,
            aad: pairing_ref.as_bytes(),
        };
        let encrypted = cipher
            .encrypt(nonce, payload)
            .map_err(|_| anyhow::anyhow!("AES-GCM encryption failed"))?;

        Ok(encrypted)
    }

    /// Builds pairing IQ for master device
    pub fn build_master_pair_iq(
        master_jid: &Jid,
        encrypted_message: Vec<u8>,
        req_id: String,
    ) -> Node {
        let response_content = NodeBuilder::new("pair-device-sign")
            .attr("jid", master_jid.to_string())
            .bytes(encrypted_message)
            .build();
        NodeBuilder::new("iq")
            .attrs([
                ("to", SERVER_JID.to_string()),
                ("type", "set".to_string()),
                ("id", req_id),
                ("xmlns", "md".to_string()),
            ])
            .children([response_content])
            .build()
    }

    /// Helper to concatenate multiple byte slices into a single Vec.
    fn concat_bytes(slices: &[&[u8]]) -> Vec<u8> {
        slices.iter().flat_map(|s| s.iter().cloned()).collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn dummy_device_state() -> DeviceState {
        use rand::TryRngCore;
        let mut rng = rand::rngs::OsRng.unwrap_err();
        DeviceState {
            identity_key: KeyPair::generate(&mut rng),
            noise_key: KeyPair::generate(&mut rng),
            adv_secret_key: [0x42u8; 32],
        }
    }

    #[test]
    fn make_qr_data_has_five_fields_with_client_type_suffix() {
        let state = dummy_device_state();
        let qr = PairUtils::make_qr_data(
            &state,
            "the-ref".to_string(),
            CompanionWebClientType::Chrome,
        );
        let parts: Vec<&str> = qr.split(',').collect();
        assert_eq!(parts.len(), 5, "expected 5 fields, got {qr:?}");
        assert_eq!(parts[0], "the-ref");
        assert_eq!(parts[4], "1", "Chrome wire value must be \"1\"");
    }

    #[test]
    fn make_qr_data_renders_each_client_type_wire_byte() {
        let state = dummy_device_state();
        for (ct, wire) in [
            (CompanionWebClientType::Chrome, "1"),
            (CompanionWebClientType::Edge, "2"),
            (CompanionWebClientType::Firefox, "3"),
            (CompanionWebClientType::Ie, "4"),
            (CompanionWebClientType::Opera, "5"),
            (CompanionWebClientType::Safari, "6"),
            (CompanionWebClientType::Electron, "7"),
            (CompanionWebClientType::Uwp, "8"),
            (CompanionWebClientType::OtherWebClient, "9"),
            (CompanionWebClientType::AndroidTablet, "d"),
            (CompanionWebClientType::AndroidPhone, "e"),
            (CompanionWebClientType::AndroidAmbiguous, "f"),
        ] {
            let qr = PairUtils::make_qr_data(&state, "r".to_string(), ct);
            assert_eq!(qr.rsplit(',').next(), Some(wire), "{ct:?}");
        }
    }

    /// Synthesize a signed pair-success payload whose HMAC is keyed by
    /// `adv_secret_for_hmac`. The outer HMAC follows `account_type` (hosted
    /// or not), the account signature follows the SIGNED `device_type` —
    /// independently, mirroring WA Web (oxidezap #594 + #1440).
    fn build_pair_success_payload(
        state: &DeviceState,
        adv_secret_for_hmac: &[u8; 32],
        is_hosted: bool,
    ) -> Vec<u8> {
        build_pair_success_payload_mixed(state, adv_secret_for_hmac, is_hosted, is_hosted)
    }

    /// Like [`build_pair_success_payload`] but with `account_type` (outer,
    /// HMAC prefix) and `device_type` (signed, account-signature prefix)
    /// chosen independently.
    fn build_pair_success_payload_mixed(
        state: &DeviceState,
        adv_secret_for_hmac: &[u8; 32],
        hosted_account: bool,
        hosted_device: bool,
    ) -> Vec<u8> {
        use rand::TryRngCore;
        let mut rng = rand::rngs::OsRng.unwrap_err();
        let account_kp = KeyPair::generate(&mut rng);
        let outer_account_type = if hosted_account {
            ADVEncryptionType::HOSTED
        } else {
            ADVEncryptionType::E2EE
        };
        let mut inner = wa::ADVDeviceIdentity::default();
        inner.raw_id = Some(1);
        inner.timestamp = Some(0);
        inner.key_index = Some(0);
        inner.account_type = Some(if hosted_account {
            ADVEncryptionType::HOSTED
        } else {
            ADVEncryptionType::E2EE
        });
        inner.device_type = Some(if hosted_device {
            ADVEncryptionType::HOSTED
        } else {
            ADVEncryptionType::E2EE
        });
        let inner = inner.encode_to_vec();
        let account_sig_prefix: &[u8] = if hosted_device {
            ADV_HOSTED_PREFIX_ACCOUNT_SIGNATURE
        } else {
            ADV_PREFIX_ACCOUNT_SIGNATURE
        };
        let mut to_sign = Vec::new();
        to_sign.extend_from_slice(account_sig_prefix);
        to_sign.extend_from_slice(&inner);
        to_sign.extend_from_slice(state.identity_key.public_key.public_key_bytes());
        let sig = account_kp
            .private_key
            .calculate_signature(&to_sign, &mut rng)
            .expect("signing must succeed");
        let mut signed = wa::ADVSignedDeviceIdentity::default();
        signed.details = Some(inner);
        signed.account_signature_key = Some(account_kp.public_key.public_key_bytes().to_vec());
        signed.account_signature = Some(sig.to_vec());
        let signed = signed.encode_to_vec();
        let mut mac = <HmacSha256 as Mac>::new_from_slice(adv_secret_for_hmac)
            .expect("HMAC accepts any key length");
        if hosted_account {
            mac.update(ADV_HOSTED_PREFIX_ACCOUNT_SIGNATURE);
        }
        mac.update(&signed);
        let hmac_bytes = mac.finalize().into_bytes().to_vec();
        let mut container = wa::ADVSignedDeviceIdentityHMAC::default();
        container.details = Some(signed);
        container.hmac = Some(hmac_bytes);
        container.account_type = Some(outer_account_type);
        container.encode_to_vec()
    }

    #[test]
    fn do_pair_crypto_accepts_matching_hmac() {
        let state = dummy_device_state();
        let payload = build_pair_success_payload(&state, &state.adv_secret_key, false);
        PairUtils::do_pair_crypto(&state, &payload).expect("matching HMAC must verify");
    }

    #[test]
    fn do_pair_crypto_rejects_mismatched_hmac() {
        let state = dummy_device_state();
        // A different secret than the companion holds: tampered/forged pair-success.
        let wrong_secret = [0xCDu8; 32];
        let payload = build_pair_success_payload(&state, &wrong_secret, false);
        let err = PairUtils::do_pair_crypto(&state, &payload)
            .expect_err("mismatched HMAC must abort pairing");
        assert_eq!(err.code, 401, "expected 401 unauthorized, got {}", err.code);
        assert_eq!(err.text, "hmac-mismatch");
    }

    #[test]
    fn do_pair_crypto_accepts_matching_hmac_for_hosted_account() {
        let state = dummy_device_state();
        let payload = build_pair_success_payload(&state, &state.adv_secret_key, true);
        PairUtils::do_pair_crypto(&state, &payload)
            .expect("hosted-account HMAC with matching secret must verify");
    }

    #[test]
    fn do_pair_crypto_rejects_mismatched_hmac_for_hosted_account() {
        let state = dummy_device_state();
        let wrong_secret = [0xCDu8; 32];
        let payload = build_pair_success_payload(&state, &wrong_secret, true);
        let err = PairUtils::do_pair_crypto(&state, &payload)
            .expect_err("hosted-account HMAC with wrong secret must abort pairing");
        assert_eq!(err.code, 401, "expected 401 unauthorized, got {}", err.code);
        assert_eq!(err.text, "hmac-mismatch");
    }

    /// Mixed chains must verify: the HMAC prefix follows the outer
    /// account_type, the account-signature prefix follows the signed
    /// device_type, and both fields can disagree (oxidezap #1440).
    #[test]
    fn do_pair_crypto_verifies_hosted_account_with_regular_device_type() {
        let state = dummy_device_state();
        let payload = build_pair_success_payload_mixed(&state, &state.adv_secret_key, true, false);
        PairUtils::do_pair_crypto(&state, &payload)
            .expect("hosted account / regular device chain must verify");
    }

    #[test]
    fn do_pair_crypto_verifies_regular_account_with_hosted_device_type() {
        let state = dummy_device_state();
        let payload = build_pair_success_payload_mixed(&state, &state.adv_secret_key, false, true);
        PairUtils::do_pair_crypto(&state, &payload)
            .expect("regular account / hosted device chain must verify");
    }
}
