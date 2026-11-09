//! Device-memo diagnostics and per-session memory/resource reports.
//!
//! Counts are monotonic for the client's lifetime; take two snapshots and
//! subtract to scope them to a workload. Numbers only, no JIDs.

use std::fmt;

use wacore::stats::{
    AllocSnapshot, CollectionStats, HttpResourceReport, StorageResourceReport,
    TransportResourceReport,
};

/// Outcomes of `resolve_group_devices_memoized`, one per call.
#[non_exhaustive]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct GroupDevicesMemoStats {
    pub hits: u64,
    pub restamps: u64,
    pub miss_absent: u64,
    pub miss_group_info: u64,
    pub miss_topology: u64,
    pub bypassed: u64,
}

impl GroupDevicesMemoStats {
    pub fn calls(&self) -> u64 {
        self.hits
            + self.restamps
            + self.miss_absent
            + self.miss_group_info
            + self.miss_topology
            + self.bypassed
    }

    /// Share of calls served without re-resolving the device list.
    pub fn served_rate(&self) -> Option<f64> {
        let calls = self.calls();
        (calls > 0).then(|| (self.hits + self.restamps) as f64 / calls as f64)
    }
}

/// Outcomes of `resolve_skdm_targets_memoized`, one per call.
#[non_exhaustive]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct SkdmTargetsMemoStats {
    pub hits: u64,
    pub miss_absent: u64,
    pub miss_devices: u64,
    pub miss_map: u64,
    pub miss_map_generation: u64,
    pub miss_sender: u64,
    pub not_stored: u64,
    pub bypassed: u64,
    pub resolve_failed: u64,
}

impl SkdmTargetsMemoStats {
    pub fn calls(&self) -> u64 {
        self.hits
            + self.miss_absent
            + self.miss_devices
            + self.miss_map
            + self.miss_map_generation
            + self.miss_sender
            + self.bypassed
            + self.resolve_failed
    }

    /// Memo hits over every call to the resolver.
    pub fn hit_rate(&self) -> Option<f64> {
        let calls = self.calls();
        (calls > 0).then(|| self.hits as f64 / calls as f64)
    }
}

/// Cumulative per-term outcomes of both device-list memos.
#[non_exhaustive]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct DeviceMemoStats {
    pub group_devices: GroupDevicesMemoStats,
    pub skdm_targets: SkdmTargetsMemoStats,
}

impl DeviceMemoStats {
    /// This snapshot minus an earlier one, saturating at zero.
    pub fn since(&self, earlier: &Self) -> Self {
        let group = &self.group_devices;
        let was = &earlier.group_devices;
        let skdm = &self.skdm_targets;
        let skdm_was = &earlier.skdm_targets;
        Self {
            group_devices: GroupDevicesMemoStats {
                hits: group.hits.saturating_sub(was.hits),
                restamps: group.restamps.saturating_sub(was.restamps),
                miss_absent: group.miss_absent.saturating_sub(was.miss_absent),
                miss_group_info: group.miss_group_info.saturating_sub(was.miss_group_info),
                miss_topology: group.miss_topology.saturating_sub(was.miss_topology),
                bypassed: group.bypassed.saturating_sub(was.bypassed),
            },
            skdm_targets: SkdmTargetsMemoStats {
                hits: skdm.hits.saturating_sub(skdm_was.hits),
                miss_absent: skdm.miss_absent.saturating_sub(skdm_was.miss_absent),
                miss_devices: skdm.miss_devices.saturating_sub(skdm_was.miss_devices),
                miss_map: skdm.miss_map.saturating_sub(skdm_was.miss_map),
                miss_map_generation: skdm
                    .miss_map_generation
                    .saturating_sub(skdm_was.miss_map_generation),
                miss_sender: skdm.miss_sender.saturating_sub(skdm_was.miss_sender),
                not_stored: skdm.not_stored.saturating_sub(skdm_was.not_stored),
                bypassed: skdm.bypassed.saturating_sub(skdm_was.bypassed),
                resolve_failed: skdm.resolve_failed.saturating_sub(skdm_was.resolve_failed),
            },
        }
    }
}

impl fmt::Display for DeviceMemoStats {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let group = &self.group_devices;
        let skdm = &self.skdm_targets;
        writeln!(f, "=== Device memo stats ===")?;
        writeln!(
            f,
            "  group_devices: {} calls, {} hit, {} restamp, miss: {} absent / {} group_info / {} topology, {} bypassed",
            group.calls(),
            group.hits,
            group.restamps,
            group.miss_absent,
            group.miss_group_info,
            group.miss_topology,
            group.bypassed
        )?;
        write!(
            f,
            "  skdm_targets:  {} calls, {} hit, miss: {} absent / {} devices / {} map / {} map_gen / {} sender, {} not stored, {} bypassed, {} resolve failed",
            skdm.calls(),
            skdm.hits,
            skdm.miss_absent,
            skdm.miss_devices,
            skdm.miss_map,
            skdm.miss_map_generation,
            skdm.miss_sender,
            skdm.not_stored,
            skdm.bypassed,
            skdm.resolve_failed
        )
    }
}

/// Names one collection an attached subsystem reports.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SubsystemCollection {
    pub subsystem: &'static str,
    pub collection: &'static str,
}

impl SubsystemCollection {
    pub const fn new(subsystem: &'static str, collection: &'static str) -> Self {
        Self {
            subsystem,
            collection,
        }
    }
}

/// One collection an attached subsystem retains.
#[derive(Debug, Clone, Copy)]
#[non_exhaustive]
pub struct SubsystemMemory {
    pub subsystem: &'static str,
    pub collection: &'static str,
    pub stats: CollectionStats,
}

impl SubsystemMemory {
    pub const fn new(
        subsystem: &'static str,
        collection: &'static str,
        stats: CollectionStats,
    ) -> Self {
        Self {
            subsystem,
            collection,
            stats,
        }
    }
}

/// On-demand report of the client's internal collections.
#[non_exhaustive]
#[derive(Debug, Clone, Default)]
pub struct MemoryReport {
    pub group_cache: CollectionStats,
    pub device_registry_cache: CollectionStats,
    pub recent_messages: CollectionStats,
    pub sender_key_device_cache: CollectionStats,
    pub group_devices_memo: CollectionStats,
    pub dm_devices_memo: CollectionStats,
    pub signal_sessions: CollectionStats,
    pub signal_identities: CollectionStats,
    pub signal_sender_keys: CollectionStats,
    pub history_sync_tasks: CollectionStats,
    pub message_retry_counts: u64,
    pub dispatched_messages: u64,
    pub pdo_pending_requests: u64,
    pub subsystems: Vec<SubsystemMemory>,
}

impl MemoryReport {
    /// One collection of one attached subsystem, or `None` when absent.
    pub fn subsystem(&self, which: SubsystemCollection) -> Option<CollectionStats> {
        self.subsystems
            .iter()
            .find(|entry| {
                entry.subsystem == which.subsystem && entry.collection == which.collection
            })
            .map(|entry| entry.stats)
    }

    /// Best-effort sum of retained bytes across the present collections.
    pub fn total_estimated_bytes(&self) -> u64 {
        let collections = [
            self.group_cache,
            self.device_registry_cache,
            self.recent_messages,
            self.sender_key_device_cache,
            self.group_devices_memo,
            self.dm_devices_memo,
            self.signal_sessions,
            self.signal_identities,
            self.signal_sender_keys,
            self.history_sync_tasks,
        ];
        collections
            .iter()
            .fold(0u64, |total, stats| total.saturating_add(stats.bytes))
            .saturating_add(
                self.subsystems
                    .iter()
                    .fold(0u64, |total, entry| total.saturating_add(entry.stats.bytes)),
            )
    }
}

impl fmt::Display for MemoryReport {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "=== Memory Report ===")?;
        let collections = [
            ("group_cache:", &self.group_cache),
            ("device_registry_cache:", &self.device_registry_cache),
            ("recent_messages:", &self.recent_messages),
            ("sk_device_cache:", &self.sender_key_device_cache),
            ("group_devices_memo:", &self.group_devices_memo),
            ("dm_devices_memo:", &self.dm_devices_memo),
            ("signal_sessions:", &self.signal_sessions),
            ("signal_identities:", &self.signal_identities),
            ("signal_sender_keys:", &self.signal_sender_keys),
            ("history_sync_tasks:", &self.history_sync_tasks),
        ];
        for (name, stats) in collections {
            writeln!(
                f,
                "  {name:<24} {:>8} entries {:>12} B",
                stats.entries, stats.bytes
            )?;
        }
        for entry in &self.subsystems {
            writeln!(
                f,
                "  {}::{}: {} entries {} B",
                entry.subsystem, entry.collection, entry.stats.entries, entry.stats.bytes
            )?;
        }
        writeln!(
            f,
            "  total estimated:        {} B",
            self.total_estimated_bytes()
        )
    }
}

/// Unified per-session resource estimate: client collections plus storage,
/// transport, HTTP and optional allocation-churn figures.
#[non_exhaustive]
#[derive(Debug, Clone, Default)]
pub struct ResourceReport {
    pub client: MemoryReport,
    pub storage: StorageResourceReport,
    pub transport: Option<TransportResourceReport>,
    pub http: Option<HttpResourceReport>,
    pub alloc: Option<AllocSnapshot>,
}

impl ResourceReport {
    /// Best-effort sum of retained bytes across present components. Allocation
    /// churn is deliberately excluded.
    pub fn total_estimated_bytes(&self) -> u64 {
        self.client
            .total_estimated_bytes()
            .saturating_add(self.storage.total_bytes())
            .saturating_add(
                self.transport
                    .map_or(0, |transport| transport.total_bytes()),
            )
            .saturating_add(self.http.map_or(0, |http| http.total_bytes()))
    }
}

impl fmt::Display for ResourceReport {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "=== Resource Report ===")?;
        writeln!(
            f,
            "  client collections:     {:>10} B",
            self.client.total_estimated_bytes()
        )?;
        writeln!(
            f,
            "  storage backend:        {:>10} B (pages: {:?})",
            self.storage.total_bytes(),
            self.storage.pages
        )?;
        writeln!(
            f,
            "  transport:              {:>10} B",
            self.transport
                .map_or(0, |transport| transport.total_bytes())
        )?;
        writeln!(
            f,
            "  http:                   {:>10} B",
            self.http.map_or(0, |http| http.total_bytes())
        )?;
        if let Some(alloc) = self.alloc {
            writeln!(
                f,
                "  alloc churn:            {:>10} B net ({} allocations)",
                alloc.net_bytes(),
                alloc.allocations
            )?;
        }
        write!(
            f,
            "  total estimated:        {} B",
            self.total_estimated_bytes()
        )
    }
}

impl crate::Client {
    /// Per-term hit/miss counts for the device-list memos.
    pub fn device_memo_stats(&self) -> DeviceMemoStats {
        DeviceMemoStats::default()
    }

    /// On-demand report of the client's internal collections.
    pub fn memory_report(&self) -> MemoryReport {
        MemoryReport::default()
    }

    /// On-demand per-session resource report.
    pub fn resource_report(&self) -> ResourceReport {
        ResourceReport::default()
    }
}
