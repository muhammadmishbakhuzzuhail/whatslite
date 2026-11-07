//! Reusable visual primitives. These are deliberately token-driven: no
//! component in this file owns a literal colour.

use std::sync::atomic::{AtomicUsize, Ordering};

use gpui_kit::assets::IconName;
use gpui_kit::component::Icon;
use gpui_kit::prelude::*;
use gpui_kit::{
    div, px, Div, ElementId, FontWeight, Rgba, Role, Stateful,
};

static NEXT_ID: AtomicUsize = AtomicUsize::new(1);

fn next_id(prefix: &str) -> String {
    format!("{prefix}-{}", NEXT_ID.fetch_add(1, Ordering::Relaxed))
}

use crate::theme::{Palette, RADIUS_LG, RADIUS_MD, RADIUS_XL, TEXT_BODY, TEXT_CAPTION, TEXT_SMALL};

pub fn icon(name: IconName, size: f32, color: Rgba) -> Icon {
    Icon::new(name).text_color(color).size(px(size))
}

pub fn avatar(initials: impl Into<String>, size: f32, online: bool, p: Palette) -> Div {
    let initials = initials.into();
    div()
        .relative()
        .flex_shrink_0()
        .size(px(size))
        .child(
            div()
                .size_full()
                .flex()
                .items_center()
                .justify_center()
                .rounded_full()
                .bg(p.bg_search)
                .text_color(p.text_primary)
                .text_size(px(size * 0.34))
                .font_weight(FontWeight::SEMIBOLD)
                .child(initials),
        )
        .when(online, |el| {
            el.child(
                div()
                    .absolute()
                    .bottom_0()
                    .right_0()
                    .size(px(size * 0.24))
                    .rounded_full()
                    .bg(p.accent)
                    .border_2()
                    .border_color(p.bg_list),
            )
        })
}

pub fn avatar_with_ring(initials: impl Into<String>, size: f32, seen: bool, p: Palette) -> Div {
    let initials = initials.into();
    div()
        .flex_shrink_0()
        .size(px(size + 8.0))
        .flex()
        .items_center()
        .justify_center()
        .rounded_full()
        .border_2()
        .border_color(if seen { p.border } else { p.accent })
        .child(avatar(initials, size, false, p))
}

pub fn icon_button(
    id: impl Into<ElementId>,
    name: IconName,
    size: f32,
    color: Rgba,
    hover: Rgba,
    label: &'static str,
) -> Stateful<Div> {
    div()
        .id(id)
        .role(Role::Button)
        .tab_index(0)
        .aria_label(label)
        .cursor_pointer()
        .size(px(36.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_full()
        .hover(move |style| style.bg(hover))
        .child(icon(name, size, color))
}

pub fn icon_button_small(
    id: impl Into<ElementId>,
    name: IconName,
    size: f32,
    color: Rgba,
    hover: Rgba,
    label: &'static str,
) -> Stateful<Div> {
    div()
        .id(id)
        .role(Role::Button)
        .tab_index(0)
        .aria_label(label)
        .cursor_pointer()
        .size(px(28.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_md()
        .hover(move |style| style.bg(hover))
        .child(icon(name, size, color))
}

pub fn primary_button(
    id: impl Into<ElementId>,
    label: impl Into<String>,
    p: Palette,
) -> Stateful<Div> {
    let label = label.into();
    div()
        .id(id)
        .role(Role::Button)
        .tab_index(0)
        .aria_label(label.clone())
        .cursor_pointer()
        .h(px(36.))
        .px(px(16.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_lg()
        .bg(p.accent)
        .text_color(p.text_inverse)
        .text_size(px(TEXT_BODY))
        .font_weight(FontWeight::MEDIUM)
        .hover(move |style| style.bg(p.accent))
        .child(label)
}

pub fn secondary_button(
    id: impl Into<ElementId>,
    label: impl Into<String>,
    p: Palette,
) -> Stateful<Div> {
    let label = label.into();
    div()
        .id(id)
        .role(Role::Button)
        .tab_index(0)
        .aria_label(label.clone())
        .cursor_pointer()
        .h(px(36.))
        .px(px(16.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_lg()
        .bg(p.bg_list)
        .border_1()
        .border_color(p.border)
        .text_color(p.text_primary)
        .text_size(px(TEXT_BODY))
        .font_weight(FontWeight::MEDIUM)
        .hover(move |style| style.bg(p.hover()))
        .child(label)
}

pub fn danger_button(
    id: impl Into<ElementId>,
    label: impl Into<String>,
    p: Palette,
) -> Stateful<Div> {
    let label = label.into();
    div()
        .id(id)
        .role(Role::Button)
        .tab_index(0)
        .aria_label(label.clone())
        .cursor_pointer()
        .h(px(36.))
        .px(px(16.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_lg()
        .bg(p.danger())
        .text_color(p.text_inverse)
        .text_size(px(TEXT_BODY))
        .font_weight(FontWeight::MEDIUM)
        .child(label)
}

pub fn badge(count: u32, p: Palette) -> Div {
    let label = if count > 99 {
        "99+".to_owned()
    } else {
        count.to_string()
    };
    div()
        .min_w(px(20.))
        .h(px(20.))
        .px(px(6.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_full()
        .bg(p.accent)
        .text_color(p.text_inverse)
        .text_size(px(TEXT_CAPTION))
        .font_weight(FontWeight::SEMIBOLD)
        .child(label)
}

pub fn dot_badge(p: Palette) -> Div {
    div().size(px(10.)).rounded_full().bg(p.accent)
}

pub fn chip(label: impl Into<String>, selected: bool, p: Palette) -> Stateful<Div> {
    let label = label.into();
    div()
        .id(next_id("chip"))
        .h(px(34.))
        .px(px(10.))
        .flex()
        .items_center()
        .justify_center()
        .rounded_full()
        .border_1()
        .border_color(if selected { p.accent_soft } else { p.border })
        .bg(if selected { p.accent_soft } else { p.bg_list })
        .text_color(if selected { p.unread_text } else { p.text_primary })
        .text_size(px(13.))
        .font_weight(FontWeight::MEDIUM)
        .child(label)
}

pub fn section_label(label: impl Into<String>, p: Palette) -> Div {
    div()
        .px(px(12.))
        .py(px(6.))
        .text_size(px(TEXT_SMALL))
        .font_weight(FontWeight::SEMIBOLD)
        .text_color(p.text_secondary)
        .child(label.into())
}

pub fn divider(p: Palette) -> Div {
    div().h(px(1.)).w_full().bg(p.divider)
}

pub fn empty_state(
    title: impl Into<String>,
    body: impl Into<String>,
    action: Option<Stateful<Div>>,
    p: Palette,
) -> Div {
    let title = title.into();
    let body = body.into();
    div()
        .flex()
        .flex_col()
        .items_center()
        .justify_center()
        .gap(px(12.))
        .p(px(32.))
        .child(
            div()
                .size(px(88.))
                .flex()
                .items_center()
                .justify_center()
                .rounded_full()
                .bg(p.accent_soft)
                .child(icon(IconName::Zap, 34.0, p.accent)),
        )
        .child(
            div()
                .text_size(px(20.))
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(p.text_primary)
                .child(title),
        )
        .child(
            div()
                .max_w(px(420.))
                .text_center()
                .text_size(px(TEXT_BODY))
                .text_color(p.text_secondary)
                .child(body),
        )
        .when_some(action, |el, action| el.child(action))
}

pub fn empty_state_compact(
    title: impl Into<String>,
    body: impl Into<String>,
    action: Option<Stateful<Div>>,
    p: Palette,
) -> Div {
    let title = title.into();
    let body = body.into();
    div()
        .flex()
        .flex_col()
        .items_center()
        .justify_center()
        .gap(px(8.))
        .p(px(16.))
        .child(
            div()
                .size(px(64.))
                .flex()
                .items_center()
                .justify_center()
                .rounded_full()
                .bg(p.accent_soft)
                .child(icon(IconName::Zap, 26.0, p.accent)),
        )
        .child(
            div()
                .text_size(px(16.))
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(p.text_primary)
                .text_center()
                .child(title),
        )
        .child(
            div()
                .w(px(240.))
                .text_center()
                .text_size(px(TEXT_SMALL))
                .text_color(p.text_secondary)
                .child(body),
        )
        .when_some(action, |el, action| el.child(action))
}

pub fn skeleton_row(p: Palette) -> Div {
    div()
        .h(px(80.))
        .flex()
        .items_center()
        .gap(px(12.))
        .px(px(12.))
        .border_b_1()
        .border_color(p.divider)
        .child(div().size(px(44.)).rounded_full().bg(p.bg_search))
        .child(
            div()
                .flex_1()
                .flex()
                .flex_col()
                .gap(px(8.))
                .child(div().h(px(12.)).w(px(140.)).rounded_md().bg(p.bg_search))
                .child(div().h(px(10.)).w(px(210.)).rounded_md().bg(p.bg_search)),
        )
}

pub fn wallpaper(dark: bool) -> Div {
    crate::wallpaper::wallpaper(dark)
}

pub fn settings_row(
    title: impl Into<String>,
    description: impl Into<String>,
    value: Option<String>,
    p: Palette,
) -> Stateful<Div> {
    let title = title.into();
    let description = description.into();
    div()
        .id(next_id("settings-row"))
        .flex()
        .items_center()
        .justify_between()
        .gap(px(16.))
        .px(px(20.))
        .py(px(14.))
        .border_b_1()
        .border_color(p.divider)
        .child(
            div()
                .flex()
                .flex_col()
                .gap(px(2.))
                .child(
                    div()
                        .text_size(px(TEXT_BODY))
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(p.text_primary)
                        .child(title),
                )
                .child(
                    div()
                        .text_size(px(TEXT_SMALL))
                        .text_color(p.text_secondary)
                        .child(description),
                ),
        )
        .child(
            div()
                .flex()
                .items_center()
                .gap(px(6.))
                .when_some(value, |el, value| {
                    el.child(
                        div()
                            .text_size(px(TEXT_SMALL))
                            .text_color(p.text_secondary)
                            .child(value),
                    )
                })
                .child(icon(IconName::ChevronRight, 16.0, p.text_secondary)),
        )
}

pub fn live_pill(p: Palette) -> Div {
    div()
        .h(px(20.))
        .px(px(8.))
        .flex()
        .items_center()
        .rounded_full()
        .border_1()
        .border_color(p.border)
        .text_size(px(10.))
        .font_weight(FontWeight::SEMIBOLD)
        .text_color(p.text_secondary)
        .child("LIVE")
}

pub fn soon_pill(p: Palette) -> Div {
    div()
        .h(px(20.))
        .px(px(8.))
        .flex()
        .items_center()
        .rounded_full()
        .bg(p.accent_soft)
        .text_size(px(10.))
        .font_weight(FontWeight::SEMIBOLD)
        .text_color(p.unread_text)
        .child("COMING SOON")
}

pub fn media_tile(
    label: impl Into<String>,
    meta: impl Into<String>,
    icon_name: IconName,
    accent: usize,
    p: Palette,
) -> Div {
    let colors = [p.attach_orange, p.attach_blue, p.attach_green, p.attach_red, p.accent, p.tick_read];
    let bg = colors[accent % colors.len()];
    let label = label.into();
    let meta = meta.into();
    div()
        .relative()
        .h(px(140.))
        .w_full()
        .overflow_hidden()
        .rounded(px(RADIUS_LG))
        .bg(bg)
        .child(
            div()
                .absolute()
                .top_0()
                .left_0()
                .size_full()
                .opacity(0.18)
                .child(icon(icon_name, 160.0, p.text_inverse)),
        )
        .child(
            div()
                .absolute()
                .left(px(8.))
                .bottom(px(8.))
                .px(px(6.))
                .py(px(2.))
                .rounded_md()
                .bg(p.text_primary)
                .text_color(p.text_inverse)
                .text_size(px(10.))
                .child(meta),
        )
        .child(
            div()
                .absolute()
                .left(px(10.))
                .top(px(10.))
                .max_w(px(150.))
                .text_size(px(11.))
                .font_weight(FontWeight::MEDIUM)
                .text_color(p.text_inverse)
                .child(label),
        )
}

pub fn call_direction_icon(missed: bool, incoming: bool, p: Palette) -> Icon {
    let (name, color) = if missed {
        (IconName::PhoneMissed, p.danger())
    } else if incoming {
        (IconName::PhoneIncoming, p.attach_green)
    } else {
        (IconName::PhoneOutgoing, p.attach_green)
    };
    icon(name, 14.0, color)
}

pub fn toast_shell(message: impl Into<String>, p: Palette) -> Div {
    div()
        .flex()
        .items_center()
        .gap(px(10.))
        .px(px(16.))
        .py(px(10.))
        .rounded(px(RADIUS_MD))
        .bg(p.text_primary)
        .text_color(p.text_inverse)
        .text_size(px(TEXT_SMALL))
        .child(icon(IconName::Info, 16.0, p.text_inverse))
        .child(message.into())
}

pub fn modal_shell(p: Palette) -> Div {
    div()
        .flex()
        .flex_col()
        .gap(px(16.))
        .w(px(420.))
        .p(px(24.))
        .rounded(px(RADIUS_XL))
        .bg(p.bg_list)
        .border_1()
        .border_color(p.border)
}

pub fn context_menu_shell(p: Palette) -> Div {
    div()
        .flex()
        .flex_col()
        .w(px(200.))
        .p(px(6.))
        .rounded(px(RADIUS_LG))
        .bg(p.bg_list)
        .border_1()
        .border_color(p.border)
}

pub fn menu_item(label: impl Into<String>, icon_name: IconName, p: Palette) -> Stateful<Div> {
    div()
        .id(next_id("menu-item"))
        .flex()
        .items_center()
        .gap(px(10.))
        .h(px(36.))
        .px(px(10.))
        .rounded_md()
        .cursor_pointer()
        .text_size(px(TEXT_BODY))
        .text_color(p.text_primary)
        .hover(move |style| style.bg(p.hover()))
        .child(icon(icon_name, 16.0, p.text_secondary))
        .child(label.into())
}
