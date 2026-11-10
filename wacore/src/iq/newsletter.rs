//! Newsletter (Channel) IQ specifications.
//!
//! Newsletters use two protocol layers:
//! - Mex (GraphQL) for metadata/management operations, which live outside this
//!   module and carry their own document IDs and typed variables
//! - Standard IQ (xmlns="newsletter") for message operations, specified here

/// IQ namespace for newsletter operations (message history, reactions, live updates).
pub const NEWSLETTER_XMLNS: &str = "newsletter";

use crate::iq::node::{IqParseError, optional_child, required_child};
use crate::iq::spec::IqSpec;
use crate::request::InfoQuery;
use anyhow::{Result, anyhow};
use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::{Jid, SERVER_JID};
use wacore_binary::node::{Node, NodeContent};

/// This account's own add-ons (its reaction and its poll vote) on one
/// newsletter message, as [`MyAddOnsSpec`] reads them.
///
/// The public tallies cannot answer whether this account's vote landed: they
/// are counts across every follower. This is the server's record of the
/// account's own choice.
#[derive(Debug, Clone, PartialEq, Eq, bon::Builder)]
#[non_exhaustive]
pub struct NewsletterMyAddOns {
    /// The message's server-assigned id.
    pub server_id: u64,
    /// This account's reaction, `None` when it has none on this message.
    pub reaction: Option<NewsletterMyReaction>,
    /// This account's poll vote, `None` when it never voted on this message.
    /// A vote that was removed is still `Some`, with no option hashes: the
    /// server keeps the removal as a dated, empty selection.
    pub poll_vote: Option<NewsletterMyPollVote>,
}

/// This account's own reaction on a newsletter message.
#[derive(Debug, Clone, PartialEq, Eq, bon::Builder)]
#[non_exhaustive]
pub struct NewsletterMyReaction {
    /// The reaction emoji.
    pub code: String,
    /// When it was set (Unix seconds).
    pub timestamp: u64,
}

/// This account's own vote on a newsletter poll.
#[derive(Debug, Clone, PartialEq, Eq, bon::Builder)]
#[non_exhaustive]
pub struct NewsletterMyPollVote {
    /// When the selection was last sent (Unix seconds). This is the `t` of the
    /// server's ack to that vote.
    pub timestamp: u64,
    /// The selected options, each the SHA-256 of an option name
    /// ([`crate::poll::compute_option_hash`]), the key channel poll tallies
    /// are counted by. Empty after the vote was removed.
    pub option_hashes: Vec<[u8; 32]>,
}

/// Read this account's own add-ons on a newsletter's recent messages:
/// `<my_addons limit jid/>` (`makeMyAddOnsRequest`).
///
/// Addressed to the server like the history IQ, naming the channel in the
/// node's `jid` attribute. `makeMyAddOnsRequest` makes that attribute
/// optional, but what the server answers without it has not been observed,
/// so the spec always names one channel.
#[derive(Debug, Clone)]
pub struct MyAddOnsSpec {
    jid: Jid,
    limit: u32,
}

impl MyAddOnsSpec {
    pub fn new(jid: &Jid, limit: u32) -> Self {
        Self {
            jid: jid.clone(),
            limit,
        }
    }
}

impl IqSpec for MyAddOnsSpec {
    type Response = Vec<NewsletterMyAddOns>;

    fn build_iq(&self) -> InfoQuery<'static> {
        InfoQuery::get(
            NEWSLETTER_XMLNS,
            Jid::new("", SERVER_JID),
            Some(NodeContent::Nodes(vec![
                NodeBuilder::new("my_addons")
                    .attr("limit", self.limit.to_string())
                    .jid_attr("jid", self.jid.clone())
                    .build(),
            ])),
        )
    }

    /// ```xml
    /// <my_addons>
    ///   <messages jid="NL_JID">
    ///     <message server_id="777">
    ///       <reaction code="👍" t="TS"/>
    ///       <votes t="TS"><vote>…32-byte option hash…</vote></votes>
    ///     </message>
    ///   </messages>
    /// </my_addons>
    /// ```
    ///
    /// Read as strictly as `WASmaxInNewslettersMyAddOnsResponseSuccess` reads
    /// it: a response that parser would reject is an error here, not a shorter
    /// list. Unlike the history tallies, this is the account's own selection,
    /// and a vote read with one hash dropped is a different vote, not an
    /// approximate one. `<messages>` groups for another channel are ignored,
    /// since a `server_id` only identifies a message within its channel, but a
    /// group whose channel cannot be read is an error: skipping it would pass
    /// off its add-ons as absent.
    fn parse_response(&self, response: &Node) -> Result<Self::Response> {
        let my_addons = required_child(response, "my_addons")?;

        let mut result = Vec::new();
        for group in my_addons.get_children_by_tag("messages") {
            let channel: Jid = required_attr(group, "jid")?
                .parse()
                .map_err(|e| anyhow!("invalid attribute jid: {e}"))?;
            if channel != self.jid {
                continue;
            }
            for msg_node in group.get_children_by_tag("message") {
                result.push(parse_my_addons_message(msg_node)?);
            }
        }
        Ok(result)
    }
}

/// A required attribute rendered as text, including attributes the binary
/// protocol stores as structured JIDs (`<messages jid/>`).
fn required_attr(node: &Node, key: &str) -> Result<String> {
    node.attrs
        .get(key)
        .map(|value| value.to_string_value())
        .ok_or_else(|| {
            IqParseError::MissingAttribute {
                name: key.to_owned(),
            }
            .into()
        })
}

/// A required integer attribute, on top of [`required_attr`] so a missing
/// one reads like every other missing attribute.
fn required_u64(node: &Node, key: &str) -> Result<u64> {
    required_attr(node, key)?
        .parse()
        .map_err(|e| anyhow!("invalid attribute {key}: {e}"))
}

/// The `<vote>` payload on an owned node, where [`NodeRef::content_bytes`]
/// would be available on a borrowed one.
fn content_bytes(node: &Node) -> Option<&[u8]> {
    match &node.content {
        Some(NodeContent::Bytes(bytes)) => Some(bytes),
        _ => None,
    }
}

fn parse_my_addons_message(msg_node: &Node) -> Result<NewsletterMyAddOns> {
    let server_id = required_u64(msg_node, "server_id")?;

    let reaction = match optional_child(msg_node, "reaction") {
        Some(node) => Some(NewsletterMyReaction {
            code: required_attr(node, "code")?,
            timestamp: required_u64(node, "t")?,
        }),
        None => None,
    };

    let poll_vote = match optional_child(msg_node, "votes") {
        Some(node) => {
            let timestamp = required_u64(node, "t")?;
            let option_hashes = node
                .get_children_by_tag("vote")
                .map(|vote| {
                    content_bytes(vote)
                        .and_then(|bytes| <[u8; 32]>::try_from(bytes).ok())
                        .ok_or_else(|| anyhow!("<vote> is not a 32-byte option hash"))
                })
                .collect::<Result<_>>()?;
            Some(NewsletterMyPollVote {
                timestamp,
                option_hashes,
            })
        }
        None => None,
    };

    Ok(NewsletterMyAddOns {
        server_id,
        reaction,
        poll_vote,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::poll::compute_option_hash;
    use wacore_binary::builder::NodeBuilder;
    use wacore_binary::jid::{Jid, SERVER_JID};
    use wacore_binary::node::{Node, NodeContent};

    fn newsletter_jid() -> Jid {
        "120363000000000001@newsletter".parse().expect("jid")
    }

    const GOOD_MORNING_HASH: &str =
        "2e090fda1d75dab720e00f72f5b06d88020d56c637d25a5d64529b51faeb6acf";
    const MONDAYS_HASH: &str = "ea53ff01672231aa49905b2a74164330b2a006d6f731d1227171fd92d3779322";

    fn hash(hex_digest: &str) -> [u8; 32] {
        hex::decode(hex_digest)
            .expect("hex")
            .try_into()
            .expect("32 bytes")
    }

    fn vote(bytes: &[u8]) -> Node {
        NodeBuilder::new("vote").bytes(bytes).build()
    }

    fn votes(t: u64, hashes: &[[u8; 32]]) -> Node {
        NodeBuilder::new("votes")
            .attr("t", t.to_string())
            .children(hashes.iter().map(|h| vote(h)))
            .build()
    }

    fn message(server_id: u64, children: Vec<Node>) -> Node {
        NodeBuilder::new("message")
            .attr("server_id", server_id.to_string())
            .children(children)
            .build()
    }

    fn response_for(jid: &Jid, messages: Vec<Node>) -> Node {
        NodeBuilder::new("iq")
            .attr("type", "result")
            .children([NodeBuilder::new("my_addons")
                .children([NodeBuilder::new("messages")
                    .jid_attr("jid", jid.clone())
                    .children(messages)
                    .build()])
                .build()])
            .build()
    }

    fn parse(response: &Node) -> anyhow::Result<Vec<NewsletterMyAddOns>> {
        MyAddOnsSpec::new(&newsletter_jid(), 20).parse_response(response)
    }

    /// `<iq to="s.whatsapp.net" xmlns="newsletter" type="get"><my_addons
    /// limit jid/></iq>`, as WA Web sent it: addressed to the server, the
    /// channel named on the node.
    #[test]
    fn request_goes_to_the_server_and_names_the_channel() {
        let query = MyAddOnsSpec::new(&newsletter_jid(), 20).build_iq();

        assert_eq!(query.namespace, NEWSLETTER_XMLNS);
        assert_eq!(query.query_type, crate::request::InfoQueryType::Get);
        assert_eq!(query.to, Jid::new("", SERVER_JID));
        assert!(query.target.is_none());

        let Some(NodeContent::Nodes(children)) = query.content.as_ref() else {
            panic!("my_addons carries one child node");
        };
        assert_eq!(children.len(), 1);
        let node = &children[0];
        assert_eq!(node.tag, "my_addons");
        let mut attrs = node.attrs();
        assert_eq!(attrs.optional_u64("limit"), Some(20));
        assert_eq!(attrs.optional_jid("jid"), Some(newsletter_jid()));
        assert_eq!(node.attrs.len(), 2);
        assert!(node.content.is_none());
    }

    /// The captured answer after voting for two options.
    #[test]
    fn a_vote_is_read_with_its_time_and_every_option() {
        let hashes = [hash(GOOD_MORNING_HASH), hash(MONDAYS_HASH)];
        let response = response_for(
            &newsletter_jid(),
            vec![message(777, vec![votes(1790340039, &hashes)])],
        );

        let addons = parse(&response).expect("valid response");

        assert_eq!(addons.len(), 1);
        assert_eq!(addons[0].server_id, 777);
        assert_eq!(addons[0].reaction, None);
        let vote = addons[0].poll_vote.as_ref().expect("a vote");
        assert_eq!(vote.timestamp, 1790340039);
        assert_eq!(vote.option_hashes, hashes);
    }

    /// The option hashes are the same SHA-256 digests the poll module counts
    /// tallies by, so a vote built from `compute_option_hash` reads back
    /// unchanged.
    #[test]
    fn option_hashes_round_trip_through_compute_option_hash() {
        let hashes = [compute_option_hash("Yes"), compute_option_hash("No")];
        let response = response_for(
            &newsletter_jid(),
            vec![message(1, vec![votes(1700000000, &hashes)])],
        );

        let addons = parse(&response).expect("valid response");
        let vote = addons[0].poll_vote.as_ref().expect("a vote");
        assert_eq!(vote.option_hashes, hashes);
    }

    /// A removed vote stays on the server as a dated empty `<votes t/>`,
    /// which is not the same fact as never having voted.
    #[test]
    fn a_removed_vote_is_an_empty_selection_not_an_absent_one() {
        let response = response_for(
            &newsletter_jid(),
            vec![
                message(777, vec![votes(1790340162, &[])]),
                message(
                    778,
                    vec![
                        NodeBuilder::new("reaction")
                            .attr("code", "👍")
                            .attr("t", 1790340200u64.to_string())
                            .build(),
                    ],
                ),
            ],
        );

        let addons = parse(&response).expect("valid response");

        let removed = addons[0]
            .poll_vote
            .as_ref()
            .expect("a removed vote is kept");
        assert_eq!(removed.timestamp, 1790340162);
        assert!(removed.option_hashes.is_empty());
        assert_eq!(addons[1].poll_vote, None, "no <votes> means never voted");
    }

    /// The reaction shape comes from `newsletterMyReactionMixin` in
    /// the IR: `<reaction code t/>` on the same `<message>` as the vote.
    #[test]
    fn a_reaction_is_read_beside_a_vote() {
        let response = response_for(
            &newsletter_jid(),
            vec![message(
                777,
                vec![
                    NodeBuilder::new("reaction")
                        .attr("code", "❤️")
                        .attr("t", 1790340100u64.to_string())
                        .build(),
                    votes(1790340039, &[hash(MONDAYS_HASH)]),
                ],
            )],
        );

        let addons = parse(&response).expect("valid response");

        assert_eq!(
            addons[0].reaction,
            Some(NewsletterMyReaction {
                code: "❤️".into(),
                timestamp: 1790340100,
            })
        );
        assert_eq!(
            addons[0]
                .poll_vote
                .as_ref()
                .map(|v| v.option_hashes.clone()),
            Some(vec![hash(MONDAYS_HASH)])
        );
    }

    #[test]
    fn an_answer_with_no_messages_is_an_empty_list() {
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .children([NodeBuilder::new("my_addons").build()])
            .build();

        assert_eq!(parse(&response).expect("valid response"), []);
    }

    /// A `server_id` only means something within its channel, so a group
    /// for another channel is not read into this one's add-ons.
    #[test]
    fn a_group_for_another_channel_is_ignored() {
        let other: Jid = "120363000000000009@newsletter".parse().expect("jid");
        let response = response_for(&other, vec![message(777, vec![votes(1, &[])])]);

        assert_eq!(parse(&response).expect("valid response"), []);
    }

    /// Every node WA Web's parser requires is required here too: a partly
    /// read selection would misstate the account's own vote.
    #[test]
    fn a_response_wa_web_would_reject_is_an_error() {
        let short_hash = NodeBuilder::new("votes")
            .attr("t", "1")
            .children([vote(&[0u8; 31])])
            .build();
        let untimed_votes = NodeBuilder::new("votes")
            .children([vote(&hash(MONDAYS_HASH))])
            .build();
        let untimed_reaction = NodeBuilder::new("reaction").attr("code", "👍").build();
        let codeless_reaction = NodeBuilder::new("reaction").attr("t", "1").build();

        for (what, messages) in [
            ("a <vote> of 31 bytes", vec![message(777, vec![short_hash])]),
            (
                "a <votes> with no t",
                vec![message(777, vec![untimed_votes])],
            ),
            (
                "a <reaction> with no t",
                vec![message(777, vec![untimed_reaction])],
            ),
            (
                "a <reaction> with no code",
                vec![message(777, vec![codeless_reaction])],
            ),
            (
                "a <message> with no server_id",
                vec![
                    NodeBuilder::new("message")
                        .children([votes(1, &[])])
                        .build(),
                ],
            ),
        ] {
            assert!(
                parse(&response_for(&newsletter_jid(), messages)).is_err(),
                "{what}"
            );
        }

        let no_my_addons = NodeBuilder::new("iq").attr("type", "result").build();
        assert!(parse(&no_my_addons).is_err());
    }

    /// A group whose channel cannot be read is an error, not a group for some
    /// other channel: skipping it would report its add-ons as absent.
    #[test]
    fn a_group_without_a_readable_channel_is_an_error() {
        let group = |jid: Option<&str>| {
            let mut messages = NodeBuilder::new("messages");
            if let Some(jid) = jid {
                messages = messages.attr("jid", jid);
            }
            NodeBuilder::new("iq")
                .attr("type", "result")
                .children([NodeBuilder::new("my_addons")
                    .children([messages
                        .children([message(777, vec![votes(1, &[])])])
                        .build()])
                    .build()])
                .build()
        };

        assert!(parse(&group(None)).is_err(), "no jid");
        // This crate's JID parser is lenient about unknown servers when a
        // user part is present, so the malformed case is a server-only value
        // that names no known server.
        assert!(parse(&group(Some("not a jid"))).is_err(), "unparseable jid");
        assert_eq!(
            parse(&group(Some("120363000000000001@newsletter"))).expect("valid"),
            [NewsletterMyAddOns {
                server_id: 777,
                reaction: None,
                poll_vote: Some(NewsletterMyPollVote {
                    timestamp: 1,
                    option_hashes: Vec::new(),
                }),
            }]
        );
    }
}
