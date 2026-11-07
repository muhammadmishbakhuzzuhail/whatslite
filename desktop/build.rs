use std::path::Path;

fn main() {
    render_doodle();

    // GPUI links xkbcommon-x11. Some Linux distributions ship the runtime
    // library (`libxkbcommon-x11.so.0`) without the development symlink
    // (`libxkbcommon-x11.so`). Create a local linker symlink when the system
    // one is missing, without requiring a system package change.
    #[cfg(target_os = "linux")]
    {
        use std::path::PathBuf;

        let out_dir = PathBuf::from(std::env::var("OUT_DIR").expect("OUT_DIR is set by Cargo"));
        let link = out_dir.join("libxkbcommon-x11.so");
        if !link.exists() {
            for candidate in [
                "/lib64/libxkbcommon-x11.so.0",
                "/usr/lib64/libxkbcommon-x11.so.0",
                "/lib/x86_64-linux-gnu/libxkbcommon-x11.so.0",
                "/usr/lib/x86_64-linux-gnu/libxkbcommon-x11.so.0",
            ] {
                if Path::new(candidate).exists() {
                    let _ = std::os::unix::fs::symlink(candidate, &link);
                    break;
                }
            }
        }
        println!("cargo:rustc-link-search=native={}", out_dir.display());
    }
}

/// Rasterise the editable doodle master at 1x, 2x and 3x.
///
/// The SVG is the source of truth in `design/machine/wallpaper-doodle-source.svg`.
/// Its strokes are opaque, so each PNG is an alpha mask; the runtime tints it
/// with `theme::wallpaper_style` and repeats it at 800pt.
fn render_doodle() {
    let out_dir = std::path::PathBuf::from(std::env::var("OUT_DIR").expect("OUT_DIR is set by Cargo"));
    let source = Path::new("../design/machine/wallpaper-doodle-source.svg");
    println!("cargo:rerun-if-changed={}", source.display());
    let bytes = std::fs::read(source).expect("wallpaper-doodle-source.svg must exist");
    let tree = resvg::usvg::Tree::from_data(&bytes, &resvg::usvg::Options::default())
        .expect("wallpaper-doodle-source.svg must parse");
    let size = tree.size();

    const TILE: u32 = 800;
    for scale in [1u32, 2, 3] {
        let edge = TILE * scale;
        let mut pixmap = resvg::tiny_skia::Pixmap::new(edge, edge)
            .expect("doodle tile dimensions must be valid");
        let transform = resvg::tiny_skia::Transform::from_scale(
            edge as f32 / size.width(),
            edge as f32 / size.height(),
        );
        resvg::render(&tree, transform, &mut pixmap.as_mut());
        pixmap
            .save_png(out_dir.join(format!("doodle-{scale}x.png")))
            .expect("doodle tile must encode as PNG");
    }
}
