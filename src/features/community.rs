//! Community surface.

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum CommunityError {
    #[error("community operation failed: {0}")]
    Failed(String),
}

#[derive(Debug, Clone, Default)]
pub struct CreateCommunityOptions {
    pub name: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CreateCommunityResult {
    pub jid: String,
}

#[derive(Debug, Clone, Default)]
pub struct CreateSubgroupOptions {
    pub name: String,
}

#[derive(Debug, Clone, Default)]
pub struct LinkSubgroupOptions {
    pub subgroup: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CommunitySubgroup {
    pub jid: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SubgroupFailure {
    pub jid: String,
    pub error: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SubgroupVisibility {
    Visible,
    Hidden,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Community<'a> {
    pub _marker: std::marker::PhantomData<&'a ()>,
}
