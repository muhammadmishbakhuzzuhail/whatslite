//! Minimal RTCP packet header.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RtcpPacket {
    pub packet_type: u8,
    pub payload: Vec<u8>,
}
impl RtcpPacket {
    pub fn encode(&self) -> Vec<u8> {
        let mut out = vec![0x80, self.packet_type];
        let len = (self.payload.len() / 4) as u16;
        out.extend_from_slice(&len.to_be_bytes());
        out.extend_from_slice(&self.payload);
        out
    }
    pub fn decode(data: &[u8]) -> Option<Self> {
        if data.len() < 4 || data[0] >> 6 != 2 {
            return None;
        }
        let words = u16::from_be_bytes([data[2], data[3]]) as usize;
        let end = 4 + words * 4;
        if data.len() < end {
            return None;
        }
        Some(Self {
            packet_type: data[1],
            payload: data[4..end].to_vec(),
        })
    }
}
