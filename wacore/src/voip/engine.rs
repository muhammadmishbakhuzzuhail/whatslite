//! Sans-IO call engine state machine.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EngineState {
    Idle,
    Offering,
    Ringing,
    Connected,
    Closing,
    Closed,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum EngineCommand {
    Offer { call_id: String },
    Accept,
    Reject,
    Hangup,
    Tick,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum EngineEvent {
    Incoming { call_id: String },
    Connected,
    Ended,
}

#[derive(Debug)]
pub struct CallEngine {
    state: EngineState,
    call_id: Option<String>,
    events: Vec<EngineEvent>,
}

impl Default for CallEngine {
    fn default() -> Self {
        Self::new()
    }
}

impl CallEngine {
    pub fn new() -> Self {
        Self {
            state: EngineState::Idle,
            call_id: None,
            events: Vec::new(),
        }
    }

    pub fn state(&self) -> EngineState {
        self.state
    }

    pub fn call_id(&self) -> Option<&str> {
        self.call_id.as_deref()
    }

    pub fn handle(&mut self, command: EngineCommand) -> &[EngineEvent] {
        self.events.clear();
        match (self.state, command) {
            (EngineState::Idle, EngineCommand::Offer { call_id }) => {
                self.call_id = Some(call_id.clone());
                self.state = EngineState::Ringing;
                self.events.push(EngineEvent::Incoming { call_id });
            }
            (EngineState::Ringing, EngineCommand::Accept) => {
                self.state = EngineState::Connected;
                self.events.push(EngineEvent::Connected);
            }
            (EngineState::Ringing, EngineCommand::Reject)
            | (EngineState::Connected, EngineCommand::Hangup) => {
                self.state = EngineState::Closed;
                self.events.push(EngineEvent::Ended);
            }
            (EngineState::Closing, EngineCommand::Tick) => {
                self.state = EngineState::Closed;
                self.events.push(EngineEvent::Ended);
            }
            _ => {}
        }
        &self.events
    }
}

/// Driver owns one engine and drains its events.
#[derive(Debug, Default)]
pub struct Driver {
    engine: CallEngine,
}

impl Driver {
    pub fn new() -> Self {
        Self::default()
    }
    pub fn engine(&self) -> &CallEngine {
        &self.engine
    }
    pub fn engine_mut(&mut self) -> &mut CallEngine {
        &mut self.engine
    }
}

/// Session metadata over an engine.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Session {
    pub call_id: String,
    pub state: EngineState,
}

#[derive(Debug, Default)]
pub struct GroupAudio;
#[derive(Debug, Default)]
pub struct GroupMedia;
