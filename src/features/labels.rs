//! Label and sublist surface.

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LabelSublist {
    pub id: String,
    pub name: String,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Labels;
