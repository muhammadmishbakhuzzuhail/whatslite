#![cfg(feature = "bundled-sqlite")]
//! Phase 5 SQLite storage profile contract.

use wa_rs_sqlite_storage::SqliteStore;

#[tokio::test]
async fn sqlite_store_opens_and_scopes_devices() {
    let path = std::env::temp_dir().join(format!(
        "wa-rs-parity-storage-{}-{}.db",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    let url = format!("sqlite://{}", path.display());

    let store = SqliteStore::new(&url).await.expect("open store");
    assert_eq!(store.device_id(), 1);
    let device = SqliteStore::new_for_device(&url, 7)
        .await
        .expect("open device store");
    assert_eq!(device.device_id(), 7);

    drop(device);
    drop(store);
    let _ = std::fs::remove_file(&path);
    let _ = std::fs::remove_file(format!("{}-wal", path.display()));
    let _ = std::fs::remove_file(format!("{}-shm", path.display()));
}
