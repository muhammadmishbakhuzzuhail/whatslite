//! Media option and streaming contracts shared by download/upload paths.

use std::time::Duration;

/// Per-image send options.
#[derive(Debug, Clone, Default)]
pub struct ImageOptions {
    pub caption: Option<String>,
    pub view_once: bool,
}

/// Per-video send options.
#[derive(Debug, Clone, Default)]
pub struct VideoOptions {
    pub caption: Option<String>,
    pub gif_playback: bool,
}

/// Per-document send options.
#[derive(Debug, Clone, Default)]
pub struct DocumentOptions {
    pub file_name: Option<String>,
    pub caption: Option<String>,
}

/// Per-audio send options.
#[derive(Debug, Clone, Default)]
pub struct AudioOptions {
    pub ptt: bool,
    pub seconds: Option<u32>,
}

/// Byte domain a download is allowed to read.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum MediaByteDomain {
    Full,
    Head { length: u64 },
    Range { start: u64, end: u64 },
}

/// Download limits and policy.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DownloadLimits {
    pub max_bytes: u64,
    pub timeout: Duration,
}

impl Default for DownloadLimits {
    fn default() -> Self {
        Self {
            max_bytes: 64 * 1024 * 1024,
            timeout: Duration::from_secs(60),
        }
    }
}

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum DownloadPolicyError {
    #[error("download exceeds the configured byte limit")]
    TooLarge,
    #[error("invalid byte range")]
    InvalidRange,
}

impl DownloadLimits {
    pub fn validate(
        &self,
        domain: MediaByteDomain,
        content_length: Option<u64>,
    ) -> Result<(), DownloadPolicyError> {
        match domain {
            MediaByteDomain::Range { start, end } if start > end => {
                return Err(DownloadPolicyError::InvalidRange);
            }
            _ => {}
        }
        if content_length.is_some_and(|len| len > self.max_bytes) {
            return Err(DownloadPolicyError::TooLarge);
        }
        Ok(())
    }
}

/// Parameters for one download.
#[derive(Debug, Clone)]
pub struct DownloadParams {
    pub url: String,
    pub domain: MediaByteDomain,
    pub limits: DownloadLimits,
}

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum MediaDownloadError {
    #[error("download policy rejected the request: {0}")]
    Policy(#[from] DownloadPolicyError),
    #[error("download failed: {0}")]
    Transport(String),
}

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum ClientDownloadError {
    #[error(transparent)]
    Media(#[from] MediaDownloadError),
    #[error("client is not connected")]
    NotConnected,
}

/// Streaming downloader seam.
#[async_trait::async_trait]
pub trait MediaDownloader: Send + Sync {
    async fn download(&self, params: DownloadParams) -> Result<Vec<u8>, MediaDownloadError>;
}

/// Upload options.
#[derive(Debug, Clone, Default)]
pub struct UploadOptions {
    pub filename: Option<String>,
    pub mimetype: Option<String>,
}

/// Reupload lifecycle.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum MediaReuploadPhase {
    Requested,
    Uploading,
    Completed,
    Failed,
}

#[derive(Debug, Clone)]
pub struct MediaReuploadRequest {
    pub message_id: String,
    pub phase: MediaReuploadPhase,
}

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum MediaReuploadError {
    #[error("reupload request is not pending")]
    NotPending,
    #[error("reupload failed: {0}")]
    Failed(String),
}

/// Small state machine for media reupload requests.
#[derive(Debug, Default)]
pub struct MediaReupload {
    requests: Vec<MediaReuploadRequest>,
}

impl MediaReupload {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn request(&mut self, message_id: impl Into<String>) {
        self.requests.push(MediaReuploadRequest {
            message_id: message_id.into(),
            phase: MediaReuploadPhase::Requested,
        });
    }

    pub fn advance(
        &mut self,
        message_id: &str,
        phase: MediaReuploadPhase,
    ) -> Result<(), MediaReuploadError> {
        let request = self
            .requests
            .iter_mut()
            .find(|request| request.message_id == message_id)
            .ok_or(MediaReuploadError::NotPending)?;
        request.phase = phase;
        Ok(())
    }

    pub fn phase(&self, message_id: &str) -> Option<MediaReuploadPhase> {
        self.requests
            .iter()
            .find(|request| request.message_id == message_id)
            .map(|request| request.phase)
    }
}
