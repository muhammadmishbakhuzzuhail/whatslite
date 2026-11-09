//! Build-time client plugins and their capability-scoped host.
//!
//! The host validates manifests, orders dependencies, and exposes the
//! capability-scoped surfaces a trusted native plugin receives. The runtime
//! callback queue is owned by the client lifecycle; this module keeps the
//! public contract and a synchronous registry for validation/stats.

mod events;
mod iq_reply;

pub use events::{
    PluginEventEndpointConfig, PluginEventEnvelope, PluginEventOverflow,
    PluginEventPayloadEncoding, PluginEventPublishError, PluginEventPublishReport,
    PluginEventPublisherStats, PluginEventReceiveError, PluginEventRouteError, PluginEventRouter,
    PluginEventRouterStats, PluginEventSelector, PluginEventSubscribeError,
    PluginEventSubscription, PluginEventTopic, PluginEventTryReceiveError, PluginEvents,
};
pub use iq_reply::{Interception, PluginIqInterceptor, PluginIqReply};

use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use std::time::Duration;

use wacore::runtime::{BoxFuture, ShutdownSignal};
use wacore::sync_marker::MaybeSendSync;
use wacore_binary::jid::Jid;

use crate::client::extension_lifecycle::{ConnectionScope, ConnectionScopeState};

const CAP_CORE_EVENTS: u64 = 1 << 0;
const CAP_TASKS: u64 = 1 << 1;
const CAP_MESSAGING: u64 = 1 << 2;
const CAP_IQ: u64 = 1 << 3;
const CAP_PLUGIN_EVENTS: u64 = 1 << 4;
const CAP_STANZA_INTERCEPTION: u64 = 1 << 5;
const CAP_CLIENT_INFO: u64 = 1 << 6;
const CAP_IQ_REPLY: u64 = 1 << 7;

const DEFAULT_PLUGIN_INSTALL_TIMEOUT: Duration = Duration::from_secs(30);
const DEFAULT_PLUGIN_CALLBACK_TIMEOUT: Duration = Duration::from_secs(5);
const DEFAULT_PLUGIN_TASK_DRAIN_TIMEOUT: Duration = Duration::from_secs(5);

/// A capability a plugin asks the host to expose during installation.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[non_exhaustive]
pub enum PluginCapability {
    CoreEvents,
    Tasks,
    Messaging,
    Iq,
    PluginEvents,
    StanzaInterception,
    ClientInfo,
    /// Reply once to a claimed, authenticated inbound IQ request. Requires
    /// `StanzaInterception` to register its handler.
    IqReply,
}

impl PluginCapability {
    pub const fn identifier(self) -> &'static str {
        match self {
            Self::CoreEvents => "events.core.observe",
            Self::Tasks => "tasks.spawn",
            Self::Messaging => "messaging.send",
            Self::Iq => "iq.execute",
            Self::PluginEvents => "events.plugin.publish",
            Self::StanzaInterception => "stanza.intercept",
            Self::ClientInfo => "client.info.read",
            Self::IqReply => "iq.reply",
        }
    }

    const fn bit(self) -> u64 {
        match self {
            Self::CoreEvents => CAP_CORE_EVENTS,
            Self::Tasks => CAP_TASKS,
            Self::Messaging => CAP_MESSAGING,
            Self::Iq => CAP_IQ,
            Self::PluginEvents => CAP_PLUGIN_EVENTS,
            Self::StanzaInterception => CAP_STANZA_INTERCEPTION,
            Self::ClientInfo => CAP_CLIENT_INFO,
            Self::IqReply => CAP_IQ_REPLY,
        }
    }
}

/// Compact set of capabilities requested by one plugin.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct PluginCapabilities(u64);

impl PluginCapabilities {
    pub const NONE: Self = Self(0);

    pub const fn with(self, capability: PluginCapability) -> Self {
        Self(self.0 | capability.bit())
    }

    pub const fn contains(self, capability: PluginCapability) -> bool {
        self.0 & capability.bit() != 0
    }
}

/// Deadlines applied by the native plugin host.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PluginHostConfig {
    install_timeout: Duration,
    callback_timeout: Duration,
    task_drain_timeout: Duration,
}

impl PluginHostConfig {
    pub const fn new() -> Self {
        Self {
            install_timeout: DEFAULT_PLUGIN_INSTALL_TIMEOUT,
            callback_timeout: DEFAULT_PLUGIN_CALLBACK_TIMEOUT,
            task_drain_timeout: DEFAULT_PLUGIN_TASK_DRAIN_TIMEOUT,
        }
    }

    pub const fn with_install_timeout(mut self, timeout: Duration) -> Self {
        self.install_timeout = timeout;
        self
    }

    pub const fn with_callback_timeout(mut self, timeout: Duration) -> Self {
        self.callback_timeout = timeout;
        self
    }

    pub const fn with_task_drain_timeout(mut self, timeout: Duration) -> Self {
        self.task_drain_timeout = timeout;
        self
    }

    pub const fn install_timeout(self) -> Duration {
        self.install_timeout
    }

    pub const fn callback_timeout(self) -> Duration {
        self.callback_timeout
    }

    pub const fn task_drain_timeout(self) -> Duration {
        self.task_drain_timeout
    }
}

impl Default for PluginHostConfig {
    fn default() -> Self {
        Self::new()
    }
}

/// Build-time declaration used for validation, ordering, and future foreign adapters.
#[derive(Debug, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub struct PluginManifest {
    id: String,
    version: String,
    dependencies: Vec<String>,
    capabilities: PluginCapabilities,
}

impl PluginManifest {
    pub fn new(id: impl Into<String>, version: impl Into<String>) -> Self {
        Self {
            id: id.into(),
            version: version.into(),
            dependencies: Vec::new(),
            capabilities: PluginCapabilities::NONE,
        }
    }

    pub fn with_dependency(mut self, plugin_id: impl Into<String>) -> Self {
        self.dependencies.push(plugin_id.into());
        self
    }

    pub const fn with_capability(mut self, capability: PluginCapability) -> Self {
        self.capabilities = self.capabilities.with(capability);
        self
    }

    pub fn id(&self) -> &str {
        &self.id
    }

    pub fn version(&self) -> &str {
        &self.version
    }

    pub fn dependencies(&self) -> &[String] {
        &self.dependencies
    }

    pub const fn capabilities(&self) -> PluginCapabilities {
        self.capabilities
    }
}

/// Target-correct future returned by native plugin entry points.
pub type PluginFuture<'a, T> = BoxFuture<'a, T>;

/// A trusted native plugin installed exactly once while the client is still inert.
pub trait ClientPlugin: MaybeSendSync + 'static {
    type Api: MaybeSendSync + 'static;

    fn manifest(&self) -> PluginManifest;

    fn install(&self, context: PluginContext) -> PluginFuture<'_, anyhow::Result<Arc<Self::Api>>>;

    fn on_ready(&self, _scope: PluginConnectionScope) -> PluginFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }

    fn on_closed(&self, _scope: PluginConnectionScope) -> PluginFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }

    fn shutdown(&self) -> PluginFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }
}

/// A trusted plugin instance identified only by its manifest ID.
pub trait UntypedClientPlugin: MaybeSendSync + 'static {
    fn manifest(&self) -> PluginManifest;

    fn install(&self, context: PluginContext) -> PluginFuture<'_, anyhow::Result<()>>;

    fn on_ready(&self, _scope: PluginConnectionScope) -> PluginFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }

    fn on_closed(&self, _scope: PluginConnectionScope) -> PluginFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }

    fn shutdown(&self) -> PluginFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }
}

/// Manifest validation or dependency-ordering failure.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum PluginPlanError {
    #[error("plugin {plugin_type} panicked while producing its manifest")]
    ManifestPanicked { plugin_type: &'static str },
    #[error("invalid plugin id `{id}`")]
    InvalidId { id: String },
    #[error("plugin `{plugin_id}` has an invalid version `{version}`")]
    InvalidVersion { plugin_id: String, version: String },
    #[error("plugin id `{id}` is registered more than once")]
    DuplicateId { id: String },
    #[error("plugin marker type `{plugin_type}` is registered more than once")]
    DuplicateType { plugin_type: &'static str },
    #[error("plugin `{plugin_id}` lists dependency `{dependency}` more than once")]
    DuplicateDependency {
        plugin_id: String,
        dependency: String,
    },
    #[error("plugin `{plugin_id}` requires missing plugin `{dependency}`")]
    MissingDependency {
        plugin_id: String,
        dependency: String,
    },
    #[error("plugin `{plugin_id}` capability {capability:?} requires {required:?}")]
    MissingCapabilityDependency {
        plugin_id: String,
        capability: PluginCapability,
        required: PluginCapability,
    },
    #[error("plugin dependency cycle involves: {plugins:?}")]
    DependencyCycle { plugins: Vec<String> },
}

/// Capability use after the client or plugin scope has ended.
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginResourceError {
    #[error("the client is no longer available")]
    ClientUnavailable,
    #[error("the plugin host has not started yet")]
    NotActive,
    #[error("the plugin scope is shutting down")]
    ShuttingDown,
    #[error("the plugin task capacity is exhausted")]
    TaskCapacityExceeded,
}

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum PluginMessagingError {
    #[error("{0}")]
    Resource(#[from] PluginResourceError),
    #[error("{0}")]
    Send(#[from] crate::send::SendError),
}

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum PluginIqError {
    #[error("{0}")]
    Resource(#[from] PluginResourceError),
    #[error("{0}")]
    Iq(#[from] crate::request::IqError),
}

/// Failure to admit or send one inbound IQ reply.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum PluginIqReplyError {
    #[error("{0}")]
    Resource(#[from] PluginResourceError),
    #[error("the IQ reply capability was not requested")]
    CapabilityRequired,
    #[error("the interceptor did not claim this IQ")]
    NotClaimed,
    #[error("this IQ reply grant has already been used")]
    AlreadyUsed,
    #[error("the IQ's connection has retired")]
    ConnectionRetired,
    #[error("IQ error codes must be between 400 and 599")]
    InvalidErrorCode,
    #[error("IQ error text must not be empty")]
    InvalidErrorText,
    #[error("{0}")]
    Send(#[from] crate::client::ClientError),
}

/// Lifecycle state of one installed plugin.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginState {
    Installing,
    Active,
    ShuttingDown,
    /// The bounded shutdown attempt completed.
    Stopped,
}

impl Default for PluginState {
    fn default() -> Self {
        Self::Installing
    }
}

/// Sticky health derived from cumulative host and event-router failures.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
#[non_exhaustive]
pub enum PluginHealth {
    #[default]
    Healthy,
    Degraded,
}

/// On-demand runtime snapshot for one plugin.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
#[non_exhaustive]
pub struct PluginStats {
    pub plugin_id: String,
    pub state: PluginState,
    pub health: PluginHealth,
    pub callbacks_completed: u64,
    pub callback_failures: u64,
    pub callback_timeouts: u64,
    pub task_drain_timeouts: u64,
    pub task_panics: u64,
    pub core_events_delivered: u64,
    pub core_event_panics: u64,
    pub resource_teardown_panics: u64,
    pub install_tasks: u64,
    pub connection_tasks: u64,
    pub connection_generations: u64,
    pub core_event_subscriptions: u64,
    pub stanza_interceptors: u64,
    pub stanza_interception_panics: u64,
    pub events: Option<PluginEventPublisherStats>,
}

/// On-demand aggregate for the native plugin host.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
#[non_exhaustive]
pub struct PluginHostStats {
    pub terminal: bool,
    pub health: PluginHealth,
    pub upstream_callback_failures: u64,
    pub upstream_callback_timeouts: u64,
    pub plugins: Vec<PluginStats>,
    pub event_router: Option<PluginEventRouterStats>,
}

/// Read-only identity facts exposed to plugins that request `ClientInfo`.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
#[non_exhaustive]
pub struct PluginClientInfo {
    pub jid: Option<Jid>,
    pub push_name: Option<String>,
}

impl PluginClientInfo {
    pub fn new(jid: Option<Jid>, push_name: Option<String>) -> Self {
        Self { jid, push_name }
    }
}

/// Capability-scoped handles passed to one plugin installation.
#[derive(Debug, Clone)]
#[non_exhaustive]
pub struct PluginContext {
    plugin_id: String,
    capabilities: PluginCapabilities,
    client_info: Option<PluginClientInfo>,
}

impl PluginContext {
    pub fn new(plugin_id: impl Into<String>, capabilities: PluginCapabilities) -> Self {
        Self {
            plugin_id: plugin_id.into(),
            capabilities,
            client_info: None,
        }
    }

    pub fn with_client_info(mut self, info: PluginClientInfo) -> Self {
        self.client_info = Some(info);
        self
    }

    pub fn plugin_id(&self) -> &str {
        &self.plugin_id
    }

    pub const fn capabilities(&self) -> PluginCapabilities {
        self.capabilities
    }

    pub const fn has_capability(&self, capability: PluginCapability) -> bool {
        self.capabilities.contains(capability)
    }

    pub fn client_info(&self) -> Option<&PluginClientInfo> {
        self.client_info.as_ref()
    }
}

/// One connection generation exposed to a plugin lifecycle callback.
#[derive(Debug, Clone)]
pub struct PluginConnectionScope {
    scope: ConnectionScope,
}

impl PluginConnectionScope {
    pub fn new(scope: ConnectionScope) -> Self {
        Self { scope }
    }

    pub fn generation(&self) -> u64 {
        self.scope.generation()
    }

    pub fn state(&self) -> ConnectionScopeState {
        self.scope.state()
    }

    pub fn is_cancelled(&self) -> bool {
        self.scope.is_cancelled()
    }

    pub fn cancellation_signal(&self) -> ShutdownSignal {
        self.scope.cancellation_signal()
    }
}

/// Synchronous plugin registry used for manifest validation, lifecycle state
/// and event-router wiring. Runtime installation remains a client concern.
pub struct PluginHost {
    config: PluginHostConfig,
    manifests: Vec<PluginManifest>,
    states: HashMap<String, PluginState>,
    router: PluginEventRouter,
    terminal: bool,
}

impl PluginHost {
    pub fn new(config: PluginHostConfig) -> Self {
        Self {
            config,
            manifests: Vec::new(),
            states: HashMap::new(),
            router: PluginEventRouter::default(),
            terminal: false,
        }
    }

    pub fn config(&self) -> PluginHostConfig {
        self.config
    }

    pub fn manifests(&self) -> &[PluginManifest] {
        &self.manifests
    }

    /// Validate and add one manifest transactionally.
    pub fn register(&mut self, manifest: PluginManifest) -> Result<(), PluginPlanError> {
        self.register_all([manifest])
    }

    /// Validate and add a batch of manifests atomically. This is the only way
    /// to register a dependency cycle for validation, since incremental
    /// registration rejects missing dependencies.
    pub fn register_all(
        &mut self,
        manifests: impl IntoIterator<Item = PluginManifest>,
    ) -> Result<(), PluginPlanError> {
        if self.terminal {
            return Err(PluginPlanError::ManifestPanicked {
                plugin_type: "plugin host is terminal",
            });
        }
        let start = self.manifests.len();
        self.manifests.extend(manifests);
        match validate_plan(&self.manifests) {
            Ok(()) => {
                let new_ids: Vec<String> = self.manifests[start..]
                    .iter()
                    .map(|manifest| manifest.id().to_string())
                    .collect();
                for id in new_ids {
                    self.states.insert(id.clone(), PluginState::Installing);
                    self.router.register_publisher(id);
                }
                Ok(())
            }
            Err(error) => {
                self.manifests.truncate(start);
                Err(error)
            }
        }
    }

    pub fn activate(&mut self, plugin_id: &str) -> bool {
        match self.states.get_mut(plugin_id) {
            Some(state @ (PluginState::Installing | PluginState::Active)) => {
                *state = PluginState::Active;
                true
            }
            _ => false,
        }
    }

    pub fn stop(&mut self, plugin_id: &str) -> bool {
        match self.states.get_mut(plugin_id) {
            Some(state) if *state != PluginState::Stopped => {
                *state = PluginState::Stopped;
                true
            }
            _ => false,
        }
    }

    pub fn state(&self, plugin_id: &str) -> Option<PluginState> {
        self.states.get(plugin_id).copied()
    }

    pub fn event_router(&self) -> PluginEventRouter {
        self.router.clone()
    }

    pub fn stats(&self) -> PluginHostStats {
        let plugins = self
            .manifests
            .iter()
            .map(|manifest| {
                let state = self.states.get(manifest.id()).copied().unwrap_or_default();
                PluginStats {
                    plugin_id: manifest.id().to_string(),
                    state,
                    health: PluginHealth::Healthy,
                    events: self.router.publisher_stats(manifest.id()),
                    ..PluginStats::default()
                }
            })
            .collect();
        PluginHostStats {
            terminal: self.terminal,
            health: PluginHealth::Healthy,
            upstream_callback_failures: 0,
            upstream_callback_timeouts: 0,
            plugins,
            event_router: Some(self.router.stats()),
        }
    }

    /// Stop every plugin and close the event router.
    pub fn shutdown(&mut self) {
        self.terminal = true;
        for state in self.states.values_mut() {
            *state = PluginState::Stopped;
        }
        self.router.close();
    }
}

impl Default for PluginHost {
    fn default() -> Self {
        Self::new(PluginHostConfig::default())
    }
}

fn validate_manifest(manifest: &PluginManifest) -> Result<(), PluginPlanError> {
    if !valid_plugin_id(&manifest.id) {
        return Err(PluginPlanError::InvalidId {
            id: manifest.id.clone(),
        });
    }
    if manifest.version.is_empty()
        || manifest.version.len() > 64
        || !manifest.version.bytes().all(|byte| byte.is_ascii_graphic())
    {
        return Err(PluginPlanError::InvalidVersion {
            plugin_id: manifest.id.clone(),
            version: manifest.version.clone(),
        });
    }
    if manifest.capabilities.contains(PluginCapability::IqReply)
        && !manifest
            .capabilities
            .contains(PluginCapability::StanzaInterception)
    {
        return Err(PluginPlanError::MissingCapabilityDependency {
            plugin_id: manifest.id.clone(),
            capability: PluginCapability::IqReply,
            required: PluginCapability::StanzaInterception,
        });
    }
    Ok(())
}

fn validate_plan(manifests: &[PluginManifest]) -> Result<(), PluginPlanError> {
    let mut ids = HashSet::new();
    for manifest in manifests {
        validate_manifest(manifest)?;
        if !ids.insert(manifest.id.clone()) {
            return Err(PluginPlanError::DuplicateId {
                id: manifest.id.clone(),
            });
        }
    }

    for manifest in manifests {
        let mut dependencies = HashSet::new();
        for dependency in &manifest.dependencies {
            if !dependencies.insert(dependency.clone()) {
                return Err(PluginPlanError::DuplicateDependency {
                    plugin_id: manifest.id.clone(),
                    dependency: dependency.clone(),
                });
            }
            if !ids.contains(dependency) {
                return Err(PluginPlanError::MissingDependency {
                    plugin_id: manifest.id.clone(),
                    dependency: dependency.clone(),
                });
            }
        }
    }

    let by_id: HashMap<&str, &PluginManifest> = manifests
        .iter()
        .map(|manifest| (manifest.id.as_str(), manifest))
        .collect();
    let mut done = HashSet::new();
    for manifest in manifests {
        let mut visiting = Vec::new();
        if let Err(cycle) = visit(manifest.id.as_str(), &by_id, &mut visiting, &mut done) {
            return Err(PluginPlanError::DependencyCycle { plugins: cycle });
        }
    }
    Ok(())
}

fn visit<'a>(
    id: &'a str,
    by_id: &HashMap<&'a str, &'a PluginManifest>,
    visiting: &mut Vec<&'a str>,
    done: &mut HashSet<&'a str>,
) -> Result<(), Vec<String>> {
    if done.contains(id) {
        return Ok(());
    }
    if let Some(position) = visiting.iter().position(|entry| *entry == id) {
        return Err(visiting[position..]
            .iter()
            .map(|id| id.to_string())
            .collect());
    }
    visiting.push(id);
    if let Some(manifest) = by_id.get(id) {
        for dependency in &manifest.dependencies {
            visit(dependency, by_id, visiting, done)?;
        }
    }
    visiting.pop();
    done.insert(id);
    Ok(())
}

pub(crate) fn valid_plugin_id(id: &str) -> bool {
    if id.is_empty() || id.len() > 128 {
        return false;
    }
    let mut previous_separator = true;
    for byte in id.bytes() {
        let separator = matches!(byte, b'.' | b'-' | b'_');
        if separator {
            if previous_separator {
                return false;
            }
        } else if !byte.is_ascii_lowercase() && !byte.is_ascii_digit() {
            return false;
        }
        previous_separator = separator;
    }
    !previous_separator && id.as_bytes()[0].is_ascii_lowercase()
}
