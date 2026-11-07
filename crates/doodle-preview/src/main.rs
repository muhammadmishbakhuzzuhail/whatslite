use std::collections::HashMap;
use std::sync::{Arc, Mutex, OnceLock};

use gpui_kit::prelude::*;
use gpui_kit::{
    canvas, div, img, point, px, rgb, rgba, size, Bounds, Corners, Div, ImageSource, ObjectFit,
    RenderImage, Rgba, WindowOptions,
};

const TILE_1X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-1x.png"));
const TILE_2X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-2x.png"));
const TILE_3X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-3x.png"));
const LIGHT_BG: u32 = 0xF4F1EB;
const DARK_BG: u32 = 0x0B141A;
const LINE: u32 = 0xC9B9A3;
const DARK_LINE: u32 = 0xD8CCBC;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
struct Key {
    edge: i32,
    tint: u32,
}

fn edge(scale: f32) -> i32 {
    if scale > 2.0 {
        2400
    } else if scale > 1.0 {
        1600
    } else {
        800
    }
}

fn tinted(scale: f32, line: Rgba) -> Option<Arc<RenderImage>> {
    let edge = edge(scale);
    let key = Key {
        edge,
        tint: (line.r.to_bits() << 24) ^ (line.g.to_bits() << 16) ^ line.b.to_bits(),
    };
    static CACHE: OnceLock<Mutex<HashMap<Key, Arc<RenderImage>>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(HashMap::new()));
    if let Ok(cache) = cache.lock() {
        if let Some(image) = cache.get(&key) {
            return Some(image.clone());
        }
    }
    let bytes = match edge {
        2400 => TILE_3X,
        1600 => TILE_2X,
        _ => TILE_1X,
    };
    let mut pixels = image::load_from_memory(bytes).ok()?.to_rgba8();
    let red = (line.r.clamp(0.0, 1.0) * 255.0).round() as u8;
    let green = (line.g.clamp(0.0, 1.0) * 255.0).round() as u8;
    let blue = (line.b.clamp(0.0, 1.0) * 255.0).round() as u8;
    let table: [u8; 256] =
        std::array::from_fn(|i| ((i as u16 * (line.a.clamp(0.0, 1.0) * 255.0).round() as u16 + 127) / 255) as u8);
    let raw: &mut [u8] = pixels.as_mut();
    for pixel in raw.chunks_exact_mut(4) {
        let alpha = pixel[3];
        pixel[0] = blue;
        pixel[1] = green;
        pixel[2] = red;
        pixel[3] = table[alpha as usize];
    }
    let image = Arc::new(RenderImage::new([image::Frame::new(pixels)]));
    if let Ok(mut cache) = cache.lock() {
        cache.insert(key, image.clone());
    }
    Some(image)
}

fn doodle(tile: f32, line: u32, opacity: f32) -> Div {
    let line = rgb(line);
    div()
        .absolute()
        .top_0()
        .left_0()
        .right_0()
        .bottom_0()
        .opacity(opacity)
        .child(
            canvas(
                move |bounds, window, _cx| (bounds, tinted(window.scale_factor(), line)),
                move |_bounds, (bounds, texture), window, _cx| {
                    let Some(texture) = texture else { return };
                    let cols = (f32::from(bounds.size.width) / tile).ceil() as i32 + 1;
                    let rows = (f32::from(bounds.size.height) / tile).ceil() as i32 + 1;
                    for row in 0..rows {
                        for col in 0..cols {
                            let origin = point(
                                bounds.origin.x + px(col as f32 * tile),
                                bounds.origin.y + px(row as f32 * tile),
                            );
                            let tile = Bounds {
                                origin,
                                size: size(px(tile), px(tile)),
                            };
                            let _ = window.paint_image(
                                tile,
                                tile,
                                Corners::default(),
                                texture.clone(),
                                0,
                                false,
                            );
                        }
                    }
                },
            )
            .absolute()
            .top_0()
            .left_0()
            .right_0()
            .bottom_0(),
        )
}

fn load_reference() -> Option<Arc<RenderImage>> {
    let path = std::env::var("DOODLE_REFERENCE").ok()?;
    let mut image = image::open(path).ok()?.to_rgba8();
    let raw: &mut [u8] = image.as_mut();
    for pixel in raw.chunks_exact_mut(4) {
        pixel.swap(0, 2);
    }
    Some(Arc::new(RenderImage::new([image::Frame::new(image)])))
}

struct Preview {
    reference: Option<Arc<RenderImage>>,
    tile: f32,
    line: u32,
    light_opacity: f32,
    dark_opacity: f32,
}

impl Render for Preview {
    fn render(&mut self, _window: &mut gpui_kit::Window, _cx: &mut gpui_kit::Context<Self>) -> impl IntoElement {
        let panel = |title: String, background: u32, content: gpui_kit::AnyElement| {
            div()
                .flex_1()
                .h_full()
                .relative()
                .overflow_hidden()
                .bg(rgb(background))
                .child(content)
                .child(
                    div()
                        .absolute()
                        .top(px(12.))
                        .left(px(12.))
                        .px(px(8.))
                        .py(px(4.))
                        .rounded(px(6.))
                        .bg(rgba(0x00000099))
                        .text_size(px(12.))
                        .text_color(rgb(0xFFFFFFFF))
                        .child(title),
                )
                .into_any_element()
        };
        let reference = self.reference.clone();
        let reference_panel = if let Some(reference) = reference {
            panel(
                "WHATSAPP REFERENCE".to_string(),
                0x1A1A1A,
                div()
                    .size_full()
                    .flex()
                    .items_center()
                    .justify_center()
                    .child(
                        img(ImageSource::Render(reference))
                            .w_full()
                            .h_full()
                            .object_fit(ObjectFit::Contain),
                    )
                    .into_any_element(),
            )
        } else {
            panel("WHATSAPP REFERENCE · set DOODLE_REFERENCE".to_string(), 0x1A1A1A, div().into_any_element())
        };

        div()
            .size_full()
            .flex()
            .font_family("Inter")
            .child(panel(
                format!("LIGHT · {}PT TILE", self.tile),
                LIGHT_BG,
                doodle(self.tile, self.line, self.light_opacity).into_any_element(),
            ))
            .child(panel(
                format!("DARK · {}PT TILE", self.tile),
                DARK_BG,
                doodle(self.tile, DARK_LINE, self.dark_opacity).into_any_element(),
            ))
            .child(reference_panel)
    }
}

fn main() {
    gpui_kit::application().run(|cx| {
        gpui_kit::init(cx);
        let reference = load_reference();
        let tile = std::env::var("DOODLE_TILE").ok().and_then(|v| v.parse().ok()).unwrap_or(800.0);
        let line = std::env::var("DOODLE_LINE").ok().and_then(|v| u32::from_str_radix(v.trim_start_matches('#'), 16).ok()).unwrap_or(LINE);
        let light_opacity = std::env::var("DOODLE_LIGHT_OPACITY").ok().and_then(|v| v.parse().ok()).unwrap_or(0.60);
        let dark_opacity = std::env::var("DOODLE_DARK_OPACITY").ok().and_then(|v| v.parse().ok()).unwrap_or(0.20);
        gpui_kit::open_window(
            WindowOptions {
                window_bounds: Some(gpui_kit::WindowBounds::Windowed(gpui_kit::Bounds::centered(
                    None,
                    size(px(1500.), px(900.)),
                    cx,
                ))),
                titlebar: Some(gpui_kit::TitlebarOptions {
                    title: Some("Doodle Preview".into()),
                    ..Default::default()
                }),
                ..Default::default()
            },
            cx,
            move |_window, cx| cx.new(|_| Preview { reference, tile, line, light_opacity, dark_opacity }),
        )
        .expect("Failed to open doodle preview");
    });
}
