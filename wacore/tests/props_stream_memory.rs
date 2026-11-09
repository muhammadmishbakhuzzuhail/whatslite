use wa_rs_core::iq::props::AbProp;
#[test]
fn prop_clone_is_independent() {
    let prop = AbProp {
        config_code: 2,
        config_value: "v".into(),
        config_expo_key: Some(3),
    };
    let cloned = prop.clone();
    assert_eq!(cloned.config_code, 2);
    assert_eq!(cloned.config_expo_key, Some(3));
}
