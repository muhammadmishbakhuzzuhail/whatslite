//! Shared creation result types for polls and events.

macro_rules! creation_types {
    ($($name:ident),* $(,)?) => {
        $(
            #[derive(Debug, Clone, PartialEq, Eq)]
            pub struct $name {
                pub id: String,
            }
        )*
    };
}

creation_types!(CreatedPoll, CreatedEvent, PollRef, EventRef);
