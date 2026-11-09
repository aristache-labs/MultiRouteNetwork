package icon

import (
	"bytes"
	"image/png"
	"testing"
)

func TestAppIconIsTileWithMark(t *testing.T) {
	img := App(256)
	corner := img.NRGBAAt(0, 0)
	if corner.A != 0 {
		t.Fatalf("tile corner should be transparent (rounded), got alpha %d", corner.A)
	}
	centre := img.NRGBAAt(128, 128)
	if centre.A != 255 {
		t.Fatalf("tile centre should be opaque, got alpha %d", centre.A)
	}
	// The right stem is the bright tone; sample inside it (x=53, y=45 on the grid, 0.76 scale).
	rx, ry := int((32+(53-32)*0.76)*4), int((32+(45-32)*0.76)*4)
	stem := img.NRGBAAt(rx, ry)
	if stem.R < 140 || stem.G < 230 || stem.B < 230 {
		t.Fatalf("right stem should be the bright teal, got %+v", stem)
	}
}

func TestTrayGlyphHasTransparentBackground(t *testing.T) {
	for _, on := range []bool{true, false} {
		img, err := png.Decode(bytes.NewReader(PNG(on)))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
			t.Fatalf("on=%v: tray corner should be transparent", on)
		}
		if _, _, _, a := img.At(3, 16).RGBA(); a == 0 {
			t.Fatalf("on=%v: left stem should be drawn", on)
		}
	}
}

func TestICOLayout(t *testing.T) {
	b := AppICO()
	if len(b) < 6 || b[2] != 1 || b[4] != 7 {
		t.Fatalf("expected a 7-image icon, header %v", b[:6])
	}
	if b[6] != 16 || b[6+16*6] != 0 { // 16 px first, 256 px (written as 0) last
		t.Fatalf("unexpected entry sizes %d, %d", b[6], b[6+16*6])
	}
}
