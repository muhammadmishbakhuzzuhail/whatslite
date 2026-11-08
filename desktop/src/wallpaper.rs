//! Tiled doodle wallpaper.
//!
//! The source is a single original, seamlessly wrapped SVG tile. It is
//! rasterised at 1x, 2x and 3x by `build.rs`; the stage repeats the matching
//! texture at a fixed 800pt logical size. The pre-rendered tile is used as a
//! monochrome alpha mask: the runtime tints it with the wallpaper colour from
//! `theme::wallpaper_style`, so light, dark and future wallpaper themes share
//! one artwork.

use std::collections::HashMap;
use std::sync::{Arc, Mutex, OnceLock};

use gpui_kit::prelude::*;
use gpui_kit::{Bounds, Corners, Div, RenderImage, Rgba, canvas, div, point, px, size};

use crate::theme::wallpaper_style;

const TILE_1X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-1x.png"));
const TILE_2X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-2x.png"));
const TILE_3X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-3x.png"));

/// Logical size of one repeat. 3x maps 1:1 to the 2400px texture produced by
/// the build script.
pub const TILE_LOGICAL: f32 = 800.0;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
struct TextureKey {
    edge: i32,
    red: u32,
    green: u32,
    blue: u32,
    alpha: u32,
}

pub(crate) fn texture_edge(scale: f32) -> i32 {
    if scale > 2.0 {
        2400
    } else if scale > 1.0 {
        1600
    } else {
        800
    }
}

fn texture(scale: f32, line: Rgba) -> Option<Arc<RenderImage>> {
    let edge = texture_edge(scale);
    let key = TextureKey {
        edge,
        red: line.r.to_bits(),
        green: line.g.to_bits(),
        blue: line.b.to_bits(),
        alpha: line.a.to_bits(),
    };

    static CACHE: OnceLock<Mutex<HashMap<TextureKey, Arc<RenderImage>>>> = OnceLock::new();
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
    let tint_alpha = (line.a.clamp(0.0, 1.0) * 255.0).round() as u16;
    let alpha_table: [u8; 256] =
        std::array::from_fn(|index| ((index as u16 * tint_alpha + 127) / 255) as u8);
    let raw: &mut [u8] = pixels.as_mut();
    for pixel in raw.chunks_exact_mut(4) {
        let alpha = pixel[3];
        pixel[0] = blue;
        pixel[1] = green;
        pixel[2] = red;
        pixel[3] = alpha_table[alpha as usize];
    }

    let image = RenderImage::new([image::Frame::new(pixels)]);
    let image = Arc::new(image);
    if let Ok(mut cache) = cache.lock() {
        cache.insert(key, image.clone());
    }
    Some(image)
}

/// Warm the texture for the current scale before the first frame, so the
/// tinting pass never stalls the first paint.
pub fn prewarm(scale: f32, dark: bool) {
    let style = wallpaper_style(dark);
    let _ = texture(scale, style.line);
}

/// Absolutely positioned wallpaper that fills its parent and repeats the
/// doodle tile at `TILE_LOGICAL`.
pub fn wallpaper(dark: bool) -> Div {
    let style = wallpaper_style(dark);
    div()
        .absolute()
        .top_0()
        .left_0()
        .right_0()
        .bottom_0()
        .opacity(style.opacity)
        .child(
            canvas(
                move |bounds, window, _cx| (bounds, texture(window.scale_factor(), style.line)),
                move |_bounds, (bounds, texture), window, _cx| {
                    let Some(texture) = texture else {
                        return;
                    };
                    let cols = (f32::from(bounds.size.width) / TILE_LOGICAL).ceil() as i32 + 1;
                    let rows = (f32::from(bounds.size.height) / TILE_LOGICAL).ceil() as i32 + 1;
                    for row in 0..rows {
                        for col in 0..cols {
                            let origin = point(
                                bounds.origin.x + px(col as f32 * TILE_LOGICAL),
                                bounds.origin.y + px(row as f32 * TILE_LOGICAL),
                            );
                            let tile_bounds = Bounds {
                                origin,
                                size: size(px(TILE_LOGICAL), px(TILE_LOGICAL)),
                            };
                            let _ = window.paint_image(
                                tile_bounds,
                                tile_bounds,
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

#[cfg(test)]
mod tests {
    use super::*;
    use gpui_kit::{DevicePixels, rgb};

    #[test]
    fn picks_the_1x_2x_3x_texture() {
        assert_eq!(texture_edge(1.0), 800);
        assert_eq!(texture_edge(1.5), 1600);
        assert_eq!(texture_edge(2.0), 1600);
        assert_eq!(texture_edge(2.5), 2400);
        assert_eq!(texture_edge(3.0), 2400);
    }

    #[test]
    fn pre_rendered_tiles_tint_at_1x_2x_3x() {
        for (scale, edge) in [(1.0, 800), (2.0, 1600), (3.0, 2400)] {
            let image = texture(scale, rgb(0xD8CCBC)).expect("the doodle PNG must decode and tint");
            let rendered = image.size(0);
            assert_eq!(rendered.width, DevicePixels(edge));
            assert_eq!(rendered.height, DevicePixels(edge));
            let bytes = image.as_bytes(0).expect("tinted texture must have a frame");
            let pixel = bytes
                .chunks_exact(4)
                .find(|pixel| pixel[3] > 0)
                .expect("tinted texture must have visible pixels");
            assert_eq!(pixel[0], 188, "blue channel");
            assert_eq!(pixel[1], 204, "green channel");
            assert_eq!(pixel[2], 216, "red channel");
        }
    }
}
