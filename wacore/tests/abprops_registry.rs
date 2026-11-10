use wa_rs_core::iq::abprops::{self, AbDefault, AbPropType};

#[test]
fn registry_slices_match_catalog_sizes() {
    assert_eq!(abprops::web::ALL.len(), 2886);
    assert_eq!(abprops::group::ALL.len(), 19);
    assert_eq!(abprops::hybrid::ALL.len(), 438);
    assert_eq!(abprops::ALL.len(), 3);
}

#[test]
fn flag_constants_keep_their_codes_and_defaults() {
    let tctoken_duration = abprops::web::TCTOKEN_DURATION;
    assert_eq!(tctoken_duration.name, "tctoken_duration");
    assert_eq!(tctoken_duration.code, 865);
    assert_eq!(tctoken_duration.value_type, AbPropType::Int);
    assert_eq!(tctoken_duration.default, AbDefault::Int(604800));

    let receipt_mode = abprops::web::RECEIPT_MODE_BITMASK_ENABLED;
    assert_eq!(receipt_mode.name, "receipt_mode_bitmask_enabled");
    assert_eq!(receipt_mode.code, 30084);
    assert_eq!(receipt_mode.value_type, AbPropType::Bool);
    assert_eq!(receipt_mode.default, AbDefault::Bool(false));
}

#[test]
fn stale_constants_stay_reachable_through_the_web_module() {
    let retained = abprops::web::WEB_TEST_ABPROP_DELETE_ME;
    assert_eq!(retained.name, "web_test_abprop_delete_me");
    assert_eq!(retained.code, 27274);
    assert_eq!(
        wa_rs_core::iq::props::stale::WEB_TEST_ABPROP_DELETE_ME.code,
        retained.code
    );
}
