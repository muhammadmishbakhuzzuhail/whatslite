use wa_rs::features::app_state_resync::{AppStateResyncMode, AppStateResyncReport};
use wa_rs::features::app_state_settings::AppStateSettings;
#[test]
fn app_state_types_are_stable() {
    assert_eq!(AppStateResyncMode::Full, AppStateResyncMode::Full);
    assert_eq!(AppStateResyncReport::default().collections, 0);
    assert!(AppStateSettings::default().settings.is_empty());
}
