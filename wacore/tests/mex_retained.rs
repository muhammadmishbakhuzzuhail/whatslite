use wa_rs_core::iq::mex::{MexDoc, MexQuerySpec};
use wa_rs_core::iq::spec::IqSpec;
use wa_rs_core::request::InfoQueryType;
use wacore_binary::NodeContent;

#[test]
fn mex_query_spec_keeps_doc_and_variables() {
    let doc = MexDoc {
        name: "WAWebMexTestQuery",
        id: "doc",
    };
    let spec = MexQuerySpec::new(doc, &serde_json::json!({"a": 1})).expect("serialize variables");
    assert_eq!(spec.doc.name, "WAWebMexTestQuery");
    assert_eq!(spec.doc.id, "doc");

    let iq = spec.build_iq();
    assert_eq!(iq.namespace, "w:mex");
    assert_eq!(iq.query_type, InfoQueryType::Get);
    let Some(NodeContent::Nodes(nodes)) = &iq.content else {
        panic!("expected query child");
    };
    assert_eq!(nodes[0].attrs.get("query_id").unwrap(), "doc");
    let Some(NodeContent::Bytes(bytes)) = &nodes[0].content else {
        panic!("expected wire payload");
    };
    assert_eq!(bytes.as_slice(), br#"{"variables":{"a":1}}"#);
}
