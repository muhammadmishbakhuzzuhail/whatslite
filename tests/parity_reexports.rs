//! Phase 0 architecture contract: upstream-compatible crate aliases and
//! public re-exports stay reachable from the root crate.

#[test]
fn crate_aliases_are_reachable() {
    let _: fn() = || {
        let _ = std::any::type_name::<wa_rs::wacore::types::events::Event>();
        let _ = std::any::type_name::<wa_rs::wacore_binary::jid::Jid>();
        let _ = std::any::type_name::<wa_rs::waproto::whatsapp::Message>();
    };
}

#[test]
fn client_and_feature_surface_are_reachable() {
    let _: fn() = || {
        let _ = std::any::type_name::<wa_rs::Client>();
        let _ = std::any::type_name::<wa_rs::bot::Bot>();
        let _ = std::any::type_name::<wa_rs::store::SqliteStore>();
        let _ = std::any::type_name::<wa_rs::pair_code::PairCodeOptions>();
        let _ = std::any::type_name::<wa_rs::features::Blocking>();
    };
}

#[test]
fn event_delivery_is_exported_from_bot() {
    let _: fn() = || {
        let _ = std::any::type_name::<wa_rs::bot::EventDelivery>();
        let _ = std::any::type_name::<wa_rs::bot::CallbackEventHandler>();
        let _ = std::any::type_name::<wa_rs::types::events::EventKind>();
    };
}
