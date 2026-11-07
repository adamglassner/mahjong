//go:build !headless

// The tiles, drawn from scratch at the size they appear on screen: faces,
// two-tone bodies and the three tile sets. (c) 2026 Ironite
package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type TileStyle struct {
	name                 string
	face0, face1, band   color.NRGBA
	back0, back1, border color.NRGBA
}

var tileStyles = map[string]TileStyle{
	"ivory":    {"Ivory", rgb(0xfbf5e4), rgb(0xeee2c4), rgb(0xf1e6cb), rgb(0x2f8a63), rgb(0x1a5a3f), rgb(0xcdbd98)},
	"jade":     {"Jade", rgb(0xf1f8ef), rgb(0xd6e9d6), rgb(0xe1efdf), rgb(0x2b5d76), rgb(0x173a4d), rgb(0xa9c7ae)},
	"rosewood": {"Rosewood", rgb(0xfdf6e6), rgb(0xf0e0c0), rgb(0xf2e4c6), rgb(0x8a3424), rgb(0x55190f), rgb(0xd2b98e)},
}

// TileGeom is the size of a tile on screen in device pixels.
type TileGeom struct {
	W, H  float64 // face
	D     float64 // thickness (and the shift per layer)
	Pad   float64 // margin in the sprite around the tile
	Style string
	Marks bool
}

func (g TileGeom) spriteSize() (int, int) {
	return int(math.Ceil(g.W + g.D + 2*g.Pad)), int(math.Ceil(g.H + g.D + 2*g.Pad))
}

// renderTile draws one tile (body and face) into a new image.
func renderTile(k int, g TileGeom) *ebiten.Image {
	sw, sh := g.spriteSize()
	im := ebiten.NewImage(sw, sh)
	c := &Cv{dst: im, s: 1}
	st := tileStyles[g.Style]
	x, y, w, h, d := g.Pad, g.Pad, g.W, g.H, g.D
	r := w * .12
	// body: dark back, then the ivory band, then the face
	c.RRectGrad(x+d, y+d, w, h, r, st.back0, st.back1)
	c.RRectGrad(x+d*.42, y+d*.42, w, h, r, st.band, mix(st.band, st.back1, .25))
	c.RRectGrad(x, y, w, h, r, st.face0, st.face1)
	c.RRectStroke(x+.5, y+.5, w-1, h-1, r, math.Max(1, w*.012), withA(st.border, .9))
	// a soft highlight along the top edge
	c.RRectGrad(x+w*.06, y+h*.02, w*.88, h*.12, r*.6, rgba(0xffffff, .55), rgba(0xffffff, 0))
	if k >= 0 {
		drawFace(c, k, x, y, w, h, g.Marks)
	}
	return im
}

// renderBlank: a face-down tile (used while shuffling).
func renderBack(g TileGeom) *ebiten.Image {
	sw, sh := g.spriteSize()
	im := ebiten.NewImage(sw, sh)
	c := &Cv{dst: im, s: 1}
	st := tileStyles[g.Style]
	x, y, w, h, d := g.Pad, g.Pad, g.W, g.H, g.D
	r := w * .12
	c.RRectGrad(x+d, y+d, w, h, r, st.back1, mix(st.back1, cInk, .3))
	c.RRectGrad(x, y, w, h, r, st.back0, st.back1)
	c.RRectStroke(x+w*.12, y+h*.1, w*.76, h*.8, r*.6, w*.03, withA(cGold, .6))
	c.Circle(x+w/2, y+h/2, w*.14, withA(cGold, .5))
	return im
}

// ---------------------------------------------------------------- faces

func glyph(c *Cv, s string, cx, cy, size float64, col color.NRGBA) {
	c.Text(s, cx, cy+size*.38, size, FGlyph, col, AlignCenter)
}

var numerals = []string{"一", "二", "三", "四", "五", "六", "七", "八", "九"}

// drawFace paints tile kind k into the face rectangle.
func drawFace(c *Cv, k int, x, y, w, h float64, marks bool) {
	r := rankOf(k)
	switch suitOf(k) {
	case 0:
		drawDots(c, r, x, y, w, h)
	case 1:
		drawBamboo(c, r, x, y, w, h)
	case 2:
		glyph(c, numerals[r-1], x+w/2, y+h*.29, h*.36, rgb(0x1d2a52))
		glyph(c, "萬", x+w/2, y+h*.7, h*.4, cRed)
	case 3:
		glyph(c, []string{"東", "南", "西", "北"}[r-1], x+w/2, y+h*.53, h*.6, rgb(0x1b1a20))
	case 4:
		switch r {
		case 1:
			glyph(c, "中", x+w/2, y+h*.52, h*.66, cRed)
		case 2:
			glyph(c, "發", x+w/2, y+h*.52, h*.6, cGreen)
		default:
			c.RRectStroke(x+w*.17, y+h*.14, w*.66, h*.72, w*.04, w*.075, cBlue)
			c.RRectStroke(x+w*.29, y+h*.24, w*.42, h*.52, w*.02, w*.035, withA(cBlue, .8))
		}
	case 5:
		drawFlower(c, r, x, y, w, h)
		glyph(c, []string{"梅", "蘭", "菊", "竹"}[r-1], x+w/2, y+h*.76, h*.3, rgb(0x2a2440))
	case 6:
		drawSeason(c, r, x, y, w, h)
		glyph(c, []string{"春", "夏", "秋", "冬"}[r-1], x+w/2, y+h*.76, h*.3, rgb(0x2a2440))
	}
	if marks {
		mark, col := cornerMark(k)
		fs := h * .15
		c.Text(mark, x+w*.09, y+fs*.95+h*.02, fs, FBold, withA(col, .9), AlignLeft)
	}
}

func cornerMark(k int) (string, color.NRGBA) {
	r := rankOf(k)
	switch suitOf(k) {
	case 0:
		return itoa(r), cBlue
	case 1:
		return itoa(r), cGreen
	case 2:
		return itoa(r), cRed
	case 3:
		return []string{"E", "S", "W", "N"}[r-1], cInk
	case 4:
		return []string{"R", "G", "Wh"}[r-1], []color.NRGBA{cRed, cGreen, cBlue}[r-1]
	case 5:
		return "F", rgb(0xb0457a)
	}
	return "Se", rgb(0x3d6fa0)
}

var dotPos = [][][2]float64{
	{{.5, .5}},
	{{.5, .29}, {.5, .71}},
	{{.27, .22}, {.5, .5}, {.73, .78}},
	{{.3, .29}, {.7, .29}, {.3, .71}, {.7, .71}},
	{{.28, .24}, {.72, .24}, {.5, .5}, {.28, .76}, {.72, .76}},
	{{.3, .21}, {.7, .21}, {.3, .54}, {.7, .54}, {.3, .82}, {.7, .82}},
	{{.22, .15}, {.5, .26}, {.78, .37}, {.3, .62}, {.7, .62}, {.3, .85}, {.7, .85}},
	{{.3, .15}, {.7, .15}, {.3, .38}, {.7, .38}, {.3, .62}, {.7, .62}, {.3, .85}, {.7, .85}},
	{{.22, .19}, {.5, .19}, {.78, .19}, {.22, .5}, {.5, .5}, {.78, .5}, {.22, .81}, {.5, .81}, {.78, .81}},
}
var dotRad = []float64{.36, .19, .15, .17, .15, .14, .12, .12, .12}

func dotColor(r, i int) color.NRGBA {
	switch r {
	case 2:
		return []color.NRGBA{cGreen, cBlue}[i]
	case 3:
		return []color.NRGBA{cBlue, cRed, cGreen}[i]
	case 4:
		return []color.NRGBA{cBlue, cGreen, cGreen, cBlue}[i]
	case 5:
		return []color.NRGBA{cBlue, cGreen, cRed, cGreen, cBlue}[i]
	case 6:
		if i < 2 {
			return cGreen
		}
		return cRed
	case 7:
		if i < 3 {
			return cGreen
		}
		return cRed
	case 8:
		return cBlue
	case 9:
		return []color.NRGBA{cBlue, cRed, cGreen}[i/3]
	}
	return cBlue
}

func oneDot(c *Cv, cx, cy, rr float64, col color.NRGBA) {
	c.Circle(cx, cy, rr, col)
	c.Circle(cx, cy, rr*.74, cPaper)
	c.Circle(cx, cy, rr*.58, col)
	c.Circle(cx, cy, rr*.3, cPaper)
	c.Circle(cx, cy, rr*.16, mix(col, cInk, .3))
}

func drawDots(c *Cv, r int, x, y, w, h float64) {
	if r == 1 {
		cx, cy, R := x+w/2, y+h/2, w*.38
		c.Circle(cx, cy, R, cGreen)
		c.Circle(cx, cy, R*.86, cPaper)
		for i := 0; i < 12; i++ { // a ring of petals
			a := float64(i) / 12 * 2 * math.Pi
			c.Circle(cx+R*.7*math.Cos(a), cy+R*.7*math.Sin(a), R*.12, cRed)
		}
		c.Circle(cx, cy, R*.52, cBlue)
		c.Circle(cx, cy, R*.38, cPaper)
		c.Circle(cx, cy, R*.24, cRed)
		c.Circle(cx, cy, R*.1, cPaper)
		return
	}
	for i, p := range dotPos[r-1] {
		oneDot(c, x+p[0]*w, y+p[1]*h, dotRad[r-1]*w, dotColor(r, i))
	}
}

var bambooPos = [][][2]float64{
	nil,
	{{.5, .29}, {.5, .71}},
	{{.5, .29}, {.3, .71}, {.7, .71}},
	{{.32, .29}, {.68, .29}, {.32, .71}, {.68, .71}},
	{{.25, .29}, {.75, .29}, {.5, .5}, {.25, .71}, {.75, .71}},
	{{.22, .29}, {.5, .29}, {.78, .29}, {.22, .71}, {.5, .71}, {.78, .71}},
	{{.5, .18}, {.22, .5}, {.5, .5}, {.78, .5}, {.22, .82}, {.5, .82}, {.78, .82}},
	{{.17, .29}, {.39, .29}, {.61, .29}, {.83, .29}, {.17, .71}, {.39, .71}, {.61, .71}, {.83, .71}},
	{{.22, .18}, {.5, .18}, {.78, .18}, {.22, .5}, {.5, .5}, {.78, .5}, {.22, .82}, {.5, .82}, {.78, .82}},
}

func bambooColor(r, i int) color.NRGBA {
	switch {
	case r == 5 && i == 2, r == 7 && i == 0, r == 9 && i%3 == 1:
		return cRed
	case r == 8 && (i == 1 || i == 2 || i == 5 || i == 6):
		return cBlue
	}
	return cGreen
}

func stick(c *Cv, cx, cy, sw, sh float64, col color.NRGBA) {
	dark := mix(col, cInk, .35)
	lite := mix(col, cPaper, .35)
	c.RRectGrad(cx-sw/2, cy-sh/2, sw, sh, sw*.45, lite, dark)
	c.Line(cx-sw*.12, cy-sh*.38, cx-sw*.12, cy+sh*.38, sw*.18, withA(cPaper, .45))
	// joints: ends and middle
	for _, t := range []float64{-.5, 0, .5} {
		yy := cy + t*sh*.92
		c.RRect(cx-sw*.62, yy-sw*.16, sw*1.24, sw*.32, sw*.16, dark)
	}
}

func drawBamboo(c *Cv, r int, x, y, w, h float64) {
	if r == 1 {
		drawKoi(c, x, y, w, h)
		return
	}
	pos := bambooPos[r-1]
	sh := h * .34
	if r == 7 || r == 9 {
		sh = h * .26
	}
	sw := w * .12
	if r == 8 {
		sw = w * .11
	}
	for i, p := range pos {
		stick(c, x+p[0]*w, y+p[1]*h, sw, sh, bambooColor(r, i))
	}
}

// drawKoi: the One of Bamboo is a koi in this garden, where most sets have a bird.
func drawKoi(c *Cv, x, y, w, h float64) {
	cx, cy := x+w*.5, y+h*.5
	// ripples
	c.CircleStroke(cx, cy, w*.4, w*.02, rgba(0x6aa9c9, .55))
	c.CircleStroke(cx, cy, w*.3, w*.016, rgba(0x6aa9c9, .4))
	s := w * 1.05
	ang := -math.Pi / 2.6
	ca, sa := math.Cos(ang), math.Sin(ang)
	T := func(px, py float64) (float64, float64) {
		return cx + (px*ca-py*sa)*s, cy + (px*sa+py*ca)*s
	}
	path := func(pts [][2]float64, kind []byte) *LP {
		p := c.Path()
		i := 0
		for _, k := range kind {
			switch k {
			case 'M':
				a, b := T(pts[i][0], pts[i][1])
				p.M(a, b)
				i++
			case 'C':
				a, b := T(pts[i][0], pts[i][1])
				a2, b2 := T(pts[i+1][0], pts[i+1][1])
				a3, b3 := T(pts[i+2][0], pts[i+2][1])
				p.C(a, b, a2, b2, a3, b3)
				i += 3
			case 'Q':
				a, b := T(pts[i][0], pts[i][1])
				a2, b2 := T(pts[i+1][0], pts[i+1][1])
				p.Q(a, b, a2, b2)
				i += 2
			}
		}
		return p.Z()
	}
	orange, deep := rgb(0xee7a2c), rgb(0xc8401f)
	// tail and fins (behind the body)
	path([][2]float64{{-.28, 0}, {-.38, -.04}, {-.46, -.2}, {-.56, -.23}, {-.5, -.08}, {-.5, .08}, {-.56, .23}, {-.46, .2}, {-.38, .04}, {-.28, 0}}, []byte("MCCC")).Fill(withA(orange, .9))
	path([][2]float64{{.14, .1}, {.1, .3}, {-.04, .3}, {.04, .17}, {.08, .1}}, []byte("MQQ")).Fill(withA(deep, .85))
	path([][2]float64{{.14, -.1}, {.1, -.3}, {-.04, -.3}, {.04, -.17}, {.08, -.1}}, []byte("MQQ")).Fill(withA(deep, .85))
	// body
	path([][2]float64{{.44, 0}, {.42, -.15}, {.15, -.19}, {-.05, -.16}, {-.2, -.13}, {-.29, -.05}, {-.31, 0}, {-.29, .05}, {-.2, .13}, {-.05, .16}, {.15, .19}, {.42, .15}, {.44, 0}}, []byte("MCCCC")).Fill(orange)
	// patches
	px, py := T(.05, -.05)
	c.Circle(px, py, s*.09, cPaper)
	px, py = T(-.15, .04)
	c.Circle(px, py, s*.07, cPaper)
	px, py = T(.25, .05)
	c.Circle(px, py, s*.06, deep)
	px, py = T(-.05, .08)
	c.Circle(px, py, s*.045, deep)
	// eye and whiskers
	px, py = T(.34, -.06)
	c.Circle(px, py, s*.025, cInk)
	px, py = T(.34, .06)
	c.Circle(px, py, s*.025, cInk)
	a, b := T(.43, -.03)
	a2, b2 := T(.52, -.12)
	c.Line(a, b, a2, b2, s*.012, deep)
	a, b = T(.43, .03)
	a2, b2 = T(.52, .12)
	c.Line(a, b, a2, b2, s*.012, deep)
}

func petals(c *Cv, cx, cy, R float64, n int, col, mid color.NRGBA) {
	for i := 0; i < n; i++ {
		a := -math.Pi/2 + float64(i)/float64(n)*2*math.Pi
		c.Circle(cx+R*.55*math.Cos(a), cy+R*.55*math.Sin(a), R*.45, col)
	}
	c.Circle(cx, cy, R*.32, mid)
}

func leaf(c *Cv, cx, cy, l, wd, ang float64, col color.NRGBA) {
	ca, sa := math.Cos(ang), math.Sin(ang)
	P := func(u, v float64) (float64, float64) { return cx + u*ca - v*sa, cy + u*sa + v*ca }
	x0, y0 := P(0, 0)
	x1, y1 := P(l*.5, -wd)
	x2, y2 := P(l, 0)
	x3, y3 := P(l*.5, wd)
	c.Path().M(x0, y0).Q(x1, y1, x2, y2).Q(x3, y3, x0, y0).Z().Fill(col)
}

func drawFlower(c *Cv, r int, x, y, w, h float64) {
	cx, cy, R := x+w*.5, y+h*.33, w*.27
	switch r {
	case 1: // plum
		c.Line(cx-R*1.1, cy+R*.9, cx+R*.9, cy-R*.6, w*.035, rgb(0x5a3a2a))
		petals(c, cx, cy, R, 5, rgb(0xe9849f), rgb(0xf6d36b))
		petals(c, cx+R*.95, cy-R*.65, R*.45, 5, rgb(0xf2a8bc), rgb(0xf6d36b))
	case 2: // orchid
		for i, a := range []float64{-2.4, -1.57, -.74} {
			leaf(c, cx, cy+R*.2, R*1.1, R*.32, a, []color.NRGBA{rgb(0xa77bd0), rgb(0x8e58c0), rgb(0xa77bd0)}[i])
		}
		leaf(c, cx, cy+R*.2, R*.8, R*.28, 2.6, rgb(0x8e58c0))
		leaf(c, cx, cy+R*.2, R*.8, R*.28, .55, rgb(0x8e58c0))
		c.Circle(cx, cy+R*.2, R*.18, rgb(0xf6d36b))
	case 3: // chrysanthemum
		for i := 0; i < 16; i++ {
			a := float64(i) / 16 * 2 * math.Pi
			leaf(c, cx, cy, R*1.05, R*.17, a, rgb(0xe5a92a))
		}
		c.Circle(cx, cy, R*.32, rgb(0xc9701c))
	case 4: // bamboo sprig
		c.Line(cx-R*.2, cy+R*1.1, cx+R*.1, cy-R*1.1, w*.05, cGreen)
		for i, a := range []float64{-2.5, -.6, -2.9, -.3} {
			leaf(c, cx-R*.05+float64(i%2)*R*.05, cy-R*.3+float64(i/2)*R*.6, R*.95, R*.2, a, []color.NRGBA{rgb(0x3f9a5c), rgb(0x2e7d4a)}[i%2])
		}
	}
	c.Text(itoa(r), x+w*.86, y+h*.2, h*.13, FBold, withA(cRed, .85), AlignCenter)
}

func drawSeason(c *Cv, r int, x, y, w, h float64) {
	cx, cy, R := x+w*.5, y+h*.33, w*.27
	switch r {
	case 1: // spring blossom
		petals(c, cx, cy, R, 5, rgb(0xf4a6c0), rgb(0xd9466e))
		for i := 0; i < 5; i++ {
			a := -math.Pi/2 + float64(i)/5*2*math.Pi
			c.Circle(cx+R*.28*math.Cos(a), cy+R*.28*math.Sin(a), R*.06, rgb(0xfff2a8))
		}
	case 2: // summer sun over water
		for i := 0; i < 12; i++ {
			a := float64(i) / 12 * 2 * math.Pi
			c.Line(cx+R*.62*math.Cos(a), cy+R*.62*math.Sin(a), cx+R*1.02*math.Cos(a), cy+R*1.02*math.Sin(a), w*.035, rgb(0xf0a030))
		}
		c.Circle(cx, cy, R*.52, rgb(0xe25a24))
		c.Circle(cx, cy, R*.36, rgb(0xf08a34))
	case 3: // autumn maple leaf
		pts := []float64{}
		for i := 0; i < 10; i++ {
			a := -math.Pi/2 + float64(i)/10*2*math.Pi
			rr := R
			if i%2 == 1 {
				rr = R * .45
			}
			if i == 4 || i == 6 {
				rr *= .75
			}
			pts = append(pts, cx+rr*math.Cos(a), cy+rr*math.Sin(a))
		}
		c.Poly(rgb(0xd9531e), pts...)
		c.Line(cx, cy, cx, cy+R*1.1, w*.03, rgb(0x8a3214))
	case 4: // winter snowflake
		col := rgb(0x4b86c2)
		for i := 0; i < 6; i++ {
			a := float64(i) / 6 * 2 * math.Pi
			ex, ey := cx+R*math.Cos(a), cy+R*math.Sin(a)
			c.Line(cx, cy, ex, ey, w*.035, col)
			for _, s := range []float64{-1, 1} {
				mx, my := cx+R*.6*math.Cos(a), cy+R*.6*math.Sin(a)
				c.Line(mx, my, mx+R*.3*math.Cos(a+s*.8), my+R*.3*math.Sin(a+s*.8), w*.03, col)
			}
		}
		c.Circle(cx, cy, R*.13, col)
	}
	c.Text(itoa(r), x+w*.86, y+h*.2, h*.13, FBold, withA(cBlue, .85), AlignCenter)
}
