//! Group notification action types used by the event model.

/// Group notification action carried by a `GroupUpdate` event.
///
/// Phase 0 carries the variants the event model needs; the full parser lands
/// with the social phase.
#[derive(Debug, Clone, serde::Serialize)]
#[serde(rename_all = "snake_case")]
#[non_exhaustive]
pub enum GroupNotificationAction {
    Unlocked,
    Other(String),
}
