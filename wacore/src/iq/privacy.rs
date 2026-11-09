//! Privacy settings IQ specification.
//!
//! Fetches and sets the user's privacy settings.
//!
//! ## Wire Format
//! ```xml
//! <!-- GET request -->
//! <iq xmlns="privacy" type="get" to="s.whatsapp.net" id="...">
//!   <privacy/>
//! </iq>
//!
//! <!-- GET response -->
//! <iq from="s.whatsapp.net" id="..." type="result">
//!   <privacy>
//!     <category name="last" value="all"/>
//!     <category name="online" value="all"/>
//!     <category name="profile" value="contacts"/>
//!     <category name="status" value="contacts"/>
//!     <category name="groupadd" value="contacts"/>
//!     ...
//!   </privacy>
//! </iq>
//!
//! <!-- SET request (simple) -->
//! <iq xmlns="privacy" type="set" to="s.whatsapp.net" id="...">
//!   <privacy>
//!     <category name="{category}" value="{value}"/>
//!   </privacy>
//! </iq>
//!
//! <!-- SET request (with disallowed list, LID addressing) -->
//! <iq xmlns="privacy" type="set" to="s.whatsapp.net" id="...">
//!   <privacy addressing_mode="lid">
//!     <category name="{category}" value="contact_blacklist" dhash="{hash}">
//!       <user action="add" jid="{lid}" pn_jid="{pn}"/>
//!       <user action="remove" jid="{lid}" pn_jid="{pn}"/>
//!     </category>
//!   </privacy>
//! </iq>
//! ```
//!
//! Verified against WhatsApp Web JS:
//! - `WAWebQueryPrivacySettingsJob` (GET parsing)
//! - `WAWebSetPrivacyJob` (SET building)

use crate::iq::spec::IqSpec;
use crate::request::InfoQuery;
use crate::types::message::AddressingMode;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::{Jid, SERVER_JID};
use wacore_binary::node::{Node, NodeContent};

pub use crate::types::wire_enums::DisallowedListAction;

/// IQ namespace for privacy settings.
pub const PRIVACY_NAMESPACE: &str = "privacy";

/// Privacy setting category name.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PrivacyCategory {
    /// Last seen visibility
    Last,
    /// Online status visibility
    Online,
    /// Profile photo visibility
    Profile,
    /// Status visibility
    Status,
    /// Group add permissions
    GroupAdd,
    /// Read receipts
    ReadReceipts,
    /// Call add permissions
    CallAdd,
    /// Message permissions / anti-brigading
    Messages,
    /// Defense mode
    DefenseMode,
    /// Other/unknown category
    Other(String),
}

impl PrivacyCategory {
    pub fn as_str(&self) -> &str {
        match self {
            PrivacyCategory::Last => "last",
            PrivacyCategory::Online => "online",
            PrivacyCategory::Profile => "profile",
            PrivacyCategory::Status => "status",
            PrivacyCategory::GroupAdd => "groupadd",
            PrivacyCategory::ReadReceipts => "readreceipts",
            PrivacyCategory::CallAdd => "calladd",
            PrivacyCategory::Messages => "messages",
            PrivacyCategory::DefenseMode => "defense",
            PrivacyCategory::Other(s) => s.as_str(),
        }
    }

    /// Check whether a value is valid for this category per `WAWebPrivacySettings`.
    pub fn is_valid_value(&self, value: &PrivacyValue) -> bool {
        match self {
            Self::Last | Self::Profile | Self::Status | Self::GroupAdd => matches!(
                value,
                PrivacyValue::All
                    | PrivacyValue::Contacts
                    | PrivacyValue::ContactBlacklist
                    | PrivacyValue::None
            ),
            Self::ReadReceipts => matches!(value, PrivacyValue::All | PrivacyValue::None),
            Self::Online => matches!(value, PrivacyValue::All | PrivacyValue::MatchLastSeen),
            Self::CallAdd => matches!(
                value,
                PrivacyValue::All | PrivacyValue::Known | PrivacyValue::Contacts
            ),
            Self::Messages => matches!(value, PrivacyValue::All | PrivacyValue::Contacts),
            Self::DefenseMode => matches!(value, PrivacyValue::Off | PrivacyValue::OnStandard),
            // Unknown categories: allow anything
            Self::Other(_) => true,
        }
    }

    /// Whether this category supports disallowed lists (`contact_blacklist` with user list).
    pub fn supports_disallowed_list(&self) -> bool {
        matches!(
            self,
            Self::Last | Self::Profile | Self::Status | Self::GroupAdd
        )
    }
}

impl From<&str> for PrivacyCategory {
    fn from(s: &str) -> Self {
        match s {
            "last" => PrivacyCategory::Last,
            "online" => PrivacyCategory::Online,
            "profile" => PrivacyCategory::Profile,
            "status" => PrivacyCategory::Status,
            "groupadd" => PrivacyCategory::GroupAdd,
            "readreceipts" => PrivacyCategory::ReadReceipts,
            "calladd" => PrivacyCategory::CallAdd,
            "messages" => PrivacyCategory::Messages,
            "defense" => PrivacyCategory::DefenseMode,
            other => PrivacyCategory::Other(other.to_string()),
        }
    }
}

/// Privacy setting value.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PrivacyValue {
    /// Visible to everyone
    All,
    /// Visible only to contacts
    Contacts,
    /// Not visible to anyone
    None,
    /// Visible to contacts except specific list
    ContactBlacklist,
    /// Match their settings (for online/last)
    MatchLastSeen,
    /// `calladd` only
    Known,
    /// `defense` only
    Off,
    /// `defense` only
    OnStandard,
    /// Other/unknown value
    Other(String),
}

impl PrivacyValue {
    pub fn as_str(&self) -> &str {
        match self {
            PrivacyValue::All => "all",
            PrivacyValue::Contacts => "contacts",
            PrivacyValue::None => "none",
            PrivacyValue::ContactBlacklist => "contact_blacklist",
            PrivacyValue::MatchLastSeen => "match_last_seen",
            PrivacyValue::Known => "known",
            PrivacyValue::Off => "off",
            PrivacyValue::OnStandard => "on_standard",
            PrivacyValue::Other(s) => s.as_str(),
        }
    }
}

impl From<&str> for PrivacyValue {
    fn from(s: &str) -> Self {
        match s {
            "all" => PrivacyValue::All,
            "contacts" => PrivacyValue::Contacts,
            "none" => PrivacyValue::None,
            "contact_blacklist" => PrivacyValue::ContactBlacklist,
            "match_last_seen" => PrivacyValue::MatchLastSeen,
            "known" => PrivacyValue::Known,
            "off" => PrivacyValue::Off,
            "on_standard" => PrivacyValue::OnStandard,
            other => PrivacyValue::Other(other.to_string()),
        }
    }
}

/// A single privacy setting.
#[derive(Debug, Clone)]
pub struct PrivacySetting {
    /// The category name (e.g., "last", "profile", etc.)
    pub category: PrivacyCategory,
    /// The privacy value (e.g., "all", "contacts", "none")
    pub value: PrivacyValue,
}

/// Response from privacy settings query.
#[derive(Debug, Clone, Default)]
pub struct PrivacySettingsResponse {
    /// The list of privacy settings.
    pub settings: Vec<PrivacySetting>,
}

impl PrivacySettingsResponse {
    /// Get a privacy setting by category.
    pub fn get(&self, category: &PrivacyCategory) -> Option<&PrivacySetting> {
        self.settings.iter().find(|s| &s.category == category)
    }

    /// Get the value for a category.
    pub fn get_value(&self, category: &PrivacyCategory) -> Option<&PrivacyValue> {
        self.get(category).map(|s| &s.value)
    }
}

/// Fetches privacy settings from the server.
#[derive(Debug, Clone, Default)]
pub struct PrivacySettingsSpec;

impl PrivacySettingsSpec {
    /// Create a new privacy settings spec.
    pub fn new() -> Self {
        Self
    }
}

impl IqSpec for PrivacySettingsSpec {
    type Response = PrivacySettingsResponse;

    fn build_iq(&self) -> InfoQuery<'static> {
        InfoQuery::get(
            PRIVACY_NAMESPACE,
            Jid::new("", SERVER_JID),
            Some(NodeContent::Nodes(vec![
                NodeBuilder::new("privacy").build(),
            ])),
        )
    }

    fn parse_response(&self, response: &Node) -> Result<Self::Response, anyhow::Error> {
        use crate::iq::node::{optional_attr, required_child};

        let privacy_node = required_child(response, "privacy")?;

        let mut settings = Vec::new();
        for child in privacy_node.get_children_by_tag("category") {
            let name = optional_attr(child, "name")
                .ok_or_else(|| anyhow::anyhow!("missing name in category"))?;
            let value = optional_attr(child, "value")
                .ok_or_else(|| anyhow::anyhow!("missing value in category"))?;

            settings.push(PrivacySetting {
                category: PrivacyCategory::from(name),
                value: PrivacyValue::from(value),
            });
        }

        Ok(PrivacySettingsResponse { settings })
    }
}

/// A single entry in a `contact_blacklist` disallowed list.
#[derive(Debug, Clone)]
pub struct DisallowedListUserEntry {
    pub action: DisallowedListAction,
    pub jid: Jid,
    pub pn_jid: Option<Jid>,
}

/// Update for the `contact_blacklist` disallowed list (user exclusions).
#[derive(Debug, Clone)]
pub struct DisallowedListUpdate {
    /// Hash of the current disallowed list for conflict detection (409 handling).
    /// The server returns the updated dhash in the response.
    pub dhash: String,
    /// User additions/removals.
    pub users: Vec<DisallowedListUserEntry>,
}

/// Response from setting a privacy category.
/// Mirrors `WAWebSetPrivacyJob`'s setPrivacyParser, which extracts `{name, value, dhash}`.
#[derive(Debug, Clone, Default)]
pub struct SetPrivacySettingResponse {
    pub dhash: Option<String>,
}

/// Sets a single privacy category.
#[derive(Debug, Clone)]
pub struct SetPrivacySettingSpec {
    pub category: PrivacyCategory,
    pub value: PrivacyValue,
    pub disallowed_list: Option<DisallowedListUpdate>,
}

impl SetPrivacySettingSpec {
    pub fn new(category: PrivacyCategory, value: PrivacyValue) -> Self {
        debug_assert!(
            category.is_valid_value(&value),
            "{:?} does not accept {:?}",
            category,
            value
        );
        Self {
            category,
            value,
            disallowed_list: None,
        }
    }

    /// Automatically sets the value to `ContactBlacklist`.
    pub fn with_disallowed_list(category: PrivacyCategory, update: DisallowedListUpdate) -> Self {
        debug_assert!(
            category.supports_disallowed_list(),
            "{:?} does not support disallowed lists",
            category
        );
        Self {
            category,
            value: PrivacyValue::ContactBlacklist,
            disallowed_list: Some(update),
        }
    }
}

impl IqSpec for SetPrivacySettingSpec {
    type Response = SetPrivacySettingResponse;

    fn build_iq(&self) -> InfoQuery<'static> {
        let mut category_node = NodeBuilder::new("category")
            .attr("name", self.category.as_str())
            .attr("value", self.value.as_str());

        let mut privacy_node = NodeBuilder::new("privacy");

        if let Some(ref update) = self.disallowed_list {
            category_node = category_node.attr("dhash", update.dhash.clone());

            let user_nodes: Vec<Node> = update
                .users
                .iter()
                .map(|entry| {
                    let mut user = NodeBuilder::new("user")
                        .attr("action", entry.action.as_str())
                        .jid_attr("jid", entry.jid.clone());
                    if let Some(ref pn) = entry.pn_jid {
                        user = user.jid_attr("pn_jid", pn.clone());
                    }
                    user.build()
                })
                .collect();

            category_node = category_node.children(user_nodes);
            privacy_node = privacy_node.attr("addressing_mode", AddressingMode::Lid.as_str());
        }

        InfoQuery::set(
            PRIVACY_NAMESPACE,
            Jid::new("", SERVER_JID),
            Some(NodeContent::Nodes(vec![
                privacy_node.children([category_node.build()]).build(),
            ])),
        )
    }

    /// WA Web's `setPrivacyParser` extracts `{name, value, dhash}` per category;
    /// only the dhash matters here, for disallowed-list conflict resolution.
    fn parse_response(&self, response: &Node) -> Result<Self::Response, anyhow::Error> {
        use crate::iq::node::optional_attr;

        let dhash = response.get_optional_child("privacy").and_then(|privacy| {
            privacy
                .get_children_by_tag("category")
                .next()
                .and_then(|cat| optional_attr(cat, "dhash").map(str::to_string))
        });

        Ok(SetPrivacySettingResponse { dhash })
    }
}

/// Set the default disappearing messages duration.
///
/// ```xml
/// <iq xmlns="disappearing_mode" type="set" to="s.whatsapp.net">
///   <disappearing_mode duration="{seconds}"/>
/// </iq>
/// ```
#[derive(Debug, Clone)]
pub struct SetDefaultDisappearingModeSpec {
    pub duration: u32,
}

impl SetDefaultDisappearingModeSpec {
    pub fn new(duration: u32) -> Self {
        Self { duration }
    }
}

impl IqSpec for SetDefaultDisappearingModeSpec {
    type Response = ();

    fn build_iq(&self) -> InfoQuery<'static> {
        InfoQuery::set(
            "disappearing_mode",
            Jid::new("", SERVER_JID),
            Some(NodeContent::Nodes(vec![
                NodeBuilder::new("disappearing_mode")
                    .attr("duration", self.duration.to_string())
                    .build(),
            ])),
        )
    }

    fn parse_response(&self, _response: &Node) -> Result<Self::Response, anyhow::Error> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_privacy_settings_spec_build_iq() {
        let spec = PrivacySettingsSpec::new();
        let iq = spec.build_iq();

        assert_eq!(iq.namespace, PRIVACY_NAMESPACE);
        assert_eq!(iq.query_type, crate::request::InfoQueryType::Get);

        if let Some(NodeContent::Nodes(nodes)) = &iq.content {
            assert_eq!(nodes.len(), 1);
            assert_eq!(nodes[0].tag, "privacy");
        } else {
            panic!("Expected NodeContent::Nodes");
        }
    }

    #[test]
    fn test_privacy_settings_spec_parse_response() {
        let spec = PrivacySettingsSpec::new();
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .children([NodeBuilder::new("privacy")
                .children([
                    NodeBuilder::new("category")
                        .attr("name", "last")
                        .attr("value", "all")
                        .build(),
                    NodeBuilder::new("category")
                        .attr("name", "profile")
                        .attr("value", "contacts")
                        .build(),
                    NodeBuilder::new("category")
                        .attr("name", "status")
                        .attr("value", "none")
                        .build(),
                ])
                .build()])
            .build();

        let result = spec.parse_response(&response).unwrap();
        assert_eq!(result.settings.len(), 3);

        assert_eq!(result.settings[0].category, PrivacyCategory::Last);
        assert_eq!(result.settings[0].value, PrivacyValue::All);

        assert_eq!(result.settings[1].category, PrivacyCategory::Profile);
        assert_eq!(result.settings[1].value, PrivacyValue::Contacts);

        assert_eq!(result.settings[2].category, PrivacyCategory::Status);
        assert_eq!(result.settings[2].value, PrivacyValue::None);
    }

    #[test]
    fn test_privacy_settings_response_get() {
        let response = PrivacySettingsResponse {
            settings: vec![
                PrivacySetting {
                    category: PrivacyCategory::Last,
                    value: PrivacyValue::All,
                },
                PrivacySetting {
                    category: PrivacyCategory::Profile,
                    value: PrivacyValue::Contacts,
                },
            ],
        };

        assert_eq!(
            response.get_value(&PrivacyCategory::Last),
            Some(&PrivacyValue::All)
        );
        assert_eq!(
            response.get_value(&PrivacyCategory::Profile),
            Some(&PrivacyValue::Contacts)
        );
        assert_eq!(response.get_value(&PrivacyCategory::Online), None);
    }

    #[test]
    fn test_privacy_category_from_str() {
        assert_eq!(PrivacyCategory::from("last"), PrivacyCategory::Last);
        assert_eq!(PrivacyCategory::from("online"), PrivacyCategory::Online);
        assert_eq!(
            PrivacyCategory::from("unknown"),
            PrivacyCategory::Other("unknown".to_string())
        );
    }

    #[test]
    fn test_privacy_value_from_str() {
        assert_eq!(PrivacyValue::from("all"), PrivacyValue::All);
        assert_eq!(PrivacyValue::from("contacts"), PrivacyValue::Contacts);
        assert_eq!(PrivacyValue::from("none"), PrivacyValue::None);
        assert_eq!(
            PrivacyValue::from("unknown"),
            PrivacyValue::Other("unknown".to_string())
        );
    }

    #[test]
    fn set_privacy_builds_simple_category_node() {
        let spec = SetPrivacySettingSpec::new(PrivacyCategory::Last, PrivacyValue::Contacts);
        let iq = spec.build_iq();

        assert_eq!(iq.namespace, PRIVACY_NAMESPACE);
        assert_eq!(iq.query_type, crate::request::InfoQueryType::Set);
        let Some(NodeContent::Nodes(nodes)) = &iq.content else {
            panic!("Expected NodeContent::Nodes");
        };
        assert_eq!(nodes.len(), 1);
        assert_eq!(nodes[0].tag, "privacy");
        assert!(nodes[0].attrs.get("addressing_mode").is_none());

        let categories: Vec<&Node> = nodes[0].get_children_by_tag("category").collect();
        assert_eq!(categories.len(), 1);
        assert_eq!(
            categories[0].attrs.get("name").and_then(|v| v.as_str()),
            Some("last")
        );
        assert_eq!(
            categories[0].attrs.get("value").and_then(|v| v.as_str()),
            Some("contacts")
        );
        assert!(categories[0].attrs.get("dhash").is_none());
    }

    #[test]
    fn set_privacy_disallowed_list_uses_lid_addressing_and_users() {
        let lid: Jid = "100000000000001@lid".parse().unwrap();
        let pn: Jid = "15551234567@s.whatsapp.net".parse().unwrap();
        let update = DisallowedListUpdate {
            dhash: "hash".to_string(),
            users: vec![DisallowedListUserEntry {
                action: DisallowedListAction::Add,
                jid: lid,
                pn_jid: Some(pn),
            }],
        };
        let spec = SetPrivacySettingSpec::with_disallowed_list(PrivacyCategory::Profile, update);
        let iq = spec.build_iq();

        let Some(NodeContent::Nodes(nodes)) = &iq.content else {
            panic!("Expected NodeContent::Nodes");
        };
        let privacy = &nodes[0];
        assert_eq!(
            privacy
                .attrs
                .get("addressing_mode")
                .and_then(|v| v.as_str()),
            Some("lid")
        );

        let category = privacy
            .get_children_by_tag("category")
            .next()
            .expect("category");
        assert_eq!(
            category.attrs.get("value").and_then(|v| v.as_str()),
            Some("contact_blacklist")
        );
        assert_eq!(
            category.attrs.get("dhash").and_then(|v| v.as_str()),
            Some("hash")
        );

        let user = category.get_children_by_tag("user").next().expect("user");
        assert_eq!(
            user.attrs.get("action").and_then(|v| v.as_str()),
            Some("add")
        );
        assert_eq!(
            user.attrs.get("jid").map(|v| v.to_string_value()),
            Some("100000000000001@lid".to_string())
        );
        assert_eq!(
            user.attrs.get("pn_jid").map(|v| v.to_string_value()),
            Some("15551234567@s.whatsapp.net".to_string())
        );
    }

    #[test]
    fn set_privacy_parses_dhash_from_response() {
        let spec = SetPrivacySettingSpec::new(PrivacyCategory::Last, PrivacyValue::All);
        let response = NodeBuilder::new("iq")
            .children([NodeBuilder::new("privacy")
                .children([NodeBuilder::new("category")
                    .attr("name", "last")
                    .attr("value", "all")
                    .attr("dhash", "server-hash")
                    .build()])
                .build()])
            .build();

        let parsed = spec.parse_response(&response).unwrap();
        assert_eq!(parsed.dhash.as_deref(), Some("server-hash"));
    }

    #[test]
    fn default_disappearing_mode_builds_duration_node() {
        let spec = SetDefaultDisappearingModeSpec::new(86_400);
        let iq = spec.build_iq();

        assert_eq!(iq.namespace, "disappearing_mode");
        assert_eq!(iq.query_type, crate::request::InfoQueryType::Set);
        let Some(NodeContent::Nodes(nodes)) = &iq.content else {
            panic!("Expected NodeContent::Nodes");
        };
        assert_eq!(nodes[0].tag, "disappearing_mode");
        assert_eq!(
            nodes[0].attrs.get("duration").and_then(|v| v.as_str()),
            Some("86400")
        );
    }

    #[test]
    fn privacy_category_validity_matches_wa_web() {
        assert!(PrivacyCategory::Last.is_valid_value(&PrivacyValue::ContactBlacklist));
        assert!(PrivacyCategory::CallAdd.is_valid_value(&PrivacyValue::Known));
        assert!(!PrivacyCategory::ReadReceipts.is_valid_value(&PrivacyValue::Contacts));
        assert!(PrivacyCategory::DefenseMode.is_valid_value(&PrivacyValue::OnStandard));
        assert!(PrivacyCategory::Last.supports_disallowed_list());
        assert!(!PrivacyCategory::Online.supports_disallowed_list());
    }
}
