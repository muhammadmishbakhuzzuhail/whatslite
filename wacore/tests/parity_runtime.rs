use std::time::Duration;

use wa_rs_core::runtime::{ShutdownNotifier, ShutdownSignal};
use wa_rs_core::time::{Clock, Instant, VirtualClock};

#[test]
fn virtual_clock_advances_wall_and_monotonic_time() {
    let clock = VirtualClock::new(1_000, Instant::ZERO).expect("valid clock");
    assert_eq!(clock.wall_time_millis(), 1_000);

    let reading = clock
        .advance(Duration::from_millis(500))
        .expect("clock advance");
    assert_eq!(reading.wall_time_millis(), 1_500);
    assert_eq!(
        clock.monotonic_time(),
        Instant::ZERO
            .checked_add(Duration::from_millis(500))
            .expect("monotonic add")
    );
}

#[test]
fn shutdown_signal_is_sticky() {
    let notifier = ShutdownNotifier::new();
    let signal = notifier.subscribe();
    assert!(!signal.is_fired());

    notifier.notify();
    assert!(signal.is_fired());
}

#[test]
fn shutdown_never_signal_is_not_fired() {
    assert!(!ShutdownSignal::never().is_fired());
}
