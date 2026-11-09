#![cfg(all(feature = "tracing", feature = "metrics"))]
//! Phase 5 metrics, tracing and diagnostics contract.

use wa_rs::telemetry::{self, IQ_DURATION};
use wa_rs::{
    AllocSnapshot, CollectionStats, DeviceMemoStats, MemoryReport, ResourceReport, SessionStats,
    StatsSnapshot, StorageResourceReport, SubsystemCollection, SubsystemMemory, Timer,
    TransportResourceReport,
};

#[test]
fn session_stats_snapshot_counts_frames_and_messages() {
    let stats = SessionStats::new();
    stats.record_frame_sent(10);
    stats.record_recv_batch(20, 2);
    stats.record_message_sent();
    stats.record_message_received();

    let snapshot: StatsSnapshot = stats.snapshot();
    assert_eq!(snapshot.bytes_sent, 10);
    assert_eq!(snapshot.bytes_received, 20);
    assert_eq!(snapshot.frames_sent, 1);
    assert_eq!(snapshot.frames_received, 2);
    assert_eq!(snapshot.messages_sent, 1);
    assert_eq!(snapshot.messages_received, 1);
}

#[test]
fn metric_emission_and_timer_do_not_require_a_recorder() {
    telemetry::recv("decrypted");
    telemetry::send("text");
    telemetry::iq("result");
    telemetry::describe();
    {
        let _timer: Timer = telemetry::timer(IQ_DURATION);
    }
}

struct AlwaysSubscriber;

impl tracing::Subscriber for AlwaysSubscriber {
    fn enabled(&self, _metadata: &tracing::Metadata<'_>) -> bool {
        true
    }

    fn new_span(&self, _span: &tracing::span::Attributes<'_>) -> tracing::span::Id {
        static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);
        tracing::span::Id::from_u64(NEXT.fetch_add(1, std::sync::atomic::Ordering::Relaxed))
    }

    fn record(&self, _span: &tracing::span::Id, _values: &tracing::span::Record<'_>) {}

    fn record_follows_from(&self, _span: &tracing::span::Id, _follows: &tracing::span::Id) {}

    fn event(&self, _event: &tracing::Event<'_>) {}

    fn enter(&self, _span: &tracing::span::Id) {}

    fn exit(&self, _span: &tracing::span::Id) {}
}

#[test]
fn tracing_span_is_created() {
    let dispatch = tracing::Dispatch::new(AlwaysSubscriber);
    tracing::dispatcher::with_default(&dispatch, || {
        let span = wa_rs::span("connect");
        assert!(!span.is_disabled());
        assert_eq!(
            span.metadata().map(|metadata| metadata.name()),
            Some("wa_operation")
        );
        let _guard = span.enter();
    });
}

#[test]
fn memory_and_resource_reports_compose() {
    let mut report = MemoryReport::default();
    report.group_cache = CollectionStats::new(2, 128);
    report.signal_sessions = CollectionStats::new(1, 64);
    report.subsystems.push(SubsystemMemory::new(
        "voip",
        "active_calls",
        CollectionStats::new(1, 32),
    ));

    assert_eq!(report.total_estimated_bytes(), 224);
    let subsystem = report
        .subsystem(SubsystemCollection::new("voip", "active_calls"))
        .expect("subsystem collection");
    assert_eq!((subsystem.entries, subsystem.bytes), (1, 32));
    assert!(
        report
            .subsystem(SubsystemCollection::new("voip", "missing"))
            .is_none()
    );
    assert!(report.to_string().contains("Memory Report"));

    let mut alloc = AllocSnapshot::default();
    alloc.allocated_bytes = 50;
    alloc.freed_bytes = 20;
    alloc.allocations = 1;

    let mut resource = ResourceReport::default();
    resource.client = report;
    resource.storage = StorageResourceReport {
        memory_bytes: Some(100),
        ..Default::default()
    };
    resource.transport = Some(TransportResourceReport {
        read_buffer_bytes: Some(10),
        ..Default::default()
    });
    resource.alloc = Some(alloc);

    assert_eq!(resource.total_estimated_bytes(), 334);
    assert!(resource.to_string().contains("Resource Report"));
}

#[test]
fn device_memo_stats_since_and_rates() {
    let earlier = DeviceMemoStats::default();
    let mut later = DeviceMemoStats::default();
    later.group_devices.hits = 3;
    later.group_devices.miss_absent = 1;
    later.skdm_targets.hits = 1;
    later.skdm_targets.resolve_failed = 1;

    let delta = later.since(&earlier);
    assert_eq!(delta.group_devices.calls(), 4);
    assert_eq!(delta.group_devices.served_rate(), Some(0.75));
    assert_eq!(delta.skdm_targets.calls(), 2);
    assert_eq!(delta.skdm_targets.hit_rate(), Some(0.5));
    assert!(later.to_string().contains("Device memo stats"));

    let reversed = earlier.since(&later);
    assert_eq!(reversed.group_devices.calls(), 0);
    assert_eq!(reversed.group_devices.served_rate(), None);
}
