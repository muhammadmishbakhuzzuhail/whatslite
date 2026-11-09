//! App-state resync surface.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AppStateResyncMode {
    Normal,
    Full,
}
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AppStateResyncReport {
    pub collections: usize,
}
