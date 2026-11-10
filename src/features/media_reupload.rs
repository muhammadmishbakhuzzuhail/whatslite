//! Recover expired media with a correlated receipt ACK and mediaretry notification.
//!
//! Protocol reference: WAWebSendServerErrorReceiptJob and
//! WAWebRequestMediaReuploadManager, WA Web 2.3000.1045368834.

use crate::client::node_io::NodeFilter;
use crate::client::{Client, ClientError};
use futures::FutureExt;
use futures::future::{Shared, WeakShared};
use std::collections::HashMap;
use std::sync::{Arc, Mutex, Weak};
use std::time::Duration;
pub use wacore::media_retry::MediaRetryResult;
use wacore::media_retry::{
    build_media_retry_receipt, encrypt_media_retry_receipt, parse_media_retry_notification,
};
use wacore::runtime::BoxFuture;
use wacore::stanza::wire_tags::{NotificationType, StanzaTag};
use wacore::types::message_ref::{MessageId, MessageRef};
use wacore_binary::Jid;
use wacore_binary::node::Node;

const MEDIA_RETRY_TIMEOUT: Duration = Duration::from_secs(30);
const MEDIA_REUPLOAD_CONCURRENCY: usize = 32;

/// The bounded phase in which a reupload stopped making progress.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum MediaReuploadPhase {
    /// Sending the receipt and waiting for its correlated ACK share one budget.
    SendAndAck,
    /// Waiting for the mediaretry notification after the receipt was accepted.
    Notification,
}

/// Failure of a media reupload. Shared operations preserve the original cause
/// for every subscriber, including transport and parsing failures.
#[derive(Clone)]
#[non_exhaustive]
pub enum MediaReuploadError {
    Client(Arc<ClientError>),
    /// Reupload requires the local account's LID. There is no PN fallback.
    NotLoggedIn,
    /// Another target or media key already owns this wire ID.
    Conflict(MessageId),
    Timeout {
        phase: MediaReuploadPhase,
    },
    /// The waiter channel closed. Dropping the caller's future produces no error.
    Cancelled {
        phase: MediaReuploadPhase,
    },
    /// A rejected receipt. Unknown or missing codes remain in the full response.
    ///
    /// The response is the ACK stanza itself. Its `Debug` output names only the
    /// tag so a server-supplied payload never reaches logs implicitly.
    Rejected {
        code: Option<u16>,
        response: Arc<Node>,
    },
    Internal(Arc<anyhow::Error>),
}

impl std::fmt::Display for MediaReuploadError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Client(error) => std::fmt::Display::fmt(error, f),
            Self::NotLoggedIn => f.write_str("local LID is unavailable"),
            Self::Conflict(id) => write!(f, "conflicting media reupload for message {id}"),
            Self::Timeout { phase } => write!(f, "media retry {phase:?} timed out"),
            Self::Cancelled { phase } => write!(f, "media retry {phase:?} waiter cancelled"),
            Self::Rejected { code, .. } => {
                write!(f, "media retry receipt rejected (code: {code:?})")
            }
            Self::Internal(error) => std::fmt::Display::fmt(error, f),
        }
    }
}

impl std::fmt::Debug for MediaReuploadError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            // Opaque causes may include keys or signed URLs. Keep them available
            // through source(), but never format them implicitly in diagnostics.
            Self::Client(_) => f.write_str("Client([REDACTED])"),
            Self::Internal(_) => f.write_str("Internal([REDACTED])"),
            Self::NotLoggedIn => f.write_str("NotLoggedIn"),
            Self::Conflict(id) => f.debug_tuple("Conflict").field(id).finish(),
            Self::Timeout { phase } => f.debug_struct("Timeout").field("phase", phase).finish(),
            Self::Cancelled { phase } => f.debug_struct("Cancelled").field("phase", phase).finish(),
            Self::Rejected { code, response } => f
                .debug_struct("Rejected")
                .field("code", code)
                .field("response_tag", &response.tag)
                .finish(),
        }
    }
}

impl std::error::Error for MediaReuploadError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        // Expose the original typed cause, not the Arc used to share it.
        // A derived source on Arc<ClientError> exposes the Arc as the cause.
        match self {
            Self::Client(error) => Some(error.as_ref()),
            Self::Internal(error) => Some(error.as_ref().as_ref()),
            _ => None,
        }
    }
}

impl From<ClientError> for MediaReuploadError {
    fn from(error: ClientError) -> Self {
        Self::Client(Arc::new(error))
    }
}

impl From<anyhow::Error> for MediaReuploadError {
    fn from(error: anyhow::Error) -> Self {
        Self::Internal(Arc::new(error))
    }
}

/// A validated message target and its media key. Media keys are distinct from
/// the message secrets used by add-ons. Debug never prints key bytes.
#[derive(Clone)]
pub struct MediaReuploadRequest<'a> {
    pub target: MessageRef<'a>,
    pub media_key: &'a [u8; 32],
}

impl std::fmt::Debug for MediaReuploadRequest<'_> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("MediaReuploadRequest")
            .field("target", &self.target)
            .field("media_key", &"[REDACTED]")
            .finish()
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct ReuploadTarget {
    id: MessageId,
    chat: Jid,
    participant: Option<Jid>,
    from_me: bool,
    media_key: [u8; 32],
}

type ReuploadFuture = BoxFuture<'static, Result<MediaRetryResult, MediaReuploadError>>;
type PendingReupload = WeakShared<ReuploadFuture>;

pub(crate) struct MediaReuploadInFlight {
    target: Arc<ReuploadTarget>,
    // The registry must not own the future: the final subscriber's Drop must
    // release its waiters and key even when no more traffic arrives.
    future: PendingReupload,
}

pub(crate) type MediaReuploadMap = HashMap<String, MediaReuploadInFlight>;

struct ReuploadGuard {
    pending: Arc<Mutex<MediaReuploadMap>>,
    target: Arc<ReuploadTarget>,
}

impl Drop for ReuploadGuard {
    fn drop(&mut self) {
        let mut pending = self
            .pending
            .lock()
            .unwrap_or_else(|poison| poison.into_inner());
        if pending
            .get(self.target.id.as_str())
            .is_some_and(|entry| Arc::ptr_eq(&entry.target, &self.target))
        {
            pending.remove(self.target.id.as_str());
        }
    }
}

/// Notification waiter for a correlated `<notification type="mediaretry">`.
fn notification_filter(media_id: &str) -> NodeFilter {
    NodeFilter::tag(StanzaTag::Notification.as_str())
        .attr("type", NotificationType::MediaRetry.as_str())
        .attr("id", media_id)
}

/// ACK waiter for the server-error receipt. The server ACK omits
/// `participant`; a group-echoed ACK carries one and must not satisfy us.
fn ack_filter(media_id: &str, own_jid: &Jid) -> NodeFilter {
    NodeFilter::tag(StanzaTag::Ack.as_str())
        .attr("id", media_id)
        .attr("class", StanzaTag::Receipt.as_str())
        .attr("type", "server-error")
        .from_jid(own_jid)
        .without_attr("participant")
}

/// `None` when the ACK accepted the receipt. `Some(code)` when it carries an
/// `error` attribute or `<error>` child; the inner `None` means unparsable.
fn receipt_rejection(ack: &Node) -> Option<Option<u16>> {
    if let Some(error) = ack.attrs.get("error") {
        return Some(error.as_str().and_then(|value| value.parse().ok()));
    }
    ack.get_optional_child_by_tag(&["error"]).map(|error| {
        error
            .attrs
            .get("code")
            .and_then(|value| value.as_str())
            .and_then(|value| value.parse().ok())
    })
}

/// Build the operation target from a caller's reference. The chat is kept
/// verbatim: WA Web substitutes accountLid only when `is_lid_migrated()` holds,
/// and this fork has no migration flag. A PN→LID mapping alone is not
/// evidence — an unmigrated account keeps 1:1 chats on PN even with a known
/// mapping, and `<rmr jid>` must address the message's own chat.
fn reupload_target(req: &MediaReuploadRequest<'_>) -> ReuploadTarget {
    ReuploadTarget {
        id: req.target.id().clone(),
        chat: req.target.chat().to_non_ad(),
        participant: req.target.receipt_sender().map(Jid::to_non_ad),
        from_me: req.target.from_me(),
        media_key: *req.media_key,
    }
}

/// Race `future` against the phase budget, naming the phase that stalled.
/// The budget is a parameter so the mapping is testable offline.
async fn within_phase<F, T>(
    phase: MediaReuploadPhase,
    budget: Duration,
    future: F,
) -> Result<T, MediaReuploadError>
where
    F: std::future::Future<Output = T>,
{
    tokio::time::timeout(budget, future)
        .await
        .map_err(|_| MediaReuploadError::Timeout { phase })
}

pub struct MediaReupload<'a> {
    client: &'a Client,
}

impl<'a> MediaReupload<'a> {
    pub(crate) fn new(client: &'a Client) -> Self {
        Self { client }
    }

    fn subscribe(
        &self,
        req: &MediaReuploadRequest<'_>,
    ) -> Result<Shared<ReuploadFuture>, MediaReuploadError> {
        let target = reupload_target(req);
        let mut pending = self
            .client
            .media_reuploads
            .lock()
            .unwrap_or_else(|poison| poison.into_inner());
        if let Some(entry) = pending.get(target.id.as_str())
            && let Some(future) = entry.future.upgrade()
        {
            let identical = *entry.target == target;
            // Dropping the last Shared subscriber runs ReuploadGuard. Release
            // the map before dropping a conflicting temporary subscriber.
            drop(pending);
            return if identical {
                Ok(future)
            } else {
                Err(MediaReuploadError::Conflict(target.id))
            };
        }
        let client = self
            .client
            .self_weak
            .get()
            .and_then(Weak::upgrade)
            .ok_or_else(|| anyhow::anyhow!("client is no longer available"))?;
        let target = Arc::new(target);
        let guard = ReuploadGuard {
            pending: Arc::clone(&self.client.media_reuploads),
            target: target.clone(),
        };
        let request = target.clone();
        let future: ReuploadFuture = Box::pin(async move {
            let _guard = guard;
            client.media_reupload().request_inner(&request).await
        });
        let future = future.shared();
        let weak = future
            .downgrade()
            .ok_or_else(|| anyhow::anyhow!("new shared future is pending"))?;
        pending.insert(
            target.id.to_string(),
            MediaReuploadInFlight {
                target,
                future: weak,
            },
        );
        Ok(future)
    }

    /// Recover one message. Concurrent identical requests share the operation.
    /// A conflicting target/key with the same wire ID returns `Conflict`.
    /// Cancelling one caller leaves other subscribers running; cancelling the
    /// last subscriber removes the operation and its node waiters immediately.
    pub async fn request(
        &self,
        req: &MediaReuploadRequest<'_>,
    ) -> Result<MediaRetryResult, MediaReuploadError> {
        self.subscribe(req)?.await
    }

    async fn request_inner(
        &self,
        req: &ReuploadTarget,
    ) -> Result<MediaRetryResult, MediaReuploadError> {
        let lid = self
            .client
            .persistence_manager
            .get_device_snapshot()
            .await
            .lid
            .clone()
            .ok_or(MediaReuploadError::NotLoggedIn)?;
        let own_jid = lid.to_non_ad();
        // WAWebSendServerErrorReceiptJob substitutes accountLid only for a
        // migrated regular-user chat. Without a migration flag, keep the
        // message's own chat; genuinely LID-addressed chats already are LID.
        let chat = req.chat.clone();
        // MessageRef requires the author for received group/broadcast messages.
        // Preserve the supplied sender; the Web job omits participant when its
        // message model has no sender (possible for an own-send reference).
        let participant = req.participant.as_ref();
        let (ciphertext, iv) = encrypt_media_retry_receipt(&req.media_key, req.id.as_str())?;
        let notification = self
            .client
            .wait_for_node(notification_filter(req.id.as_str()));
        let ack = self
            .client
            .wait_for_node(ack_filter(req.id.as_str(), &own_jid));
        let receipt = build_media_retry_receipt(
            &own_jid,
            req.id.as_str(),
            &chat,
            req.from_me,
            participant,
            &ciphertext,
            &iv,
        );
        // Both subscriptions exist before send. A notification received during
        // send/ACK remains buffered, but cannot hide a rejected receipt.
        within_phase(MediaReuploadPhase::SendAndAck, MEDIA_RETRY_TIMEOUT, async {
            self.client.send_node(receipt).await?;
            let ack = ack.await.map_err(|_| MediaReuploadError::Cancelled {
                phase: MediaReuploadPhase::SendAndAck,
            })?;
            if let Some(code) = receipt_rejection(&ack) {
                return Err(MediaReuploadError::Rejected {
                    code,
                    response: ack,
                });
            }
            Ok(())
        })
        .await??;
        let node = within_phase(
            MediaReuploadPhase::Notification,
            MEDIA_RETRY_TIMEOUT,
            notification,
        )
        .await?
        .map_err(|_| MediaReuploadError::Cancelled {
            phase: MediaReuploadPhase::Notification,
        })?;
        Ok(parse_media_retry_notification(
            &node.as_node_ref(),
            req.media_key.as_slice(),
        )?)
    }

    /// Recover a batch, preserving input order and cardinality. Identical inputs
    /// share one receipt even when separated by more than the concurrency window.
    /// The first occurrence reserves each wire ID; conflicting inputs receive
    /// `Conflict` in their own result slot. Up to 32 distinct operations run at
    /// once; duplicates share a slot and completed operations release it
    /// without waiting for earlier inputs.
    pub async fn request_many(
        &self,
        reqs: &[MediaReuploadRequest<'_>],
    ) -> Vec<Result<MediaRetryResult, MediaReuploadError>> {
        use futures::StreamExt;
        let mut results = vec![None; reqs.len()];
        let mut operations: Vec<(Shared<ReuploadFuture>, Vec<usize>)> = Vec::new();
        let mut by_id = std::collections::HashMap::<&str, usize>::new();
        // Reserve every input before polling: duplicates beyond the window
        // must share even if the first operation finishes immediately.
        for (index, req) in reqs.iter().enumerate() {
            match self.subscribe(req) {
                Err(error) => results[index] = Some(Err(error)),
                Ok(subscription) => {
                    if let Some(&operation) = by_id.get(req.target.id().as_str())
                        && operations[operation].0.ptr_eq(&subscription)
                    {
                        operations[operation].1.push(index);
                    } else {
                        // An external subscriber can complete an operation
                        // during reservation. Only group the same shared future.
                        by_id.insert(req.target.id().as_str(), operations.len());
                        operations.push((subscription, vec![index]));
                    }
                }
            }
        }
        async fn complete(
            (operation, indices): (Shared<ReuploadFuture>, Vec<usize>),
        ) -> (Result<MediaRetryResult, MediaReuploadError>, Vec<usize>) {
            (operation.await, indices)
        }
        // Collect before awaiting: retaining the map in the stream prevents
        // MSRV callers from proving this future is Send.
        let operations: Vec<_> = operations.into_iter().map(complete).collect();
        let mut pending =
            futures::stream::iter(operations).buffer_unordered(MEDIA_REUPLOAD_CONCURRENCY);
        while let Some((result, indices)) = pending.next().await {
            for index in indices {
                results[index] = Some(result.clone());
            }
        }
        results
            .into_iter()
            .map(|result| {
                result.unwrap_or_else(|| {
                    // Every input above either reserved a slot or stored a
                    // conflict; this arm is unreachable, so it must not panic.
                    Err(MediaReuploadError::Internal(Arc::new(anyhow::anyhow!(
                        "media reupload result slot was not reserved"
                    ))))
                })
            })
            .collect()
    }
}

impl Client {
    pub fn media_reupload(&self) -> MediaReupload<'_> {
        MediaReupload::new(self)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::client::ClientError;
    use std::error::Error;
    use wacore::types::message_ref::{MessageId, MessageRef};
    use wacore_binary::builder::NodeBuilder;

    fn pn_target<'a>(chat: &'a Jid, id: &str) -> MessageRef<'a> {
        MessageRef::new(
            chat,
            MessageId::new(id).expect("test message id"),
            None,
            false,
        )
        .expect("test message reference")
    }

    #[test]
    fn debug_never_prints_media_key_bytes() {
        let chat = Jid::pn("15551234567");
        let target = pn_target(&chat, "MSG1");
        let media_key = [0xAB; 32];
        let request = MediaReuploadRequest {
            target,
            media_key: &media_key,
        };

        let debug = format!("{request:?}");

        assert!(debug.contains("[REDACTED]"), "{debug}");
        assert!(!debug.contains("ab"), "{debug}");
        assert!(!debug.contains("171"), "{debug}");
    }

    #[test]
    fn display_messages_are_stable() {
        assert_eq!(
            MediaReuploadError::NotLoggedIn.to_string(),
            "local LID is unavailable"
        );
        assert_eq!(
            MediaReuploadError::Conflict(MessageId::new("MSG1").expect("id")).to_string(),
            "conflicting media reupload for message MSG1"
        );
        assert_eq!(
            MediaReuploadError::Timeout {
                phase: MediaReuploadPhase::SendAndAck
            }
            .to_string(),
            "media retry SendAndAck timed out"
        );
        assert_eq!(
            MediaReuploadError::Cancelled {
                phase: MediaReuploadPhase::Notification
            }
            .to_string(),
            "media retry Notification waiter cancelled"
        );
    }

    #[test]
    fn phases_are_distinct() {
        assert_ne!(
            MediaReuploadPhase::SendAndAck,
            MediaReuploadPhase::Notification
        );
    }

    #[test]
    fn debug_redacts_opaque_causes() {
        let client = MediaReuploadError::Client(Arc::new(ClientError::NotConnected));
        assert_eq!(format!("{client:?}"), "Client([REDACTED])");

        let internal = MediaReuploadError::Internal(Arc::new(anyhow::anyhow!("secret detail")));
        assert_eq!(format!("{internal:?}"), "Internal([REDACTED])");
    }

    #[test]
    fn error_source_exposes_original_cause() {
        let client = MediaReuploadError::from(ClientError::NotConnected);
        assert!(client.source().is_some());

        let internal = MediaReuploadError::from(anyhow::anyhow!("boom"));
        assert!(internal.source().is_some());

        assert!(MediaReuploadError::NotLoggedIn.source().is_none());
    }

    #[test]
    fn rejection_reads_attribute_then_child_code() {
        let attr = NodeBuilder::new("ack").attr("error", "401").build();
        assert_eq!(receipt_rejection(&attr), Some(Some(401)));

        let child = NodeBuilder::new("ack")
            .children([NodeBuilder::new("error").attr("code", "500").build()])
            .build();
        assert_eq!(receipt_rejection(&child), Some(Some(500)));

        let unknown = NodeBuilder::new("ack").attr("error", "boom").build();
        assert_eq!(receipt_rejection(&unknown), Some(None));

        let accepted = NodeBuilder::new("ack").attr("class", "receipt").build();
        assert_eq!(receipt_rejection(&accepted), None);
    }

    #[test]
    fn ack_filter_matches_correlated_server_error_receipt() {
        let own = Jid::pn("15550001111");
        let filter = ack_filter("MSG1", &own);

        let matching = NodeBuilder::new("ack")
            .attr("id", "MSG1")
            .attr("class", "receipt")
            .attr("type", "server-error")
            .attr("from", own.to_string())
            .build();
        assert!(filter.matches(&matching.as_node_ref()));

        let other_from = NodeBuilder::new("ack")
            .attr("id", "MSG1")
            .attr("class", "receipt")
            .attr("type", "server-error")
            .attr("from", Jid::pn("15550002222").to_string())
            .build();
        assert!(!filter.matches(&other_from.as_node_ref()));

        let with_participant = NodeBuilder::new("ack")
            .attr("id", "MSG1")
            .attr("class", "receipt")
            .attr("type", "server-error")
            .attr("from", own.to_string())
            .attr("participant", Jid::pn("15550003333").to_string())
            .build();
        assert!(!filter.matches(&with_participant.as_node_ref()));

        let wrong_id = NodeBuilder::new("ack")
            .attr("id", "MSG2")
            .attr("class", "receipt")
            .attr("type", "server-error")
            .attr("from", own.to_string())
            .build();
        assert!(!filter.matches(&wrong_id.as_node_ref()));
    }

    #[test]
    fn notification_filter_matches_mediaretry() {
        let filter = notification_filter("MSG1");

        let matching = NodeBuilder::new("notification")
            .attr("type", "mediaretry")
            .attr("id", "MSG1")
            .build();
        assert!(filter.matches(&matching.as_node_ref()));

        let other_id = NodeBuilder::new("notification")
            .attr("type", "mediaretry")
            .attr("id", "MSG2")
            .build();
        assert!(!filter.matches(&other_id.as_node_ref()));

        let other_type = NodeBuilder::new("notification")
            .attr("type", "w:gp2")
            .attr("id", "MSG1")
            .build();
        assert!(!filter.matches(&other_type.as_node_ref()));
    }

    #[test]
    fn target_keeps_every_chat_kind_verbatim() {
        // Regression: a PN chat with a known PN→LID mapping must stay PN. Only
        // `is_lid_migrated()` justifies substitution and this fork has no such
        // flag, so the receipt must address the message's own chat.
        let pn = Jid::pn("15551234567");
        let lid = Jid::lid("100000012345678");
        let group = Jid::group("120363040237990503");
        let sender = Jid::pn("15550001111");
        let media_key = [1u8; 32];

        let group_target = MessageRef::new(
            &group,
            MessageId::new("MSG1").expect("test message id"),
            Some(&sender),
            false,
        )
        .expect("test group reference");
        for (chat, target) in [
            (pn.clone(), pn_target(&pn, "MSG1")),
            (lid.clone(), pn_target(&lid, "MSG1")),
            (group.clone(), group_target),
        ] {
            let request = MediaReuploadRequest {
                target,
                media_key: &media_key,
            };
            let target = reupload_target(&request);
            assert_eq!(target.chat, chat, "target chat must not be rewritten");
        }
    }

    #[test]
    fn reupload_targets_distinguish_key_and_author() {
        let base = ReuploadTarget {
            id: MessageId::new("MSG1").expect("id"),
            chat: Jid::pn("15551234567"),
            participant: None,
            from_me: false,
            media_key: [1; 32],
        };
        let same = base.clone();
        assert_eq!(base, same);

        let mut other_key = base.clone();
        other_key.media_key = [2; 32];
        assert_ne!(base, other_key);

        let mut other_author = base.clone();
        other_author.participant = Some(Jid::pn("15550002222"));
        assert_ne!(base, other_author);
    }

    #[tokio::test]
    async fn client_wait_for_node_resolves_matching_node() {
        let client = crate::test_utils::create_test_client().await;
        let mut waiter = client.wait_for_node(NodeFilter::tag("ack").attr("id", "WAITER1"));

        client
            .process_decrypted_node(NodeBuilder::new("ack").attr("id", "OTHER").build())
            .await;
        assert!(matches!(waiter.try_recv(), Ok(None)));

        client
            .process_decrypted_node(NodeBuilder::new("ack").attr("id", "WAITER1").build())
            .await;
        let received: Arc<Node> = waiter.await.expect("waiter must resolve");
        assert_eq!(
            received.attrs.get("id").and_then(|value| value.as_str()),
            Some("WAITER1")
        );
    }

    #[tokio::test]
    async fn identical_targets_share_and_conflicts_are_rejected() {
        let client = crate::test_utils::create_test_client().await;
        let chat = Jid::pn("15551234567");
        let media_key_a = [1u8; 32];
        let media_key_b = [2u8; 32];
        let request_a = MediaReuploadRequest {
            target: pn_target(&chat, "MSG1"),
            media_key: &media_key_a,
        };
        let request_b = MediaReuploadRequest {
            target: pn_target(&chat, "MSG1"),
            media_key: &media_key_b,
        };
        let reupload = client.media_reupload();

        let first = reupload
            .subscribe(&request_a)
            .expect("first operation reserves the id");
        let duplicate = reupload
            .subscribe(&request_a)
            .expect("identical request shares the operation");
        assert!(
            first.ptr_eq(&duplicate),
            "duplicates must share one operation"
        );

        match reupload.subscribe(&request_b) {
            Err(MediaReuploadError::Conflict(id)) => assert_eq!(id.as_str(), "MSG1"),
            other => panic!("expected Conflict, got {other:?}"),
        }

        // Dropping one of two subscribers keeps the operation and its id.
        drop(duplicate);
        assert_eq!(
            client
                .media_reuploads
                .lock()
                .unwrap_or_else(|poison| poison.into_inner())
                .len(),
            1
        );

        // Dropping the last subscriber cancels the operation and frees the id.
        drop(first);
        assert!(
            client
                .media_reuploads
                .lock()
                .unwrap_or_else(|poison| poison.into_inner())
                .is_empty()
        );
    }

    #[tokio::test]
    async fn request_many_preserves_order_and_reports_conflicts_in_place() {
        let client = crate::test_utils::create_test_client().await;
        let chat = Jid::pn("15551234567");
        let media_key_a = [1u8; 32];
        let media_key_b = [2u8; 32];
        let requests = [
            MediaReuploadRequest {
                target: pn_target(&chat, "MSG1"),
                media_key: &media_key_a,
            },
            MediaReuploadRequest {
                target: pn_target(&chat, "MSG1"),
                media_key: &media_key_b,
            },
            MediaReuploadRequest {
                target: pn_target(&chat, "MSG1"),
                media_key: &media_key_a,
            },
            MediaReuploadRequest {
                target: pn_target(&chat, "MSG2"),
                media_key: &media_key_a,
            },
        ];

        let results = client.media_reupload().request_many(&requests).await;

        assert_eq!(results.len(), requests.len(), "cardinality is preserved");
        assert!(matches!(results[0], Err(MediaReuploadError::NotLoggedIn)));
        assert!(
            matches!(results[1], Err(MediaReuploadError::Conflict(ref id)) if id.as_str() == "MSG1")
        );
        assert!(
            matches!(results[2], Err(MediaReuploadError::NotLoggedIn)),
            "the duplicate shares the first operation's slot"
        );
        assert!(
            matches!(results[3], Err(MediaReuploadError::NotLoggedIn)),
            "a distinct id runs its own operation"
        );
        assert!(
            client
                .media_reuploads
                .lock()
                .unwrap_or_else(|poison| poison.into_inner())
                .is_empty()
        );
    }

    #[tokio::test]
    async fn phase_budget_reports_the_stalled_phase() {
        for phase in [
            MediaReuploadPhase::SendAndAck,
            MediaReuploadPhase::Notification,
        ] {
            let error = within_phase(
                phase,
                Duration::from_millis(5),
                std::future::pending::<()>(),
            )
            .await
            .expect_err("a pending future must time out");
            assert!(
                matches!(error, MediaReuploadError::Timeout { phase: stalled } if stalled == phase),
                "{error:?}"
            );
        }

        let value = within_phase(
            MediaReuploadPhase::SendAndAck,
            Duration::from_secs(1),
            async { 7 },
        )
        .await;
        assert_eq!(value.expect("a fast future completes"), 7);
    }

    #[tokio::test]
    async fn request_without_lid_reports_not_logged_in_and_releases_registry() {
        let client = crate::test_utils::create_test_client().await;
        let chat = Jid::pn("15551234567");
        let target = pn_target(&chat, "MSG1");
        let key = [9u8; 32];
        let request = MediaReuploadRequest {
            target,
            media_key: &key,
        };

        let result = client.media_reupload().request(&request).await;

        assert!(matches!(result, Err(MediaReuploadError::NotLoggedIn)));
        assert!(
            client
                .media_reuploads
                .lock()
                .unwrap_or_else(|poison| poison.into_inner())
                .is_empty()
        );
    }
}
