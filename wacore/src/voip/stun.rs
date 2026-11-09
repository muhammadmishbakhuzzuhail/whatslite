//! Minimal STUN header parsing.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StunMessage {
    pub message_type: u16,
    pub transaction_id: [u8; 12],
    pub body: Vec<u8>,
}
impl StunMessage {
    pub fn decode(data: &[u8]) -> Option<Self> {
        if data.len() < 20 || data[0] & 0xc0 != 0 {
            return None;
        }
        let message_type = u16::from_be_bytes([data[0], data[1]]);
        let length = u16::from_be_bytes([data[2], data[3]]) as usize;
        if data.len() < 20 + length {
            return None;
        }
        let mut transaction_id = [0u8; 12];
        transaction_id.copy_from_slice(&data[8..20]);
        Some(Self {
            message_type,
            transaction_id,
            body: data[20..20 + length].to_vec(),
        })
    }
}
