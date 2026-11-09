//
// Copyright 2021 Signal Messenger, LLC.
// SPDX-License-Identifier: AGPL-3.0-only
//

mod error;
mod hash;

mod aes_cbc;
mod aes_ctr;
mod aes_gcm;

pub use aes_cbc::{
    DecryptionError, EncryptionError, aes_256_cbc_decrypt_into, aes_256_cbc_encrypt_into,
};
pub use aes_ctr::Aes256Ctr32;
pub use aes_gcm::{Aes256GcmDecryption, Aes256GcmEncryption};
pub use error::{Error, Result};
pub use hash::{
    CryptographicHash, CryptographicMac, SHA1_OUTPUT_SIZE, SHA256_OUTPUT_SIZE, SHA512_OUTPUT_SIZE,
};

/// Caller-owned symmetric crypto port.
///
/// Phase 0 needs the runtime seam only; the full upstream method surface is
/// ported with the libsignal crypto task.
pub trait SymmetricCryptoProvider: Send + Sync + 'static {}

/// AES-256-GCM encrypt; appends ciphertext followed by the 16-byte tag.
pub fn aes_256_gcm_encrypt(
    key: &[u8],
    nonce: &[u8],
    aad: &[u8],
    plaintext: &[u8],
    out: &mut Vec<u8>,
) -> Result<()> {
    let mut cipher = Aes256GcmEncryption::new(key, nonce, aad)?;
    let mut buffer = plaintext.to_vec();
    cipher.encrypt(&mut buffer);
    let tag = cipher.compute_tag();
    out.extend_from_slice(&buffer);
    out.extend_from_slice(&tag);
    Ok(())
}

/// AES-256-GCM decrypt; input is ciphertext followed by the 16-byte tag.
pub fn aes_256_gcm_decrypt(
    key: &[u8],
    nonce: &[u8],
    aad: &[u8],
    payload: &[u8],
    out: &mut Vec<u8>,
) -> Result<()> {
    if payload.len() < 16 {
        return Err(Error::InvalidInputSize);
    }
    let (ciphertext, tag) = payload.split_at(payload.len() - 16);
    let mut cipher = Aes256GcmDecryption::new(key, nonce, aad)?;
    let mut buffer = ciphertext.to_vec();
    cipher.decrypt(&mut buffer);
    cipher.verify_tag(tag)?;
    out.extend_from_slice(&buffer);
    Ok(())
}
