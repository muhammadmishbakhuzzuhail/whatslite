//! Newsletter surface.

use wacore_binary::node::NodeRef;

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum NewsletterError {
    #[error("newsletter operation failed: {0}")]
    Failed(String),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewsletterMetadata {
    pub jid: String,
    pub name: String,
    pub state: NewsletterState,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewsletterMessage {
    pub id: String,
    pub text: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NewsletterAdminInfo {
    pub admins: Vec<String>,
}

/// A poll option's vote tally on a newsletter message.
///
/// Channel polls are counted server-side, so unlike a DM or group vote (which
/// arrives encrypted) there is nothing to decrypt here: the server hands over
/// the totals directly, keyed by `option_hash`, the SHA-256 of the option name.
#[derive(Debug, Clone, PartialEq, Eq, bon::Builder)]
#[non_exhaustive]
pub struct NewsletterPollVote {
    pub option_hash: [u8; 32],
    pub count: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NewsletterRole {
    Owner,
    Admin,
    Subscriber,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NewsletterState {
    Active,
    Suspended,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Newsletter<'a> {
    pub _marker: std::marker::PhantomData<&'a ()>,
}

/// A reaction count on a newsletter message.
#[derive(Debug, Clone, bon::Builder)]
#[non_exhaustive]
pub struct NewsletterReactionCount {
    pub code: String,
    pub count: u64,
}

// ─── Shared parsing helpers ────────────────────────────────────────────

/// Parse reaction counts from a `<reactions>` node.
/// Used by both message history parsing and notification handling.
pub(crate) fn parse_reaction_counts(node: &NodeRef<'_>) -> Vec<NewsletterReactionCount> {
    let mut reactions = Vec::new();
    if let Some(reactions_node) = node.get_optional_child("reactions")
        && let Some(children) = reactions_node.children()
    {
        for r in children.iter().filter(|n| n.tag.as_ref() == "reaction") {
            let Some(code) = r
                .get_attr("code")
                .and_then(|v| v.as_str())
                .filter(|s| !s.is_empty())
                .map(str::to_owned)
            else {
                continue;
            };
            let count = r
                .get_attr("count")
                .and_then(|v| v.as_str())
                .and_then(|s| s.parse::<u64>().ok())
                .unwrap_or(0);
            reactions.push(NewsletterReactionCount { code, count });
        }
    }
    reactions
}

/// Parse per-option tallies from a `<votes>` node.
///
/// Not gated on the message's `type`: history responses can carry `<votes>`
/// on a poll envelope whose type is missing or is not yet known, so gating
/// here would make the parser brittle as the wire grows. Live updates carry
/// `<votes>` in the same shape and are read by this helper too.
///
/// A `<vote>` whose content is not a 32-byte digest is skipped rather than
/// truncated or padded — the hash is the only handle on which option was
/// voted for, and a wrong one silently attributes votes to the wrong option.
pub(crate) fn parse_poll_votes(node: &NodeRef<'_>) -> Vec<NewsletterPollVote> {
    let mut votes = Vec::new();
    if let Some(votes_node) = node.get_optional_child("votes")
        && let Some(children) = votes_node.children()
    {
        for v in children.iter().filter(|n| n.tag.as_ref() == "vote") {
            let Some(option_hash) = v
                .content_bytes()
                .and_then(|bytes| <[u8; 32]>::try_from(bytes).ok())
            else {
                continue;
            };
            let Some(count) = v.attr_parser().optional_u64("count") else {
                // `count` is required by the history contract; an absent or
                // malformed value is not the same fact as count zero.
                continue;
            };
            votes.push(NewsletterPollVote { option_hash, count });
        }
    }
    votes
}

/// Read a `<tag count="N"/>` child, the shape `<forwards_count>`,
/// `<views_count>` and `<responses_count>` share.
///
/// `<views_count>` has a second form upstream, `<views_count type="views"
/// count="N"/>`, which puts the count in the same place; both are read here.
///
/// The server omits the node instead of sending a zero, so `None` means "no
/// counter" and does not collapse with a count of zero.
pub(crate) fn parse_count_child(node: &NodeRef<'_>, tag: &str) -> Option<u64> {
    node.get_optional_child(tag)?
        .attr_parser()
        .optional_u64("count")
}

#[cfg(test)]
mod tests {
    use super::*;
    use wacore_binary::builder::NodeBuilder;

    #[test]
    fn reaction_without_code_is_skipped_and_missing_count_is_zero() {
        let node = NodeBuilder::new("message")
            .children([NodeBuilder::new("reactions")
                .children([
                    NodeBuilder::new("reaction").attr("count", "4").build(),
                    NodeBuilder::new("reaction").attr("code", "").build(),
                    NodeBuilder::new("reaction").attr("code", "👍").build(),
                ])
                .build()])
            .build();

        let reactions = parse_reaction_counts(&node.as_node_ref());

        assert_eq!(reactions.len(), 1);
        assert_eq!(reactions[0].code, "👍");
        assert_eq!(reactions[0].count, 0);
    }

    #[test]
    fn vote_with_wrong_hash_length_or_missing_count_is_skipped() {
        let node = NodeBuilder::new("message")
            .children([NodeBuilder::new("votes")
                .children([
                    NodeBuilder::new("vote")
                        .attr("count", "1")
                        .bytes(vec![1u8; 31])
                        .build(),
                    NodeBuilder::new("vote")
                        .attr("count", "1")
                        .bytes(vec![1u8; 33])
                        .build(),
                    NodeBuilder::new("vote").bytes(vec![2u8; 32]).build(),
                    NodeBuilder::new("vote")
                        .attr("count", "9")
                        .bytes(vec![3u8; 32])
                        .build(),
                ])
                .build()])
            .build();

        let votes = parse_poll_votes(&node.as_node_ref());

        assert_eq!(votes.len(), 1);
        assert_eq!(votes[0].option_hash, [3u8; 32]);
        assert_eq!(votes[0].count, 9);
    }

    #[test]
    fn count_child_distinguishes_absent_from_zero() {
        let node = NodeBuilder::new("message")
            .children([NodeBuilder::new("forwards_count")
                .attr("count", "0")
                .build()])
            .build();
        let node_ref = node.as_node_ref();

        assert_eq!(parse_count_child(&node_ref, "forwards_count"), Some(0));
        assert_eq!(parse_count_child(&node_ref, "views_count"), None);
    }
}
