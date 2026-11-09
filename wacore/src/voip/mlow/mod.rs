//! MLOW/Opus packet framing seam.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AudioConfig {
    pub sample_rate: u32,
    pub channels: u8,
    pub frame_ms: u16,
}

impl Default for AudioConfig {
    fn default() -> Self {
        Self {
            sample_rate: 48_000,
            channels: 1,
            frame_ms: 20,
        }
    }
}

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum OpusMlowPacketError {
    #[error("empty opus payload")]
    Empty,
    #[error("opus payload too large")]
    TooLarge,
    #[error("truncated mlow packet")]
    Truncated,
}

/// Frame an Opus payload for MLOW with a big-endian u16 length prefix.
pub fn packetize_opus_for_mlow(payload: &[u8]) -> Result<Vec<u8>, OpusMlowPacketError> {
    if payload.is_empty() {
        return Err(OpusMlowPacketError::Empty);
    }
    if payload.len() > u16::MAX as usize {
        return Err(OpusMlowPacketError::TooLarge);
    }
    let mut out = Vec::with_capacity(2 + payload.len());
    out.extend_from_slice(&(payload.len() as u16).to_be_bytes());
    out.extend_from_slice(payload);
    Ok(out)
}

/// Reverse [`packetize_opus_for_mlow`].
pub fn depacketize_opus_from_mlow(data: &[u8]) -> Result<Vec<u8>, OpusMlowPacketError> {
    if data.len() < 2 {
        return Err(OpusMlowPacketError::Truncated);
    }
    let len = u16::from_be_bytes([data[0], data[1]]) as usize;
    if len == 0 {
        return Err(OpusMlowPacketError::Empty);
    }
    if data.len() < 2 + len {
        return Err(OpusMlowPacketError::Truncated);
    }
    Ok(data[2..2 + len].to_vec())
}
