#![cfg(feature = "plugins")]
//! Phase 5 plugin host, lifecycle and event/IQ reply contract.

use std::sync::Arc;
use std::sync::atomic::{AtomicUsize, Ordering};

use bytes::Bytes;
use wa_rs::wacore::runtime::BoxFuture;
use wa_rs::wacore_binary::jid::Jid;
use wa_rs::{
    ClientLifecycle, ClientPlugin, ConnectionScope, ConnectionScopeState, Interception,
    NodeBuilder, PluginCapabilities, PluginCapability, PluginClientInfo, PluginContext,
    PluginEventEndpointConfig, PluginEventOverflow, PluginEventPayloadEncoding,
    PluginEventPublishError, PluginEventRouter, PluginEventSelector, PluginEventTopic,
    PluginEvents, PluginFuture, PluginHost, PluginHostConfig, PluginIqInterceptor, PluginIqReply,
    PluginIqReplyError, PluginManifest, PluginPlanError, PluginState, UntypedClientPlugin,
};

struct DemoPlugin {
    installed: Arc<AtomicUsize>,
}

impl ClientPlugin for DemoPlugin {
    type Api = ();

    fn manifest(&self) -> PluginManifest {
        PluginManifest::new("demo.plugin", "1.0.0").with_capability(PluginCapability::PluginEvents)
    }

    fn install(&self, context: PluginContext) -> PluginFuture<'_, anyhow::Result<Arc<()>>> {
        self.installed.fetch_add(1, Ordering::SeqCst);
        assert_eq!(context.plugin_id(), "demo.plugin");
        Box::pin(async { Ok(Arc::new(())) })
    }
}

struct DemoUntyped;

impl UntypedClientPlugin for DemoUntyped {
    fn manifest(&self) -> PluginManifest {
        PluginManifest::new("demo.untyped", "1.0.0")
    }

    fn install(&self, _context: PluginContext) -> PluginFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }
}

struct RecordingLifecycle {
    ready: Arc<AtomicUsize>,
    closed: Arc<AtomicUsize>,
    shutdown: Arc<AtomicUsize>,
}

impl ClientLifecycle for RecordingLifecycle {
    fn on_ready<'a>(&'a self, _scope: ConnectionScope) -> BoxFuture<'a, anyhow::Result<()>> {
        self.ready.fetch_add(1, Ordering::SeqCst);
        Box::pin(async { Ok(()) })
    }

    fn on_closed<'a>(&'a self, _scope: ConnectionScope) -> BoxFuture<'a, anyhow::Result<()>> {
        self.closed.fetch_add(1, Ordering::SeqCst);
        Box::pin(async { Ok(()) })
    }

    fn signal_shutdown(&self) {
        self.shutdown.fetch_add(1, Ordering::SeqCst);
    }
}

#[test]
fn capabilities_are_a_compact_set() {
    let caps = PluginCapabilities::NONE
        .with(PluginCapability::CoreEvents)
        .with(PluginCapability::PluginEvents);
    assert!(caps.contains(PluginCapability::CoreEvents));
    assert!(caps.contains(PluginCapability::PluginEvents));
    assert!(!caps.contains(PluginCapability::Iq));
    assert_eq!(PluginCapability::IqReply.identifier(), "iq.reply");
    assert_eq!(
        PluginCapability::ClientInfo.identifier(),
        "client.info.read"
    );
}

#[test]
fn plugin_host_validates_manifests_and_lifecycle() {
    let mut host = PluginHost::new(PluginHostConfig::new());
    host.register(PluginManifest::new("demo.base", "1.0.0"))
        .unwrap();
    host.register(
        PluginManifest::new("demo.plugin", "1.0.0")
            .with_dependency("demo.base")
            .with_capability(PluginCapability::PluginEvents),
    )
    .unwrap();

    assert_eq!(host.manifests().len(), 2);
    assert_eq!(host.state("demo.plugin"), Some(PluginState::Installing));
    assert!(host.activate("demo.plugin"));
    assert_eq!(host.state("demo.plugin"), Some(PluginState::Active));

    let stats = host.stats();
    assert_eq!(stats.plugins.len(), 2);
    assert_eq!(stats.plugins[1].state, PluginState::Active);
    assert!(stats.event_router.is_some());

    assert!(matches!(
        host.register(PluginManifest::new("demo.plugin", "1.0.0")),
        Err(PluginPlanError::DuplicateId { .. })
    ));
    assert!(matches!(
        host.register(PluginManifest::new("Demo", "1.0.0")),
        Err(PluginPlanError::InvalidId { .. })
    ));
    assert!(matches!(
        host.register(PluginManifest::new("demo.orphan", "1.0.0").with_dependency("demo.missing")),
        Err(PluginPlanError::MissingDependency { .. })
    ));
    assert!(matches!(
        host.register(
            PluginManifest::new("demo.reply", "1.0.0").with_capability(PluginCapability::IqReply)
        ),
        Err(PluginPlanError::MissingCapabilityDependency { .. })
    ));
    assert_eq!(host.manifests().len(), 2);

    host.stop("demo.plugin");
    assert_eq!(host.state("demo.plugin"), Some(PluginState::Stopped));
    host.shutdown();
    assert!(host.stats().terminal);
    assert_eq!(host.state("demo.base"), Some(PluginState::Stopped));
}

#[test]
fn plugin_host_rejects_dependency_cycles() {
    let mut host = PluginHost::new(PluginHostConfig::default());
    let result = host.register_all([
        PluginManifest::new("cycle.a", "1.0.0").with_dependency("cycle.b"),
        PluginManifest::new("cycle.b", "1.0.0").with_dependency("cycle.a"),
    ]);
    assert!(matches!(
        result,
        Err(PluginPlanError::DependencyCycle { .. })
    ));
    assert!(host.manifests().is_empty());
}

#[tokio::test]
async fn client_plugin_install_receives_context() {
    let installed = Arc::new(AtomicUsize::new(0));
    let plugin = DemoPlugin {
        installed: installed.clone(),
    };
    let context =
        PluginContext::new("demo.plugin", plugin.manifest().capabilities()).with_client_info(
            PluginClientInfo::new(Some(Jid::pn("15551234567")), Some("Demo".to_string())),
        );
    let api = plugin.install(context).await.unwrap();
    assert_eq!(installed.load(Ordering::SeqCst), 1);
    assert_eq!(Arc::strong_count(&api), 1);

    let untyped = DemoUntyped;
    assert_eq!(untyped.manifest().id(), "demo.untyped");
    untyped
        .install(PluginContext::new("demo.untyped", PluginCapabilities::NONE))
        .await
        .unwrap();
}

#[tokio::test]
async fn client_lifecycle_defaults_and_scope_states() {
    let ready = Arc::new(AtomicUsize::new(0));
    let closed = Arc::new(AtomicUsize::new(0));
    let shutdown = Arc::new(AtomicUsize::new(0));
    let lifecycle = RecordingLifecycle {
        ready: ready.clone(),
        closed: closed.clone(),
        shutdown: shutdown.clone(),
    };

    lifecycle
        .install(std::sync::Weak::<wa_rs::Client>::new())
        .await
        .unwrap();
    let scope = ConnectionScope::new(3);
    assert_eq!(scope.generation(), 3);
    assert_eq!(scope.state(), ConnectionScopeState::Open);
    lifecycle.on_ready(scope.clone()).await.unwrap();
    lifecycle.on_closed(scope.clone()).await.unwrap();
    lifecycle.shutdown().await.unwrap();
    lifecycle.signal_shutdown();
    assert_eq!(ready.load(Ordering::SeqCst), 1);
    assert_eq!(closed.load(Ordering::SeqCst), 1);
    assert_eq!(shutdown.load(Ordering::SeqCst), 1);

    let signal = scope.cancellation_signal();
    assert!(!signal.is_fired());
    assert!(scope.mark_ready());
    assert_eq!(scope.state(), ConnectionScopeState::Ready);
    scope.cancel();
    assert!(signal.is_fired());
    assert!(scope.is_cancelled());
    scope.close();
    assert_eq!(scope.state(), ConnectionScopeState::Closed);
}

#[tokio::test]
async fn plugin_events_route_exact_topics_and_overflow() {
    let router = PluginEventRouter::new(["demo.plugin".to_string()]);
    let topic = PluginEventTopic::new("updates").unwrap();
    let selector = PluginEventSelector::new("demo.plugin", topic.clone()).unwrap();
    let subscription = router
        .subscribe(
            [selector],
            PluginEventEndpointConfig::new(1, PluginEventOverflow::DropOldest),
        )
        .unwrap();
    let events = PluginEvents::new("demo.plugin", router.clone());
    assert!(events.has_subscribers(&topic));

    let first = events
        .publish(
            &topic,
            1,
            PluginEventPayloadEncoding::Json,
            Bytes::from_static(b"one"),
        )
        .unwrap();
    assert_eq!(first.enqueued, 1);
    let second = events
        .publish(
            &topic,
            1,
            PluginEventPayloadEncoding::Json,
            Bytes::from_static(b"two"),
        )
        .unwrap();
    assert_eq!(second.dropped, 1);

    let envelope = subscription.recv().await.unwrap();
    assert_eq!(envelope.topic.as_str(), "updates");
    assert_eq!(envelope.payload, Bytes::from_static(b"two"));
    assert_eq!(subscription.selectors()[0].plugin_id(), "demo.plugin");
    assert_eq!(router.stats().delivered, 1);
    assert_eq!(
        router.publisher_stats("demo.plugin").unwrap().state,
        PluginState::Active
    );

    assert!(matches!(
        router.subscribe(
            Vec::<PluginEventSelector>::new(),
            PluginEventEndpointConfig::new(1, PluginEventOverflow::DropNewest)
        ),
        Err(wa_rs::PluginEventSubscribeError::EmptySelectors)
    ));

    router.close();
    assert!(matches!(
        events.publish(
            &topic,
            1,
            PluginEventPayloadEncoding::Json,
            Bytes::from_static(b"closed")
        ),
        Err(PluginEventPublishError::Closed)
    ));
}

#[test]
fn iq_reply_grants_are_single_use_and_validated() {
    let jid: Jid = "15551234567@s.whatsapp.net".parse().unwrap();
    let reply = PluginIqReply::new("iq-1", jid.clone(), true);
    assert_eq!(reply.id(), "iq-1");
    assert_eq!(reply.destination().user, "15551234567");
    assert!(matches!(
        reply.error(399, "bad"),
        Err(PluginIqReplyError::InvalidErrorCode)
    ));
    assert!(matches!(
        reply.error(400, ""),
        Err(PluginIqReplyError::InvalidErrorText)
    ));
    reply.result(Vec::new()).unwrap();
    assert!(reply.is_used());
    assert!(matches!(
        reply.result(Vec::new()),
        Err(PluginIqReplyError::AlreadyUsed)
    ));

    let ungranted = PluginIqReply::new("iq-2", jid, false);
    assert!(matches!(
        ungranted.result(Vec::new()),
        Err(PluginIqReplyError::CapabilityRequired)
    ));

    let interceptor =
        |_node: &wa_rs::wacore_binary::node::Node, _reply: PluginIqReply| Interception::Handled;
    let node = NodeBuilder::new("iq").build();
    assert!(interceptor.intercept(&node, ungranted).is_handled());
    assert!(!Interception::Pass.is_handled());
}
