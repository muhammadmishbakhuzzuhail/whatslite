//! Patch list parsing (snapshot + patches) - partial port of Go appstate/decode.go

use anyhow::{Result, anyhow};
use std::str::FromStr;
use wacore_binary::node::Node;
use waproto::whatsapp as wa;

/// Why a collection sync request failed.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CollectionSyncError {
    Conflict { has_more: bool },
    Fatal { code: u16, text: String },
    Retry { code: u16, text: String },
}

impl std::fmt::Display for CollectionSyncError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Conflict { has_more } => write!(f, "conflict (has_more={has_more})"),
            Self::Fatal { code, text } => write!(f, "fatal error {code}: {text}"),
            Self::Retry { code, text } => write!(f, "retryable error {code}: {text}"),
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WAPatchName {
    CriticalBlock,
    CriticalUnblockLow,
    RegularLow,
    RegularHigh,
    Regular,
    Unknown,
}

impl WAPatchName {
    pub fn as_str(&self) -> &'static str {
        match self {
            Self::CriticalBlock => "critical_block",
            Self::CriticalUnblockLow => "critical_unblock_low",
            Self::RegularLow => "regular_low",
            Self::RegularHigh => "regular_high",
            Self::Regular => "regular",
            Self::Unknown => "unknown",
        }
    }
}

impl FromStr for WAPatchName {
    type Err = ();
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        Ok(match s {
            "critical_block" => Self::CriticalBlock,
            "critical_unblock_low" => Self::CriticalUnblockLow,
            "regular_low" => Self::RegularLow,
            "regular_high" => Self::RegularHigh,
            "regular" => Self::Regular,
            _ => Self::Unknown,
        })
    }
}

#[derive(Debug, Clone)]
pub struct PatchList {
    pub name: WAPatchName,
    pub has_more_patches: bool,
    pub patches: Vec<wa::SyncdPatch>,
    pub snapshot: Option<wa::SyncdSnapshot>, // filled only if already present inline (currently never)
    pub snapshot_ref: Option<wa::ExternalBlobReference>, // external reference to fetch
    /// Per-collection error from the server (None = success).
    pub error: Option<CollectionSyncError>,
}

/// Parse an incoming app state collection node into a PatchList.
/// Node path: sync -> collection (attributes: name, has_more_patches)
pub fn parse_patch_list(node: &Node) -> Result<PatchList> {
    let collection = node
        .get_optional_child_by_tag(&["sync", "collection"]) // naive path descent
        .ok_or_else(|| anyhow!("missing sync/collection"))?;
    let mut ag = collection.attrs();
    let name_str = ag
        .optional_string("name")
        .ok_or_else(|| anyhow!("collection missing 'name' attribute"))?
        .to_string();
    let has_more = ag.optional_bool("has_more_patches");

    // Check for per-collection error (WA Web: `3JJWKHeu5-P.js:54222-54254`)
    let col_type = ag.optional_string("type");
    let error = parse_collection_error(collection, col_type.as_deref());

    ag.finish()?; // propagate attr parse errors

    // snapshot (optional)
    let mut snapshot_ref = None;
    // A `<snapshot>` we cannot read is not the same as no snapshot. Dropping it
    // silently made the collection look like a bootstrap the server answered
    // with nothing, which is recorded as a synced empty collection -- after
    // which it never asks for the snapshot again and a non-genesis patch over
    // its empty ltHash is refused for good.
    let mut unreadable_snapshot = false;
    if let Some(snapshot_node) = collection.get_optional_child("snapshot") {
        match &snapshot_node.content {
            Some(wacore_binary::node::NodeContent::Bytes(raw)) => {
                match waproto::codec::external_blob_reference_decode(raw.as_slice()) {
                    Ok(ext_ref) => snapshot_ref = Some(ext_ref),
                    Err(_) => unreadable_snapshot = true,
                }
            }
            _ => unreadable_snapshot = true,
        }
    }
    let snapshot = None; // external only currently

    // patches list
    let mut patches: Vec<wa::SyncdPatch> = Vec::new();
    if let Some(patches_node) = collection.get_optional_child("patches")
        && let Some(children) = patches_node.children()
    {
        for child in children {
            if child.tag == "patch"
                && let Some(wacore_binary::node::NodeContent::Bytes(raw)) = &child.content
            {
                match waproto::codec::syncd_patch_decode(raw.as_slice()) {
                    Ok(p) => patches.push(p),
                    Err(e) => return Err(anyhow!("failed to unmarshal patch: {e}")),
                }
            }
        }
    }

    Ok(PatchList {
        name: WAPatchName::from_str(&name_str).unwrap_or(WAPatchName::Unknown),
        has_more_patches: has_more,
        patches,
        snapshot,
        snapshot_ref,
        error: error.or_else(|| {
            unreadable_snapshot.then(|| CollectionSyncError::Retry {
                code: 0,
                text: "unreadable snapshot reference".to_string(),
            })
        }),
    })
}

/// Parse per-collection error from `<collection type="error"><error code="..." text="..."/>`.
/// Returns `None` for successful collections.
fn parse_collection_error(
    collection: &Node,
    col_type: Option<&str>,
) -> Option<CollectionSyncError> {
    if col_type? != "error" {
        return None;
    }

    // Parse error details from child node, or fall back to a default retryable
    // error if the <error> child is missing/malformed.
    let (code, text) = if let Some(error_node) = collection.get_optional_child("error") {
        let mut error_attrs = error_node.attrs();
        let code_str = error_attrs.optional_string("code");
        let text = error_attrs
            .optional_string("text")
            .as_deref()
            .unwrap_or("")
            .to_string();
        let code: u16 = code_str.as_deref().unwrap_or("0").parse().unwrap_or(0);
        (code, text)
    } else {
        (0u16, "missing <error> child".to_string())
    };

    Some(match code {
        409 => CollectionSyncError::Conflict {
            has_more: collection.attrs().optional_bool("has_more_patches"),
        },
        400 | 404 => CollectionSyncError::Fatal { code, text },
        _ => CollectionSyncError::Retry { code, text },
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use wacore_binary::builder::NodeBuilder;

    fn collection(attrs: &[(&str, &str)], children: Vec<Node>) -> Node {
        let mut builder = NodeBuilder::new("collection").children(children);
        for (key, value) in attrs {
            builder = builder.attr(*key, *value);
        }
        NodeBuilder::new("iq")
            .children([NodeBuilder::new("sync").children([builder.build()]).build()])
            .build()
    }

    #[test]
    fn a_successful_collection_has_no_error() {
        let node = collection(&[("name", "regular")], vec![]);
        let list = parse_patch_list(&node).expect("collection parses");
        assert!(list.error.is_none());
    }

    #[test]
    fn a_fatal_collection_error_is_parsed() {
        let node = collection(
            &[("name", "regular"), ("type", "error")],
            vec![
                NodeBuilder::new("error")
                    .attr("code", "404")
                    .attr("text", "synthetic gone")
                    .build(),
            ],
        );
        let list = parse_patch_list(&node).expect("collection parses");
        assert_eq!(
            list.error,
            Some(CollectionSyncError::Fatal {
                code: 404,
                text: "synthetic gone".into()
            })
        );
    }

    #[test]
    fn a_conflict_error_carries_has_more() {
        let node = collection(
            &[
                ("name", "regular"),
                ("type", "error"),
                ("has_more_patches", "true"),
            ],
            vec![
                NodeBuilder::new("error")
                    .attr("code", "409")
                    .attr("text", "synthetic conflict")
                    .build(),
            ],
        );
        let list = parse_patch_list(&node).expect("collection parses");
        assert_eq!(
            list.error,
            Some(CollectionSyncError::Conflict { has_more: true })
        );
    }

    #[test]
    fn an_unknown_error_code_is_retryable() {
        let node = collection(
            &[("name", "regular"), ("type", "error")],
            vec![
                NodeBuilder::new("error")
                    .attr("code", "503")
                    .attr("text", "synthetic retry")
                    .build(),
            ],
        );
        let list = parse_patch_list(&node).expect("collection parses");
        assert_eq!(
            list.error,
            Some(CollectionSyncError::Retry {
                code: 503,
                text: "synthetic retry".into()
            })
        );
    }

    #[test]
    fn a_missing_error_child_is_retryable() {
        let node = collection(&[("name", "regular"), ("type", "error")], vec![]);
        let list = parse_patch_list(&node).expect("collection parses");
        assert_eq!(
            list.error,
            Some(CollectionSyncError::Retry {
                code: 0,
                text: "missing <error> child".into()
            })
        );
    }

    #[test]
    fn an_unreadable_snapshot_reference_is_retryable() {
        let node = collection(
            &[("name", "regular")],
            vec![NodeBuilder::new("snapshot").bytes(vec![0xFF, 0xFF]).build()],
        );
        let list = parse_patch_list(&node).expect("collection parses");
        assert!(list.snapshot_ref.is_none());
        assert_eq!(
            list.error,
            Some(CollectionSyncError::Retry {
                code: 0,
                text: "unreadable snapshot reference".into()
            })
        );
    }

    #[test]
    fn a_readable_snapshot_reference_is_kept() {
        let mut reference = wa::ExternalBlobReference::default();
        reference.direct_path = Some("/synthetic".into());
        reference.media_key = Some(vec![1; 32]);
        let bytes = waproto::buffa::Message::encode_to_vec(&reference);
        let node = collection(
            &[("name", "regular")],
            vec![NodeBuilder::new("snapshot").bytes(bytes).build()],
        );
        let list = parse_patch_list(&node).expect("collection parses");
        assert!(list.error.is_none());
        assert_eq!(
            list.snapshot_ref
                .and_then(|reference| reference.direct_path),
            Some("/synthetic".into())
        );
    }
}
