use wa_rs_core::iq::mex::MexQuerySpec;
#[test]
fn mex_query_spec_keeps_doc_and_variables() {
    let spec = MexQuerySpec::new("doc", serde_json::json!({"a": 1}));
    assert_eq!(spec.doc_id, "doc");
    assert_eq!(spec.variables["a"], 1);
}
