//! Minimal SRTP protection context (Phase 4 seam).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SrtpContext {
    pub ssrc: u32,
    pub key: [u8; 32],
}
impl SrtpContext {
    pub fn new(ssrc: u32, key: [u8; 32]) -> Self {
        Self { ssrc, key }
    }
    pub fn protect(&self, payload: &[u8]) -> Vec<u8> {
        let mut out = payload.to_vec();
        for (i, byte) in out.iter_mut().enumerate() {
            *byte ^= self.key[i % self.key.len()];
        }
        out
    }
    pub fn unprotect(&self, payload: &[u8]) -> Vec<u8> {
        self.protect(payload)
    }
}
