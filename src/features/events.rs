//! Events surface.

pub use crate::features::creation::{CreatedEvent, EventRef};

#[derive(Debug, Clone, Default)]
pub struct EventCreationParams {
    pub name: String,
    pub start_time: i64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EventResponseType {
    Going,
    Maybe,
    NotGoing,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Events;
