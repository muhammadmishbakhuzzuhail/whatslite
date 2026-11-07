use std::sync::OnceLock;

use gpui_kit::component::theme::{Theme, ThemeMode as GpuiThemeMode};
use gpui_kit::{px, App, Rgba, rgb};
use serde_json::Value;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ThemeMode {
    Light,
    Dark,
}

impl ThemeMode {
    pub const fn name(self) -> &'static str {
        match self {
            Self::Light => "light",
            Self::Dark => "dark",
        }
    }

}

#[derive(Clone, Copy, Debug)]
pub struct Palette {
    pub bg_sidebar: Rgba,
    pub bg_list: Rgba,
    pub bg_conversation: Rgba,
    pub bg_header: Rgba,
    pub bg_search: Rgba,
    pub bg_selected: Rgba,
    pub bg_row_selected: Rgba,
    pub bg_composer: Rgba,
    pub bg_notice_encrypt: Rgba,
    pub bg_notice_system: Rgba,
    pub bubble_in: Rgba,
    pub bubble_out: Rgba,
    pub text_primary: Rgba,
    pub text_secondary: Rgba,
    pub text_inverse: Rgba,
    pub accent: Rgba,
    pub accent_soft: Rgba,
    pub unread_text: Rgba,
    pub tick_read: Rgba,
    pub border: Rgba,
    pub divider: Rgba,
    pub attach_blue: Rgba,
    pub attach_orange: Rgba,
    pub attach_red: Rgba,
    pub attach_green: Rgba,
    pub traffic_red: Rgba,
    pub traffic_yellow: Rgba,
    pub traffic_green: Rgba,
}

impl Palette {
    pub fn danger(self) -> Rgba {
        self.attach_red
    }

    pub fn hover(self) -> Rgba {
        self.bg_row_selected
    }

}

fn tokens() -> &'static Value {
    static TOKENS: OnceLock<Value> = OnceLock::new();
    TOKENS.get_or_init(|| {
        serde_json::from_str(include_str!("../../design/machine/theme.json"))
            .expect("design/machine/theme.json must be valid JSON")
    })
}

fn color(mode: ThemeMode, name: &str) -> Rgba {
    let value = tokens()["theme"]["colors"][name][mode.name()]
        .as_str()
        .unwrap_or_else(|| panic!("missing color token: {name}"));
    let hex = value
        .trim_start_matches('#')
        .get(..6)
        .unwrap_or_else(|| panic!("color token {name} must be a six digit RGB value"));
    let value = u32::from_str_radix(hex, 16)
        .unwrap_or_else(|_| panic!("color token {name} must be hexadecimal"));
    rgb(value)
}

pub fn palette(mode: ThemeMode) -> Palette {
    Palette {
        bg_sidebar: color(mode, "bg_sidebar"),
        bg_list: color(mode, "bg_list"),
        bg_conversation: color(mode, "bg_conversation"),
        bg_header: color(mode, "bg_header"),
        bg_search: color(mode, "bg_search"),
        bg_selected: color(mode, "bg_selected"),
        bg_row_selected: color(mode, "bg_row_selected"),
        bg_composer: color(mode, "bg_composer"),
        bg_notice_encrypt: color(mode, "bg_notice_encrypt"),
        bg_notice_system: color(mode, "bg_notice_system"),
        bubble_in: color(mode, "bubble_in"),
        bubble_out: color(mode, "bubble_out"),
        text_primary: color(mode, "text_primary"),
        text_secondary: color(mode, "text_secondary"),
        text_inverse: color(mode, "text_inverse"),
        accent: color(mode, "accent"),
        accent_soft: color(mode, "accent_soft"),
        unread_text: color(mode, "unread_text"),
        tick_read: color(mode, "tick_read"),
        border: color(mode, "border"),
        divider: color(mode, "divider"),
        attach_blue: color(mode, "attach_blue"),
        attach_orange: color(mode, "attach_orange"),
        attach_red: color(mode, "attach_red"),
        attach_green: color(mode, "attach_green"),
        traffic_red: color(mode, "traffic_red"),
        traffic_yellow: color(mode, "traffic_yellow"),
        traffic_green: color(mode, "traffic_green"),
    }
}

/// Keep GPUI Kit's own controls (inputs, switches, dialogs) on the same
/// palette as the WhatsLite surfaces.
pub fn apply_theme(cx: &mut App, mode: ThemeMode) {
    apply_theme_scaled(cx, mode, 1.0);
}

pub fn apply_theme_scaled(cx: &mut App, mode: ThemeMode, scale: f32) {
    let gpui_mode = match mode {
        ThemeMode::Light => GpuiThemeMode::Light,
        ThemeMode::Dark => GpuiThemeMode::Dark,
    };
    Theme::change(gpui_mode, None, cx);
    let p = palette(mode);
    Theme::update(cx, |theme| {
        theme.font_family = "Inter".into();
        theme.font_size = px(14. * scale);
        theme.radius = px(8.);
        theme.radius_lg = px(12.);

        theme.colors.background = p.bg_list.into();
        theme.colors.foreground = p.text_primary.into();
        theme.colors.border = p.border.into();
        theme.colors.primary = p.accent.into();
        theme.colors.primary_foreground = p.text_inverse.into();
        theme.colors.primary_hover = p.accent.into();
        theme.colors.primary_active = p.accent.into();
        theme.colors.secondary = p.bg_search.into();
        theme.colors.secondary_foreground = p.text_primary.into();
        theme.colors.muted = p.bg_selected.into();
        theme.colors.muted_foreground = p.text_secondary.into();
        theme.colors.accent = p.accent_soft.into();
        theme.colors.accent_foreground = p.text_primary.into();
        theme.colors.popover = p.bg_list.into();
        theme.colors.popover_foreground = p.text_primary.into();
        theme.colors.input = p.bg_search.into();
        theme.colors.caret = p.accent.into();
        theme.colors.ring = p.accent.into();
        theme.colors.danger = p.attach_red.into();
        theme.colors.danger_foreground = p.text_inverse.into();
        theme.colors.danger_hover = p.attach_red.into();
        theme.colors.danger_active = p.attach_red.into();
        theme.colors.list = p.bg_list.into();
        theme.colors.list_hover = p.bg_row_selected.into();
        theme.colors.list_active = p.bg_selected.into();
        theme.colors.list_active_border = p.accent.into();
        theme.colors.button = p.bg_search.into();
        theme.colors.button_hover = p.bg_selected.into();
        theme.colors.button_active = p.bg_selected.into();
        theme.colors.button_foreground = p.text_primary.into();
        theme.colors.button_primary = p.accent.into();
        theme.colors.button_primary_hover = p.accent.into();
        theme.colors.button_primary_active = p.accent.into();
        theme.colors.button_primary_foreground = p.text_inverse.into();
        theme.colors.button_secondary = p.bg_search.into();
        theme.colors.button_secondary_hover = p.bg_selected.into();
        theme.colors.button_secondary_active = p.bg_selected.into();
        theme.colors.button_secondary_foreground = p.text_primary.into();
        theme.colors.button_danger = p.attach_red.into();
        theme.colors.button_danger_hover = p.attach_red.into();
        theme.colors.button_danger_active = p.attach_red.into();
        theme.colors.button_danger_foreground = p.text_inverse.into();
        theme.colors.scrollbar = p.bg_search.into();
        theme.colors.scrollbar_thumb = p.border.into();
        theme.colors.progress_bar = p.accent.into();
    });
}

pub const TITLEBAR_HEIGHT: f32 = 32.0;
pub const SIDEBAR_WIDTH: f32 = 200.0;
pub const RAIL_WIDTH: f32 = 64.0;
pub const LIST_WIDTH: f32 = 300.0;
pub const CHAT_ROW_HEIGHT: f32 = 80.0;
pub const CONVERSATION_HEADER_HEIGHT: f32 = 58.0;
pub const COMPOSER_HEIGHT: f32 = 56.0;
pub const INFO_PANEL_WIDTH: f32 = 380.0;
pub const SETTINGS_NAV_WIDTH: f32 = 300.0;
pub const COMPACT_BREAKPOINT: f32 = 1100.0;
pub const PANEL_INLINE_BREAKPOINT: f32 = 1280.0;

pub const RADIUS_SM: f32 = 4.0;
pub const RADIUS_MD: f32 = 8.0;
pub const RADIUS_LG: f32 = 10.0;
pub const RADIUS_XL: f32 = 12.0;
pub const RADIUS_BUBBLE: f32 = 18.0;

pub const TEXT_CAPTION: f32 = 11.0;
pub const TEXT_SMALL: f32 = 12.0;
pub const TEXT_BODY: f32 = 14.0;
pub const TEXT_HEADING: f32 = 20.0;
