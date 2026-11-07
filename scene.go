//go:build !headless

// The three garden scenes behind the board, painted once per window size,
// and their little living touches (fireflies, lantern glow, petals). (c) 2026 Ironite
package main

import (
	"image/color"
	"math"
	"math/rand"
)

var sceneNames = map[string]string{"moon": "Moon Garden", "lantern": "Lantern Festival", "dawn": "Plum Blossom Dawn"}

// paintScene fills the whole canvas. The canvas is set up so the 1600x900
// logical frame covers the window.
func paintScene(c *Cv, name string) {
	switch name {
	case "lantern":
		paintLantern(c)
	case "dawn":
		paintDawn(c)
	default:
		paintMoon(c)
	}
	// vignette: keeps the eye on the board
	c.RRectGrad(-400, 760, 2400, 600, 0, rgba(0x000000, 0), rgba(0x000000, .35))
}

func stars(c *Cv, n int, seed int64, maxY float64, col color.NRGBA) {
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < n; i++ {
		x, y := r.Float64()*2000-200, r.Float64()*maxY-200
		s := .6 + r.Float64()*1.6
		c.Circle(x, y, s, withA(col, .3+r.Float64()*.6))
	}
}

func ridge(c *Cv, seed int64, base, amp, rough float64, top, bot color.NRGBA) {
	r := rand.New(rand.NewSource(seed))
	p := c.Path().M(-400, 1300)
	ph := r.Float64() * 10
	for x := -400.0; x <= 2000; x += 20 {
		y := base - amp*(.55*math.Sin(x/310+ph)+.3*math.Sin(x/131+ph*2)+.15*math.Sin(x/57+ph*3)) - rough*(r.Float64()-.5)
		p.L(x, y)
	}
	p.L(2000, 1300).Z().Grad(top, bot, base-amp, base+200)
}

// pavilion: a silhouette with upswept eaves. (x, ground y, width)
func pavilion(c *Cv, x, gy, w float64, col color.NRGBA) {
	h := w * .9
	// platform and steps
	c.Rect(x-w*.08, gy-h*.08, w*1.16, h*.08, col)
	// pillars
	for _, f := range []float64{.08, .36, .64, .92} {
		c.Rect(x+w*f-w*.025, gy-h*.62, w*.05, h*.56, col)
	}
	// railing
	c.Rect(x, gy-h*.24, w, h*.03, col)
	for i := 0; i <= 12; i++ {
		c.Rect(x+w*float64(i)/12-1.5, gy-h*.24, 3, h*.16, col)
	}
	// lower roof
	roof := func(y0, ww, hh float64) {
		cx := x + w/2
		c.Path().M(cx-ww*.62, y0-hh*.35).
			Q(cx-ww*.5, y0+hh*.02, cx-ww*.36, y0).
			L(cx+ww*.36, y0).
			Q(cx+ww*.5, y0+hh*.02, cx+ww*.62, y0-hh*.35).
			Q(cx+ww*.45, y0-hh*.25, cx+ww*.3, y0-hh).
			L(cx-ww*.3, y0-hh).
			Q(cx-ww*.45, y0-hh*.25, cx-ww*.62, y0-hh*.35).Z().Fill(col)
	}
	roof(gy-h*.6, w*1.25, h*.28)
	c.Rect(x+w*.2, gy-h*.95, w*.6, h*.12, col)
	roof(gy-h*.9, w*.9, h*.26)
	c.Rect(x+w/2-3, gy-h*1.3, 6, h*.2, col)
	c.Circle(x+w/2, gy-h*1.3, 7, col)
}

func willow(c *Cv, x0, y0 float64, n int, seed int64, col color.NRGBA) {
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < n; i++ {
		x := x0 + r.Float64()*420
		l := 180 + r.Float64()*320
		sway := (r.Float64() - .3) * 60
		p := c.Path().M(x, y0)
		p.Q(x+sway, y0+l*.5, x+sway*.6, y0+l)
		p.Stroke(2.2, col)
		for t := .15; t < 1; t += .07 {
			bx := x + sway*2*t*(1-t) + sway*.6*t*t
			by := y0 + l*t
			leaf(c, bx, by, 16, 4, math.Pi/2+(r.Float64()-.5)*1.2, withA(col, .9))
		}
	}
}

func paintMoon(c *Cv) {
	c.RRectGrad(-400, -300, 2400, 760, 0, rgb(0x070d22), rgb(0x1a3550))
	c.RRectGrad(-400, 450, 2400, 300, 0, rgb(0x1a3550), rgb(0x2b5262))
	stars(c, 260, 3, 600, rgb(0xf3ecd2))
	// moon
	mx, my := 1290.0, 165.0
	c.Glow(mx, my, 420, rgb(0xf6e7b0), .55)
	c.Circle(mx, my, 82, rgb(0xfaf2d6))
	for _, cr := range [][3]float64{{-22, -18, 16}, {20, 12, 12}, {-6, 30, 9}, {30, -26, 7}} {
		c.Circle(mx+cr[0], my+cr[1], cr[2], rgba(0xd9cfa8, .45))
	}
	// mountains and mist
	ridge(c, 11, 520, 110, 6, rgb(0x223e55), rgb(0x1a3346))
	c.RRectGrad(-400, 520, 2400, 120, 0, rgba(0x9fb9c7, 0), rgba(0x9fb9c7, .18))
	ridge(c, 12, 610, 70, 4, rgb(0x172c3c), rgb(0x112230))
	// pond
	c.RRectGrad(-400, 690, 2400, 600, 0, rgb(0x16303c), rgb(0x08141a))
	for i := 0; i < 26; i++ { // moon on the water
		y := 705 + float64(i)*9
		w := 120 - float64(i)*3 + 30*math.Sin(float64(i)*1.7)
		c.RRect(mx-w/2+10*math.Sin(float64(i)), y, w, 3, 1.5, rgba(0xf6e7b0, .32-float64(i)*.011))
	}
	// lily pads
	r := rand.New(rand.NewSource(5))
	for i := 0; i < 14; i++ {
		x, y := r.Float64()*1800-100, 740+r.Float64()*200
		s := 18 + r.Float64()*26
		c.Path().M(x, y).L(x+s, y-s*.12).Q(x+s*1.1, y+s*.5, x, y+s*.42).Q(x-s*1.05, y+s*.3, x-s, y-s*.05).Z().Fill(rgba(0x1d4a35, .85))
		if i%4 == 0 {
			petals(c, x+s*.2, y+s*.05, s*.45, 6, rgba(0xf2b3c4, .9), rgb(0xf6d36b))
		}
	}
	pavilion(c, 40, 712, 330, rgb(0x0a161e))
	willow(c, -120, -20, 16, 8, rgb(0x0f2a1f))
	// far lanterns glowing on the water
	for _, l := range [][2]float64{{560, 760}, {980, 820}, {1450, 780}} {
		c.Glow(l[0], l[1], 50, rgb(0xffb35c), .8)
		c.RRect(l[0]-7, l[1]-9, 14, 18, 6, rgb(0xffcf86))
	}
}

func lanternShape(c *Cv, x, y, s float64, body color.NRGBA) {
	c.Glow(x, y, s*3.2, rgb(0xff8a3c), .55)
	c.Rect(x-s*.32, y-s*.75, s*.64, s*.16, rgb(0x2a1a10))
	c.Rect(x-s*.32, y+s*.6, s*.64, s*.16, rgb(0x2a1a10))
	c.Path().M(x-s*.35, y-s*.6).Q(x-s*1.05, y, x-s*.35, y+s*.6).L(x+s*.35, y+s*.6).Q(x+s*1.05, y, x+s*.35, y-s*.6).Z().
		Grad(mix(body, rgb(0xffd27a), .25), mix(body, cInk, .2), y-s*.6, y+s*.6)
	for _, f := range []float64{-.45, 0, .45} {
		c.Path().M(x+s*f*.75, y-s*.58).Q(x+s*f*1.6, y, x+s*f*.75, y+s*.58).Stroke(s*.05, withA(rgb(0x5a1010), .6))
	}
	c.Line(x, y+s*.76, x, y+s*1.25, s*.08, rgb(0xe3b65c))
	c.Circle(x, y+s*1.28, s*.09, rgb(0xe3b65c))
}

func paintLantern(c *Cv) {
	c.RRectGrad(-400, -300, 2400, 650, 0, rgb(0x1c0b2a), rgb(0x5c1f3e))
	c.RRectGrad(-400, 350, 2400, 450, 0, rgb(0x5c1f3e), rgb(0xc8683a))
	stars(c, 120, 21, 300, rgb(0xffe9c4))
	ridge(c, 31, 600, 60, 3, rgb(0x3a1530), rgb(0x2a0f22))
	// a pagoda against the glow
	px, gy := 1260.0, 640.0
	for i := 0; i < 5; i++ {
		w := 150 - float64(i)*22
		y := gy - float64(i)*58
		c.Rect(px-w*.32, y-44, w*.64, 44, rgb(0x1e0b18))
		c.Path().M(px-w*.62, y-38).Q(px-w*.4, y-48, px-w*.32, y-62).L(px+w*.32, y-62).Q(px+w*.4, y-48, px+w*.62, y-38).Z().Fill(rgb(0x1e0b18))
	}
	c.Rect(px-3, gy-330, 6, 50, rgb(0x1e0b18))
	// rooftops along the street
	r := rand.New(rand.NewSource(9))
	for x := -380.0; x < 2000; {
		w := 220 + r.Float64()*160
		h := 120 + r.Float64()*90
		y := 900 - h
		col := rgb(0x16080f)
		c.Rect(x+w*.08, y, w*.84, 600, col)
		c.Path().M(x-20, y+10).Q(x+w*.1, y-6, x+w*.2, y-40).L(x+w*.8, y-40).Q(x+w*.9, y-6, x+w+20, y+10).Z().Fill(col)
		// lit windows
		for k := 0; k < 3; k++ {
			if r.Float64() < .6 {
				wx, wy := x+w*(.22+.22*float64(k)), y+30+r.Float64()*30
				c.Glow(wx+12, wy+16, 40, rgb(0xffa34d), .6)
				c.RRect(wx, wy, 24, 32, 3, rgba(0xffc070, .85))
			}
		}
		x += w + 20
	}
	// strings of lanterns
	for row, yy := range []float64{40, 150} {
		sag := 70.0
		x0, x1 := -200.0, 1800.0
		p := c.Path().M(x0, yy).Q((x0+x1)/2, yy+sag*2, x1, yy)
		p.Stroke(2, rgba(0x1a0a0a, .9))
		n := 16
		for i := 1; i < n; i++ {
			t := float64(i) / float64(n)
			x := x0 + (x1-x0)*t
			y := yy + sag*2*2*t*(1-t)
			s := 22.0 - float64(row)*4
			if (i+row)%2 == 0 {
				c.Line(x, y, x, y+14, 1.5, rgba(0x1a0a0a, .9))
				lanternShape(c, x, y+14+s*.75, s, []color.NRGBA{rgb(0xd12f22), rgb(0xe0582a)}[(i/2+row)%2])
			}
		}
	}
}

func branch(c *Cv, pts [][2]float64, w float64, col color.NRGBA) {
	p := c.Path().M(pts[0][0], pts[0][1])
	for i := 1; i+1 < len(pts); i += 2 {
		p.Q(pts[i][0], pts[i][1], pts[i+1][0], pts[i+1][1])
	}
	p.Stroke(w, col)
}

func paintDawn(c *Cv) {
	c.RRectGrad(-400, -300, 2400, 700, 0, rgb(0x8fa9cf), rgb(0xf0c7b4))
	c.RRectGrad(-400, 400, 2400, 400, 0, rgb(0xf0c7b4), rgb(0xfbe3c4))
	c.Glow(380, 640, 520, rgb(0xfff1c8), .9)
	c.Circle(380, 640, 70, rgba(0xfff6dc, .9))
	ridge(c, 41, 560, 120, 4, rgba(0x9aa7c2, .85), rgba(0xb8bfd2, .85))
	c.RRectGrad(-400, 560, 2400, 140, 0, rgba(0xfdf0e2, 0), rgba(0xfdf0e2, .55))
	ridge(c, 42, 660, 80, 3, rgba(0x7d8aa8, .9), rgba(0x99a3bd, .9))
	c.RRectGrad(-400, 650, 2400, 120, 0, rgba(0xfdf0e2, 0), rgba(0xfdf0e2, .5))
	ridge(c, 43, 770, 50, 2, rgb(0x5f6c86), rgb(0x47526a))
	// garden wall with tiled coping
	c.Rect(-400, 800, 2400, 200, rgb(0xe9e0d2))
	c.Rect(-400, 790, 2400, 18, rgb(0x3f4a5c))
	for x := -400.0; x < 2000; x += 26 {
		c.RRect(x, 782, 22, 14, 6, rgb(0x4b576b))
	}
	// plum branches from the top right
	dark := rgb(0x3b2620)
	branch(c, [][2]float64{{1900, -40}, {1600, 60}, {1420, 40}, {1300, 30}, {1180, 120}}, 16, dark)
	branch(c, [][2]float64{{1560, 50}, {1520, 150}, {1440, 210}, {1400, 240}, {1330, 300}}, 9, dark)
	branch(c, [][2]float64{{1420, 40}, {1380, -10}, {1300, -30}}, 7, dark)
	branch(c, [][2]float64{{1700, 30}, {1720, 160}, {1650, 260}}, 8, dark)
	r := rand.New(rand.NewSource(77))
	for _, b := range [][2]float64{{1180, 120}, {1250, 70}, {1330, 300}, {1400, 230}, {1470, 180}, {1300, -20}, {1650, 260}, {1700, 150}, {1600, 70}, {1360, 35}, {1520, 120}} {
		for k := 0; k < 3; k++ {
			x, y := b[0]+(r.Float64()-.5)*50, b[1]+(r.Float64()-.5)*40
			s := 10 + r.Float64()*9
			petals(c, x, y, s, 5, []color.NRGBA{rgb(0xf3a5bd), rgb(0xe9849f), rgb(0xfac3d3)}[r.Intn(3)], rgb(0xb8344f))
		}
	}
}

// ---------------------------------------------------------------- ambience

type mote struct {
	x, y, vx, vy, ph, size, life float64
}

// ambient moves and draws the scene's small living things (logical units).
type ambient struct {
	scene string
	m     []mote
	rng   *rand.Rand
	t     float64
}

func (a *ambient) step(dt float64, scene string) {
	if a.rng == nil {
		a.rng = rand.New(rand.NewSource(1))
	}
	if scene != a.scene {
		a.scene, a.m = scene, nil
	}
	a.t += dt
	want := map[string]int{"moon": 18, "lantern": 10, "dawn": 22}[scene]
	for len(a.m) < want {
		m := mote{x: a.rng.Float64() * 1700, y: a.rng.Float64() * 900, ph: a.rng.Float64() * 6.28, size: .6 + a.rng.Float64()*.8}
		switch scene {
		case "moon":
			m.vx, m.vy = (a.rng.Float64()-.5)*14, (a.rng.Float64()-.5)*10
			m.y = 380 + a.rng.Float64()*480
		case "lantern": // sky lanterns drifting up
			m.vx, m.vy = 4+a.rng.Float64()*6, -(10 + a.rng.Float64()*12)
			m.y = 900 + a.rng.Float64()*500
		case "dawn": // petals falling
			m.vx, m.vy = -(12 + a.rng.Float64()*20), 22+a.rng.Float64()*20
			m.y = -a.rng.Float64() * 900
			m.x = 600 + a.rng.Float64()*1400
		}
		a.m = append(a.m, m)
	}
	for i := range a.m {
		m := &a.m[i]
		m.ph += dt
		m.x += (m.vx + 8*math.Sin(m.ph*.7)) * dt
		m.y += m.vy * dt
		if m.y < -150 || m.y > 1000 || m.x < -150 || m.x > 1800 {
			a.m[i] = a.m[len(a.m)-1]
			a.m = a.m[:len(a.m)-1]
			break
		}
	}
}

func (a *ambient) draw(c *Cv) {
	for _, m := range a.m {
		switch a.scene {
		case "moon":
			b := .5 + .5*math.Sin(m.ph*2.3)
			c.Glow(m.x, m.y, 16*m.size, rgb(0xd8f27a), .7*b)
			c.Circle(m.x, m.y, 1.8*m.size, withA(rgb(0xf4ffc0), .4+.6*b))
		case "lantern":
			s := 9 * m.size
			c.Glow(m.x, m.y, s*4, rgb(0xffa040), .5)
			c.Path().M(m.x-s*.7, m.y-s).L(m.x+s*.7, m.y-s).L(m.x+s*.5, m.y+s).L(m.x-s*.5, m.y+s).Z().Grad(rgb(0xffd890), rgb(0xf07a2a), m.y-s, m.y+s)
		case "dawn":
			s := 6 * m.size
			a := m.ph * 1.7
			leaf(c, m.x, m.y, s*2, s*.8, a, rgba(0xf3a5bd, .9))
		}
	}
}
