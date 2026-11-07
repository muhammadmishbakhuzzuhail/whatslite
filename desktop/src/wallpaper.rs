//! Tiled doodle wallpaper.
//!
//! The source is a single original, seamlessly wrapped SVG tile. It is
//! rasterised at 1x, 2x and 3x by `build.rs`; the stage repeats the matching
//! texture at a fixed 400pt logical size. This keeps 3x displays sharp
//! without paying for an SVG rasterisation on the first frame.

use std::collections::HashMap;
use std::sync::{Arc, Mutex, OnceLock};

use gpui_kit::prelude::*;
use gpui_kit::{
    canvas, div, point, px, size, App, Bounds, Corners, Div, Image, ImageFormat, RenderImage,
};

const TILE_1X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-1x.png"));
const TILE_2X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-2x.png"));
const TILE_3X: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/doodle-3x.png"));

/// Logical size of one repeat. 3x maps 1:1 to the 1200px texture produced by
/// the build script.
pub const TILE_LOGICAL: f32 = 400.0;

pub(crate) fn texture_edge(scale: f32) -> i32 {
    if scale > 2.0 {
        1200
    } else if scale > 1.0 {
        800
    } else {
        400
    }
}

fn texture(scale: f32, cx: &App) -> Option<Arc<RenderImage>> {
    let edge = texture_edge(scale);
    let bytes = match edge {
        1200 => TILE_3X,
        800 => TILE_2X,
        _ => TILE_1X,
    };

    static CACHE: OnceLock<Mutex<HashMap<i32, Arc<RenderImage>>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(HashMap::new()));
    if let Ok(cache) = cache.lock() {
        if let Some(image) = cache.get(&edge) {
            return Some(image.clone());
        }
    }

    let image = Image::from_bytes(ImageFormat::Png, bytes.to_vec());
    let image = image.to_image_data(cx.svg_renderer()).ok()?;
    if let Ok(mut cache) = cache.lock() {
        cache.insert(edge, image.clone());
    }
    Some(image)
}

/// Absolutely positioned wallpaper that fills its parent and repeats the
/// doodle tile at `TILE_LOGICAL`.
pub fn wallpaper(dark: bool) -> Div {
    div()
        .absolute()
        .top_0()
        .left_0()
        .right_0()
        .bottom_0()
        .opacity(if dark { 0.12 } else { 0.72 })
        .child(
            canvas(
                move |bounds, window, cx| (bounds, texture(window.scale_factor(), cx)),
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
    use gpui_kit::{DevicePixels, SvgRenderer};

    #[test]
    fn picks_the_1x_2x_3x_texture() {
        assert_eq!(texture_edge(1.0), 400);
        assert_eq!(texture_edge(1.5), 800);
        assert_eq!(texture_edge(2.0), 800);
        assert_eq!(texture_edge(2.5), 1200);
        assert_eq!(texture_edge(3.0), 1200);
    }

    #[test]
    fn pre_rendered_tiles_have_the_expected_sizes() {
        let renderer = SvgRenderer::new(std::sync::Arc::new(crate::assets::Assets));
        for (bytes, edge) in [(TILE_1X, 400), (TILE_2X, 800), (TILE_3X, 1200)] {
            let image = Image::from_bytes(ImageFormat::Png, bytes.to_vec());
            let image = image
                .to_image_data(renderer.clone())
                .expect("the pre-rendered doodle PNG must decode");
            let rendered = image.size(0);
            assert_eq!(rendered.width, DevicePixels(edge));
            assert_eq!(rendered.height, DevicePixels(edge));
        }
    }
}
