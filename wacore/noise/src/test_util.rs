//! Test fixtures shared between this crate's unit tests and downstream
//! integration tests. Visible only under `#[cfg(test)]` (this crate) or
//! when the `test-util` feature is enabled.

use waproto::buffa::Message as _;
use waproto::whatsapp::{self as wa, cert_chain::noise_certificate};

/// Builds a minimal `CertChain` blob whose leaf.key matches `server_static_pub`.
///
/// The validity windows are pinned (`not_before = 1_700_000_000` for both
/// certs, `not_after` slightly under `1_900_000_000`) so callers can exercise
/// `select_pattern`'s clock checks against deterministic boundaries.
///
/// Signatures are zero-filled — the client today does NOT verify the
/// intermediate's Ed25519 signature against `WA_CERT_PUB_KEY`, so the bytes
/// only need to round-trip through prost.
pub fn build_cert_chain_bytes(server_static_pub: &[u8; 32]) -> Vec<u8> {
    let mut intermediate_details = noise_certificate::Details::default();
    intermediate_details.serial = Some(1);
    intermediate_details.issuer_serial = Some(0);
    intermediate_details.key = Some(vec![0xCC; 32]);
    intermediate_details.not_before = Some(1_700_000_000);
    intermediate_details.not_after = Some(1_900_000_000);
    let mut intermediate_details_bytes = Vec::new();
    intermediate_details.encode(&mut intermediate_details_bytes);

    let mut leaf_details = noise_certificate::Details::default();
    leaf_details.serial = Some(2);
    leaf_details.issuer_serial = Some(1);
    leaf_details.key = Some(server_static_pub.to_vec());
    leaf_details.not_before = Some(1_700_000_500);
    leaf_details.not_after = Some(1_899_999_500);
    let mut leaf_details_bytes = Vec::new();
    leaf_details.encode(&mut leaf_details_bytes);

    let mut leaf = wa::cert_chain::NoiseCertificate::default();
    leaf.details = Some(leaf_details_bytes);
    leaf.signature = Some(vec![0u8; 64]);
    let mut intermediate = wa::cert_chain::NoiseCertificate::default();
    intermediate.details = Some(intermediate_details_bytes);
    intermediate.signature = Some(vec![0u8; 64]);

    let mut chain = wa::CertChain::default();
    chain.leaf = leaf.into();
    chain.intermediate = intermediate.into();
    let mut bytes = Vec::new();
    chain.encode(&mut bytes);
    bytes
}
