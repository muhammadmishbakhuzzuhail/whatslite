#![cfg(feature = "voip")]
//! Phase 4 sans-IO engine contract.
use wa_rs_core::voip::engine::{
    CallEngine, Driver, EngineCommand, EngineEvent, EngineState, Session,
};
#[test]
fn engine_walks_offer_accept_hangup() {
    let mut engine = CallEngine::new();
    assert_eq!(engine.state(), EngineState::Idle);
    assert_eq!(
        engine.handle(EngineCommand::Offer {
            call_id: "c1".into()
        }),
        &[EngineEvent::Incoming {
            call_id: "c1".into()
        }]
    );
    assert_eq!(engine.state(), EngineState::Ringing);
    assert_eq!(
        engine.handle(EngineCommand::Accept),
        &[EngineEvent::Connected]
    );
    assert_eq!(engine.state(), EngineState::Connected);
    assert_eq!(engine.handle(EngineCommand::Hangup), &[EngineEvent::Ended]);
    assert_eq!(engine.state(), EngineState::Closed);
}
#[test]
fn driver_and_session_are_reachable() {
    let mut driver = Driver::new();
    driver.engine_mut().handle(EngineCommand::Offer {
        call_id: "c2".into(),
    });
    let session = Session {
        call_id: "c2".into(),
        state: driver.engine().state(),
    };
    assert_eq!(session.state, EngineState::Ringing);
}
