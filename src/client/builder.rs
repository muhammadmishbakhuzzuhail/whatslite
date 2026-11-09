//! Client construction seam. The existing `Client::new` remains the
//! compatibility constructor; this builder adds the upstream option surface.

use std::sync::Arc;

use tokio::sync::mpsc;

use crate::client::Client;
use crate::http::HttpClient;
use crate::store::persistence_manager::PersistenceManager;
use crate::sync_task::MajorSyncTask;
use crate::transport::TransportFactory;
use crate::types::connect_admission::ConnectAdmission;
use crate::types::decrypt_failure_admission::DecryptFailureAdmission;
use crate::types::durability_hook::{HistorySyncCaptureHook, InboundDurabilityHook};
use crate::types::history_sync_admission::HistorySyncAdmission;
use crate::types::retry_admission::RetryAdmission;

#[derive(Debug, Clone, Default)]
pub struct ClientOptions {
    /// `None` keeps automatic version selection.
    pub override_version: Option<(u32, u32, u32)>,
    /// Acknowledge history-sync notifications without processing payloads.
    pub skip_history_sync: bool,
}

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum ClientBuilderError {
    #[error("persistence manager is required")]
    MissingPersistence,
    #[error("transport factory is required")]
    MissingTransport,
    #[error("HTTP client is required")]
    MissingHttpClient,
    #[error("unsupported durability backend: {0}")]
    UnsupportedDurabilityBackend(String),
}

pub struct ClientBuilder {
    options: ClientOptions,
    persistence: Option<Arc<PersistenceManager>>,
    transport: Option<Arc<dyn TransportFactory>>,
    http: Option<Arc<dyn HttpClient>>,
    connect_admission: Option<Arc<dyn ConnectAdmission>>,
    decrypt_failure_admission: Option<Arc<dyn DecryptFailureAdmission>>,
    retry_admission: Option<Arc<dyn RetryAdmission>>,
    history_sync_admission: Option<Arc<dyn HistorySyncAdmission>>,
    history_sync_capture_hook: Option<Arc<dyn HistorySyncCaptureHook>>,
    inbound_durability_hook: Option<Arc<dyn InboundDurabilityHook>>,
}

impl Default for ClientBuilder {
    fn default() -> Self {
        Self::new()
    }
}

impl ClientBuilder {
    pub fn new() -> Self {
        Self {
            options: ClientOptions::default(),
            persistence: None,
            transport: None,
            http: None,
            connect_admission: None,
            decrypt_failure_admission: None,
            retry_admission: None,
            history_sync_admission: None,
            history_sync_capture_hook: None,
            inbound_durability_hook: None,
        }
    }

    pub fn with_options(mut self, options: ClientOptions) -> Self {
        self.options = options;
        self
    }

    pub fn options(&self) -> &ClientOptions {
        &self.options
    }

    pub fn with_persistence_manager(mut self, persistence: Arc<PersistenceManager>) -> Self {
        self.persistence = Some(persistence);
        self
    }

    pub fn with_transport_factory<T>(mut self, transport: T) -> Self
    where
        T: TransportFactory + 'static,
    {
        self.transport = Some(Arc::new(transport));
        self
    }

    pub fn with_transport_factory_arc(mut self, transport: Arc<dyn TransportFactory>) -> Self {
        self.transport = Some(transport);
        self
    }

    pub fn with_http_client<H>(mut self, http: H) -> Self
    where
        H: HttpClient + 'static,
    {
        self.http = Some(Arc::new(http));
        self
    }

    pub fn with_http_client_arc(mut self, http: Arc<dyn HttpClient>) -> Self {
        self.http = Some(http);
        self
    }

    pub fn with_version_override(mut self, version: (u32, u32, u32)) -> Self {
        self.options.override_version = Some(version);
        self
    }

    pub fn with_skip_history_sync(mut self, skip: bool) -> Self {
        self.options.skip_history_sync = skip;
        self
    }

    pub fn with_connect_admission_arc(mut self, admission: Arc<dyn ConnectAdmission>) -> Self {
        self.connect_admission = Some(admission);
        self
    }

    pub fn with_connect_admission<A: ConnectAdmission + 'static>(mut self, admission: A) -> Self {
        self.connect_admission = Some(Arc::new(admission));
        self
    }

    pub fn with_decrypt_failure_admission_arc(
        mut self,
        admission: Arc<dyn DecryptFailureAdmission>,
    ) -> Self {
        self.decrypt_failure_admission = Some(admission);
        self
    }

    pub fn with_decrypt_failure_admission<A: DecryptFailureAdmission + 'static>(
        mut self,
        admission: A,
    ) -> Self {
        self.decrypt_failure_admission = Some(Arc::new(admission));
        self
    }

    pub fn with_retry_admission_arc(mut self, admission: Arc<dyn RetryAdmission>) -> Self {
        self.retry_admission = Some(admission);
        self
    }

    pub fn with_retry_admission<A: RetryAdmission + 'static>(mut self, admission: A) -> Self {
        self.retry_admission = Some(Arc::new(admission));
        self
    }

    pub fn with_history_sync_admission_arc(
        mut self,
        admission: Arc<dyn HistorySyncAdmission>,
    ) -> Self {
        self.history_sync_admission = Some(admission);
        self
    }

    pub fn with_history_sync_capture_hook_arc(
        mut self,
        hook: Arc<dyn HistorySyncCaptureHook>,
    ) -> Self {
        self.history_sync_capture_hook = Some(hook);
        self
    }

    pub fn with_inbound_durability_hook_arc(
        mut self,
        hook: Arc<dyn InboundDurabilityHook>,
    ) -> Self {
        self.inbound_durability_hook = Some(hook);
        self
    }

    pub async fn build(self) -> Result<ClientBuild, ClientBuilderError> {
        let persistence = self
            .persistence
            .ok_or(ClientBuilderError::MissingPersistence)?;
        let transport = self.transport.ok_or(ClientBuilderError::MissingTransport)?;
        let http = self.http.ok_or(ClientBuilderError::MissingHttpClient)?;

        let (client, sync_rx) =
            Client::new(persistence, transport, http, self.options.override_version).await;
        if self.options.skip_history_sync {
            client.set_skip_history_sync(true);
        }
        Ok(ClientBuild { client, sync_rx })
    }
}

pub struct ClientBuild {
    client: Arc<Client>,
    sync_rx: mpsc::Receiver<MajorSyncTask>,
}

impl ClientBuild {
    pub fn into_client(self) -> Arc<Client> {
        self.client
    }

    pub fn client(&self) -> &Arc<Client> {
        &self.client
    }

    pub fn into_parts(self) -> (Arc<Client>, mpsc::Receiver<MajorSyncTask>) {
        (self.client, self.sync_rx)
    }
}
