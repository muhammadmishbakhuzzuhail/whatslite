//! Phase 1 session-engine contract: exchange phases, lifecycle transitions,
//! reconnect policy and backoff sampling.

use std::time::Duration;

use wa_rs_core::session_engine::exchange::RegisteredExchange;
use wa_rs_core::session_engine::lifecycle::{
    AuthenticationPhase, DesiredConnection, DriveKind, DriveState, Lifecycle, LifecycleError,
    NextConnection, ReconnectPolicy, StopReason, TransportPhase,
};
use wa_rs_core::time::Instant;

#[test]
fn exchange_phases_release_registration_before_response() {
    let reply = RegisteredExchange::new(7u8)
        .begin_send()
        .sent()
        .response(42u32)
        .into_response();
    assert_eq!(reply, 42);
}

#[test]
fn lifecycle_walks_drive_transport_and_authentication() {
    let mut lifecycle = Lifecycle::default();
    assert_eq!(lifecycle.snapshot().transport(), TransportPhase::Absent);
    assert_eq!(lifecycle.snapshot().desired(), DesiredConnection::Active);
    assert!(!lifecycle.snapshot().driving());

    let lease = lifecycle.start_drive(DriveKind::Supervisor).unwrap();
    assert!(matches!(
        lifecycle.snapshot().drive(),
        DriveState::Supervisor { .. }
    ));

    let attempt = lifecycle.begin_attempt().unwrap();
    lifecycle.transport_opened(attempt).unwrap();
    let generation = lifecycle.handshake_ready(attempt).unwrap();
    assert!(lifecycle.finish_attempt(attempt));
    assert_eq!(
        lifecycle.snapshot().transport(),
        TransportPhase::Established
    );
    assert_eq!(lifecycle.snapshot().generation(), generation);

    let auth = lifecycle.claim_authentication().unwrap();
    assert!(matches!(
        lifecycle.snapshot().authentication(),
        AuthenticationPhase::Claimed(_)
    ));
    assert!(lifecycle.admit_authentication(auth, Instant::ZERO));
    assert!(lifecycle.ready(auth));
    assert!(lifecycle.snapshot().ready());
    assert!(lifecycle.snapshot().can_reach_server());

    lifecycle.stop_driving();
    assert!(matches!(lifecycle.snapshot().drive(), DriveState::Idle));
    let _ = lease;
}

#[test]
fn lifecycle_refuses_second_drive_and_second_attempt() {
    let mut lifecycle = Lifecycle::default();
    lifecycle.start_drive(DriveKind::Supervisor).unwrap();
    assert_eq!(
        lifecycle.start_drive(DriveKind::Supervisor).unwrap_err(),
        LifecycleError::AlreadyDriving
    );

    let _attempt = lifecycle.begin_attempt().unwrap();
    assert_eq!(
        lifecycle.begin_attempt().unwrap_err(),
        LifecycleError::AlreadyConnecting
    );
}

#[test]
fn reconnect_policy_and_shutdown_control_next_connection() {
    let mut lifecycle = Lifecycle::new(ReconnectPolicy::Disabled);
    lifecycle.start_drive(DriveKind::Supervisor).unwrap();
    assert_eq!(
        lifecycle.next_connection(Instant::ZERO).unwrap(),
        NextConnection::Stop(StopReason::ReconnectDisabled)
    );

    lifecycle.set_reconnect(true);
    lifecycle.shutdown();
    assert_eq!(
        lifecycle.next_connection(Instant::ZERO).unwrap(),
        NextConnection::Stop(StopReason::Shutdown)
    );
}

#[test]
fn backoff_sampling_is_bounded_by_fibonacci_base() {
    let mut lifecycle = Lifecycle::default();
    lifecycle.start_drive(DriveKind::Supervisor).unwrap();

    let NextConnection::SampleBackoff(request) = lifecycle.next_connection(Instant::ZERO).unwrap()
    else {
        panic!("expected backoff sample");
    };
    assert_eq!(request.attempt(), 0);
    let delay = lifecycle.sample_backoff(request, 0).unwrap();
    assert!(delay >= Duration::from_millis(900));
    assert!(delay <= Duration::from_millis(1200));
}
