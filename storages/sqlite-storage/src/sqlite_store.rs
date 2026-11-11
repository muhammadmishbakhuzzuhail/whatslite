use crate::schema::*;
use async_trait::async_trait;
use diesel::prelude::*;
use diesel::r2d2::{ConnectionManager, Pool};
use diesel::sql_query;
use diesel::sqlite::SqliteConnection;
use diesel::upsert::excluded;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use log::warn;
use std::sync::Arc;
use wacore::appstate::hash::HashState;
use wacore::appstate::processor::AppStateMutationMAC;
use wacore::libsignal::protocol::{KeyPair, PrivateKey, PublicKey};
use wacore::store::Device as CoreDevice;
use wacore::store::error::{Result, StoreError};
use wacore::store::traits::*;
use wacore_binary::jid::Jid;
use waproto::buffa::Message as _;
use waproto::whatsapp as wa;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");

/// One upsert statement per chunk. Seven columns per row keeps a chunk well
/// under SQLite's default 999-variable limit.
const MSG_SECRET_INSERT_CHUNK_SIZE: usize = 100;

type SqlitePool = Pool<ConnectionManager<SqliteConnection>>;
type DeviceRow = (
    i32,
    String,
    String,
    i32,
    Vec<u8>,
    Vec<u8>,
    Vec<u8>,
    i32,
    Vec<u8>,
    Vec<u8>,
    Option<Vec<u8>>,
    String,
    i32,
    i32,
    i64,
    i64,
    Option<Vec<u8>>,
    Option<String>,
    Option<Vec<u8>>,
);

#[derive(Clone)]
pub struct SqliteStore {
    pub(crate) pool: SqlitePool,
    pub(crate) db_semaphore: Arc<tokio::sync::Semaphore>,
    pub(crate) database_path: String,
    device_id: i32,
}

#[derive(Debug, Clone, Copy)]
struct ConnectionOptions;

impl diesel::r2d2::CustomizeConnection<SqliteConnection, diesel::r2d2::Error>
    for ConnectionOptions
{
    fn on_acquire(
        &self,
        conn: &mut SqliteConnection,
    ) -> std::result::Result<(), diesel::r2d2::Error> {
        diesel::sql_query("PRAGMA busy_timeout = 30000;")
            .execute(conn)
            .map_err(diesel::r2d2::Error::QueryError)?;
        diesel::sql_query("PRAGMA synchronous = NORMAL;")
            .execute(conn)
            .map_err(diesel::r2d2::Error::QueryError)?;
        diesel::sql_query("PRAGMA cache_size = 512;")
            .execute(conn)
            .map_err(diesel::r2d2::Error::QueryError)?;
        diesel::sql_query("PRAGMA temp_store = memory;")
            .execute(conn)
            .map_err(diesel::r2d2::Error::QueryError)?;
        diesel::sql_query("PRAGMA foreign_keys = ON;")
            .execute(conn)
            .map_err(diesel::r2d2::Error::QueryError)?;
        Ok(())
    }
}

fn parse_database_path(database_url: &str) -> Result<String> {
    // Reject in-memory databases
    if database_url == ":memory:" {
        return Err(StoreError::Database(
            "Snapshot not supported for in-memory databases".to_string(),
        ));
    }

    // Strip query string and fragment
    let path = database_url
        .split(['?', '#'])
        .next()
        .unwrap_or(database_url);

    // Remove sqlite:// prefix if present
    let path = path.trim_start_matches("sqlite://");

    // Check if the resulting path looks like an in-memory marker
    if path == ":memory:" || path.starts_with(":memory:?") {
        return Err(StoreError::Database(
            "Snapshot not supported for in-memory databases".to_string(),
        ));
    }

    Ok(path.to_string())
}

impl SqliteStore {
    pub async fn new(database_url: &str) -> std::result::Result<Self, StoreError> {
        let manager = ConnectionManager::<SqliteConnection>::new(database_url);

        let pool_size = 2;

        let pool = Pool::builder()
            .max_size(pool_size)
            .connection_customizer(Box::new(ConnectionOptions))
            .build(manager)
            .map_err(|e| StoreError::Connection(e.to_string()))?;

        let pool_clone = pool.clone();
        tokio::task::spawn_blocking(move || -> std::result::Result<(), StoreError> {
            let mut conn = pool_clone
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            diesel::sql_query("PRAGMA journal_mode = WAL;")
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;

            conn.run_pending_migrations(MIGRATIONS)
                .map_err(|e| StoreError::Migration(e.to_string()))?;

            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;

        let database_path = parse_database_path(database_url)?;

        Ok(Self {
            pool,
            db_semaphore: Arc::new(tokio::sync::Semaphore::new(1)),
            database_path,
            device_id: 1,
        })
    }

    pub async fn new_for_device(
        database_url: &str,
        device_id: i32,
    ) -> std::result::Result<Self, StoreError> {
        let mut store = Self::new(database_url).await?;
        store.device_id = device_id;
        Ok(store)
    }

    pub fn device_id(&self) -> i32 {
        self.device_id
    }

    async fn with_semaphore<F, T>(&self, f: F) -> Result<T>
    where
        F: FnOnce() -> Result<T> + Send + 'static,
        T: Send + 'static,
    {
        let permit = self
            .db_semaphore
            .clone()
            .acquire_owned()
            .await
            .map_err(|e| StoreError::Database(format!("Semaphore error: {}", e)))?;
        let result = tokio::task::spawn_blocking(move || {
            let res = f();
            drop(permit);
            res
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(result)
    }

    fn serialize_keypair(&self, key_pair: &KeyPair) -> Result<Vec<u8>> {
        let mut bytes = Vec::with_capacity(64);
        bytes.extend_from_slice(key_pair.private_key.serialize());
        bytes.extend_from_slice(key_pair.public_key.public_key_bytes());
        Ok(bytes)
    }

    fn deserialize_keypair(&self, bytes: &[u8]) -> Result<KeyPair> {
        if bytes.len() != 64 {
            return Err(StoreError::Serialization(format!(
                "Invalid KeyPair length: {}",
                bytes.len()
            )));
        }

        let private_key = PrivateKey::deserialize(&bytes[0..32])
            .map_err(|e| StoreError::Serialization(e.to_string()))?;
        let public_key = PublicKey::from_djb_public_key_bytes(&bytes[32..64])
            .map_err(|e| StoreError::Serialization(e.to_string()))?;

        Ok(KeyPair::new(public_key, private_key))
    }

    pub async fn save_device_data_for_device(
        &self,
        device_id: i32,
        device_data: &CoreDevice,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let noise_key_data = self.serialize_keypair(&device_data.noise_key)?;
        let identity_key_data = self.serialize_keypair(&device_data.identity_key)?;
        let signed_pre_key_data = self.serialize_keypair(&device_data.signed_pre_key)?;
        let account_data = device_data
            .account
            .as_ref()
            .map(|account| account.encode_to_vec());
        let registration_id = device_data.registration_id as i32;
        let signed_pre_key_id = device_data.signed_pre_key_id as i32;
        let signed_pre_key_signature: Vec<u8> = device_data.signed_pre_key_signature.to_vec();
        let adv_secret_key: Vec<u8> = device_data.adv_secret_key.to_vec();
        let push_name = device_data.push_name.clone();
        let app_version_primary = device_data.app_version_primary as i32;
        let app_version_secondary = device_data.app_version_secondary as i32;
        let app_version_tertiary = device_data.app_version_tertiary as i64;
        let app_version_last_fetched_ms = device_data.app_version_last_fetched_ms;
        let edge_routing_info = device_data.edge_routing_info.clone();
        let props_hash = device_data.props_hash.clone();
        let server_cert_chain: Option<Vec<u8>> = device_data
            .server_cert_chain
            .as_ref()
            .map(|chain| {
                bincode::serde::encode_to_vec(chain, bincode::config::standard())
                    .map_err(|e| StoreError::Serialization(e.to_string()))
            })
            .transpose()?;
        let new_lid = device_data
            .lid
            .as_ref()
            .map(|j| j.to_string())
            .unwrap_or_default();
        let new_pn = device_data
            .pn
            .as_ref()
            .map(|j| j.to_string())
            .unwrap_or_default();

        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            diesel::insert_into(device::table)
                .values((
                    device::id.eq(device_id),
                    device::lid.eq(&new_lid),
                    device::pn.eq(&new_pn),
                    device::registration_id.eq(registration_id),
                    device::noise_key.eq(&noise_key_data),
                    device::identity_key.eq(&identity_key_data),
                    device::signed_pre_key.eq(&signed_pre_key_data),
                    device::signed_pre_key_id.eq(signed_pre_key_id),
                    device::signed_pre_key_signature.eq(&signed_pre_key_signature[..]),
                    device::adv_secret_key.eq(&adv_secret_key[..]),
                    device::account.eq(account_data.clone()),
                    device::push_name.eq(&push_name),
                    device::app_version_primary.eq(app_version_primary),
                    device::app_version_secondary.eq(app_version_secondary),
                    device::app_version_tertiary.eq(app_version_tertiary),
                    device::app_version_last_fetched_ms.eq(app_version_last_fetched_ms),
                    device::edge_routing_info.eq(edge_routing_info.clone()),
                    device::props_hash.eq(props_hash.clone()),
                    device::server_cert_chain.eq(server_cert_chain.clone()),
                ))
                .on_conflict(device::id)
                .do_update()
                .set((
                    device::lid.eq(&new_lid),
                    device::pn.eq(&new_pn),
                    device::registration_id.eq(registration_id),
                    device::noise_key.eq(&noise_key_data),
                    device::identity_key.eq(&identity_key_data),
                    device::signed_pre_key.eq(&signed_pre_key_data),
                    device::signed_pre_key_id.eq(signed_pre_key_id),
                    device::signed_pre_key_signature.eq(&signed_pre_key_signature[..]),
                    device::adv_secret_key.eq(&adv_secret_key[..]),
                    device::account.eq(account_data.clone()),
                    device::push_name.eq(&push_name),
                    device::app_version_primary.eq(app_version_primary),
                    device::app_version_secondary.eq(app_version_secondary),
                    device::app_version_tertiary.eq(app_version_tertiary),
                    device::app_version_last_fetched_ms.eq(app_version_last_fetched_ms),
                    device::edge_routing_info.eq(edge_routing_info),
                    device::props_hash.eq(props_hash),
                    device::server_cert_chain.eq(server_cert_chain),
                ))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;

            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;

        Ok(())
    }

    pub async fn create_new_device(&self) -> Result<i32> {
        use crate::schema::device;

        let pool = self.pool.clone();
        tokio::task::spawn_blocking(move || -> Result<i32> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            let new_device = wacore::store::Device::new();

            let noise_key_data = {
                let mut bytes = Vec::with_capacity(64);
                bytes.extend_from_slice(new_device.noise_key.private_key.serialize());
                bytes.extend_from_slice(new_device.noise_key.public_key.public_key_bytes());
                bytes
            };
            let identity_key_data = {
                let mut bytes = Vec::with_capacity(64);
                bytes.extend_from_slice(new_device.identity_key.private_key.serialize());
                bytes.extend_from_slice(new_device.identity_key.public_key.public_key_bytes());
                bytes
            };
            let signed_pre_key_data = {
                let mut bytes = Vec::with_capacity(64);
                bytes.extend_from_slice(new_device.signed_pre_key.private_key.serialize());
                bytes.extend_from_slice(new_device.signed_pre_key.public_key.public_key_bytes());
                bytes
            };

            diesel::insert_into(device::table)
                .values((
                    device::lid.eq(""),
                    device::pn.eq(""),
                    device::registration_id.eq(new_device.registration_id as i32),
                    device::noise_key.eq(&noise_key_data),
                    device::identity_key.eq(&identity_key_data),
                    device::signed_pre_key.eq(&signed_pre_key_data),
                    device::signed_pre_key_id.eq(new_device.signed_pre_key_id as i32),
                    device::signed_pre_key_signature.eq(&new_device.signed_pre_key_signature[..]),
                    device::adv_secret_key.eq(&new_device.adv_secret_key[..]),
                    device::account.eq(None::<Vec<u8>>),
                    device::push_name.eq(&new_device.push_name),
                    device::app_version_primary.eq(new_device.app_version_primary as i32),
                    device::app_version_secondary.eq(new_device.app_version_secondary as i32),
                    device::app_version_tertiary.eq(new_device.app_version_tertiary as i64),
                    device::app_version_last_fetched_ms.eq(new_device.app_version_last_fetched_ms),
                    device::edge_routing_info.eq(None::<Vec<u8>>),
                    device::props_hash.eq(None::<String>),
                    device::server_cert_chain.eq(None::<Vec<u8>>),
                ))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;

            use diesel::sql_types::Integer;

            #[derive(QueryableByName)]
            struct LastInsertedId {
                #[diesel(sql_type = Integer)]
                last_insert_rowid: i32,
            }

            let device_id: i32 = sql_query("SELECT last_insert_rowid() as last_insert_rowid")
                .get_result::<LastInsertedId>(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?
                .last_insert_rowid;

            Ok(device_id)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    pub async fn device_exists(&self, device_id: i32) -> Result<bool> {
        use crate::schema::device;

        let pool = self.pool.clone();
        tokio::task::spawn_blocking(move || -> Result<bool> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            let count: i64 = device::table
                .filter(device::id.eq(device_id))
                .count()
                .get_result(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;

            Ok(count > 0)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    pub async fn load_device_data_for_device(&self, device_id: i32) -> Result<Option<CoreDevice>> {
        use crate::schema::device;

        let pool = self.pool.clone();
        let row = tokio::task::spawn_blocking(move || -> Result<Option<DeviceRow>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let result = device::table
                .filter(device::id.eq(device_id))
                .first::<DeviceRow>(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(result)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;

        if let Some((
            _device_id,
            lid_str,
            pn_str,
            registration_id,
            noise_key_data,
            identity_key_data,
            signed_pre_key_data,
            signed_pre_key_id,
            signed_pre_key_signature_data,
            adv_secret_key_data,
            account_data,
            push_name,
            app_version_primary,
            app_version_secondary,
            app_version_tertiary,
            app_version_last_fetched_ms,
            edge_routing_info,
            props_hash,
            server_cert_chain_blob,
        )) = row
        {
            let id = if !pn_str.is_empty() {
                pn_str.parse().ok()
            } else {
                None
            };
            let lid = if !lid_str.is_empty() {
                lid_str.parse().ok()
            } else {
                None
            };

            let noise_key = self.deserialize_keypair(&noise_key_data)?;
            let identity_key = self.deserialize_keypair(&identity_key_data)?;
            let signed_pre_key = self.deserialize_keypair(&signed_pre_key_data)?;

            let signed_pre_key_signature: [u8; 64] =
                signed_pre_key_signature_data.try_into().map_err(|_| {
                    StoreError::Serialization("Invalid signed_pre_key_signature length".to_string())
                })?;

            let adv_secret_key: [u8; 32] = adv_secret_key_data.try_into().map_err(|_| {
                StoreError::Serialization("Invalid adv_secret_key length".to_string())
            })?;

            let account = account_data
                .map(|data| {
                    wa::ADVSignedDeviceIdentity::decode_from_slice(&data[..])
                        .map_err(|e| StoreError::Serialization(e.to_string()))
                })
                .transpose()?;

            Ok(Some(CoreDevice {
                pn: id,
                lid,
                registration_id: registration_id as u32,
                noise_key,
                identity_key,
                signed_pre_key,
                signed_pre_key_id: signed_pre_key_id as u32,
                signed_pre_key_signature,
                adv_secret_key,
                account,
                push_name,
                app_version_primary: app_version_primary as u32,
                app_version_secondary: app_version_secondary as u32,
                app_version_tertiary: app_version_tertiary.try_into().unwrap_or(0u32),
                app_version_last_fetched_ms,
                device_props: {
                    use wacore::store::device::DEVICE_PROPS;
                    DEVICE_PROPS.clone()
                },
                edge_routing_info,
                props_hash,
                server_cert_chain: server_cert_chain_blob
                    .as_deref()
                    .and_then(|bytes| {
                        // The cert chain is a perf cache, not load-bearing
                        // identity. A corrupt blob (truncated row, format
                        // change between versions) must NOT block startup —
                        // log it and degrade to None so the next connect
                        // simply pays one XX handshake to repopulate.
                        match bincode::serde::decode_from_slice(
                            bytes,
                            bincode::config::standard(),
                        ) {
                            Ok((chain, _)) => Some(chain),
                            Err(e) => {
                                log::warn!(
                                    "device {} server_cert_chain blob ({} bytes) failed to decode: {};                                      dropping cache, next connect will use XX",
                                    self.device_id,
                                    bytes.len(),
                                    e,
                                );
                                None
                            }
                        }
                    }),
            }))
        } else {
            Ok(None)
        }
    }

    pub async fn put_identity_for_device(
        &self,
        address: &str,
        key: [u8; 32],
        device_id: i32,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let db_semaphore = self.db_semaphore.clone();
        let address_owned = address.to_string();
        let key_vec = key.to_vec();

        const MAX_RETRIES: u32 = 5;

        for attempt in 0..=MAX_RETRIES {
            let permit =
                db_semaphore.clone().acquire_owned().await.map_err(|e| {
                    StoreError::Database(format!("Failed to acquire semaphore: {}", e))
                })?;

            let pool_clone = pool.clone();
            let address_clone = address_owned.clone();
            let key_clone = key_vec.clone();

            let result = tokio::task::spawn_blocking(move || -> Result<()> {
                let mut conn = pool_clone
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                diesel::insert_into(identities::table)
                    .values((
                        identities::address.eq(address_clone),
                        identities::key.eq(&key_clone[..]),
                        identities::device_id.eq(device_id),
                    ))
                    .on_conflict((identities::address, identities::device_id))
                    .do_update()
                    .set(identities::key.eq(&key_clone[..]))
                    .execute(&mut conn)
                    .map_err(|e| StoreError::Database(e.to_string()))?;
                Ok(())
            })
            .await;

            drop(permit);

            match result {
                Ok(Ok(())) => return Ok(()),
                Ok(Err(e)) => {
                    let error_msg = e.to_string();
                    if (error_msg.contains("locked") || error_msg.contains("busy"))
                        && attempt < MAX_RETRIES
                    {
                        let delay_ms = 10 * 2u64.pow(attempt);
                        warn!(
                            "Identity write failed (attempt {}/{}): {}. Retrying in {}ms...",
                            attempt + 1,
                            MAX_RETRIES + 1,
                            error_msg,
                            delay_ms
                        );
                        tokio::time::sleep(std::time::Duration::from_millis(delay_ms)).await;
                        continue;
                    }
                    return Err(e);
                }
                Err(e) => return Err(StoreError::Database(format!("Task join error: {}", e))),
            }
        }

        Err(StoreError::Database(format!(
            "Identity write failed after {} attempts",
            MAX_RETRIES + 1
        )))
    }

    pub async fn delete_identity_for_device(&self, address: &str, device_id: i32) -> Result<()> {
        let pool = self.pool.clone();
        let address_owned = address.to_string();

        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                identities::table
                    .filter(identities::address.eq(address_owned))
                    .filter(identities::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;

        Ok(())
    }

    pub async fn load_identity_for_device(
        &self,
        address: &str,
        device_id: i32,
    ) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        let address = address.to_string();
        let result = self
            .with_semaphore(move || -> Result<Option<Vec<u8>>> {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                let res: Option<Vec<u8>> = identities::table
                    .select(identities::key)
                    .filter(identities::address.eq(address))
                    .filter(identities::device_id.eq(device_id))
                    .first(&mut conn)
                    .optional()
                    .map_err(|e| StoreError::Database(e.to_string()))?;
                Ok(res)
            })
            .await?;

        Ok(result)
    }

    pub async fn get_session_for_device(
        &self,
        address: &str,
        device_id: i32,
    ) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        let address_for_query = address.to_string();
        let result = self
            .with_semaphore(move || -> Result<Option<Vec<u8>>> {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                let res: Option<Vec<u8>> = sessions::table
                    .select(sessions::record)
                    .filter(sessions::address.eq(address_for_query.clone()))
                    .filter(sessions::device_id.eq(device_id))
                    .first(&mut conn)
                    .optional()
                    .map_err(|e| StoreError::Database(e.to_string()))?;

                Ok(res)
            })
            .await?;

        Ok(result)
    }

    pub async fn put_session_for_device(
        &self,
        address: &str,
        session: &[u8],
        device_id: i32,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let db_semaphore = self.db_semaphore.clone();
        let address_owned = address.to_string();
        let session_vec = session.to_vec();

        const MAX_RETRIES: u32 = 5;

        for attempt in 0..=MAX_RETRIES {
            let permit =
                db_semaphore.clone().acquire_owned().await.map_err(|e| {
                    StoreError::Database(format!("Failed to acquire semaphore: {}", e))
                })?;

            let pool_clone = pool.clone();
            let address_clone = address_owned.clone();
            let session_clone = session_vec.clone();

            let result = tokio::task::spawn_blocking(move || -> Result<()> {
                let mut conn = pool_clone
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                diesel::insert_into(sessions::table)
                    .values((
                        sessions::address.eq(address_clone),
                        sessions::record.eq(&session_clone),
                        sessions::device_id.eq(device_id),
                    ))
                    .on_conflict((sessions::address, sessions::device_id))
                    .do_update()
                    .set(sessions::record.eq(&session_clone))
                    .execute(&mut conn)
                    .map_err(|e| StoreError::Database(e.to_string()))?;
                Ok(())
            })
            .await;

            drop(permit);

            match result {
                Ok(Ok(())) => {
                    return Ok(());
                }
                Ok(Err(e)) => {
                    let error_msg = e.to_string();
                    if (error_msg.contains("locked") || error_msg.contains("busy"))
                        && attempt < MAX_RETRIES
                    {
                        let delay_ms = 10 * 2u64.pow(attempt);
                        warn!(
                            "Session write failed (attempt {}/{}): {}. Retrying in {}ms...",
                            attempt + 1,
                            MAX_RETRIES + 1,
                            error_msg,
                            delay_ms
                        );
                        tokio::time::sleep(std::time::Duration::from_millis(delay_ms)).await;
                        continue;
                    }
                    return Err(e);
                }
                Err(e) => return Err(StoreError::Database(format!("Task join error: {}", e))),
            }
        }

        Err(StoreError::Database(format!(
            "Session write failed after {} attempts",
            MAX_RETRIES + 1
        )))
    }

    pub async fn delete_session_for_device(&self, address: &str, device_id: i32) -> Result<()> {
        let pool = self.pool.clone();
        let address_owned = address.to_string();

        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                sessions::table
                    .filter(sessions::address.eq(address_owned))
                    .filter(sessions::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;

        Ok(())
    }

    pub async fn put_sender_key_for_device(
        &self,
        address: &str,
        record: &[u8],
        device_id: i32,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let address = address.to_string();
        let record_vec = record.to_vec();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(sender_keys::table)
                .values((
                    sender_keys::address.eq(address),
                    sender_keys::record.eq(&record_vec),
                    sender_keys::device_id.eq(device_id),
                ))
                .on_conflict((sender_keys::address, sender_keys::device_id))
                .do_update()
                .set(sender_keys::record.eq(&record_vec))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    pub async fn get_sender_key_for_device(
        &self,
        address: &str,
        device_id: i32,
    ) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        let address = address.to_string();
        tokio::task::spawn_blocking(move || -> Result<Option<Vec<u8>>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let res: Option<Vec<u8>> = sender_keys::table
                .select(sender_keys::record)
                .filter(sender_keys::address.eq(address))
                .filter(sender_keys::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(res)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    pub async fn delete_sender_key_for_device(&self, address: &str, device_id: i32) -> Result<()> {
        let pool = self.pool.clone();
        let address = address.to_string();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                sender_keys::table
                    .filter(sender_keys::address.eq(address))
                    .filter(sender_keys::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    pub async fn get_app_state_sync_key_for_device(
        &self,
        key_id: &[u8],
        device_id: i32,
    ) -> Result<Option<AppStateSyncKey>> {
        let pool = self.pool.clone();
        let key_id = key_id.to_vec();
        let res: Option<Vec<u8>> =
            tokio::task::spawn_blocking(move || -> Result<Option<Vec<u8>>> {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                let res: Option<Vec<u8>> = app_state_keys::table
                    .select(app_state_keys::key_data)
                    .filter(app_state_keys::key_id.eq(&key_id))
                    .filter(app_state_keys::device_id.eq(device_id))
                    .first(&mut conn)
                    .optional()
                    .map_err(|e| StoreError::Database(e.to_string()))?;
                Ok(res)
            })
            .await
            .map_err(|e| StoreError::Database(e.to_string()))??;

        if let Some(data) = res {
            let (key, _) = bincode::serde::decode_from_slice(&data, bincode::config::standard())
                .map_err(|e| StoreError::Serialization(e.to_string()))?;
            Ok(Some(key))
        } else {
            Ok(None)
        }
    }

    pub async fn set_app_state_sync_key_for_device(
        &self,
        key_id: &[u8],
        key: AppStateSyncKey,
        device_id: i32,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let key_id = key_id.to_vec();
        let data = bincode::serde::encode_to_vec(&key, bincode::config::standard())
            .map_err(|e| StoreError::Serialization(e.to_string()))?;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(app_state_keys::table)
                .values((
                    app_state_keys::key_id.eq(&key_id),
                    app_state_keys::key_data.eq(&data),
                    app_state_keys::device_id.eq(device_id),
                ))
                .on_conflict((app_state_keys::key_id, app_state_keys::device_id))
                .do_update()
                .set(app_state_keys::key_data.eq(&data))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    pub async fn get_app_state_version_for_device(
        &self,
        name: &str,
        device_id: i32,
    ) -> Result<HashState> {
        let pool = self.pool.clone();
        let name = name.to_string();
        let res: Option<Vec<u8>> =
            tokio::task::spawn_blocking(move || -> Result<Option<Vec<u8>>> {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                let res: Option<Vec<u8>> = app_state_versions::table
                    .select(app_state_versions::state_data)
                    .filter(app_state_versions::name.eq(name))
                    .filter(app_state_versions::device_id.eq(device_id))
                    .first(&mut conn)
                    .optional()
                    .map_err(|e| StoreError::Database(e.to_string()))?;
                Ok(res)
            })
            .await
            .map_err(|e| StoreError::Database(e.to_string()))??;

        if let Some(data) = res {
            let (state, _) = bincode::serde::decode_from_slice(&data, bincode::config::standard())
                .map_err(|e| StoreError::Serialization(e.to_string()))?;
            Ok(state)
        } else {
            Ok(HashState::default())
        }
    }

    pub async fn set_app_state_version_for_device(
        &self,
        name: &str,
        state: HashState,
        device_id: i32,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let name = name.to_string();
        let data = bincode::serde::encode_to_vec(&state, bincode::config::standard())
            .map_err(|e| StoreError::Serialization(e.to_string()))?;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(app_state_versions::table)
                .values((
                    app_state_versions::name.eq(&name),
                    app_state_versions::state_data.eq(&data),
                    app_state_versions::device_id.eq(device_id),
                ))
                .on_conflict((app_state_versions::name, app_state_versions::device_id))
                .do_update()
                .set(app_state_versions::state_data.eq(&data))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    pub async fn put_app_state_mutation_macs_for_device(
        &self,
        name: &str,
        version: u64,
        mutations: &[AppStateMutationMAC],
        device_id: i32,
    ) -> Result<()> {
        if mutations.is_empty() {
            return Ok(());
        }
        let pool = self.pool.clone();
        let name = name.to_string();
        let mutations: Vec<AppStateMutationMAC> = mutations.to_vec();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            let records: Vec<_> = mutations
                .iter()
                .map(|m| {
                    (
                        app_state_mutation_macs::name.eq(&name),
                        app_state_mutation_macs::version.eq(version as i64),
                        app_state_mutation_macs::index_mac.eq(&m.index_mac),
                        app_state_mutation_macs::value_mac.eq(&m.value_mac),
                        app_state_mutation_macs::device_id.eq(device_id),
                    )
                })
                .collect();

            // SQLite variable limit is typically 999 or 32766.
            // Each row has 5 columns. 100 rows * 5 = 500 params, which is safe.
            const CHUNK_SIZE: usize = 100;

            for chunk in records.chunks(CHUNK_SIZE) {
                diesel::insert_into(app_state_mutation_macs::table)
                    .values(chunk)
                    .on_conflict((
                        app_state_mutation_macs::name,
                        app_state_mutation_macs::index_mac,
                        app_state_mutation_macs::device_id,
                    ))
                    .do_update()
                    .set((
                        app_state_mutation_macs::version
                            .eq(excluded(app_state_mutation_macs::version)),
                        app_state_mutation_macs::value_mac
                            .eq(excluded(app_state_mutation_macs::value_mac)),
                    ))
                    .execute(&mut conn)
                    .map_err(|e| StoreError::Database(e.to_string()))?;
            }
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    pub async fn delete_app_state_mutation_macs_for_device(
        &self,
        name: &str,
        index_macs: &[Vec<u8>],
        device_id: i32,
    ) -> Result<()> {
        if index_macs.is_empty() {
            return Ok(());
        }
        let pool = self.pool.clone();
        let name = name.to_string();
        let index_macs: Vec<Vec<u8>> = index_macs.to_vec();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            // SQLite variable limit is usually 999 or higher.
            // We use a safe chunk size to stay well within limits.
            const CHUNK_SIZE: usize = 500;

            for chunk in index_macs.chunks(CHUNK_SIZE) {
                diesel::delete(
                    app_state_mutation_macs::table.filter(
                        app_state_mutation_macs::name
                            .eq(&name)
                            .and(app_state_mutation_macs::index_mac.eq_any(chunk))
                            .and(app_state_mutation_macs::device_id.eq(device_id)),
                    ),
                )
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            }
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    /// Batched read of previous-MAC values for many index MACs in one query
    /// (single spawn_blocking + `index_mac IN (...)`), replacing the
    /// per-mutation N+1 in appstate sync.
    pub async fn get_app_state_mutation_macs_batch_for_device(
        &self,
        name: &str,
        index_macs: &[[u8; 32]],
        device_id: i32,
    ) -> Result<std::collections::HashMap<[u8; 32], Vec<u8>>> {
        if index_macs.is_empty() {
            return Ok(std::collections::HashMap::new());
        }
        let pool = self.pool.clone();
        let name = name.to_string();
        let index_macs: Vec<[u8; 32]> = index_macs.to_vec();
        tokio::task::spawn_blocking(
            move || -> Result<std::collections::HashMap<[u8; 32], Vec<u8>>> {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                let mut out = std::collections::HashMap::with_capacity(index_macs.len());
                // SQLite variable limit is usually 999 or higher.
                const CHUNK_SIZE: usize = 500;
                for chunk in index_macs.chunks(CHUNK_SIZE) {
                    let chunk_slices: Vec<&[u8]> = chunk.iter().map(|m| m.as_slice()).collect();
                    let rows: Vec<(Vec<u8>, Vec<u8>)> = app_state_mutation_macs::table
                        .select((
                            app_state_mutation_macs::index_mac,
                            app_state_mutation_macs::value_mac,
                        ))
                        .filter(app_state_mutation_macs::name.eq(&name))
                        .filter(app_state_mutation_macs::index_mac.eq_any(chunk_slices))
                        .filter(app_state_mutation_macs::device_id.eq(device_id))
                        .load(&mut conn)
                        .map_err(|e| StoreError::Database(e.to_string()))?;
                    // Rows with a non-32-byte index_mac cannot have come from the
                    // 32-byte keys we just queried; skip defensively.
                    out.extend(rows.into_iter().filter_map(|(k, v)| {
                        <[u8; 32]>::try_from(k.as_slice()).ok().map(|k| (k, v))
                    }));
                }
                Ok(out)
            },
        )
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    /// Persist one applied patch atomically: version, removals and additions
    /// become visible together within one transaction.
    pub async fn commit_app_state_patch_for_device(
        &self,
        name: &str,
        state: &HashState,
        removed_index_macs: &[Vec<u8>],
        added: &[AppStateMutationMAC],
        device_id: i32,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let name = name.to_string();
        let version = state.version;
        let data = bincode::serde::encode_to_vec(state, bincode::config::standard())
            .map_err(|e| StoreError::Serialization(e.to_string()))?;
        let removed: Vec<Vec<u8>> = removed_index_macs.to_vec();
        let added: Vec<AppStateMutationMAC> = added.to_vec();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            conn.transaction::<_, diesel::result::Error, _>(|conn| {
                diesel::insert_into(app_state_versions::table)
                    .values((
                        app_state_versions::name.eq(&name),
                        app_state_versions::state_data.eq(&data),
                        app_state_versions::device_id.eq(device_id),
                    ))
                    .on_conflict((app_state_versions::name, app_state_versions::device_id))
                    .do_update()
                    .set(app_state_versions::state_data.eq(&data))
                    .execute(conn)?;

                // SQLite variable limit is usually 999 or higher.
                const CHUNK_SIZE: usize = 500;
                for chunk in removed.chunks(CHUNK_SIZE) {
                    diesel::delete(
                        app_state_mutation_macs::table.filter(
                            app_state_mutation_macs::name
                                .eq(&name)
                                .and(app_state_mutation_macs::index_mac.eq_any(chunk))
                                .and(app_state_mutation_macs::device_id.eq(device_id)),
                        ),
                    )
                    .execute(conn)?;
                }

                let records: Vec<_> = added
                    .iter()
                    .map(|m| {
                        (
                            app_state_mutation_macs::name.eq(&name),
                            app_state_mutation_macs::version.eq(version as i64),
                            app_state_mutation_macs::index_mac.eq(&m.index_mac),
                            app_state_mutation_macs::value_mac.eq(&m.value_mac),
                            app_state_mutation_macs::device_id.eq(device_id),
                        )
                    })
                    .collect();
                // Each row has 5 columns; 100 rows is safe under the limit.
                const INSERT_CHUNK_SIZE: usize = 100;
                for chunk in records.chunks(INSERT_CHUNK_SIZE) {
                    diesel::insert_into(app_state_mutation_macs::table)
                        .values(chunk)
                        .on_conflict((
                            app_state_mutation_macs::name,
                            app_state_mutation_macs::index_mac,
                            app_state_mutation_macs::device_id,
                        ))
                        .do_update()
                        .set((
                            app_state_mutation_macs::version
                                .eq(excluded(app_state_mutation_macs::version)),
                            app_state_mutation_macs::value_mac
                                .eq(excluded(app_state_mutation_macs::value_mac)),
                        ))
                        .execute(conn)?;
                }
                Ok(())
            })
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    pub async fn get_latest_app_state_sync_key_id_for_device(
        &self,
        device_id: i32,
    ) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        tokio::task::spawn_blocking(move || -> Result<Option<Vec<u8>>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // Return the latest key whose blob actually decodes. A corrupt row
            // still sits in the table with a possibly higher key_id; selecting
            // it would make an outbound build_patch fail later in
            // get_app_state_key with KeyNotFound.
            let candidates: Vec<(Vec<u8>, Vec<u8>)> = app_state_keys::table
                .select((app_state_keys::key_id, app_state_keys::key_data))
                .filter(app_state_keys::device_id.eq(device_id))
                .order(app_state_keys::key_id.desc())
                .load(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(candidates
                .into_iter()
                .find(|(_, data)| {
                    bincode::serde::decode_from_slice::<AppStateSyncKey, _>(
                        data,
                        bincode::config::standard(),
                    )
                    .is_ok()
                })
                .map(|(key_id, _)| key_id))
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    pub async fn clear_app_state_mutation_macs_for_device(
        &self,
        name: &str,
        device_id: i32,
    ) -> Result<()> {
        let pool = self.pool.clone();
        let name = name.to_string();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                app_state_mutation_macs::table.filter(
                    app_state_mutation_macs::name
                        .eq(&name)
                        .and(app_state_mutation_macs::device_id.eq(device_id)),
                ),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    pub async fn get_app_state_mutation_mac_for_device(
        &self,
        name: &str,
        index_mac: &[u8],
        device_id: i32,
    ) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        let name = name.to_string();
        let index_mac = index_mac.to_vec();
        tokio::task::spawn_blocking(move || -> Result<Option<Vec<u8>>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let res: Option<Vec<u8>> = app_state_mutation_macs::table
                .select(app_state_mutation_macs::value_mac)
                .filter(app_state_mutation_macs::name.eq(&name))
                .filter(app_state_mutation_macs::index_mac.eq(&index_mac))
                .filter(app_state_mutation_macs::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(res)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }
}

#[async_trait]
impl SignalStore for SqliteStore {
    async fn put_identity(&self, address: &str, key: [u8; 32]) -> Result<()> {
        self.put_identity_for_device(address, key, self.device_id)
            .await
    }

    async fn load_identity(&self, address: &str) -> Result<Option<Vec<u8>>> {
        self.load_identity_for_device(address, self.device_id).await
    }

    async fn delete_identity(&self, address: &str) -> Result<()> {
        self.delete_identity_for_device(address, self.device_id)
            .await
    }

    async fn get_session(&self, address: &str) -> Result<Option<Vec<u8>>> {
        self.get_session_for_device(address, self.device_id).await
    }

    async fn put_session(&self, address: &str, session: &[u8]) -> Result<()> {
        self.put_session_for_device(address, session, self.device_id)
            .await
    }

    async fn delete_session(&self, address: &str) -> Result<()> {
        self.delete_session_for_device(address, self.device_id)
            .await
    }

    async fn store_prekey(&self, id: u32, record: &[u8], uploaded: bool) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let record = record.to_vec();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(prekeys::table)
                .values((
                    prekeys::id.eq(id as i32),
                    prekeys::key.eq(&record),
                    prekeys::uploaded.eq(uploaded),
                    prekeys::device_id.eq(device_id),
                ))
                .on_conflict((prekeys::id, prekeys::device_id))
                .do_update()
                .set((prekeys::key.eq(&record), prekeys::uploaded.eq(uploaded)))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn load_prekey(&self, id: u32) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<Option<Vec<u8>>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let res: Option<Vec<u8>> = prekeys::table
                .select(prekeys::key)
                .filter(prekeys::id.eq(id as i32))
                .filter(prekeys::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(res)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn remove_prekey(&self, id: u32) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                prekeys::table
                    .filter(prekeys::id.eq(id as i32))
                    .filter(prekeys::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn max_prekey_id(&self) -> Result<Option<u32>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<Option<u32>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // MAX() returns a single row (NULL on empty table), so no `.optional()` needed.
            let res: Option<i32> = prekeys::table
                .select(diesel::dsl::max(prekeys::id))
                .filter(prekeys::device_id.eq(device_id))
                .first(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(res.map(|id| id as u32))
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn store_signed_prekey(&self, id: u32, record: &[u8]) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let record = record.to_vec();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(signed_prekeys::table)
                .values((
                    signed_prekeys::id.eq(id as i32),
                    signed_prekeys::record.eq(&record),
                    signed_prekeys::device_id.eq(device_id),
                ))
                .on_conflict((signed_prekeys::id, signed_prekeys::device_id))
                .do_update()
                .set(signed_prekeys::record.eq(&record))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn load_signed_prekey(&self, id: u32) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<Option<Vec<u8>>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let res: Option<Vec<u8>> = signed_prekeys::table
                .select(signed_prekeys::record)
                .filter(signed_prekeys::id.eq(id as i32))
                .filter(signed_prekeys::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(res)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn load_all_signed_prekeys(&self) -> Result<Vec<(u32, Vec<u8>)>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<Vec<(u32, Vec<u8>)>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let results: Vec<(i32, Vec<u8>)> = signed_prekeys::table
                .select((signed_prekeys::id, signed_prekeys::record))
                .filter(signed_prekeys::device_id.eq(device_id))
                .load(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(results
                .into_iter()
                .map(|(id, record)| (id as u32, record))
                .collect())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn remove_signed_prekey(&self, id: u32) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                signed_prekeys::table
                    .filter(signed_prekeys::id.eq(id as i32))
                    .filter(signed_prekeys::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn put_sender_key(&self, address: &str, record: &[u8]) -> Result<()> {
        self.put_sender_key_for_device(address, record, self.device_id)
            .await
    }

    async fn get_sender_key(&self, address: &str) -> Result<Option<Vec<u8>>> {
        self.get_sender_key_for_device(address, self.device_id)
            .await
    }

    async fn delete_sender_key(&self, address: &str) -> Result<()> {
        self.delete_sender_key_for_device(address, self.device_id)
            .await
    }
}

#[async_trait]
impl AppSyncStore for SqliteStore {
    async fn get_sync_key(&self, key_id: &[u8]) -> Result<Option<AppStateSyncKey>> {
        self.get_app_state_sync_key_for_device(key_id, self.device_id)
            .await
    }

    async fn set_sync_key(&self, key_id: &[u8], key: AppStateSyncKey) -> Result<()> {
        self.set_app_state_sync_key_for_device(key_id, key, self.device_id)
            .await
    }

    async fn get_version(&self, name: &str) -> Result<HashState> {
        self.get_app_state_version_for_device(name, self.device_id)
            .await
    }

    async fn set_version(&self, name: &str, state: HashState) -> Result<()> {
        self.set_app_state_version_for_device(name, state, self.device_id)
            .await
    }

    async fn put_mutation_macs(
        &self,
        name: &str,
        version: u64,
        mutations: &[AppStateMutationMAC],
    ) -> Result<()> {
        self.put_app_state_mutation_macs_for_device(name, version, mutations, self.device_id)
            .await
    }

    async fn get_mutation_mac(&self, name: &str, index_mac: &[u8]) -> Result<Option<Vec<u8>>> {
        self.get_app_state_mutation_mac_for_device(name, index_mac, self.device_id)
            .await
    }

    async fn get_mutation_macs(
        &self,
        name: &str,
        index_macs: &[[u8; 32]],
    ) -> Result<std::collections::HashMap<[u8; 32], Vec<u8>>> {
        self.get_app_state_mutation_macs_batch_for_device(name, index_macs, self.device_id)
            .await
    }

    async fn delete_mutation_macs(&self, name: &str, index_macs: &[Vec<u8>]) -> Result<()> {
        self.delete_app_state_mutation_macs_for_device(name, index_macs, self.device_id)
            .await
    }

    async fn commit_patch(
        &self,
        name: &str,
        state: HashState,
        removed_index_macs: &[Vec<u8>],
        added: &[AppStateMutationMAC],
    ) -> Result<()> {
        self.commit_app_state_patch_for_device(
            name,
            &state,
            removed_index_macs,
            added,
            self.device_id,
        )
        .await
    }

    async fn clear_mutation_macs(&self, name: &str) -> Result<()> {
        self.clear_app_state_mutation_macs_for_device(name, self.device_id)
            .await
    }

    async fn get_latest_sync_key_id(&self) -> Result<Option<Vec<u8>>> {
        self.get_latest_app_state_sync_key_id_for_device(self.device_id)
            .await
    }
}

/// A pending-row mutation owned for the lifetime of its detached worker. The
/// lease must stay alive through the transaction so budget and identity
/// exclusion outlive a cancelled caller.
enum OwnedPendingMutation {
    Replace {
        payload: Arc<PendingInboundPayload>,
        revision: PendingInboundRevision,
    },
    Delete(PendingInboundLease),
}

impl OwnedPendingMutation {
    fn lease(&self) -> PendingInboundLease {
        match self {
            Self::Replace { payload, .. } => payload.lease(),
            Self::Delete(lease) => lease.clone(),
        }
    }
}

/// Single source of the pending-inbound row insert, shared by the single-row
/// and batch write paths so a schema or conflict-strategy change cannot
/// silently diverge between them. `inserted_at` keeps its column default.
fn insert_pending_inbound_row(
    conn: &mut SqliteConnection,
    device_id: i32,
    chat: &str,
    sender: &str,
    id: &str,
    message: &[u8],
) -> QueryResult<usize> {
    diesel::replace_into(pending_inbound_messages::table)
        .values((
            pending_inbound_messages::chat.eq(chat),
            pending_inbound_messages::sender.eq(sender),
            pending_inbound_messages::id.eq(id),
            pending_inbound_messages::message.eq(message),
            pending_inbound_messages::device_id.eq(device_id),
        ))
        .execute(conn)
}

#[derive(QueryableByName)]
struct BoundedPendingPayload {
    #[diesel(sql_type = diesel::sql_types::Text)]
    storage_type: String,
    #[diesel(sql_type = diesel::sql_types::Nullable<diesel::sql_types::BigInt>)]
    byte_len: Option<i64>,
    #[diesel(sql_type = diesel::sql_types::Nullable<diesel::sql_types::Binary>)]
    payload: Option<Vec<u8>>,
}

#[derive(QueryableByName)]
struct BoundedPendingSender {
    #[diesel(sql_type = diesel::sql_types::Nullable<diesel::sql_types::Text>)]
    sender: Option<String>,
}

fn read_pending_inbound_bounded_row(
    conn: &mut SqliteConnection,
    device_id: i32,
    chat: &str,
    sender: &str,
    id: &str,
    max_bytes: usize,
    lease: PendingInboundLease,
) -> QueryResult<Result<Option<PendingInboundRecord>>> {
    use diesel::query_dsl::RunQueryDsl as _;

    // CASE keeps oversized blobs out of the Diesel value buffer. Both length
    // and payload come from one row snapshot, never separate racing queries.
    let row: Option<BoundedPendingPayload> = crate::bounded_pending_query::BoundedPendingQuery {
        max_bytes: i64::try_from(max_bytes).unwrap_or(i64::MAX),
        device_id,
        chat,
        sender,
        id,
    }
    .get_result(conn)
    .optional()?;
    Ok(match row {
        None => Ok(None),
        Some(row) if row.storage_type != "blob" => Err(StoreError::Validation(
            "pending inbound payload is not a BLOB".to_owned(),
        )),
        Some(row) => match row.payload {
            Some(bytes) => Ok(Some(PendingInboundRecord::new(bytes, lease))),
            None => Err(StoreError::Validation(format!(
                "pending inbound payload length {:?} exceeds limit {max_bytes}",
                row.byte_len,
            ))),
        },
    })
}

/// Batch/single-row shared delete; see [`insert_pending_inbound_row`].
fn delete_pending_inbound_row(
    conn: &mut SqliteConnection,
    device_id: i32,
    chat: &str,
    sender: &str,
    id: &str,
) -> QueryResult<usize> {
    diesel::delete(
        pending_inbound_messages::table
            .filter(pending_inbound_messages::chat.eq(chat))
            .filter(pending_inbound_messages::sender.eq(sender))
            .filter(pending_inbound_messages::id.eq(id))
            .filter(pending_inbound_messages::device_id.eq(device_id)),
    )
    .execute(conn)
}

#[async_trait]
impl ProtocolStore for SqliteStore {
    async fn get_skdm_recipients(&self, group_jid: &str) -> Result<Vec<Jid>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let group_jid = group_jid.to_string();
        tokio::task::spawn_blocking(move || -> Result<Vec<Jid>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let recipients: Vec<String> = skdm_recipients::table
                .select(skdm_recipients::device_jid)
                .filter(skdm_recipients::group_jid.eq(&group_jid))
                .filter(skdm_recipients::device_id.eq(device_id))
                .load(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            let jids: Vec<Jid> = recipients
                .iter()
                .filter_map(|s| match s.parse::<Jid>() {
                    Ok(jid) => Some(jid),
                    Err(e) => {
                        warn!("Failed to parse SKDM recipient '{}': {}", s, e);
                        None
                    }
                })
                .collect();
            Ok(jids)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn add_skdm_recipients(&self, group_jid: &str, device_jids: &[Jid]) -> Result<()> {
        if device_jids.is_empty() {
            return Ok(());
        }
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let group_jid = group_jid.to_string();
        let device_jid_strs: Vec<String> = device_jids.iter().map(|j| j.to_string()).collect();
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i32;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            let values: Vec<_> = device_jid_strs
                .iter()
                .map(|device_jid| {
                    (
                        skdm_recipients::group_jid.eq(&group_jid),
                        skdm_recipients::device_jid.eq(device_jid),
                        skdm_recipients::device_id.eq(device_id),
                        skdm_recipients::created_at.eq(now),
                    )
                })
                .collect();

            const CHUNK_SIZE: usize = 200; // SQLite variable limit ~999, 4 cols/row

            for chunk in values.chunks(CHUNK_SIZE) {
                diesel::insert_into(skdm_recipients::table)
                    .values(chunk)
                    .on_conflict((
                        skdm_recipients::group_jid,
                        skdm_recipients::device_jid,
                        skdm_recipients::device_id,
                    ))
                    .do_nothing()
                    .execute(&mut conn)
                    .map_err(|e| StoreError::Database(e.to_string()))?;
            }
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn clear_skdm_recipients(&self, group_jid: &str) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let group_jid = group_jid.to_string();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                skdm_recipients::table
                    .filter(skdm_recipients::group_jid.eq(&group_jid))
                    .filter(skdm_recipients::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn get_lid_mapping(&self, lid: &str) -> Result<Option<LidPnMappingEntry>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let lid = lid.to_string();
        tokio::task::spawn_blocking(move || -> Result<Option<LidPnMappingEntry>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let row: Option<(String, String, i64, String, i64)> = lid_pn_mapping::table
                .select((
                    lid_pn_mapping::lid,
                    lid_pn_mapping::phone_number,
                    lid_pn_mapping::created_at,
                    lid_pn_mapping::learning_source,
                    lid_pn_mapping::updated_at,
                ))
                .filter(lid_pn_mapping::lid.eq(&lid))
                .filter(lid_pn_mapping::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(row.map(
                |(lid, phone_number, created_at, learning_source, updated_at)| LidPnMappingEntry {
                    lid,
                    phone_number,
                    created_at,
                    updated_at,
                    learning_source,
                },
            ))
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn get_pn_mapping(&self, phone: &str) -> Result<Option<LidPnMappingEntry>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let phone = phone.to_string();
        tokio::task::spawn_blocking(move || -> Result<Option<LidPnMappingEntry>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let row: Option<(String, String, i64, String, i64)> = lid_pn_mapping::table
                .select((
                    lid_pn_mapping::lid,
                    lid_pn_mapping::phone_number,
                    lid_pn_mapping::created_at,
                    lid_pn_mapping::learning_source,
                    lid_pn_mapping::updated_at,
                ))
                .filter(lid_pn_mapping::phone_number.eq(&phone))
                .filter(lid_pn_mapping::device_id.eq(device_id))
                .order(lid_pn_mapping::updated_at.desc())
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(row.map(
                |(lid, phone_number, created_at, learning_source, updated_at)| LidPnMappingEntry {
                    lid,
                    phone_number,
                    created_at,
                    updated_at,
                    learning_source,
                },
            ))
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn put_lid_mapping(&self, entry: &LidPnMappingEntry) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let entry = entry.clone();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(lid_pn_mapping::table)
                .values((
                    lid_pn_mapping::lid.eq(&entry.lid),
                    lid_pn_mapping::phone_number.eq(&entry.phone_number),
                    lid_pn_mapping::created_at.eq(entry.created_at),
                    lid_pn_mapping::learning_source.eq(&entry.learning_source),
                    lid_pn_mapping::updated_at.eq(entry.updated_at),
                    lid_pn_mapping::device_id.eq(device_id),
                ))
                .on_conflict((lid_pn_mapping::lid, lid_pn_mapping::device_id))
                .do_update()
                .set((
                    lid_pn_mapping::phone_number.eq(&entry.phone_number),
                    lid_pn_mapping::learning_source.eq(&entry.learning_source),
                    lid_pn_mapping::updated_at.eq(entry.updated_at),
                ))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn get_all_lid_mappings(&self) -> Result<Vec<LidPnMappingEntry>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<Vec<LidPnMappingEntry>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let rows: Vec<(String, String, i64, String, i64)> = lid_pn_mapping::table
                .select((
                    lid_pn_mapping::lid,
                    lid_pn_mapping::phone_number,
                    lid_pn_mapping::created_at,
                    lid_pn_mapping::learning_source,
                    lid_pn_mapping::updated_at,
                ))
                .filter(lid_pn_mapping::device_id.eq(device_id))
                .load(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(rows
                .into_iter()
                .map(
                    |(lid, phone_number, created_at, learning_source, updated_at)| {
                        LidPnMappingEntry {
                            lid,
                            phone_number,
                            created_at,
                            updated_at,
                            learning_source,
                        }
                    },
                )
                .collect())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn save_base_key(&self, address: &str, message_id: &str, base_key: &[u8]) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let address = address.to_string();
        let message_id = message_id.to_string();
        let base_key = base_key.to_vec();
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i32;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(base_keys::table)
                .values((
                    base_keys::address.eq(&address),
                    base_keys::message_id.eq(&message_id),
                    base_keys::base_key.eq(&base_key),
                    base_keys::device_id.eq(device_id),
                    base_keys::created_at.eq(now),
                ))
                .on_conflict((
                    base_keys::address,
                    base_keys::message_id,
                    base_keys::device_id,
                ))
                .do_update()
                .set(base_keys::base_key.eq(&base_key))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn has_same_base_key(
        &self,
        address: &str,
        message_id: &str,
        current_base_key: &[u8],
    ) -> Result<bool> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let address = address.to_string();
        let message_id = message_id.to_string();
        let current_base_key = current_base_key.to_vec();
        tokio::task::spawn_blocking(move || -> Result<bool> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let stored_key: Option<Vec<u8>> = base_keys::table
                .select(base_keys::base_key)
                .filter(base_keys::address.eq(&address))
                .filter(base_keys::message_id.eq(&message_id))
                .filter(base_keys::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(stored_key.as_ref() == Some(&current_base_key))
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn delete_base_key(&self, address: &str, message_id: &str) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let address = address.to_string();
        let message_id = message_id.to_string();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                base_keys::table
                    .filter(base_keys::address.eq(&address))
                    .filter(base_keys::message_id.eq(&message_id))
                    .filter(base_keys::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn update_device_list(&self, record: DeviceListRecord) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let devices_json = serde_json::to_string(&record.devices)
            .map_err(|e| StoreError::Serialization(e.to_string()))?;
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i32;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(device_registry::table)
                .values((
                    device_registry::user_id.eq(&record.user),
                    device_registry::devices_json.eq(&devices_json),
                    device_registry::timestamp.eq(record.timestamp as i32),
                    device_registry::phash.eq(&record.phash),
                    device_registry::device_id.eq(device_id),
                    device_registry::updated_at.eq(now),
                ))
                .on_conflict((device_registry::user_id, device_registry::device_id))
                .do_update()
                .set((
                    device_registry::devices_json.eq(&devices_json),
                    device_registry::timestamp.eq(record.timestamp as i32),
                    device_registry::phash.eq(&record.phash),
                    device_registry::updated_at.eq(now),
                ))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn get_devices(&self, user: &str) -> Result<Option<DeviceListRecord>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let user = user.to_string();
        tokio::task::spawn_blocking(move || -> Result<Option<DeviceListRecord>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let row: Option<(String, String, i32, Option<String>)> = device_registry::table
                .select((
                    device_registry::user_id,
                    device_registry::devices_json,
                    device_registry::timestamp,
                    device_registry::phash,
                ))
                .filter(device_registry::user_id.eq(&user))
                .filter(device_registry::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            match row {
                Some((user, devices_json, timestamp, phash)) => {
                    let devices: Vec<DeviceInfo> = serde_json::from_str(&devices_json)
                        .map_err(|e| StoreError::Serialization(e.to_string()))?;
                    Ok(Some(DeviceListRecord {
                        user,
                        devices,
                        timestamp: timestamp as i64,
                        phash,
                    }))
                }
                None => Ok(None),
            }
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn mark_forget_sender_key(&self, group_jid: &str, participant: &str) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let group_jid = group_jid.to_string();
        let participant = participant.to_string();
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i32;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(sender_key_status::table)
                .values((
                    sender_key_status::group_jid.eq(&group_jid),
                    sender_key_status::participant.eq(&participant),
                    sender_key_status::device_id.eq(device_id),
                    sender_key_status::marked_at.eq(now),
                ))
                .on_conflict((
                    sender_key_status::group_jid,
                    sender_key_status::participant,
                    sender_key_status::device_id,
                ))
                .do_update()
                .set(sender_key_status::marked_at.eq(now))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn consume_forget_marks(&self, group_jid: &str) -> Result<Vec<String>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let group_jid = group_jid.to_string();
        tokio::task::spawn_blocking(move || -> Result<Vec<String>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let participants: Vec<String> = sender_key_status::table
                .select(sender_key_status::participant)
                .filter(sender_key_status::group_jid.eq(&group_jid))
                .filter(sender_key_status::device_id.eq(device_id))
                .load(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            diesel::delete(
                sender_key_status::table
                    .filter(sender_key_status::group_jid.eq(&group_jid))
                    .filter(sender_key_status::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(participants)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn get_tc_token(&self, jid: &str) -> Result<Option<TcTokenEntry>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let jid = jid.to_string();
        tokio::task::spawn_blocking(move || -> Result<Option<TcTokenEntry>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let row: Option<(Vec<u8>, i64, Option<i64>)> = tc_tokens::table
                .select((
                    tc_tokens::token,
                    tc_tokens::token_timestamp,
                    tc_tokens::sender_timestamp,
                ))
                .filter(tc_tokens::jid.eq(&jid))
                .filter(tc_tokens::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(
                row.map(|(token, token_timestamp, sender_timestamp)| TcTokenEntry {
                    token,
                    token_timestamp,
                    sender_timestamp,
                }),
            )
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn put_tc_token(&self, jid: &str, entry: &TcTokenEntry) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let jid = jid.to_string();
        let entry = entry.clone();
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i64;
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::insert_into(tc_tokens::table)
                .values((
                    tc_tokens::jid.eq(&jid),
                    tc_tokens::token.eq(&entry.token),
                    tc_tokens::token_timestamp.eq(entry.token_timestamp),
                    tc_tokens::sender_timestamp.eq(entry.sender_timestamp),
                    tc_tokens::device_id.eq(device_id),
                    tc_tokens::updated_at.eq(now),
                ))
                .on_conflict((tc_tokens::jid, tc_tokens::device_id))
                .do_update()
                .set((
                    tc_tokens::token.eq(&entry.token),
                    tc_tokens::token_timestamp.eq(entry.token_timestamp),
                    tc_tokens::sender_timestamp.eq(entry.sender_timestamp),
                    tc_tokens::updated_at.eq(now),
                ))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn delete_tc_token(&self, jid: &str) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let jid = jid.to_string();
        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            diesel::delete(
                tc_tokens::table
                    .filter(tc_tokens::jid.eq(&jid))
                    .filter(tc_tokens::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;
        Ok(())
    }

    async fn get_all_tc_token_jids(&self) -> Result<Vec<String>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<Vec<String>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let jids: Vec<String> = tc_tokens::table
                .select(tc_tokens::jid)
                .filter(tc_tokens::device_id.eq(device_id))
                .load(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(jids)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn delete_expired_tc_tokens(&self, cutoff_timestamp: i64) -> Result<u32> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        tokio::task::spawn_blocking(move || -> Result<u32> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let deleted = diesel::delete(
                tc_tokens::table
                    .filter(tc_tokens::token_timestamp.lt(cutoff_timestamp))
                    .filter(tc_tokens::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(deleted as u32)
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))?
    }

    async fn store_pending_inbound(
        &self,
        chat: &str,
        sender: &str,
        id: &str,
        message: &[u8],
    ) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let (chat, sender, id) = (chat.to_owned(), sender.to_owned(), id.to_owned());
        let message = message.to_vec();
        self.with_semaphore(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            insert_pending_inbound_row(&mut conn, device_id, &chat, &sender, &id, &message)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
    }

    async fn get_pending_inbound(
        &self,
        chat: &str,
        sender: &str,
        id: &str,
    ) -> Result<Option<Vec<u8>>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let (chat, sender, id) = (chat.to_owned(), sender.to_owned(), id.to_owned());
        self.with_semaphore(move || -> Result<Option<Vec<u8>>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            pending_inbound_messages::table
                .select(pending_inbound_messages::message)
                .filter(pending_inbound_messages::chat.eq(&chat))
                .filter(pending_inbound_messages::sender.eq(&sender))
                .filter(pending_inbound_messages::id.eq(&id))
                .filter(pending_inbound_messages::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))
        })
        .await
    }

    async fn delete_pending_inbound(&self, chat: &str, sender: &str, id: &str) -> Result<()> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let (chat, sender, id) = (chat.to_owned(), sender.to_owned(), id.to_owned());
        self.with_semaphore(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            delete_pending_inbound_row(&mut conn, device_id, &chat, &sender, &id)
                .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(())
        })
        .await
    }

    async fn read_pending_inbound_bounded(
        &self,
        key: PendingInboundKey<'_>,
        max_bytes: usize,
        lease: PendingInboundLease,
    ) -> Result<Option<PendingInboundRecord>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let (chat, sender, id) = (
            key.chat.to_owned(),
            key.sender.to_owned(),
            key.id.to_owned(),
        );
        self.with_semaphore(move || -> Result<Option<PendingInboundRecord>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            read_pending_inbound_bounded_row(
                &mut conn, device_id, &chat, &sender, &id, max_bytes, lease,
            )
            .map_err(|e| StoreError::Database(e.to_string()))?
        })
        .await
    }

    async fn read_pending_inbound_group_bounded(
        &self,
        key: PendingInboundKey<'_>,
        max_bytes: usize,
        lease: PendingInboundLease,
    ) -> Result<Option<(String, PendingInboundRecord)>> {
        if key.sender.len() > 128 {
            return Err(StoreError::Validation(
                "group pending sender exceeds bound".into(),
            ));
        }
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let (chat, sender, id) = (
            key.chat.to_owned(),
            key.sender.to_owned(),
            key.id.to_owned(),
        );
        self.with_semaphore(move || -> Result<Option<(String, PendingInboundRecord)>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // One read transaction prevents alias discovery and payload from
            // observing different external-writer snapshots.
            conn.transaction(
                |conn| -> std::result::Result<
                    Result<Option<(String, PendingInboundRecord)>>,
                    diesel::result::Error,
                > {
                    let rows: Vec<BoundedPendingSender> =
                        crate::bounded_pending_query::BoundedPendingSenders {
                            device_id,
                            chat: &chat,
                            id: &id,
                        }
                        .load(conn)?;
                    let physical = match pending_group_sender(
                        &mut rows.iter().map(|row| row.sender.as_deref()),
                        &sender,
                    ) {
                        Ok(Some(physical)) => physical,
                        Ok(None) => return Ok(Ok(None)),
                        Err(error) => return Ok(Err(error)),
                    };
                    Ok(read_pending_inbound_bounded_row(
                        conn,
                        device_id,
                        &chat,
                        &physical,
                        &id,
                        max_bytes,
                        lease.clone(),
                    )?
                    .map(|record| record.map(|record| (physical, record))))
                },
            )
            .map_err(|e| StoreError::Database(e.to_string()))?
        })
        .await
    }

    async fn replace_pending_inbound_if(
        &self,
        key: PendingInboundKey<'_>,
        expected: PendingInboundRevision,
        replacement: PendingInboundPayload,
        max_bytes: usize,
    ) -> Result<PendingInboundMutation> {
        if replacement.bytes().len() > max_bytes {
            return Err(StoreError::Validation(format!(
                "pending inbound replacement length {} exceeds limit {max_bytes}",
                replacement.bytes().len(),
            )));
        }
        let revision = replacement.revision();
        self.mutate_pending_inbound_owned(
            key,
            expected,
            max_bytes,
            OwnedPendingMutation::Replace {
                payload: Arc::new(replacement),
                revision,
            },
        )
        .await
    }

    async fn delete_pending_inbound_if(
        &self,
        key: PendingInboundKey<'_>,
        expected: PendingInboundRevision,
        max_bytes: usize,
        lease: PendingInboundLease,
    ) -> Result<PendingInboundMutation> {
        self.mutate_pending_inbound_owned(
            key,
            expected,
            max_bytes,
            OwnedPendingMutation::Delete(lease),
        )
        .await
    }

    async fn delete_expired_pending_inbound(&self, cutoff_timestamp: i64) -> Result<u32> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        self.with_semaphore(move || -> Result<u32> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let deleted = diesel::delete(
                pending_inbound_messages::table
                    .filter(pending_inbound_messages::inserted_at.lt(cutoff_timestamp))
                    .filter(pending_inbound_messages::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(deleted as u32)
        })
        .await
    }

    async fn store_pending_inbound_batch(&self, rows: &[PendingInboundRow<'_>]) -> Result<()> {
        if rows.is_empty() {
            return Ok(());
        }
        // One owned copy, then a single transaction amortizes the WAL commit
        // over the whole batch.
        let rows: Vec<(String, String, String, Vec<u8>)> = rows
            .iter()
            .map(|r| {
                (
                    r.chat.to_owned(),
                    r.sender.to_owned(),
                    r.id.to_owned(),
                    r.message.to_vec(),
                )
            })
            .collect();
        let pool = self.pool.clone();
        let device_id = self.device_id;
        self.with_semaphore(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // Per-row statements inside ONE transaction: the WAL commit is the
            // real per-message cost and it is already amortized.
            conn.transaction(|conn| -> std::result::Result<(), diesel::result::Error> {
                for (chat, sender, id, message) in rows.iter() {
                    insert_pending_inbound_row(conn, device_id, chat, sender, id, message)?;
                }
                Ok(())
            })
            .map_err(|e| StoreError::Database(e.to_string()))
        })
        .await
    }

    async fn delete_pending_inbound_batch(&self, keys: &[PendingInboundKey<'_>]) -> Result<()> {
        if keys.is_empty() {
            return Ok(());
        }
        let keys: Vec<(String, String, String)> = keys
            .iter()
            .map(|k| (k.chat.to_owned(), k.sender.to_owned(), k.id.to_owned()))
            .collect();
        let pool = self.pool.clone();
        let device_id = self.device_id;
        self.with_semaphore(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // Diesel's DSL cannot express a composite `(chat, sender, id) IN (...)`
            // tuple filter, and the transaction already amortizes the commit.
            conn.transaction(|conn| -> std::result::Result<(), diesel::result::Error> {
                for (chat, sender, id) in keys.iter() {
                    delete_pending_inbound_row(conn, device_id, chat, sender, id)?;
                }
                Ok(())
            })
            .map_err(|e| StoreError::Database(e.to_string()))
        })
        .await
    }
}

impl SqliteStore {
    // Share the mutation transaction body while retaining the owned lease in
    // the worker, including when its awaiting caller is cancelled.
    async fn mutate_pending_inbound_owned(
        &self,
        key: PendingInboundKey<'_>,
        expected: PendingInboundRevision,
        max_bytes: usize,
        mutation: OwnedPendingMutation,
    ) -> Result<PendingInboundMutation> {
        let (chat, sender, id) = (
            key.chat.to_owned(),
            key.sender.to_owned(),
            key.id.to_owned(),
        );
        let pool = self.pool.clone();
        let device_id = self.device_id;
        self.with_semaphore(move || -> Result<PendingInboundMutation> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // IMMEDIATE takes the write lock before the revision read, so the
            // compare and the write cannot interleave with another writer.
            let outcome: std::result::Result<
                Result<PendingInboundMutation>,
                diesel::result::Error,
            > = conn.immediate_transaction(|conn| {
                let current = match read_pending_inbound_bounded_row(
                    conn,
                    device_id,
                    &chat,
                    &sender,
                    &id,
                    max_bytes,
                    mutation.lease(),
                )? {
                    Ok(current) => current,
                    Err(error) => return Ok(Err(error)),
                };
                let revision = current.as_ref().map_or(
                    PendingInboundRevision::missing(),
                    PendingInboundRecord::revision,
                );
                // Already-replaced is Applied even if the caller's expected
                // revision is stale. Delete has no such shortcut.
                if let OwnedPendingMutation::Replace {
                    revision: replacement,
                    ..
                } = &mutation
                    && revision == *replacement
                {
                    return Ok(Ok(PendingInboundMutation::Applied));
                }
                if revision != expected {
                    return Ok(Ok(PendingInboundMutation::Conflict));
                }
                match &mutation {
                    OwnedPendingMutation::Replace { payload, .. } => {
                        insert_pending_inbound_row(
                            conn,
                            device_id,
                            &chat,
                            &sender,
                            &id,
                            payload.bytes(),
                        )?;
                    }
                    OwnedPendingMutation::Delete(_) => {
                        delete_pending_inbound_row(conn, device_id, &chat, &sender, &id)?;
                    }
                }
                Ok(Ok(PendingInboundMutation::Applied))
            });
            outcome.map_err(|e| StoreError::Database(e.to_string()))?
        })
        .await
    }
}

#[async_trait]
impl MsgSecretStore for SqliteStore {
    async fn put_msg_secrets(&self, entries: Vec<MsgSecretEntry>) -> Result<usize> {
        if entries.is_empty() {
            return Ok(0);
        }

        let pool = self.pool.clone();
        let device_id = self.device_id;
        self.with_semaphore(move || -> Result<usize> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // One transaction around every chunk keeps merge order observable
            // when a batch updates a key it inserted in an earlier statement.
            conn.transaction::<_, diesel::result::Error, _>(|conn| {
                let mut stored = 0usize;
                for chunk in entries.chunks(MSG_SECRET_INSERT_CHUNK_SIZE) {
                    let records: Vec<_> = chunk
                        .iter()
                        .map(|entry| {
                            (
                                msg_secrets::chat.eq(entry.chat.as_ref()),
                                msg_secrets::sender.eq(entry.sender.as_ref()),
                                msg_secrets::msg_id.eq(entry.msg_id.as_ref()),
                                msg_secrets::device_id.eq(device_id),
                                msg_secrets::secret.eq(entry.secret.as_slice()),
                                msg_secrets::expires_at.eq(entry.expires_at),
                                msg_secrets::message_ts.eq(entry.message_ts),
                            )
                        })
                        .collect();
                    stored += diesel::insert_into(msg_secrets::table)
                        .values(records.as_slice())
                        .on_conflict((
                            msg_secrets::chat,
                            msg_secrets::sender,
                            msg_secrets::msg_id,
                            msg_secrets::device_id,
                        ))
                        .do_update()
                        .set((
                            msg_secrets::secret.eq(excluded(msg_secrets::secret)),
                            // Later deadline wins; 0 (never) beats everything.
                            // Mirrors merge_msg_secret_expiry so a redelivery or
                            // edit re-persist never shortens an existing window.
                            msg_secrets::expires_at.eq(
                                diesel::dsl::sql::<diesel::sql_types::BigInt>(
                                    "CASE WHEN msg_secrets.expires_at = 0 \
                                 OR excluded.expires_at = 0 THEN 0 \
                                 ELSE MAX(msg_secrets.expires_at, excluded.expires_at) END",
                                ),
                            ),
                            // Parent event time is immutable; keep the known
                            // (non-zero / later) value across redeliveries.
                            msg_secrets::message_ts.eq(
                                diesel::dsl::sql::<diesel::sql_types::BigInt>(
                                    "MAX(msg_secrets.message_ts, excluded.message_ts)",
                                ),
                            ),
                        ))
                        .execute(conn)?;
                }
                Ok(stored)
            })
            .map_err(|e| StoreError::Database(e.to_string()))
        })
        .await
    }

    async fn get_stored_msg_secret(
        &self,
        chat: &str,
        sender: &str,
        msg_id: &str,
    ) -> Result<Option<StoredMessageSecret>> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        let chat = chat.to_string();
        let sender = sender.to_string();
        let msg_id = msg_id.to_string();
        self.with_semaphore(move || -> Result<Option<StoredMessageSecret>> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            let row: Option<(Vec<u8>, i64)> = msg_secrets::table
                .select((msg_secrets::secret, msg_secrets::message_ts))
                .filter(msg_secrets::chat.eq(&chat))
                .filter(msg_secrets::sender.eq(&sender))
                .filter(msg_secrets::msg_id.eq(&msg_id))
                .filter(msg_secrets::device_id.eq(device_id))
                .first(&mut conn)
                .optional()
                .map_err(|e| StoreError::Database(e.to_string()))?;
            row.map(|(secret, message_ts)| {
                StoredMessageSecret::from_stored_bytes(&secret, message_ts)
            })
            .transpose()
        })
        .await
    }

    async fn delete_expired_msg_secrets(&self, cutoff_timestamp: i64) -> Result<u32> {
        let pool = self.pool.clone();
        let device_id = self.device_id;
        self.with_semaphore(move || -> Result<u32> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;
            // expires_at = 0 means never expire; only passed deadlines go.
            let deleted = diesel::delete(
                msg_secrets::table
                    .filter(msg_secrets::expires_at.ne(0))
                    .filter(msg_secrets::expires_at.le(cutoff_timestamp))
                    .filter(msg_secrets::device_id.eq(device_id)),
            )
            .execute(&mut conn)
            .map_err(|e| StoreError::Database(e.to_string()))?;
            Ok(deleted as u32)
        })
        .await
    }
}

#[async_trait]
impl DeviceStore for SqliteStore {
    async fn save(&self, device: &CoreDevice) -> Result<()> {
        SqliteStore::save_device_data_for_device(self, self.device_id, device).await
    }

    async fn load(&self) -> Result<Option<CoreDevice>> {
        SqliteStore::load_device_data_for_device(self, self.device_id).await
    }

    async fn exists(&self) -> Result<bool> {
        SqliteStore::device_exists(self, self.device_id).await
    }

    async fn create(&self) -> Result<i32> {
        SqliteStore::create_new_device(self).await
    }

    async fn snapshot_db(&self, name: &str, extra_content: Option<&[u8]>) -> Result<()> {
        fn sanitize_snapshot_name(name: &str) -> Result<String> {
            const MAX_LENGTH: usize = 100;

            let sanitized: String = name
                .chars()
                .map(|c| {
                    if c.is_ascii_alphanumeric() || c == '_' || c == '-' || c == '.' {
                        c
                    } else {
                        '_'
                    }
                })
                .collect();

            let sanitized = sanitized
                .split('.')
                .filter(|part| !part.is_empty() && *part != "..")
                .collect::<Vec<_>>()
                .join(".");

            let sanitized = sanitized.trim_matches(['/', '\\', '.']);

            if sanitized.is_empty() {
                return Err(StoreError::Database(
                    "Snapshot name cannot be empty after sanitization".to_string(),
                ));
            }

            if sanitized.len() > MAX_LENGTH {
                return Err(StoreError::Database(format!(
                    "Snapshot name exceeds maximum length of {} characters",
                    MAX_LENGTH
                )));
            }

            Ok(sanitized.to_string())
        }

        let sanitized_name = sanitize_snapshot_name(name)?;

        let pool = self.pool.clone();
        let db_path = self.database_path.clone();
        let extra_data = extra_content.map(|b| b.to_vec());

        tokio::task::spawn_blocking(move || -> Result<()> {
            let mut conn = pool
                .get()
                .map_err(|e| StoreError::Connection(e.to_string()))?;

            let timestamp = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs();

            // Construct target path: db_path.snapshot-TIMESTAMP-SANITIZED_NAME
            let target_path = format!("{}.snapshot-{}-{}", db_path, timestamp, sanitized_name);

            // Use VACUUM INTO to create a consistent backup
            // Note: We escape single quotes in the path just in case
            let query = format!("VACUUM INTO '{}'", target_path.replace("'", "''"));

            diesel::sql_query(query)
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;

            // Save extra content if provided
            if let Some(data) = extra_data {
                let extra_path = format!("{}.json", target_path);
                std::fs::write(&extra_path, data).map_err(|e| {
                    StoreError::Database(format!("Failed to write snapshot extra content: {}", e))
                })?;
            }

            Ok(())
        })
        .await
        .map_err(|e| StoreError::Database(e.to_string()))??;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use bytes::Bytes;

    async fn create_test_store() -> SqliteStore {
        // Counter instead of a timestamp: parallel tests can get identical
        // nanosecond timestamps, and two stores sharing one shared-cache
        // in-memory DB name race their migrations ("table is locked").
        use std::sync::atomic::{AtomicU64, Ordering};
        static TEST_DB_COUNTER: AtomicU64 = AtomicU64::new(0);
        let id = TEST_DB_COUNTER.fetch_add(1, Ordering::Relaxed);
        let db_name = format!("file:memdb_test_{id}?mode=memory&cache=shared");
        SqliteStore::new(&db_name)
            .await
            .expect("Failed to create test store")
    }

    #[test]
    fn test_parse_database_path_regular_path() {
        let path = "/var/lib/whatsapp/database.db";
        let result = parse_database_path(path).unwrap();
        assert_eq!(result, "/var/lib/whatsapp/database.db");
    }

    #[test]
    fn test_parse_database_path_with_sqlite_prefix() {
        let path = "sqlite:///var/lib/whatsapp/database.db";
        let result = parse_database_path(path).unwrap();
        assert_eq!(result, "/var/lib/whatsapp/database.db");
    }

    #[test]
    fn test_parse_database_path_with_query_params() {
        let path = "file:database.db?mode=memory&cache=shared";
        let result = parse_database_path(path).unwrap();
        assert_eq!(result, "file:database.db");
    }

    #[test]
    fn test_parse_database_path_with_fragment() {
        let path = "file:database.db#fragment";
        let result = parse_database_path(path).unwrap();
        assert_eq!(result, "file:database.db");
    }

    #[test]
    fn test_parse_database_path_with_both_query_and_fragment() {
        let path = "sqlite:///var/lib/database.db?mode=ro#backup";
        let result = parse_database_path(path).unwrap();
        assert_eq!(result, "/var/lib/database.db");
    }

    #[test]
    fn test_parse_database_path_in_memory_rejected() {
        let result = parse_database_path(":memory:");
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("not supported"));
    }

    #[test]
    fn test_parse_database_path_in_memory_with_query_rejected() {
        let result = parse_database_path(":memory:?cache=shared");
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("not supported"));
    }

    #[tokio::test]
    async fn test_max_prekey_id_with_gaps() {
        let store = create_test_store().await;

        assert_eq!(store.max_prekey_id().await.unwrap(), None);

        // Store pre-keys 1..=3, then consume the middle one: one-time pre-keys
        // are removed after use, so the stored ID space has gaps in production.
        for id in 1..=3u32 {
            store.store_prekey(id, &[0xAB; 8], true).await.unwrap();
        }
        store.remove_prekey(2).await.unwrap();

        // Must return the true max, not stop at the first gap (which would
        // make new key allocation overwrite still-live keys at higher IDs).
        assert_eq!(store.max_prekey_id().await.unwrap(), Some(3));

        store.remove_prekey(3).await.unwrap();
        assert_eq!(store.max_prekey_id().await.unwrap(), Some(1));
    }

    #[tokio::test]
    async fn test_device_registry_save_and_get() {
        let store = create_test_store().await;

        let record = DeviceListRecord {
            user: "1234567890".to_string(),
            devices: vec![
                DeviceInfo {
                    device_id: 0,
                    key_index: None,
                },
                DeviceInfo {
                    device_id: 1,
                    key_index: Some(42),
                },
            ],
            timestamp: 1234567890,
            phash: Some("2:abcdef".to_string()),
        };

        store.update_device_list(record).await.expect("save failed");
        let loaded = store
            .get_devices("1234567890")
            .await
            .expect("get failed")
            .expect("record should exist");

        assert_eq!(loaded.user, "1234567890");
        assert_eq!(loaded.devices.len(), 2);
        assert_eq!(loaded.devices[0].device_id, 0);
        assert_eq!(loaded.devices[1].device_id, 1);
        assert_eq!(loaded.devices[1].key_index, Some(42));
        assert_eq!(loaded.phash, Some("2:abcdef".to_string()));
    }

    #[tokio::test]
    async fn test_device_registry_update_existing() {
        let store = create_test_store().await;

        let record1 = DeviceListRecord {
            user: "1234567890".to_string(),
            devices: vec![DeviceInfo {
                device_id: 0,
                key_index: None,
            }],
            timestamp: 1000,
            phash: Some("2:old".to_string()),
        };
        store
            .update_device_list(record1)
            .await
            .expect("save1 failed");

        let record2 = DeviceListRecord {
            user: "1234567890".to_string(),
            devices: vec![
                DeviceInfo {
                    device_id: 0,
                    key_index: None,
                },
                DeviceInfo {
                    device_id: 2,
                    key_index: None,
                },
            ],
            timestamp: 2000,
            phash: Some("2:new".to_string()),
        };
        store
            .update_device_list(record2)
            .await
            .expect("save2 failed");

        let loaded = store
            .get_devices("1234567890")
            .await
            .expect("get failed")
            .expect("record should exist");

        assert_eq!(loaded.devices.len(), 2);
        assert_eq!(loaded.phash, Some("2:new".to_string()));
    }

    #[tokio::test]
    async fn test_device_registry_get_nonexistent() {
        let store = create_test_store().await;
        let result = store.get_devices("nonexistent").await.expect("get failed");
        assert!(result.is_none());
    }

    #[tokio::test]
    async fn test_sender_key_status_mark_and_consume() {
        let store = create_test_store().await;

        let group = "group123@g.us";
        let participant = "user1@s.whatsapp.net";

        store
            .mark_forget_sender_key(group, participant)
            .await
            .expect("mark failed");

        let consumed = store
            .consume_forget_marks(group)
            .await
            .expect("consume failed");
        assert_eq!(consumed.len(), 1);
        assert!(consumed.contains(&participant.to_string()));

        let consumed = store
            .consume_forget_marks(group)
            .await
            .expect("consume failed");
        assert!(consumed.is_empty());
    }

    #[tokio::test]
    async fn test_sender_key_status_consume_multiple() {
        let store = create_test_store().await;

        let group = "group123@g.us";

        store
            .mark_forget_sender_key(group, "user1@s.whatsapp.net")
            .await
            .expect("mark failed");
        store
            .mark_forget_sender_key(group, "user2@s.whatsapp.net")
            .await
            .expect("mark failed");

        let consumed = store
            .consume_forget_marks(group)
            .await
            .expect("consume failed");
        assert_eq!(consumed.len(), 2);
        assert!(consumed.contains(&"user1@s.whatsapp.net".to_string()));
        assert!(consumed.contains(&"user2@s.whatsapp.net".to_string()));

        let consumed = store
            .consume_forget_marks(group)
            .await
            .expect("consume failed");
        assert!(consumed.is_empty());
    }

    #[tokio::test]
    async fn test_tc_token_put_and_get() {
        let store = create_test_store().await;

        let entry = TcTokenEntry {
            token: vec![1, 2, 3, 4, 5],
            token_timestamp: 1707000000,
            sender_timestamp: Some(1707000100),
        };

        store
            .put_tc_token("user@lid", &entry)
            .await
            .expect("put failed");

        let loaded = store
            .get_tc_token("user@lid")
            .await
            .expect("get failed")
            .expect("should exist");

        assert_eq!(loaded.token, vec![1, 2, 3, 4, 5]);
        assert_eq!(loaded.token_timestamp, 1707000000);
        assert_eq!(loaded.sender_timestamp, Some(1707000100));
    }

    #[tokio::test]
    async fn test_tc_token_upsert() {
        let store = create_test_store().await;

        let entry1 = TcTokenEntry {
            token: vec![1, 2, 3],
            token_timestamp: 1000,
            sender_timestamp: None,
        };
        store.put_tc_token("user@lid", &entry1).await.unwrap();

        let entry2 = TcTokenEntry {
            token: vec![4, 5, 6],
            token_timestamp: 2000,
            sender_timestamp: Some(1500),
        };
        store.put_tc_token("user@lid", &entry2).await.unwrap();

        let loaded = store.get_tc_token("user@lid").await.unwrap().unwrap();
        assert_eq!(loaded.token, vec![4, 5, 6]);
        assert_eq!(loaded.token_timestamp, 2000);
        assert_eq!(loaded.sender_timestamp, Some(1500));
    }

    #[tokio::test]
    async fn test_tc_token_delete() {
        let store = create_test_store().await;

        let entry = TcTokenEntry {
            token: vec![1, 2, 3],
            token_timestamp: 1000,
            sender_timestamp: None,
        };
        store.put_tc_token("user@lid", &entry).await.unwrap();
        store.delete_tc_token("user@lid").await.unwrap();

        let result = store.get_tc_token("user@lid").await.unwrap();
        assert!(result.is_none());
    }

    #[tokio::test]
    async fn test_tc_token_get_all_jids() {
        let store = create_test_store().await;

        let entry = TcTokenEntry {
            token: vec![1],
            token_timestamp: 1000,
            sender_timestamp: None,
        };
        store.put_tc_token("user1@lid", &entry).await.unwrap();
        store.put_tc_token("user2@lid", &entry).await.unwrap();
        store.put_tc_token("user3@lid", &entry).await.unwrap();

        let mut jids = store.get_all_tc_token_jids().await.unwrap();
        jids.sort();
        assert_eq!(jids, vec!["user1@lid", "user2@lid", "user3@lid"]);
    }

    #[tokio::test]
    async fn test_tc_token_delete_expired() {
        let store = create_test_store().await;

        let old = TcTokenEntry {
            token: vec![1],
            token_timestamp: 1000,
            sender_timestamp: None,
        };
        let recent = TcTokenEntry {
            token: vec![2],
            token_timestamp: 5000,
            sender_timestamp: None,
        };
        store.put_tc_token("old@lid", &old).await.unwrap();
        store.put_tc_token("recent@lid", &recent).await.unwrap();

        let deleted = store.delete_expired_tc_tokens(3000).await.unwrap();
        assert_eq!(deleted, 1);

        assert!(store.get_tc_token("old@lid").await.unwrap().is_none());
        assert!(store.get_tc_token("recent@lid").await.unwrap().is_some());
    }

    #[tokio::test]
    async fn test_tc_token_get_nonexistent() {
        let store = create_test_store().await;
        let result = store.get_tc_token("nonexistent@lid").await.unwrap();
        assert!(result.is_none());
    }

    #[tokio::test]
    async fn test_sender_key_status_different_groups() {
        let store = create_test_store().await;

        let group1 = "group1@g.us";
        let group2 = "group2@g.us";
        let participant = "user@s.whatsapp.net";

        store
            .mark_forget_sender_key(group1, participant)
            .await
            .expect("mark failed");

        let consumed = store.consume_forget_marks(group1).await.unwrap();
        assert_eq!(consumed.len(), 1);

        let consumed = store.consume_forget_marks(group2).await.unwrap();
        assert!(consumed.is_empty());
    }

    /// Round-trips a `CachedServerCertChain` through the SQLite schema:
    /// create → save → load. Exercises the `2026-04-26-000000_add_server_cert_chain`
    /// migration plus the bincode encode/decode path (oxidezap #598).
    #[tokio::test]
    async fn test_server_cert_chain_survives_save_load_roundtrip() {
        let store = create_test_store().await;
        let device_id = store.create_new_device().await.expect("create device");

        let mut device = store
            .load_device_data_for_device(device_id)
            .await
            .expect("load")
            .expect("device should exist after create");
        assert!(device.server_cert_chain.is_none());

        device.server_cert_chain = Some(wacore::store::CachedServerCertChain {
            intermediate: wacore::store::CachedNoiseCert {
                key: [0xAB; 32],
                not_before: 1_700_000_000,
                not_after: 1_900_000_000,
            },
            leaf: wacore::store::CachedNoiseCert {
                key: [0xCD; 32],
                not_before: 1_700_000_500,
                not_after: 1_899_999_500,
            },
        });
        store
            .save_device_data_for_device(device_id, &device)
            .await
            .expect("save with cert chain");

        let loaded = store
            .load_device_data_for_device(device_id)
            .await
            .expect("reload")
            .expect("device should still exist");
        assert_eq!(loaded.server_cert_chain, device.server_cert_chain);
    }

    #[tokio::test]
    async fn msg_secret_unknown_round_trip_has_no_message_ts() {
        const SECRET_BYTE: u8 = 177;
        let store = create_test_store().await;
        store
            .put_msg_secret("c", "s", "unknown", &[SECRET_BYTE; 32])
            .await
            .expect("put");

        let stored = store
            .get_stored_msg_secret("c", "s", "unknown")
            .await
            .expect("get")
            .expect("must exist");
        assert_eq!(stored.message_ts, None);
        assert_eq!(stored.secret.as_bytes(), &[SECRET_BYTE; 32]);
        assert!(!format!("{stored:#?}").contains("177"));

        assert!(
            store
                .get_stored_msg_secret("c", "s", "missing")
                .await
                .expect("get miss")
                .is_none()
        );
    }

    #[tokio::test]
    async fn msg_secret_merge_keeps_later_ts_and_longest_deadline() {
        let store = create_test_store().await;
        for (message_ts, expires_at) in [(120, 250), (0, 150), (90, 190), (100, 200)] {
            store
                .put_msg_secrets(vec![MsgSecretEntry {
                    chat: "c".into(),
                    sender: "s".into(),
                    msg_id: "known".into(),
                    secret: [177; 32],
                    expires_at,
                    message_ts,
                }])
                .await
                .expect("put merge");
        }
        store
            .put_msg_secrets(vec![MsgSecretEntry {
                chat: "c".into(),
                sender: "s".into(),
                msg_id: "never".into(),
                secret: [7; 32],
                expires_at: 0,
                message_ts: 0,
            }])
            .await
            .expect("put never");

        let known = store
            .get_stored_msg_secret("c", "s", "known")
            .await
            .expect("get known")
            .expect("must exist");
        assert_eq!(known.message_ts, Some(120));

        assert_eq!(
            store
                .delete_expired_msg_secrets(249)
                .await
                .expect("prune below deadline"),
            0
        );
        assert_eq!(
            store
                .delete_expired_msg_secrets(250)
                .await
                .expect("prune at deadline"),
            1
        );
        assert!(
            store
                .get_stored_msg_secret("c", "s", "known")
                .await
                .expect("get pruned")
                .is_none()
        );
        assert!(
            store
                .get_stored_msg_secret("c", "s", "never")
                .await
                .expect("get never")
                .is_some(),
            "expires_at = 0 means never expire"
        );
    }

    #[tokio::test]
    async fn msg_secret_store_is_reachable_through_backend_object() {
        let store: Arc<dyn Backend> = Arc::new(create_test_store().await);
        store
            .put_msg_secret("c", "s", "M", &[7; 32])
            .await
            .expect("put via Backend");
        let stored = store
            .get_stored_msg_secret("c", "s", "M")
            .await
            .expect("get via Backend")
            .expect("must exist");
        assert_eq!(stored.secret.as_bytes(), &[7; 32]);
        assert_eq!(
            store
                .delete_expired_msg_secrets(i64::MAX)
                .await
                .expect("prune via Backend"),
            0
        );
    }

    #[tokio::test]
    async fn msg_secret_invalid_persisted_bytes_are_typed_errors() {
        let store = create_test_store().await;
        store
            .put_msg_secret("c", "s", "M", &[177; 32])
            .await
            .expect("put");

        let pool = store.pool.clone();
        let device_id = store.device_id;
        store
            .with_semaphore(move || {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                diesel::update(
                    msg_secrets::table
                        .filter(msg_secrets::chat.eq("c"))
                        .filter(msg_secrets::sender.eq("s"))
                        .filter(msg_secrets::msg_id.eq("M"))
                        .filter(msg_secrets::device_id.eq(device_id)),
                )
                .set(msg_secrets::secret.eq(vec![177u8; 31]))
                .execute(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
                Ok(())
            })
            .await
            .expect("corrupt row");

        let error = store
            .get_stored_msg_secret("c", "s", "M")
            .await
            .expect_err("malformed secret must fail");
        match &error {
            StoreError::InvalidMessageSecret(source) => assert_eq!(source.actual, 31),
            other => panic!("expected InvalidMessageSecret, got {other:?}"),
        }
        assert!(!format!("{error:#?}").contains("177"));
        assert!(matches!(
            store.get_msg_secret("c", "s", "M").await,
            Err(StoreError::InvalidMessageSecret(_))
        ));
    }

    async fn create_test_store_for_device(device_id: i32) -> (SqliteStore, String) {
        use std::sync::atomic::{AtomicU64, Ordering};
        static PENDING_DB_COUNTER: AtomicU64 = AtomicU64::new(0);
        let id = PENDING_DB_COUNTER.fetch_add(1, Ordering::Relaxed);
        let db_name = format!("file:memdb_pending_{id}?mode=memory&cache=shared");
        let store = SqliteStore::new_for_device(&db_name, device_id)
            .await
            .expect("Failed to create test store");
        (store, db_name)
    }

    async fn pending_execute(store: &SqliteStore, sql: &'static str) {
        let pool = store.pool.clone();
        store
            .with_semaphore(move || {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                diesel::sql_query(sql)
                    .execute(&mut conn)
                    .map_err(|e| StoreError::Database(e.to_string()))?;
                Ok(())
            })
            .await
            .expect("pending test SQL");
    }

    fn pending_test_lease() -> PendingInboundLease {
        PendingInboundLease::new(Arc::new(()))
    }

    #[tokio::test]
    async fn pending_inbound_round_trip_and_missing_row() {
        let store = create_test_store().await;
        let (chat, sender, id) = ("c@s.whatsapp.net", "s@s.whatsapp.net", "MISSING");
        assert!(
            store
                .get_pending_inbound(chat, sender, id)
                .await
                .unwrap()
                .is_none()
        );
        store
            .store_pending_inbound(chat, sender, id, b"payload")
            .await
            .unwrap();
        assert_eq!(
            store
                .get_pending_inbound(chat, sender, id)
                .await
                .unwrap()
                .as_deref(),
            Some(b"payload".as_slice())
        );
        store
            .delete_pending_inbound(chat, sender, id)
            .await
            .unwrap();
        assert!(
            store
                .get_pending_inbound(chat, sender, id)
                .await
                .unwrap()
                .is_none()
        );
        // Deleting an absent row is a benign no-op.
        store
            .delete_pending_inbound(chat, sender, id)
            .await
            .unwrap();
    }

    #[tokio::test]
    async fn pending_inbound_batches_write_and_remove_every_row() {
        let store = create_test_store().await;
        let rows = [
            PendingInboundRow {
                chat: "chat-a",
                sender: "sender-a",
                id: "id-1",
                message: b"one",
            },
            PendingInboundRow {
                chat: "chat-a",
                sender: "sender-a",
                id: "id-2",
                message: b"two",
            },
            PendingInboundRow {
                chat: "chat-b",
                sender: "sender-b",
                id: "id-1",
                message: b"three",
            },
        ];
        store.store_pending_inbound_batch(&rows).await.unwrap();
        assert_eq!(
            store
                .get_pending_inbound("chat-a", "sender-a", "id-1")
                .await
                .unwrap()
                .as_deref(),
            Some(b"one".as_slice())
        );
        assert_eq!(
            store
                .get_pending_inbound("chat-a", "sender-a", "id-2")
                .await
                .unwrap()
                .as_deref(),
            Some(b"two".as_slice())
        );
        assert_eq!(
            store
                .get_pending_inbound("chat-b", "sender-b", "id-1")
                .await
                .unwrap()
                .as_deref(),
            Some(b"three".as_slice())
        );
        let keys = [
            PendingInboundKey {
                chat: "chat-a",
                sender: "sender-a",
                id: "id-1",
            },
            PendingInboundKey {
                chat: "chat-a",
                sender: "sender-a",
                id: "id-2",
            },
            PendingInboundKey {
                chat: "chat-b",
                sender: "sender-b",
                id: "id-1",
            },
        ];
        store.delete_pending_inbound_batch(&keys).await.unwrap();
        for key in keys {
            assert!(
                store
                    .get_pending_inbound(key.chat, key.sender, key.id)
                    .await
                    .unwrap()
                    .is_none()
            );
        }
        // Empty batches stay no-ops rather than opening a transaction.
        store.store_pending_inbound_batch(&[]).await.unwrap();
        store.delete_pending_inbound_batch(&[]).await.unwrap();
    }

    #[tokio::test]
    async fn pending_inbound_expiry_uses_strict_cutoff_and_device_scope() {
        let (store, db_url) = create_test_store_for_device(1).await;
        let other = SqliteStore::new_for_device(&db_url, 2).await.unwrap();
        let (chat, sender) = ("1@s.whatsapp.net", "2@s.whatsapp.net");
        for key in [&store, &other] {
            key.store_pending_inbound(chat, sender, "OLD", b"msg")
                .await
                .unwrap();
            key.store_pending_inbound(chat, sender, "BOUNDARY", b"msg")
                .await
                .unwrap();
        }
        pending_execute(
            &store,
            "UPDATE pending_inbound_messages SET inserted_at = 100 WHERE device_id = 1 AND id = 'OLD'",
        )
        .await;
        pending_execute(
            &store,
            "UPDATE pending_inbound_messages SET inserted_at = 200 WHERE device_id = 1 AND id = 'BOUNDARY'",
        )
        .await;
        pending_execute(
            &other,
            "UPDATE pending_inbound_messages SET inserted_at = 100 WHERE device_id = 2",
        )
        .await;

        // Strict `<` cutoff: a row exactly at the cutoff survives.
        assert_eq!(store.delete_expired_pending_inbound(100).await.unwrap(), 0);
        assert_eq!(store.delete_expired_pending_inbound(101).await.unwrap(), 1);
        assert!(
            store
                .get_pending_inbound(chat, sender, "OLD")
                .await
                .unwrap()
                .is_none()
        );
        assert!(
            store
                .get_pending_inbound(chat, sender, "BOUNDARY")
                .await
                .unwrap()
                .is_some()
        );
        // Device 2 keeps its own row even though device 1's was swept.
        assert!(
            other
                .get_pending_inbound(chat, sender, "OLD")
                .await
                .unwrap()
                .is_some()
        );
    }

    #[tokio::test]
    async fn pending_bounded_read_distinguishes_absence_limit_and_non_blob() {
        let store = create_test_store().await;
        let key = PendingInboundKey {
            chat: "test-chat",
            sender: "test-sender",
            id: "bounded",
        };
        assert!(
            store
                .read_pending_inbound_bounded(key, 16, pending_test_lease())
                .await
                .unwrap()
                .is_none()
        );
        store
            .store_pending_inbound(key.chat, key.sender, key.id, b"abc")
            .await
            .unwrap();

        // Exact limit succeeds; one byte over is a Validation error, not absence.
        let record = store
            .read_pending_inbound_bounded(key, 3, pending_test_lease())
            .await
            .unwrap()
            .unwrap();
        assert_eq!(record.bytes(), b"abc");
        assert_eq!(record.revision(), PendingInboundRevision::for_bytes(b"abc"));
        let error = store
            .read_pending_inbound_bounded(key, 2, pending_test_lease())
            .await
            .unwrap_err();
        assert!(matches!(error, StoreError::Validation(_)));

        // Empty bytes are a present row even at a zero limit.
        store
            .store_pending_inbound(key.chat, key.sender, key.id, b"")
            .await
            .unwrap();
        let record = store
            .read_pending_inbound_bounded(key, 0, pending_test_lease())
            .await
            .unwrap()
            .unwrap();
        assert!(record.bytes().is_empty());
        assert_ne!(record.revision(), PendingInboundRevision::missing());

        // A TEXT payload in the BLOB column is rejected, never returned.
        pending_execute(
            &store,
            "INSERT INTO pending_inbound_messages (chat, sender, id, message, device_id) \
             VALUES ('test-chat', 'test-sender', 'text-row', 'not a blob', 1)",
        )
        .await;
        let text_key = PendingInboundKey {
            id: "text-row",
            ..key
        };
        let error = store
            .read_pending_inbound_bounded(text_key, 16, pending_test_lease())
            .await
            .unwrap_err();
        assert!(matches!(error, StoreError::Validation(_)));
    }

    #[tokio::test]
    async fn pending_cas_replace_and_delete_revision_semantics() {
        let store = create_test_store().await;
        let key = PendingInboundKey {
            chat: "test-chat",
            sender: "test-sender",
            id: "cas",
        };
        let v1 = PendingInboundPayload::new(Bytes::from_static(b"v1"), pending_test_lease());
        let v1_revision = v1.revision();
        // Absence is an explicit expected revision.
        assert_eq!(
            store
                .replace_pending_inbound_if(key, PendingInboundRevision::missing(), v1, 16)
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        assert_eq!(
            store
                .get_pending_inbound(key.chat, key.sender, key.id)
                .await
                .unwrap()
                .as_deref(),
            Some(b"v1".as_slice())
        );
        // A stale expected revision leaves the row untouched.
        assert_eq!(
            store
                .replace_pending_inbound_if(
                    key,
                    PendingInboundRevision::missing(),
                    PendingInboundPayload::new(Bytes::from_static(b"stale"), pending_test_lease()),
                    16,
                )
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
        // Re-installing the exact stored revision is an idempotent success.
        assert_eq!(
            store
                .replace_pending_inbound_if(
                    key,
                    PendingInboundRevision::missing(),
                    PendingInboundPayload::new(Bytes::from_static(b"v1"), pending_test_lease()),
                    16,
                )
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        // The observed revision enables the next replacement.
        assert_eq!(
            store
                .replace_pending_inbound_if(
                    key,
                    v1_revision,
                    PendingInboundPayload::new(Bytes::from_static(b"v2"), pending_test_lease()),
                    16,
                )
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        // That revision is now stale.
        assert_eq!(
            store
                .replace_pending_inbound_if(
                    key,
                    v1_revision,
                    PendingInboundPayload::new(Bytes::from_static(b"v3"), pending_test_lease()),
                    16,
                )
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
        let record = store
            .read_pending_inbound_bounded(key, 16, pending_test_lease())
            .await
            .unwrap()
            .unwrap();
        assert_eq!(record.bytes(), b"v2");
        assert_eq!(
            store
                .delete_pending_inbound_if(key, record.revision(), 16, record.lease())
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        assert_eq!(
            store
                .delete_pending_inbound_if(key, record.revision(), 16, pending_test_lease())
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
        // Oversized replacements fail before touching the row.
        let error = store
            .replace_pending_inbound_if(
                key,
                PendingInboundRevision::missing(),
                PendingInboundPayload::new(Bytes::from_static(b"too-large"), pending_test_lease()),
                2,
            )
            .await
            .unwrap_err();
        assert!(matches!(error, StoreError::Validation(_)));
    }

    #[tokio::test]
    async fn pending_conditional_updates_preserve_noncanonical_legacy_bytes() {
        let store = create_test_store().await;
        let key = PendingInboundKey {
            chat: "test-chat",
            sender: "test-sender",
            id: "legacy",
        };
        // The same conversation encoded with a compact and an extended varint.
        let compact = [0x0a, 0x01, b'a'];
        let extended = [0x0a, 0x81, 0x00, b'a'];
        assert_eq!(
            waproto::codec::message_decode(&compact)
                .unwrap()
                .conversation,
            waproto::codec::message_decode(&extended)
                .unwrap()
                .conversation
        );
        store
            .store_pending_inbound(key.chat, key.sender, key.id, &compact)
            .await
            .unwrap();
        let original = store
            .read_pending_inbound_bounded(key, 16, pending_test_lease())
            .await
            .unwrap()
            .unwrap();
        assert_eq!(original.bytes(), compact);
        let replacement =
            PendingInboundPayload::new(Bytes::copy_from_slice(&extended), pending_test_lease());
        let installed = replacement.revision();
        assert_eq!(
            store
                .replace_pending_inbound_if(key, original.revision(), replacement, 16)
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        let successor = store
            .read_pending_inbound_bounded(key, 16, pending_test_lease())
            .await
            .unwrap()
            .unwrap();
        // The exact bytes are preserved: no decode/re-encode canonicalisation.
        assert_eq!(successor.bytes(), extended);
        assert_eq!(successor.revision(), installed);
        assert_ne!(successor.revision(), original.revision());
        assert_eq!(
            store
                .replace_pending_inbound_if(
                    key,
                    original.revision(),
                    PendingInboundPayload::new(
                        Bytes::copy_from_slice(b"stale"),
                        pending_test_lease()
                    ),
                    16
                )
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
        assert_eq!(
            store
                .delete_pending_inbound_if(key, original.revision(), 16, pending_test_lease())
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
        // Re-installing the exact current bytes succeeds idempotently.
        assert_eq!(
            store
                .replace_pending_inbound_if(
                    key,
                    original.revision(),
                    PendingInboundPayload::new(
                        Bytes::copy_from_slice(&extended),
                        pending_test_lease()
                    ),
                    16
                )
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        assert_eq!(
            store
                .get_pending_inbound(key.chat, key.sender, key.id)
                .await
                .unwrap()
                .unwrap(),
            extended
        );
    }

    #[tokio::test]
    async fn pending_bounded_group_resolves_unique_alias_and_rejects_ambiguity() {
        let store = create_test_store().await;
        let chat = "120363000000000081@g.us";
        let sender = "200:75@lid";
        let key = PendingInboundKey {
            chat,
            sender,
            id: "LEGACY_ALIAS",
        };
        let logical = PendingInboundKey {
            sender: "200@lid",
            ..key
        };
        store
            .store_pending_inbound(chat, sender, key.id, b"old")
            .await
            .unwrap();
        store
            .store_pending_inbound(chat, "200:75@s.whatsapp.net", key.id, b"other")
            .await
            .unwrap();
        let owner = Arc::new(());
        let weak = Arc::downgrade(&owner);
        let (physical, record) = store
            .read_pending_inbound_group_bounded(
                logical,
                16,
                PendingInboundLease::new(owner.clone()),
            )
            .await
            .unwrap()
            .unwrap();
        drop(owner);
        assert!(
            weak.upgrade().is_some(),
            "the returned record retains the lease owner"
        );
        assert_eq!(physical, sender);
        assert_eq!(record.bytes(), b"old");
        let stale = record.revision();
        drop(record);
        assert!(weak.upgrade().is_none());

        // CAS must target the physical spelling returned above.
        store
            .store_pending_inbound(chat, sender, key.id, b"new")
            .await
            .unwrap();
        assert_eq!(
            store
                .delete_pending_inbound_if(key, stale, 16, pending_test_lease())
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
        // Oversized payload discovered through the alias is an error.
        assert!(
            store
                .read_pending_inbound_group_bounded(logical, 2, pending_test_lease())
                .await
                .is_err()
        );
        // Two physical keys sharing the logical identity are ambiguous.
        store
            .store_pending_inbound(chat, logical.sender, key.id, b"ambiguous")
            .await
            .unwrap();
        assert!(
            store
                .read_pending_inbound_group_bounded(logical, 16, pending_test_lease())
                .await
                .is_err()
        );
        assert_eq!(
            store
                .get_pending_inbound(chat, sender, key.id)
                .await
                .unwrap()
                .as_deref(),
            Some(b"new".as_slice())
        );
        // Removing the ambiguous alias leaves the legacy physical row as the
        // only match, so the group read resolves it again.
        store
            .delete_pending_inbound(chat, logical.sender, key.id)
            .await
            .unwrap();
        let (_, record) = store
            .read_pending_inbound_group_bounded(logical, 16, pending_test_lease())
            .await
            .unwrap()
            .unwrap();
        assert_eq!(
            store
                .delete_pending_inbound_if(key, record.revision(), 16, record.lease())
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        assert!(
            store
                .read_pending_inbound_group_bounded(logical, 16, pending_test_lease())
                .await
                .unwrap()
                .is_none()
        );
    }

    #[tokio::test]
    async fn pending_group_sender_rejects_oversized_and_too_many_aliases() {
        let long = "a".repeat(129);
        let error =
            pending_group_sender(&mut [Some(long.as_str())].into_iter(), "200@lid").unwrap_err();
        assert!(matches!(error, StoreError::Validation(_)));

        let error =
            pending_group_sender(&mut [Some(long.as_str())].into_iter(), &long).unwrap_err();
        assert!(matches!(error, StoreError::Validation(_)));

        let spellings: Vec<String> = (0..401).map(|i| format!("{i}:75@lid")).collect();
        let error = pending_group_sender(
            &mut spellings.iter().map(|spelling| Some(spelling.as_str())),
            "none@lid",
        )
        .unwrap_err();
        assert!(matches!(error, StoreError::Validation(_)));

        // Exactly 400 distinct non-matching keys are within the bound.
        let spellings: Vec<String> = (0..400).map(|i| format!("{i}:75@lid")).collect();
        assert_eq!(
            pending_group_sender(
                &mut spellings.iter().map(|spelling| Some(spelling.as_str())),
                "none@lid",
            )
            .unwrap(),
            None
        );
    }

    #[test]
    fn pending_bounded_statement_reuses_cache_and_rebinds_all_parameters() {
        use diesel::connection::{InstrumentationEvent, SimpleConnection};
        use std::sync::atomic::{AtomicUsize, Ordering};

        let mut conn = SqliteConnection::establish(":memory:").unwrap();
        conn.batch_execute(
            "CREATE TABLE pending_inbound_messages \
             (device_id INTEGER, chat TEXT, sender TEXT, id TEXT, message BLOB); \
             INSERT INTO pending_inbound_messages VALUES \
             (1, 'chat-a', 'sender-a', 'id-a', x'616161'), \
             (2, 'chat-a', 'sender-a', 'id-a', x'626262'), \
             (1, 'chat-b', 'sender-a', 'id-a', x'636363'), \
             (1, 'chat-a', 'sender-b', 'id-a', x'646464'), \
             (1, 'chat-a', 'sender-a', 'id-b', x'656565');",
        )
        .unwrap();
        let cached = Arc::new(AtomicUsize::new(0));
        let observed = cached.clone();
        conn.set_instrumentation(move |event: InstrumentationEvent<'_>| {
            if let InstrumentationEvent::CacheQuery { .. } = event {
                observed.fetch_add(1, Ordering::SeqCst);
            }
        });
        for (max_bytes, device_id, chat, sender, id, expected) in [
            (3, 1, "chat-a", "sender-a", "id-a", Some(b"aaa")),
            (2, 1, "chat-a", "sender-a", "id-a", None),
            (3, 2, "chat-a", "sender-a", "id-a", Some(b"bbb")),
            (3, 1, "chat-b", "sender-a", "id-a", Some(b"ccc")),
            (3, 1, "chat-a", "sender-b", "id-a", Some(b"ddd")),
            (3, 1, "chat-a", "sender-a", "id-b", Some(b"eee")),
            (3, 1, "chat-a", "sender-a", "id-a", Some(b"aaa")),
        ] {
            let row: BoundedPendingPayload = crate::bounded_pending_query::BoundedPendingQuery {
                max_bytes,
                device_id,
                chat,
                sender,
                id,
            }
            .get_result(&mut conn)
            .unwrap();
            assert_eq!(row.storage_type, "blob");
            assert_eq!(row.byte_len, Some(3));
            assert_eq!(
                row.payload.as_deref(),
                expected.map(|value| value.as_slice())
            );
        }
        assert_eq!(cached.load(Ordering::SeqCst), 1);
    }

    #[test]
    fn pending_bounded_group_metadata_sql_limits_copying_and_rebinds_cache() {
        use diesel::connection::{InstrumentationEvent, SimpleConnection};
        use std::sync::atomic::{AtomicUsize, Ordering};
        let mut conn = SqliteConnection::establish(":memory:").unwrap();
        conn.batch_execute("CREATE TABLE pending_inbound_messages (device_id INTEGER, chat TEXT, sender TEXT, id TEXT, message BLOB);
            INSERT INTO pending_inbound_messages VALUES
            (1, 'chat-a', '200:75@lid', 'id-a', zeroblob(4194304)),
            (2, 'chat-a', '300:76@lid', 'id-a', x'01'),
            (1, 'chat-b', '400@lid', 'id-a', x'01'),
            (1, 'chat-a', '500@lid', 'id-b', x'01');").unwrap();
        diesel::sql_query(
            "INSERT INTO pending_inbound_messages VALUES (1, 'huge', ?, 'id-a', x'01')",
        )
        .bind::<diesel::sql_types::Text, _>("2".repeat(4096))
        .execute(&mut conn)
        .unwrap();
        let cached = Arc::new(AtomicUsize::new(0));
        let observed = cached.clone();
        conn.set_instrumentation(move |event: InstrumentationEvent<'_>| {
            if let InstrumentationEvent::CacheQuery { .. } = event {
                observed.fetch_add(1, Ordering::SeqCst);
            }
        });
        for (device_id, chat, id, expected) in [
            (1, "chat-a", "id-a", Some("200:75@lid")),
            (2, "chat-a", "id-a", Some("300:76@lid")),
            (1, "chat-b", "id-a", Some("400@lid")),
            (1, "chat-a", "id-b", Some("500@lid")),
            (1, "huge", "id-a", None),
        ] {
            let rows: Vec<BoundedPendingSender> =
                crate::bounded_pending_query::BoundedPendingSenders {
                    device_id,
                    chat,
                    id,
                }
                .load(&mut conn)
                .unwrap();
            assert_eq!(rows.len(), 1);
            assert_eq!(rows[0].sender.as_deref(), expected);
        }
        assert_eq!(cached.load(Ordering::SeqCst), 1);
        conn.batch_execute("WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<402)
            INSERT INTO pending_inbound_messages SELECT 1, 'many', x || '@lid', 'id-a', x'01' FROM n;").unwrap();
        let rows: Vec<BoundedPendingSender> = crate::bounded_pending_query::BoundedPendingSenders {
            device_id: 1,
            chat: "many",
            id: "id-a",
        }
        .load(&mut conn)
        .unwrap();
        assert_eq!(rows.len(), 401);
        assert!(
            pending_group_sender(&mut rows.iter().map(|row| row.sender.as_deref()), "1@lid")
                .is_err()
        );
    }

    #[tokio::test]
    async fn pending_bounded_rejects_oversized_blob_and_large_text_without_mutation() {
        let store = create_test_store().await;
        let key = PendingInboundKey {
            chat: "test-chat",
            sender: "test-sender",
            id: "oversized",
        };
        let payload = vec![0x7a; 2 * 1024 * 1024];
        store
            .store_pending_inbound(key.chat, key.sender, key.id, &payload)
            .await
            .unwrap();
        let revision = PendingInboundRevision::for_bytes(&payload);
        assert!(
            store
                .read_pending_inbound_bounded(key, 16, pending_test_lease())
                .await
                .is_err()
        );
        assert!(
            store
                .replace_pending_inbound_if(
                    key,
                    revision,
                    PendingInboundPayload::new(Bytes::from_static(b"new"), pending_test_lease()),
                    16,
                )
                .await
                .is_err()
        );
        assert!(
            store
                .delete_pending_inbound_if(key, revision, 16, pending_test_lease())
                .await
                .is_err()
        );
        assert_eq!(
            store
                .get_pending_inbound(key.chat, key.sender, key.id)
                .await
                .unwrap()
                .unwrap(),
            payload
        );
        // Corrupt the column to TEXT: every bounded operation rejects it and
        // leaves the original row in place.
        pending_execute(
            &store,
            "UPDATE pending_inbound_messages SET message = replace(hex(zeroblob(1048576)), '0', 'x') WHERE id = 'oversized'",
        )
        .await;
        assert!(
            store
                .read_pending_inbound_bounded(key, 16, pending_test_lease())
                .await
                .is_err()
        );
        assert!(
            store
                .replace_pending_inbound_if(
                    key,
                    revision,
                    PendingInboundPayload::new(Bytes::from_static(b"new"), pending_test_lease()),
                    16,
                )
                .await
                .is_err()
        );
        assert!(
            store
                .delete_pending_inbound_if(key, revision, 16, pending_test_lease())
                .await
                .is_err()
        );
        let pool = store.pool.clone();
        store
            .with_semaphore(move || {
                let mut conn = pool
                    .get()
                    .map_err(|e| StoreError::Connection(e.to_string()))?;
                let row: BoundedPendingPayload = diesel::sql_query(
                    "SELECT typeof(message) AS storage_type, NULL AS byte_len, NULL AS payload FROM pending_inbound_messages WHERE id = 'oversized'",
                )
                .get_result(&mut conn)
                .map_err(|e| StoreError::Database(e.to_string()))?;
                assert_eq!(row.storage_type, "text");
                Ok(())
            })
            .await
            .unwrap();
    }

    #[tokio::test]
    async fn pending_bounded_empty_is_present_and_delete_requires_revision() {
        let store = create_test_store().await;
        let key = PendingInboundKey {
            chat: "test-chat",
            sender: "test-sender",
            id: "empty",
        };
        assert!(
            store
                .read_pending_inbound_bounded(key, 0, pending_test_lease())
                .await
                .unwrap()
                .is_none()
        );
        assert_eq!(
            store
                .delete_pending_inbound_if(
                    key,
                    PendingInboundRevision::missing(),
                    0,
                    pending_test_lease()
                )
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        store
            .store_pending_inbound(key.chat, key.sender, key.id, b"")
            .await
            .unwrap();
        let row = store
            .read_pending_inbound_bounded(key, 0, pending_test_lease())
            .await
            .unwrap()
            .unwrap();
        assert!(row.bytes().is_empty());
        assert_ne!(row.revision(), PendingInboundRevision::missing());
        assert_eq!(
            store
                .delete_pending_inbound_if(
                    key,
                    PendingInboundRevision::missing(),
                    0,
                    pending_test_lease()
                )
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
        assert_eq!(
            store
                .delete_pending_inbound_if(key, row.revision(), 0, pending_test_lease())
                .await
                .unwrap(),
            PendingInboundMutation::Applied
        );
        assert_eq!(
            store
                .delete_pending_inbound_if(key, row.revision(), 0, pending_test_lease())
                .await
                .unwrap(),
            PendingInboundMutation::Conflict
        );
    }

    struct PendingTestOwner {
        _lock: tokio::sync::OwnedMutexGuard<()>,
        charged: Arc<std::sync::atomic::AtomicUsize>,
        amount: usize,
        released: std::sync::mpsc::Sender<()>,
    }

    impl Drop for PendingTestOwner {
        fn drop(&mut self) {
            self.charged
                .fetch_sub(self.amount, std::sync::atomic::Ordering::SeqCst);
            let _ = self.released.send(());
        }
    }

    #[tokio::test]
    async fn pending_owned_read_result_keeps_budget_and_identity_lock() {
        use std::sync::atomic::{AtomicUsize, Ordering};
        use std::time::Duration;
        let store = create_test_store().await;
        let key = PendingInboundKey {
            chat: "test-chat",
            sender: "test-sender",
            id: "owned-read",
        };
        store
            .store_pending_inbound(key.chat, key.sender, key.id, b"raw")
            .await
            .unwrap();
        let lock = Arc::new(tokio::sync::Mutex::new(()));
        let charged = Arc::new(AtomicUsize::new(16));
        let (released, completion) = std::sync::mpsc::channel();
        let lease = PendingInboundLease::new(Arc::new(PendingTestOwner {
            _lock: lock.clone().lock_owned().await,
            charged: charged.clone(),
            amount: 16,
            released,
        }));
        let record = store
            .read_pending_inbound_bounded(key, 16, lease)
            .await
            .unwrap()
            .unwrap();
        assert_eq!(record.bytes(), b"raw");
        assert!(
            lock.try_lock().is_err(),
            "the returned bytes still own their identity"
        );
        assert_eq!(charged.load(Ordering::SeqCst), 16);
        drop(record);
        completion.recv_timeout(Duration::from_secs(5)).unwrap();
        assert!(lock.try_lock().is_ok());
        assert_eq!(charged.load(Ordering::SeqCst), 0);
    }

    #[test]
    fn pending_owned_cancelled_cas_keeps_worker_budget_and_excludes_successor() {
        use std::future::{Future, poll_fn};
        use std::sync::atomic::{AtomicUsize, Ordering};
        use std::task::Poll;
        use std::time::Duration;
        let runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .max_blocking_threads(1)
            .build()
            .unwrap();
        runtime.block_on(async {
            let store = create_test_store().await;
            let key = PendingInboundKey {
                chat: "test-chat",
                sender: "test-sender",
                id: "cancelled-cas",
            };
            let lock = Arc::new(tokio::sync::Mutex::new(()));
            let charged = Arc::new(AtomicUsize::new(32));
            let (released, completion) = std::sync::mpsc::channel();
            let owner = Arc::new(PendingTestOwner {
                _lock: lock.clone().lock_owned().await,
                charged: charged.clone(),
                amount: 32,
                released,
            });
            let weak = Arc::downgrade(&owner);
            let request = PendingInboundPayload::new(
                Bytes::from_static(b"raw"),
                PendingInboundLease::new(owner),
            );
            // One blocking slot makes the actual CAS worker queue deterministic.
            let (started, entered) = std::sync::mpsc::channel();
            let (unpark, parked) = std::sync::mpsc::channel();
            let blocker = tokio::task::spawn_blocking(move || {
                started.send(()).unwrap();
                parked.recv_timeout(Duration::from_secs(10)).unwrap();
            });
            entered.recv_timeout(Duration::from_secs(5)).unwrap();
            let mut operation = Box::pin(store.replace_pending_inbound_if(
                key,
                PendingInboundRevision::missing(),
                request,
                16,
            ));
            assert!(poll_fn(|cx| Poll::Ready(operation.as_mut().poll(cx).is_pending())).await);
            drop(operation);
            assert!(
                weak.upgrade().is_some(),
                "cancelled caller must not release the queued worker"
            );
            assert!(
                lock.try_lock().is_err(),
                "a successor cannot enter before worker terminal"
            );
            assert_eq!(charged.load(Ordering::SeqCst), 32);
            unpark.send(()).unwrap();
            completion.recv_timeout(Duration::from_secs(5)).unwrap();
            blocker.await.unwrap();
            assert!(weak.upgrade().is_none());
            assert_eq!(charged.load(Ordering::SeqCst), 0);
            let successor = lock.clone().lock_owned().await;
            let record = store
                .read_pending_inbound_bounded(key, 16, pending_test_lease())
                .await
                .unwrap()
                .unwrap();
            assert_eq!(record.bytes(), b"raw");
            assert_eq!(
                store
                    .delete_pending_inbound_if(key, record.revision(), 16, pending_test_lease())
                    .await
                    .unwrap(),
                PendingInboundMutation::Applied
            );
            drop(successor);
            assert!(
                store
                    .read_pending_inbound_bounded(key, 16, pending_test_lease())
                    .await
                    .unwrap()
                    .is_none()
            );
        });
    }
}
