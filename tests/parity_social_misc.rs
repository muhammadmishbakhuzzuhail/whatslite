use wa_rs::features::business::{Business, Catalog, Collection, Order};
use wa_rs::features::pictures::{Picture, Pictures};
use wa_rs::features::profile::Profile;
use wa_rs::features::stickers::Stickers;
#[test]
fn social_misc_types_are_reachable() {
    let _ = (Business, Stickers, Pictures);
    assert_eq!(Catalog { id: "c".into() }.id, "c");
    assert_eq!(Collection { id: "c".into() }.id, "c");
    assert_eq!(Order { id: "o".into() }.id, "o");
    assert_eq!(
        Picture {
            id: "p".into(),
            url: "u".into()
        }
        .id,
        "p"
    );
    assert_eq!(Profile::default().status, None);
}
