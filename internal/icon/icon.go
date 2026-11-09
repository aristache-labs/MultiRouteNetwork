// Package icon draws the MultiRoute mark: the app icon, the hicolor PNGs and the tray icon, all from
// one description so they never drift apart. The mark is an M made of three links joined end to end
// (left stem, middle V, right stem) in three teal tones, with connector nodes where the links meet
// and two flow chevrons on the diagonals. Source of truth for the design: docs/brand/.
package icon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// The mark is designed on a 64-unit grid.
const grid = 64

// traySize is the pixel size of the tray glyph; Windows and most Linux trays show 16–32 px.
const traySize = 32

type rgb struct{ r, g, b float64 }

func hex(c uint32) rgb {
	return rgb{float64(c>>16&255) / 255, float64(c>>8&255) / 255, float64(c&255) / 255}
}

var (
	tileTop, tileBottom = hex(0x16292D), hex(0x0C1315)
	tileCut             = hex(0x0E181B)
	linkDeep            = hex(0x0E8E9E)
	linkMid             = hex(0x1EBED3)
	linkBright          = hex(0x9CF0F8)
	offDeep, offMid     = hex(0x8A8A8A), hex(0xB4B4B4)
	offBright           = hex(0xDEDEDE)
)

// App renders the app icon: the mark on a dark rounded tile. Nodes and chevrons are dropped below
// 48 px, where they would turn to mud.
func App(size int) *image.NRGBA {
	return render(size, true, linkDeep, linkMid, linkBright, size >= 48, 0.76)
}

// AppPNG is App encoded as PNG.
func AppPNG(size int) []byte { return encode(App(size)) }

// AppICO is the multi-size Windows icon for the executables and the installer.
func AppICO() []byte {
	var imgs [][]byte
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	for _, s := range sizes {
		imgs = append(imgs, AppPNG(s))
	}
	return ico(sizes, imgs)
}

// WriteHicolor writes the freedesktop hicolor PNG set under dir (dir/<s>x<s>/apps/MultiRouteNetwork.png).
func WriteHicolor(dir string) error {
	for _, s := range []int{16, 32, 48, 64, 128, 256, 512} {
		p := filepath.Join(dir, fmt.Sprintf("%dx%d", s, s), "apps", "MultiRouteNetwork.png")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, AppPNG(s), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// PNG is the tray glyph: the plain M with no tile, teal when system-wide mode is on, grey when off.
func PNG(on bool) []byte {
	if on {
		return encode(render(traySize, false, linkDeep, linkMid, linkBright, false, 0.92))
	}
	return encode(render(traySize, false, offDeep, offMid, offBright, false, 0.92))
}

// ICO wraps the tray PNG in an .ico container, which is what the Windows tray wants.
func ICO(on bool) []byte { return ico([]int{traySize}, [][]byte{PNG(on)}) }

// render draws the mark at size px. scale shrinks the mark inside the grid (1 = touching the edges).
func render(size int, tile bool, deep, mid, bright rgb, cuts bool, scale float64) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	px := float64(size) / grid // pixels per grid unit
	const half = 32.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Pixel centre in grid units, then in mark units (the mark is scaled about the centre).
			gx, gy := (float64(x)+0.5)/px, (float64(y)+0.5)/px
			mx, my := half+(gx-half)/scale, half+(gy-half)/scale
			var p pixel
			if tile {
				t := clamp01(gy / grid)
				p.over(rgb{lerp(tileTop.r, tileBottom.r, t), lerp(tileTop.g, tileBottom.g, t), lerp(tileTop.b, tileBottom.b, t)},
					cover(roundRect(gx, gy, grid, 14), px))
			}
			mpx := px * scale // pixels per mark unit
			p.over(deep, cover(capsule(mx, my, 11, 53, 11, 13, 7.5), mpx))
			p.over(mid, math.Max(cover(capsule(mx, my, 11, 13, 32, 38, 7.5), mpx), cover(capsule(mx, my, 32, 38, 53, 13, 7.5), mpx)))
			p.over(bright, cover(capsule(mx, my, 53, 13, 53, 53, 7.5), mpx))
			if cuts {
				p.over(tileCut, cover(cutouts(mx, my), mpx))
			}
			img.SetNRGBA(x, y, p.nrgba())
		}
	}
	return img
}

// cutouts is the signed distance to the connector nodes (rings) and the two flow chevrons.
func cutouts(x, y float64) float64 {
	const w = 1.3 // half the 2.6 stroke
	d := math.Inf(1)
	for _, c := range [][2]float64{{11, 13}, {53, 13}, {32, 38}} {
		d = math.Min(d, math.Abs(math.Hypot(x-c[0], y-c[1])-3.6)-w)
	}
	for _, ch := range []struct{ tx, ty, deg float64 }{{20, 23.5, 50}, {44, 23.5, 130}} {
		a := ch.deg * math.Pi / 180
		rot := func(lx, ly float64) (float64, float64) {
			return ch.tx + lx*math.Cos(a) - ly*math.Sin(a), ch.ty + lx*math.Sin(a) + ly*math.Cos(a)
		}
		ax, ay := rot(-4, -2.4)
		bx, by := rot(1, 0)
		cx, cy := rot(-4, 2.4)
		d = math.Min(d, capsule(x, y, ax, ay, bx, by, w))
		d = math.Min(d, capsule(x, y, bx, by, cx, cy, w))
	}
	return d
}

// capsule is the signed distance to a round-capped segment of radius r.
func capsule(px, py, ax, ay, bx, by, r float64) float64 {
	dx, dy := bx-ax, by-ay
	t := clamp01(((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy)) - r
}

// roundRect is the signed distance to a square of side s with corner radius r at the origin.
func roundRect(px, py, s, r float64) float64 {
	qx, qy := math.Abs(px-s/2)-(s/2-r), math.Abs(py-s/2)-(s/2-r)
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

// cover turns a signed distance (units) into pixel coverage with one pixel of anti-aliasing.
func cover(d, pxPerUnit float64) float64 { return clamp01(0.5 - d*pxPerUnit) }

type pixel struct{ r, g, b, a float64 }

// over composites colour c with coverage a onto p (straight alpha).
func (p *pixel) over(c rgb, a float64) {
	if a <= 0 {
		return
	}
	na := a + p.a*(1-a)
	p.r = (c.r*a + p.r*p.a*(1-a)) / na
	p.g = (c.g*a + p.g*p.a*(1-a)) / na
	p.b = (c.b*a + p.b*p.a*(1-a)) / na
	p.a = na
}

func (p pixel) nrgba() color.NRGBA {
	return color.NRGBA{byte8(p.r), byte8(p.g), byte8(p.b), byte8(p.a)}
}

func byte8(v float64) uint8 { return uint8(math.Round(clamp01(v) * 255)) }

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func encode(img image.Image) []byte {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// ico packs PNG images into one .ico container (ICONDIR, ICONDIRENTRY per image, then the images).
func ico(sizes []int, imgs [][]byte) []byte {
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(imgs))}) // reserved, type=icon, count
	offset := 6 + 16*len(imgs)
	for i, p := range imgs {
		dim := byte(sizes[i]) // 256 wraps to 0, which is how .ico spells 256
		out.Write([]byte{dim, dim, 0, 0})                           // width, height, colours, reserved
		binary.Write(&out, binary.LittleEndian, [2]uint16{1, 32})   // planes, bit depth
		binary.Write(&out, binary.LittleEndian, uint32(len(p)))     // image size
		binary.Write(&out, binary.LittleEndian, uint32(offset))     // image offset
		offset += len(p)
	}
	for _, p := range imgs {
		out.Write(p)
	}
	return out.Bytes()
}
