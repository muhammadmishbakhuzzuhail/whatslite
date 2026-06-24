package gioui

import (
	"image"
	"strings"
	"testing"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// TestMetaAIIconRasters memastikan SVG ikon "metaai" (pakai <g transform=rotate>)
// di-parse oksvg tanpa error DAN menghasilkan piksel (bukan blank).
func TestMetaAIIconRasters(t *testing.T) {
	p, ok := iconPaths["metaai"]
	if !ok {
		t.Fatal("metaai tak ada di iconPaths")
	}
	svg := `<svg viewBox="0 0 24 24" fill="none" stroke="#ffffff" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + p + `</svg>`
	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		t.Fatalf("oksvg gagal parse metaai (transform tak didukung?): %v", err)
	}
	const sz = 64
	icon.SetTarget(0, 0, sz, sz)
	rgba := image.NewRGBA(image.Rect(0, 0, sz, sz))
	scanner := rasterx.NewScannerGV(sz, sz, rgba, rgba.Bounds())
	icon.Draw(rasterx.NewDasher(sz, sz, scanner), 1.0)
	nonzero := 0
	for _, b := range rgba.Pix {
		if b != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatal("ikon metaai ter-raster KOSONG (path/transform tak menggambar apa pun)")
	}
}
