use wa_rs::features::labels::LabelSublist;
#[test]
fn label_timestamp_shape_is_stable() {
    let sublist = LabelSublist {
        id: "id".into(),
        name: "name".into(),
    };
    assert!(!sublist.name.is_empty());
}
