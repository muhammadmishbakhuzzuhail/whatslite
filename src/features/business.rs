//! Business/catalog surface.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Catalog {
    pub id: String,
}
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Collection {
    pub id: String,
}
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Order {
    pub id: String,
}
#[derive(Debug, Clone, Copy, Default)]
pub struct Business;
