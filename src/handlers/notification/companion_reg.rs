//! Companion-registration notifications (`link_code_companion_reg`).

use std::sync::Arc;

use wacore_binary::node::Node;

use crate::client::Client;

/// Handle a companion registration notification by delegating to the pair-code
/// lifecycle. Kept as its own module so the notification router stays small and
/// the passkey/companion split is explicit.
pub(crate) async fn handle(client: &Arc<Client>, node: &Node) -> bool {
    crate::pair_code::handle_pair_code_notification(client, node).await
}
