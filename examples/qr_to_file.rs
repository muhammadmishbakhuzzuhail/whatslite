//! Render a pairing QR raw string to a BMP image file.
//!
//! Companion to `pairing_test`: when the terminal half-block rendering is
//! not scannable (font spacing, non-square glyphs), write the QR to an
//! image instead. The raw string carries key material — never paste it
//! into third-party QR generators.
//!
//! Usage:
//!   cargo run --example qr_to_file -- <raw-qr-string> [out.bmp]

use std::env;
use std::fs::File;
use std::io::BufWriter;
use std::io::Write as _;

/// 24-bit uncompressed BMP, hand-rolled to avoid pulling in an image crate.
fn write_bmp(
    path: &str,
    img_w: u32,
    img_h: u32,
    pixel: impl Fn(u32, u32) -> u8,
) -> std::io::Result<()> {
    let row_bytes = (img_w * 3 + 3) & !3; // rows padded to 4 bytes
    let data_size = row_bytes * img_h;
    let file_size = 54 + data_size;

    let mut w = BufWriter::new(File::create(path)?);
    // BITMAPFILEHEADER
    w.write_all(b"BM")?;
    w.write_all(&file_size.to_le_bytes())?;
    w.write_all(&0u32.to_le_bytes())?;
    w.write_all(&54u32.to_le_bytes())?;
    // BITMAPINFOHEADER
    w.write_all(&40u32.to_le_bytes())?;
    w.write_all(&(img_w as i32).to_le_bytes())?;
    w.write_all(&(img_h as i32).to_le_bytes())?;
    w.write_all(&1u16.to_le_bytes())?;
    w.write_all(&24u16.to_le_bytes())?;
    w.write_all(&0u32.to_le_bytes())?; // BI_RGB
    w.write_all(&data_size.to_le_bytes())?;
    w.write_all(&2835u32.to_le_bytes())?; // 72 dpi
    w.write_all(&2835u32.to_le_bytes())?;
    w.write_all(&0u32.to_le_bytes())?;
    w.write_all(&0u32.to_le_bytes())?;

    // Bottom-up rows; grayscale so B/G/R share the value.
    let mut row = vec![0u8; row_bytes as usize];
    for y in (0..img_h).rev() {
        for x in 0..img_w {
            let v = pixel(x, y);
            let o = x as usize * 3;
            row[o] = v;
            row[o + 1] = v;
            row[o + 2] = v;
        }
        w.write_all(&row)?;
    }
    w.flush()
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() < 2 {
        eprintln!("usage: qr_to_file <raw-qr-string> [out.bmp]");
        std::process::exit(2);
    }
    let raw = &args[1];
    let out_path = args.get(2).cloned().unwrap_or_else(|| "qr.bmp".to_string());

    let qr = qrcode::QrCode::new(raw.as_bytes()).expect("encode QR");
    let modules = qr.width() as u32;
    let colors = qr.to_colors();

    let scale: u32 = 12;
    let quiet: u32 = 4; // quiet-zone modules, the spec minimum
    let img = (modules + quiet * 2) * scale;

    if out_path.ends_with(".svg") {
        // One path segment per dark module, crisp at any size.
        let mut path = String::new();
        for y in 0..modules {
            for x in 0..modules {
                if colors[y as usize * modules as usize + x as usize] == qrcode::types::Color::Dark
                {
                    path.push_str(&format!("M{} {}h1v1z ", x + quiet, y + quiet));
                }
            }
        }
        let total = modules + quiet * 2;
        std::fs::write(
            &out_path,
            format!(
                "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 {total} {total}\" shape-rendering=\"crispEdges\"><rect width=\"{total}\" height=\"{total}\" fill=\"#ffffff\"/><path fill=\"#000000\" d=\"{path}\"/></svg>\n"
            ),
        )
        .expect("write SVG");
        println!("QR written: {out_path} (SVG, {modules} modules, quiet zone {quiet})");
        return;
    }

    write_bmp(&out_path, img, img, |x, y| {
        let mx = (x / scale) as i64 - quiet as i64;
        let my = (y / scale) as i64 - quiet as i64;
        let in_matrix = mx >= 0 && my >= 0 && (mx as u32) < modules && (my as u32) < modules;
        let dark = in_matrix
            && colors[my as u32 as usize * modules as usize + mx as u32 as usize]
                == qrcode::types::Color::Dark;
        if dark { 0x00 } else { 0xFF }
    })
    .expect("write BMP");

    println!("QR written: {out_path} ({img}x{img}px, {modules} modules, scale {scale})");
}
