use wa_rs::features::{
    BotDefault, BotList, BotListEntry, BotListSection, BotListVersion, BotSectionDisplayType,
    BotSectionType, BotTheme, BotThemeMode, Bots,
};
use wa_rs::{Client, features};

fn bots<'a>(client: &'a Client) -> Bots<'a> {
    client.bots()
}

#[test]
fn bot_directory_types_and_accessor_are_public() {
    let _ = bots;
    let _: Option<features::Bots<'_>> = None;
    let _ = std::any::TypeId::of::<BotList>();
    let _ = std::any::TypeId::of::<BotListEntry>();
    let _ = std::any::TypeId::of::<BotListSection>();
    let _ = std::any::TypeId::of::<BotListVersion>();
    let _ = std::any::TypeId::of::<BotDefault>();
    let _ = std::any::TypeId::of::<BotSectionDisplayType>();
    let _ = std::any::TypeId::of::<BotSectionType>();
    let _ = std::any::TypeId::of::<BotTheme>();
    let _ = std::any::TypeId::of::<BotThemeMode>();
}
