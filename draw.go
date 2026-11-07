//go:build !headless

// Drawing helpers: shapes, gradients and text in the game's 1600x900 logical
// space, drawn at the window's real resolution. (c) 2026 Ironite
package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

const LW, LH = 1600.0, 900.0

// ---------------------------------------------------------------- colours

func rgb(h uint32) color.NRGBA {
	return color.NRGBA{uint8(h >> 16), uint8(h >> 8), uint8(h), 255}
}
func rgba(h uint32, a float64) color.NRGBA {
	c := rgb(h)
	c.A = uint8(math.Max(0, math.Min(255, a*255)))
	return c
}
func withA(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(math.Max(0, math.Min(255, float64(c.A)*a)))
	return c
}
func mix(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

var (
	cInk     = rgb(0x1b1a20)
	cPaper   = rgb(0xf6efe0)
	cGold    = rgb(0xe3b65c)
	cGoldDim = rgb(0xa9843e)
	cLacquer = rgb(0x8f1d1d)
	cJade    = rgb(0x2e7d5b)
	cJadeLt  = rgb(0x8fd0a8)
	cMuted   = rgb(0xc9bfa8)
	cPanel   = rgba(0x16141a, .9)
	cRed     = rgb(0xc8312b)
	cGreen   = rgb(0x1f7a4a)
	cBlue    = rgb(0x1f5f9a)
)

// ---------------------------------------------------------------- fonts

type FontID int

const (
	FTitle FontID = iota
	FUI
	FBold
	FItalic
	FGlyph
)

var fontFiles = map[FontID]string{
	FTitle:  "assets/fonts/YoungSerif-Regular.ttf",
	FUI:     "assets/fonts/WorkSans-Regular.ttf",
	FBold:   "assets/fonts/WorkSans-Bold.ttf",
	FItalic: "assets/fonts/WorkSans-Italic.ttf",
	FGlyph:  "assets/fonts/MahjongGlyphs.otf",
}

var fonts = map[FontID]*sfnt.Font{}

type faceKey struct {
	f  FontID
	px int // size in device pixels * 4
}

var faces = map[faceKey]*text.GoXFace{}

func loadFonts() {
	for id, p := range fontFiles {
		f, err := opentype.Parse(assetBytes(p))
		if err != nil {
			panic(p + ": " + err.Error())
		}
		fonts[id] = f
	}
}

func faceFor(f FontID, devPx float64) *text.GoXFace {
	k := faceKey{f, int(math.Round(devPx * 4))}
	if fc, ok := faces[k]; ok {
		return fc
	}
	of, err := opentype.NewFace(fonts[f], &opentype.FaceOptions{Size: float64(k.px) / 4, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	fc := text.NewGoXFace(of)
	faces[k] = fc
	return fc
}

// ---------------------------------------------------------------- canvas

// Cv draws in logical units: device = o + logical*s.
type Cv struct {
	dst    *ebiten.Image
	s      float64
	ox, oy float64
}

func (c *Cv) X(x float64) float32 { return float32(c.ox + x*c.s) }
func (c *Cv) Y(y float64) float32 { return float32(c.oy + y*c.s) }
func (c *Cv) L(v float64) float32 { return float32(v * c.s) }

var whiteImg = func() *ebiten.Image {
	im := ebiten.NewImage(3, 3)
	im.Fill(color.White)
	return im
}()
var whiteSub = whiteImg.SubImage(whiteImg.Bounds().Inset(1)).(*ebiten.Image)

// fillPath fills a device-space path with a solid colour, or a vertical
// gradient from c0 at device y0 to c1 at y1.
func fillPath(dst *ebiten.Image, p *vector.Path, c0, c1 color.NRGBA, y0, y1 float32) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	colorVerts(vs, c0, c1, y0, y1)
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleNonZero})
}

func strokePath(dst *ebiten.Image, p *vector.Path, w float32, col color.NRGBA) {
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: w, LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound})
	colorVerts(vs, col, col, 0, 1)
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleFillAll})
}

func colorVerts(vs []ebiten.Vertex, c0, c1 color.NRGBA, y0, y1 float32) {
	for i := range vs {
		t := float32(0)
		if y1 != y0 {
			t = (vs[i].DstY - y0) / (y1 - y0)
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
		}
		a := (float32(c0.A) + (float32(c1.A)-float32(c0.A))*t) / 255
		vs[i].ColorR = (float32(c0.R) + (float32(c1.R)-float32(c0.R))*t) / 255 * a
		vs[i].ColorG = (float32(c0.G) + (float32(c1.G)-float32(c0.G))*t) / 255 * a
		vs[i].ColorB = (float32(c0.B) + (float32(c1.B)-float32(c0.B))*t) / 255 * a
		vs[i].ColorA = a
		vs[i].SrcX, vs[i].SrcY = 1.5, 1.5
	}
}

// rrPath: a rounded rectangle in device space.
func rrPath(x, y, w, h, r float32) *vector.Path {
	p := &vector.Path{}
	r = float32(math.Min(float64(r), math.Min(float64(w), float64(h))/2))
	if r <= .5 {
		p.MoveTo(x, y)
		p.LineTo(x+w, y)
		p.LineTo(x+w, y+h)
		p.LineTo(x, y+h)
		p.Close()
		return p
	}
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w, y, x+w, y+r, r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w, y+h, x+w-r, y+h, r)
	p.LineTo(x+r, y+h)
	p.ArcTo(x, y+h, x, y+h-r, r)
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r)
	p.Close()
	return p
}

func (c *Cv) RRect(x, y, w, h, r float64, col color.NRGBA) {
	fillPath(c.dst, rrPath(c.X(x), c.Y(y), c.L(w), c.L(h), c.L(r)), col, col, 0, 1)
}
func (c *Cv) RRectGrad(x, y, w, h, r float64, top, bot color.NRGBA) {
	fillPath(c.dst, rrPath(c.X(x), c.Y(y), c.L(w), c.L(h), c.L(r)), top, bot, c.Y(y), c.Y(y+h))
}
func (c *Cv) RRectStroke(x, y, w, h, r, lw float64, col color.NRGBA) {
	strokePath(c.dst, rrPath(c.X(x), c.Y(y), c.L(w), c.L(h), c.L(r)), c.L(lw), col)
}
func (c *Cv) Rect(x, y, w, h float64, col color.NRGBA) {
	vector.DrawFilledRect(c.dst, c.X(x), c.Y(y), c.L(w), c.L(h), col, true)
}
func (c *Cv) Circle(cx, cy, r float64, col color.NRGBA) {
	vector.DrawFilledCircle(c.dst, c.X(cx), c.Y(cy), c.L(r), col, true)
}
func (c *Cv) CircleStroke(cx, cy, r, lw float64, col color.NRGBA) {
	vector.StrokeCircle(c.dst, c.X(cx), c.Y(cy), c.L(r), c.L(lw), col, true)
}
func (c *Cv) Line(x0, y0, x1, y1, lw float64, col color.NRGBA) {
	vector.StrokeLine(c.dst, c.X(x0), c.Y(y0), c.X(x1), c.Y(y1), c.L(lw), col, true)
}

// Glow: soft light made of stacked translucent circles.
func (c *Cv) Glow(cx, cy, r float64, col color.NRGBA, strength float64) {
	n := 32
	for i := n; i >= 1; i-- {
		t := float64(i) / float64(n)
		c.Circle(cx, cy, r*t, withA(col, strength*(1-t)*(1-t)*.16))
	}
}

// Poly fills a polygon given as logical x,y pairs.
func (c *Cv) Poly(col color.NRGBA, pts ...float64) {
	p := &vector.Path{}
	for i := 0; i+1 < len(pts); i += 2 {
		if i == 0 {
			p.MoveTo(c.X(pts[0]), c.Y(pts[1]))
		} else {
			p.LineTo(c.X(pts[i]), c.Y(pts[i+1]))
		}
	}
	p.Close()
	fillPath(c.dst, p, col, col, 0, 1)
}

// LP builds a logical-space path.
type LP struct {
	c *Cv
	p vector.Path
}

func (c *Cv) Path() *LP                         { return &LP{c: c} }
func (l *LP) M(x, y float64) *LP                { l.p.MoveTo(l.c.X(x), l.c.Y(y)); return l }
func (l *LP) L(x, y float64) *LP                { l.p.LineTo(l.c.X(x), l.c.Y(y)); return l }
func (l *LP) Q(cx, cy, x, y float64) *LP        { l.p.QuadTo(l.c.X(cx), l.c.Y(cy), l.c.X(x), l.c.Y(y)); return l }
func (l *LP) C(a, b, cc, d, x, y float64) *LP   { l.p.CubicTo(l.c.X(a), l.c.Y(b), l.c.X(cc), l.c.Y(d), l.c.X(x), l.c.Y(y)); return l }
func (l *LP) Z() *LP                            { l.p.Close(); return l }
func (l *LP) Fill(col color.NRGBA)              { fillPath(l.c.dst, &l.p, col, col, 0, 1) }
func (l *LP) Stroke(w float64, col color.NRGBA) { strokePath(l.c.dst, &l.p, l.c.L(w), col) }
func (l *LP) Grad(top, bot color.NRGBA, y0, y1 float64) {
	fillPath(l.c.dst, &l.p, top, bot, l.c.Y(y0), l.c.Y(y1))
}

// ---------------------------------------------------------------- text

const (
	AlignLeft = iota
	AlignCenter
	AlignRight
)

func (c *Cv) Measure(s string, size float64, f FontID) float64 {
	w, _ := text.Measure(s, faceFor(f, size*c.s), 0)
	return w / c.s
}

// Text draws s with its baseline at y. Returns the width in logical units.
func (c *Cv) Text(s string, x, y, size float64, f FontID, col color.NRGBA, align int) float64 {
	fc := faceFor(f, size*c.s)
	w, _ := text.Measure(s, fc, 0)
	dx := float64(c.X(x))
	switch align {
	case AlignCenter:
		dx -= w / 2
	case AlignRight:
		dx -= w
	}
	m := fc.Metrics()
	op := &text.DrawOptions{}
	op.GeoM.Translate(math.Round(dx), math.Round(float64(c.Y(y))-m.HAscent))
	op.ColorScale.ScaleWithColor(col)
	text.Draw(c.dst, s, fc, op)
	return w / c.s
}

func (c *Cv) TextShadow(s string, x, y, size float64, f FontID, col, sh color.NRGBA, off float64, align int) float64 {
	c.Text(s, x+off, y+off, size, f, sh, align)
	return c.Text(s, x, y, size, f, col, align)
}

// Wrap splits s into lines no wider than maxW.
func (c *Cv) Wrap(s string, size float64, f FontID, maxW float64) []string {
	var lines []string
	for _, para := range splitLines(s) {
		words := splitWords(para)
		cur := ""
		for _, w := range words {
			try := w
			if cur != "" {
				try = cur + " " + w
			}
			if cur != "" && c.Measure(try, size, f) > maxW {
				lines = append(lines, cur)
				cur = w
			} else {
				cur = try
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

// Para draws wrapped text and returns the height used.
func (c *Cv) Para(s string, x, y, maxW, size, lineH float64, f FontID, col color.NRGBA, align int) float64 {
	lines := c.Wrap(s, size, f, maxW)
	for i, ln := range lines {
		ax := x
		if align == AlignCenter {
			ax = x + maxW/2
		}
		c.Text(ln, ax, y+float64(i)*lineH, size, f, col, align)
	}
	return float64(len(lines)) * lineH
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	return append(out, cur)
}

func splitWords(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// ---------------------------------------------------------------- icons

func (c *Cv) Star(cx, cy, r float64, col color.NRGBA) {
	p := c.Path()
	for i := 0; i < 10; i++ {
		a := -math.Pi/2 + float64(i)*math.Pi/5
		rr := r
		if i%2 == 1 {
			rr = r * .45
		}
		x, y := cx+rr*math.Cos(a), cy+rr*math.Sin(a)
		if i == 0 {
			p.M(x, y)
		} else {
			p.L(x, y)
		}
	}
	p.Z().Fill(col)
}

func (c *Cv) NoteIcon(cx, cy, s float64, col color.NRGBA) {
	c.Circle(cx-s*.35, cy+s*.45, s*.24, col)
	c.Circle(cx+s*.4, cy+s*.3, s*.24, col)
	c.Line(cx-s*.13, cy+s*.45, cx-s*.13, cy-s*.5, s*.12, col)
	c.Line(cx+s*.62, cy+s*.3, cx+s*.62, cy-s*.65, s*.12, col)
	c.Poly(col, cx-s*.19, cy-s*.5, cx+s*.68, cy-s*.72, cx+s*.68, cy-s*.48, cx-s*.19, cy-s*.26)
}

func (c *Cv) SpeakerIcon(cx, cy, s float64, col color.NRGBA) {
	c.Poly(col, cx-s*.6, cy-s*.22, cx-s*.3, cy-s*.22, cx+s*.05, cy-s*.55, cx+s*.05, cy+s*.55, cx-s*.3, cy+s*.22, cx-s*.6, cy+s*.22)
	for i, r := range []float64{.35, .62} {
		p := c.Path()
		for k := 0; k <= 8; k++ {
			a := -math.Pi/3.5 + float64(k)/8*(2*math.Pi/3.5)
			x, y := cx+s*.1+s*r*math.Cos(a), cy+s*r*math.Sin(a)
			if k == 0 {
				p.M(x, y)
			} else {
				p.L(x, y)
			}
		}
		_ = i
		p.Stroke(s*.1, col)
	}
}

func (c *Cv) GearIcon(cx, cy, s float64, col, hole color.NRGBA) {
	p := c.Path()
	n := 8
	for i := 0; i < n*4; i++ {
		a := float64(i) / float64(n*4) * 2 * math.Pi
		r := s * .62
		if (i/2)%2 == 1 {
			r = s * .45
		}
		x, y := cx+r*math.Cos(a), cy+r*math.Sin(a)
		if i == 0 {
			p.M(x, y)
		} else {
			p.L(x, y)
		}
	}
	p.Z().Fill(col)
	c.Circle(cx, cy, s*.2, hole)
}

func (c *Cv) Slash(cx, cy, s float64, col color.NRGBA) {
	c.Line(cx-s*.6, cy+s*.6, cx+s*.6, cy-s*.6, s*.13, col)
}
