//! Key-rotation request surface.

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RotateKey {
    pub rotated: bool,
}
