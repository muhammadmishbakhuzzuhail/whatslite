//! XOR parity FEC.
pub fn xor_parity(parts: &[&[u8]]) -> Vec<u8> {
    let len = parts.iter().map(|p| p.len()).max().unwrap_or(0);
    let mut parity = vec![0u8; len];
    for part in parts {
        for (i, byte) in part.iter().enumerate() {
            parity[i] ^= byte;
        }
    }
    parity
}
pub fn recover_missing(parts: &[&[u8]], parity: &[u8], missing_index: usize) -> Vec<u8> {
    let len = parity.len();
    let mut recovered = parity.to_vec();
    for (index, part) in parts.iter().enumerate() {
        if index == missing_index {
            continue;
        }
        for (i, byte) in part.iter().enumerate().take(len) {
            recovered[i] ^= byte;
        }
    }
    recovered
}
