//! Favorites surface.

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FavoriteSticker {
    pub file_sha256: String,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Favorites;
