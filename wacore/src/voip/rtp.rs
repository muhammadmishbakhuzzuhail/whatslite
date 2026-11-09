//! Minimal RTP v2 packet parsing/building.

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RtpPacket {
    pub payload_type: u8,
    pub sequence: u16,
    pub timestamp: u32,
    pub ssrc: u32,
    pub payload: Vec<u8>,
}

impl RtpPacket {
    pub fn encode(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(12 + self.payload.len());
        out.push(0x80);
        out.push(self.payload_type & 0x7f);
        out.extend_from_slice(&self.sequence.to_be_bytes());
        out.extend_from_slice(&self.timestamp.to_be_bytes());
        out.extend_from_slice(&self.ssrc.to_be_bytes());
        out.extend_from_slice(&self.payload);
        out
    }

    pub fn decode(data: &[u8]) -> Option<Self> {
        if data.len() < 12 || data[0] >> 6 != 2 {
            return None;
        }
        let csrc_count = data[0] & 0x0f;
        let header_len = 12 + usize::from(csrc_count) * 4;
        if data.len() < header_len {
            return None;
        }
        Some(Self {
            payload_type: data[1] & 0x7f,
            sequence: u16::from_be_bytes([data[2], data[3]]),
            timestamp: u32::from_be_bytes([data[4], data[5], data[6], data[7]]),
            ssrc: u32::from_be_bytes([data[8], data[9], data[10], data[11]]),
            payload: data[header_len..].to_vec(),
        })
    }
}
