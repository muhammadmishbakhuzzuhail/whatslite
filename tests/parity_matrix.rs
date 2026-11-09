//! Phase 5 feature matrix: parity aliases must imply their canonical features.

#[test]
fn parity_core_alias_is_wired() {
    #[cfg(feature = "parity-core")]
    {
        assert!(cfg!(feature = "instance-clock"));
        assert!(cfg!(feature = "legacy-session-interop"));
    }
}

#[test]
fn parity_messaging_alias_is_wired() {
    #[cfg(feature = "parity-messaging")]
    {
        assert!(cfg!(feature = "parity-core"));
        assert!(cfg!(feature = "instance-clock"));
    }
}

#[test]
fn parity_social_alias_is_wired() {
    #[cfg(feature = "parity-social")]
    {
        assert!(cfg!(feature = "parity-messaging"));
        assert!(cfg!(feature = "parity-core"));
    }
}

#[test]
fn parity_voip_alias_is_wired() {
    #[cfg(feature = "parity-voip")]
    {
        assert!(cfg!(feature = "voip"));
    }
}

#[test]
fn parity_mobile_alias_is_wired() {
    #[cfg(feature = "parity-mobile")]
    {
        assert!(cfg!(feature = "mobile-core"));
    }
}

#[test]
fn parity_plugins_alias_is_wired() {
    #[cfg(feature = "parity-plugins")]
    {
        assert!(cfg!(feature = "plugins"));
        assert!(cfg!(feature = "client-lifecycle"));
    }
}

#[test]
fn parity_observability_alias_is_wired() {
    #[cfg(feature = "parity-observability")]
    {
        assert!(cfg!(feature = "tracing"));
        assert!(cfg!(feature = "metrics"));
    }
}

#[test]
fn parity_wasm_alias_is_wired() {
    #[cfg(feature = "parity-wasm")]
    {
        assert!(cfg!(feature = "js"));
    }
}

#[test]
fn canonical_edges_are_wired() {
    #[cfg(feature = "plugins")]
    {
        assert!(cfg!(feature = "client-lifecycle"));
    }
    #[cfg(feature = "mobile-transport-native")]
    {
        assert!(cfg!(feature = "mobile-core"));
        assert!(cfg!(feature = "tokio-transport"));
    }
    #[cfg(feature = "signal")]
    {
        assert!(cfg!(feature = "tokio-runtime"));
    }
    #[cfg(feature = "sqlite-storage-bundled")]
    {
        assert!(cfg!(feature = "sqlite-storage"));
    }
}
