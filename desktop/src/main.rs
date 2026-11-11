mod app;
mod assets;
mod components;
mod model;
mod theme;
mod wallpaper;

use std::borrow::Cow;

use gpui_kit::*;

fn main() {
    application().with_assets(assets::Assets).run(|cx| {
        init(cx);
        cx.text_system()
            .add_fonts(vec![
                Cow::Borrowed(include_bytes!("../assets/fonts/InterVariable.ttf")),
                Cow::Borrowed(include_bytes!("../assets/fonts/NotoEmoji-Regular.ttf")),
            ])
            .expect("Inter and Noto Emoji fonts must load");
        let theme_mode = if std::env::var("WHATSLITE_THEME").as_deref() == Ok("dark") {
            theme::ThemeMode::Dark
        } else {
            theme::ThemeMode::Light
        };
        theme::apply_theme(cx, theme_mode);

        let (window_width, window_height) = std::env::var("WHATSLITE_WINDOW")
            .ok()
            .and_then(|value| {
                let (width, height) = value.split_once('x')?;
                Some((
                    width.trim().parse::<f32>().ok()?,
                    height.trim().parse::<f32>().ok()?,
                ))
            })
            .unwrap_or((1440., 900.));
        open_window(
            WindowOptions {
                window_bounds: Some(WindowBounds::Windowed(Bounds::centered(
                    None,
                    size(px(window_width), px(window_height)),
                    cx,
                ))),
                titlebar: Some(TitlebarOptions {
                    title: Some("WhatsLite Desktop".into()),
                    ..TitlebarOptions::default()
                }),
                window_min_size: Some(size(px(1024.), px(700.))),
                ..WindowOptions::default()
            },
            cx,
            |window, cx| cx.new(|cx| app::App::new(window, cx)),
        )
        .expect("Failed to open window");
    });
}
