//! Phase 2 media contract.

use std::time::Duration;

use wa_rs::media::{
    AudioOptions, DocumentOptions, DownloadLimits, DownloadPolicyError, ImageOptions,
    MediaByteDomain, MediaReupload, MediaReuploadError, MediaReuploadPhase, UploadOptions,
    VideoOptions,
};

#[test]
fn media_options_default_and_fields() {
    assert!(ImageOptions::default().caption.is_none());
    assert!(!VideoOptions::default().gif_playback);
    assert!(DocumentOptions::default().file_name.is_none());
    assert!(!AudioOptions::default().ptt);
    assert!(UploadOptions::default().filename.is_none());
}

#[test]
fn download_limits_validate_ranges_and_size() {
    let limits = DownloadLimits {
        max_bytes: 10,
        timeout: Duration::from_secs(1),
    };
    assert_eq!(
        limits.validate(MediaByteDomain::Range { start: 5, end: 4 }, None),
        Err(DownloadPolicyError::InvalidRange)
    );
    assert_eq!(
        limits.validate(MediaByteDomain::Full, Some(11)),
        Err(DownloadPolicyError::TooLarge)
    );
    assert!(limits.validate(MediaByteDomain::Full, Some(10)).is_ok());
    assert!(
        limits
            .validate(MediaByteDomain::Head { length: 4 }, None)
            .is_ok()
    );
}

#[test]
fn media_reupload_advances_and_rejects_unknown_ids() {
    let mut reupload = MediaReupload::new();
    reupload.request("MSG1");
    assert_eq!(reupload.phase("MSG1"), Some(MediaReuploadPhase::Requested));
    reupload
        .advance("MSG1", MediaReuploadPhase::Uploading)
        .unwrap();
    assert_eq!(reupload.phase("MSG1"), Some(MediaReuploadPhase::Uploading));
    assert_eq!(
        reupload.advance("NOPE", MediaReuploadPhase::Completed),
        Err(MediaReuploadError::NotPending)
    );
}
