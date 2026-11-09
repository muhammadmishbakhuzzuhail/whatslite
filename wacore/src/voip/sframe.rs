//! SFrame-style authenticated frame envelope. Uses AES-GCM with a per-frame
//! nonce derived from the key id and frame counter.
use aes_gcm::aead::{Aead, KeyInit, Payload};
use aes_gcm::{Aes256Gcm, Nonce};

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum SFrameError {
    #[error("invalid key length")]
    InvalidKey,
    #[error("authentication failed")]
    Authentication,
}

pub struct SFrame {
    key_id: u32,
    cipher: Aes256Gcm,
}

impl SFrame {
    pub fn new(key_id: u32, key: &[u8; 32]) -> Self {
        Self {
            key_id,
            cipher: Aes256Gcm::new(key.into()),
        }
    }

    pub fn key_id(&self) -> u32 {
        self.key_id
    }

    pub fn encrypt(&self, counter: u64, plaintext: &[u8]) -> Result<Vec<u8>, SFrameError> {
        let mut nonce = [0u8; 12];
        nonce[..8].copy_from_slice(&counter.to_be_bytes());
        nonce[8..].copy_from_slice(&self.key_id.to_be_bytes());
        let aad = self.key_id.to_be_bytes();
        self.cipher
            .encrypt(
                Nonce::from_slice(&nonce),
                Payload {
                    msg: plaintext,
                    aad: &aad,
                },
            )
            .map_err(|_| SFrameError::Authentication)
    }

    pub fn decrypt(&self, counter: u64, ciphertext: &[u8]) -> Result<Vec<u8>, SFrameError> {
        let mut nonce = [0u8; 12];
        nonce[..8].copy_from_slice(&counter.to_be_bytes());
        nonce[8..].copy_from_slice(&self.key_id.to_be_bytes());
        let aad = self.key_id.to_be_bytes();
        self.cipher
            .decrypt(
                Nonce::from_slice(&nonce),
                Payload {
                    msg: ciphertext,
                    aad: &aad,
                },
            )
            .map_err(|_| SFrameError::Authentication)
    }
}
