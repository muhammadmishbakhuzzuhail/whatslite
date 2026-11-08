use wa_rs_core::stats::{AllocMeter, TaskInstrument};

#[test]
fn alloc_meter_records_scoped_allocations() {
    let meter = AllocMeter::new();
    meter.on_poll_start();
    AllocMeter::on_alloc(128);
    AllocMeter::on_dealloc(64);
    let snapshot = meter.snapshot();
    meter.on_poll_end();

    assert_eq!(snapshot.allocated_bytes, 128);
    assert_eq!(snapshot.freed_bytes, 64);
    assert_eq!(snapshot.allocations, 1);
    assert_eq!(snapshot.net_bytes(), 64);
}

#[test]
fn error_summary_formats_io_errors() {
    use wa_rs_core::diagnostics::ErrorSummary;

    let error = std::io::Error::other("boom");
    let text = format!("{}", ErrorSummary(&error));
    assert!(text.contains("I/O"), "{text}");
}
