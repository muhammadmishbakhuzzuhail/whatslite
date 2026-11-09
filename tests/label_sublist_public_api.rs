use wa_rs::features::labels::{LabelSublist, Labels};
#[test]
fn label_sublist_is_public() {
    let sublist = LabelSublist {
        id: "id".into(),
        name: "name".into(),
    };
    assert_eq!(sublist.id, "id");
    let _ = Labels;
}
