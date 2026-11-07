//! Framework-agnostic semantic theme constants for a GPUI Kit adapter.
//! Values are proposed reconstruction tokens, not official WhatsApp tokens.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ThemeMode { Light, Dark }
#[derive(Clone, Copy, Debug)]
pub struct Colors {
    pub navigation: &'static str,
    pub list: &'static str,
    pub conversation: &'static str,
    pub header: &'static str,
    pub text_primary: &'static str,
    pub text_secondary: &'static str,
    pub accent: &'static str,
    pub divider: &'static str,
    pub bubble_in: &'static str,
    pub bubble_out: &'static str,
    pub input: &'static str,
}
pub const LIGHT: Colors = Colors {
    navigation: "#EFF2F5", list: "#FFFFFF", conversation: "#F7F2EA", header: "#FFFFFF",
    text_primary: "#182229", text_secondary: "#65727B", accent: "#17A765", divider: "#DDE2E6",
    bubble_in: "#FFFFFF", bubble_out: "#D8FBCB", input: "#F6F7F8",
};
pub const DARK: Colors = Colors {
    navigation: "#172126", list: "#172126", conversation: "#101A1E", header: "#1D282D",
    text_primary: "#E9F0F2", text_secondary: "#A7B6BC", accent: "#1DBB73", divider: "#344247",
    bubble_in: "#26363C", bubble_out: "#075E48", input: "#26343A",
};
pub fn colors(mode: ThemeMode) -> &'static Colors { match mode { ThemeMode::Light => &LIGHT, ThemeMode::Dark => &DARK } }

pub const REFERENCE_WIDTH: f32 = 1512.0;
pub const REFERENCE_HEIGHT: f32 = 982.0;
pub const REFERENCE_MENU_BAR_HEIGHT: f32 = 25.0;
pub const NAV_DEFAULT: f32 = 208.0;
pub const NAV_COMPACT: f32 = 72.0;
pub const CHAT_LIST_DEFAULT: f32 = 360.0;
pub const CHAT_LIST_MIN: f32 = 300.0;
pub const CONVERSATION_MIN: f32 = 440.0;
pub const CHAT_ROW_HEIGHT: f32 = 76.0;
pub const HEADER_HEIGHT: f32 = 72.0;
pub const COMPOSER_MIN_HEIGHT: f32 = 72.0;

#[derive(Clone, Copy, Debug)]
pub struct PanelWidths { pub navigation: f32, pub chat_list: f32, pub conversation: f32 }
pub fn panel_widths(available_width: f32) -> PanelWidths {
    let nav = if available_width < 1170.0 { NAV_COMPACT } else { NAV_DEFAULT };
    let list = if available_width < 1170.0 { 312.0 } else { CHAT_LIST_DEFAULT };
    PanelWidths { navigation: nav, chat_list: list, conversation: (available_width - nav - list).max(0.0) }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test] fn baseline_shell_width() {
        let p = panel_widths(1512.0);
        assert_eq!(p.navigation, 208.0);
        assert_eq!(p.chat_list, 360.0);
        assert_eq!(p.conversation, 944.0);
    }
    #[test] fn compact_meets_conversation_minimum() {
        let p = panel_widths(1024.0);
        assert!(p.conversation >= CONVERSATION_MIN);
    }
}
