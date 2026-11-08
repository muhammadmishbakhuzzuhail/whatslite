use std::path::Path;

fn main() {
    let out_dir = std::path::PathBuf::from(std::env::var("OUT_DIR").expect("OUT_DIR is set by Cargo"));
    let source = Path::new("../../reference/design/machine/wallpaper-doodle-source.svg");
    println!("cargo:rerun-if-changed={}", source.display());
    let bytes = std::fs::read(source).expect("wallpaper-doodle-source.svg must exist");
    let tree = resvg::usvg::Tree::from_data(&bytes, &resvg::usvg::Options::default())
        .expect("wallpaper-doodle-source.svg must parse");
    let size = tree.size();
    const TILE: f32 = 800.0;
    for scale in [1u32, 2, 3] {
        let width = (TILE * scale as f32).round() as u32;
        let height = width;
        let mut pixmap = resvg::tiny_skia::Pixmap::new(width, height)
            .expect("doodle preview dimensions must be valid");
        let transform = resvg::tiny_skia::Transform::from_scale(
            width as f32 / size.width(),
            height as f32 / size.height(),
        );
        resvg::render(&tree, transform, &mut pixmap.as_mut());
        pixmap
            .save_png(out_dir.join(format!("doodle-{scale}x.png")))
            .expect("doodle preview PNG must encode");
    }

    #[cfg(target_os = "linux")]
    {
        use std::path::PathBuf;
        let out = PathBuf::from(std::env::var("OUT_DIR").expect("OUT_DIR is set by Cargo"));
        let link = out.join("libxkbcommon-x11.so");
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
        println!("cargo:rustc-link-search=native={}", out.display());
    }
}