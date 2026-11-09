//! App-state settings surface.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AppStateSettings {
    pub settings: Vec<(String, String)>,
}
