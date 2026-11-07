//go:build !headless

// The game screen, the title screen and every dialog. Immediate-mode: each
// frame draws everything and registers the clickable areas. (c) 2026 Ironite
package main

import (
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type hit struct {
	x, y, w, h float64
	id         string
	fn         func()
	modal      bool
}

type fly struct {
	face         int
	x, y, tx, ty float64 // device px: start and meeting point
	t            float64
	burst        bool
}

type particle struct {
	x, y, vx, vy, life, max, size, rot float64
	col                                color.NRGBA
	kind                               int // 0 sparkle, 1 petal, 2 lantern
}

type confirmInfo struct {
	title, text, yes, no string
	cb                   func(bool)
}

type App struct {
	sd    *SaveData
	synth *Synth
	sfx   SFX
	pl    *Play
	upd   *Updater

	screen string // "title" or "game"
	modal  string // "", puzzles, settings, rules, win, stuck, confirm, update
	conf   confirmInfo

	sel          int
	hint         [2]int
	hintT        float64
	shakeI       int
	shakeT       float64
	flies        []fly
	parts        []particle
	pops         map[int]float64
	streak       int
	sinceMatch   float64
	shuffleT     float64
	pendingWin   float64 // countdown until the win dialog
	pendingStuck float64
	win          struct {
		stars int
		time  float64
		best  bool
		name  string
	}
	saveT            float64
	toast            string
	toastT           float64
	tune             string
	tuneT            float64
	volDrag          bool
	rng              *rand.Rand
	quit             bool
	updateNoticeSeen bool

	// geometry
	W, H     int
	s        float64 // logical -> device
	ox, oy   float64
	geom     TileGeom
	hw, hh   float64
	bx, by   float64
	top      int
	sprites  map[int]*ebiten.Image
	backImg  *ebiten.Image
	spKey    string
	sceneImg *ebiten.Image
	scKey    string
	bigTiles map[int]*ebiten.Image
	bigKey   string

	hits, prev []hit
	hover      string
	mx, my     float64 // logical mouse
	amb        ambient
	now        float64
}

func NewApp(sd *SaveData, synth *Synth) *App {
	a := &App{sd: sd, synth: synth, sfx: SFX{synth}, screen: "title", sel: -1, hint: [2]int{-1, -1}, pops: map[int]float64{},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())), upd: newUpdater()}
	a.pl = RestorePlay(sd.Game)
	return a
}

// ---------------------------------------------------------------- game actions

func (a *App) startPuzzle(id string) {
	p := puzzleByID(id)
	if p == nil {
		return
	}
	a.pl = NewPlay(p, 0)
	a.sd.Last = id
	a.sd.Played++
	a.resetFX()
	a.screen, a.modal = "game", ""
	a.saveGame()
	a.sfx.Start()
	a.spKey = "" // new layout: re-measure the tiles
}

func (a *App) resetFX() {
	a.sel, a.hint, a.hintT = -1, [2]int{-1, -1}, 0
	a.flies, a.pops, a.streak = nil, map[int]float64{}, 0
	a.pendingWin, a.pendingStuck, a.shuffleT = 0, 0, 0
}

// askNew confirms before throwing away a game with progress.
func (a *App) askNew(id string) {
	if a.pl != nil && !a.pl.Won && len(a.pl.Hist) > 0 && a.screen == "game" {
		p := puzzleByID(id)
		a.confirm("Start a new game?", "Your game of "+a.pl.P.Name+" will be lost. Start "+p.Name+" with a fresh deal?", "New Game", "Keep Playing", func(ok bool) {
			if ok {
				a.startPuzzle(id)
			}
		})
		return
	}
	a.startPuzzle(id)
}

func (a *App) confirm(title, text, yes, no string, cb func(bool)) {
	a.conf = confirmInfo{title, text, yes, no, cb}
	a.modal = "confirm"
}

func (a *App) saveGame() {
	if a.pl != nil && !a.pl.Won {
		a.sd.Game = a.pl.Save()
	} else {
		a.sd.Game = nil
	}
	a.sd.write(false)
}

func (a *App) tileCenterDev(i int) (float64, float64) {
	x, y := a.tileXY(i)
	return x + a.geom.W/2, y + a.geom.H/2
}

func (a *App) pan(i int) float64 {
	x, _ := a.tileCenterDev(i)
	return math.Max(-.8, math.Min(.8, (x/float64(a.W))*1.6-.8))
}

func (a *App) clickTile(i int) {
	pl := a.pl
	if pl == nil || pl.Won || a.shuffleT > 0 {
		return
	}
	if !pl.B.Free(i) {
		a.shakeI, a.shakeT = i, .35
		a.sfx.Blocked()
		if a.sel >= 0 {
			a.toastMsg("That tile is blocked: it needs a free left or right side and nothing on top.")
		}
		return
	}
	switch {
	case a.sel == i:
		a.sel = -1
		a.sfx.Deselect(a.pan(i))
	case a.sel >= 0 && pl.CanMatch(a.sel, i):
		a.doMatch(a.sel, i)
	case a.sel >= 0 && tilesMatch(pl.B.T[a.sel].Face, pl.B.T[i].Face) == false:
		a.sel = i
		a.sfx.Select(a.pan(i))
	default:
		a.sel = i
		a.sfx.Select(a.pan(i))
	}
}

func (a *App) doMatch(i, j int) {
	pl := a.pl
	fi, fj := pl.B.T[i].Face, pl.B.T[j].Face
	xi, yi := a.tileXY(i)
	xj, yj := a.tileXY(j)
	mx, my := (xi+xj)/2, (yi+yj)/2-a.geom.H*.25
	a.flies = append(a.flies, fly{face: fi, x: xi, y: yi, tx: mx - a.geom.W*.55, ty: my, burst: true}, fly{face: fj, x: xj, y: yj, tx: mx + a.geom.W*.55, ty: my})
	if a.sinceMatch < 4 {
		a.streak++
	} else {
		a.streak = 0
	}
	a.sinceMatch = 0
	a.sfx.Match(a.streak, (a.pan(i)+a.pan(j))/2)
	won := pl.Match(i, j)
	a.sel, a.hint, a.hintT = -1, [2]int{-1, -1}, 0
	if won {
		a.pendingWin = .9
		a.sd.Game = nil
		a.sd.write(false)
		return
	}
	a.saveGame()
	if pl.Stuck() {
		a.pendingStuck = .7
	}
}

func (a *App) undo() {
	if a.pl == nil || a.modal != "" && a.modal != "stuck" {
		return
	}
	if h, ok := a.pl.Undo(); ok {
		a.pops[h[0]], a.pops[h[1]] = .3, .3
		a.sel, a.hint, a.hintT, a.pendingStuck = -1, [2]int{-1, -1}, 0, 0
		a.sfx.Undo()
		if a.modal == "stuck" {
			a.modal = ""
		}
		a.saveGame()
	} else {
		a.sfx.Blocked()
	}
}

func (a *App) doHint() {
	if a.pl == nil || a.pl.Won {
		return
	}
	if h, ok := a.pl.Hint(); ok {
		a.hint, a.hintT, a.sel = h, 4, -1
		a.sfx.Hint()
		a.saveGame()
	} else {
		a.showStuck()
	}
}

func (a *App) doShuffle() {
	if a.pl == nil || a.pl.Won {
		return
	}
	if a.pl.Shuffle() {
		a.shuffleT = .7
		a.sel, a.hint, a.hintT, a.pendingStuck = -1, [2]int{-1, -1}, 0, 0
		a.sfx.Shuffle()
		if a.modal == "stuck" {
			a.modal = ""
		}
		a.saveGame()
	}
}

func (a *App) newDeal() {
	if a.pl == nil {
		a.startPuzzle(a.sd.Last)
		return
	}
	id := a.pl.P.ID
	if !a.pl.Won && len(a.pl.Hist) > 0 && a.modal != "stuck" {
		a.confirm("Deal again?", "Start "+a.pl.P.Name+" over with freshly dealt tiles?", "New Deal", "Keep Playing", func(ok bool) {
			if ok {
				a.startPuzzle(id)
			}
		})
		return
	}
	a.startPuzzle(id)
}

func (a *App) showStuck() {
	a.modal = "stuck"
	a.sfx.Stuck()
}

func (a *App) finishWin() {
	pl := a.pl
	a.win.stars, a.win.time, a.win.name = pl.Stars(), pl.Elapsed, pl.P.Name
	a.win.best = a.sd.record(pl.P.ID, pl.Elapsed, a.win.stars)
	a.sd.Game = nil
	a.sd.write(true)
	a.modal = "win"
	a.sfx.Win()
	for i := 0; i < 26; i++ { // sky lanterns rise
		a.parts = append(a.parts, particle{x: 200 + a.rng.Float64()*1200, y: 920 + a.rng.Float64()*200, vx: (a.rng.Float64() - .5) * 20, vy: -(60 + a.rng.Float64()*70),
			life: 0, max: 9 + a.rng.Float64()*4, size: 10 + a.rng.Float64()*10, kind: 2})
	}
	for i := 0; i < 60; i++ {
		a.parts = append(a.parts, particle{x: 800 + (a.rng.Float64()-.5)*500, y: 300, vx: (a.rng.Float64() - .5) * 500, vy: -a.rng.Float64()*300 - 50,
			max: 3 + a.rng.Float64()*2, size: 5 + a.rng.Float64()*5, rot: a.rng.Float64() * 6, kind: 1,
			col: []color.NRGBA{rgb(0xf3a5bd), rgb(0xe3b65c), rgb(0xfac3d3), rgb(0xf6efe0)}[a.rng.Intn(4)]})
	}
}

func (a *App) nextPuzzle() string {
	i := 0
	if a.pl != nil {
		i = puzzleIndex(a.pl.P.ID)
	}
	// the next puzzle not yet solved, else simply the next one
	for k := 1; k <= len(puzzles); k++ {
		p := puzzles[(i+k)%len(puzzles)]
		if r := a.sd.Records[p.ID]; r == nil || r.Wins == 0 {
			return p.ID
		}
	}
	return puzzles[(i+1)%len(puzzles)].ID
}

func (a *App) toastMsg(s string) { a.toast, a.toastT = s, 3.2 }

func (a *App) toggleMusic() {
	a.sd.Settings.Music = !a.sd.Settings.Music
	a.applyAudio()
}
func (a *App) toggleSfx() {
	a.sd.Settings.Sfx = !a.sd.Settings.Sfx
	a.applyAudio()
	a.sfx.Click()
}
func (a *App) applyAudio() {
	st := a.sd.Settings
	a.synth.SetLevels(st.Music, st.Sfx, st.Vol)
	a.sd.write(false)
}

// ---------------------------------------------------------------- per-frame update

func (a *App) step(dt float64) {
	a.now += dt
	a.amb.step(dt, a.sd.Settings.Scene)
	if a.pl != nil && a.screen == "game" && a.modal == "" && !a.pl.Won && ebiten.IsFocused() {
		a.pl.Elapsed += dt
		a.saveT += dt
		if a.saveT > 5 {
			a.saveT = 0
			a.saveGame()
		}
	}
	a.sinceMatch += dt
	if a.hintT > 0 {
		a.hintT -= dt
		if a.hintT <= 0 {
			a.hint = [2]int{-1, -1}
		}
	}
	if a.shakeT > 0 {
		a.shakeT -= dt
	}
	if a.shuffleT > 0 {
		a.shuffleT -= dt
	}
	if a.toastT > 0 {
		a.toastT -= dt
	}
	if a.tuneT > 0 {
		a.tuneT -= dt
	}
	for k, v := range a.pops {
		if v -= dt; v <= 0 {
			delete(a.pops, k)
		} else {
			a.pops[k] = v
		}
	}
	out := a.flies[:0]
	for _, f := range a.flies {
		f.t += dt
		if f.burst && f.t >= .3 {
			f.burst = false
			lx, ly := (f.tx+a.geom.W*1.05-a.ox)/a.s, (f.ty+a.geom.H/2-a.oy)/a.s
			for i := 0; i < 22; i++ {
				ang := a.rng.Float64() * 2 * math.Pi
				sp := 80 + a.rng.Float64()*220
				a.parts = append(a.parts, particle{x: lx, y: ly, vx: math.Cos(ang) * sp, vy: math.Sin(ang)*sp - 60, max: .6 + a.rng.Float64()*.5,
					size: 2 + a.rng.Float64()*3, col: []color.NRGBA{cGold, rgb(0xfff2c0), rgb(0xf3a5bd)}[a.rng.Intn(3)]})
			}
		}
		if f.t < .62 {
			out = append(out, f)
		}
	}
	a.flies = out
	po := a.parts[:0]
	for _, p := range a.parts {
		p.life += dt
		p.x += p.vx * dt
		p.y += p.vy * dt
		switch p.kind {
		case 0:
			p.vy += 300 * dt
			p.vx *= .97
		case 1:
			p.vy += 120 * dt
			p.vx *= .98
			p.rot += dt * 3
		case 2:
			p.vx += math.Sin(p.life*1.3+p.size) * 6 * dt
		}
		if p.life < p.max {
			po = append(po, p)
		}
	}
	a.parts = po
	if a.pendingWin > 0 {
		if a.pendingWin -= dt; a.pendingWin <= 0 {
			a.finishWin()
		}
	}
	if a.pendingStuck > 0 {
		if a.pendingStuck -= dt; a.pendingStuck <= 0 && a.pl != nil && a.pl.Stuck() && a.modal == "" {
			a.showStuck()
		}
	}
	// an update found in the background: offer it once, on the title screen
	if st := a.upd.snapshot(); st.State == "available" && !a.updateNoticeSeen && a.modal == "" && st.Latest != a.sd.Skipped {
		a.updateNoticeSeen = true
		a.modal = "update"
	}
}

// ---------------------------------------------------------------- geometry

func (a *App) layout(w, h int) {
	a.W, a.H = w, h
	a.s = math.Min(float64(w)/LW, float64(h)/LH)
	a.ox = (float64(w) - LW*a.s) / 2
	a.oy = (float64(h) - LH*a.s) / 2
	if a.pl == nil {
		return
	}
	bw, bh, top := a.pl.B.Bounds()
	a.top = top
	// board area (device px): between the bars, using the full window width
	ax0, ay0 := 24*a.s, a.oy+78*a.s
	ax1, ay1 := float64(w)-24*a.s, a.oy+(852-14)*a.s
	if a.oy > 0 {
		ay0, ay1 = 78*a.s+a.oy, a.oy+838*a.s
	}
	aw, ah := ax1-ax0, ay1-ay0
	ratio := 1.32
	depth := .3 // thickness relative to half a tile's width
	hw := math.Min(aw/(float64(bw)+depth*float64(top+1)), ah/(ratio*float64(bh)+depth*float64(top+1)))
	hw = math.Min(hw, 50*a.s)
	hh := hw * ratio
	d := hw * depth
	a.hw, a.hh = hw, hh
	totW := float64(bw)*hw + float64(top+1)*d
	totH := float64(bh)*hh + float64(top+1)*d
	a.bx = ax0 + (aw-totW)/2
	a.by = ay0 + (ah-totH)/2
	g := TileGeom{W: 2 * hw, H: 2 * hh, D: d, Pad: math.Ceil(2 + hw*.04), Style: a.sd.Settings.Tiles, Marks: a.sd.Settings.Corners}
	key := strings.Join([]string{itoa(int(g.W * 10)), itoa(int(g.H * 10)), g.Style, map[bool]string{true: "m", false: ""}[g.Marks]}, "/")
	a.geom = g
	if key != a.spKey {
		for _, im := range a.sprites {
			im.Deallocate()
		}
		if a.backImg != nil {
			a.backImg.Deallocate()
		}
		a.sprites = map[int]*ebiten.Image{}
		for k := 0; k < NumKinds; k++ {
			a.sprites[k] = renderTile(k, g)
		}
		a.backImg = renderBack(g)
		a.spKey = key
	}
}

// tileXY: device position of a tile's face (top-left).
func (a *App) tileXY(i int) (float64, float64) {
	t := a.pl.B.T[i]
	d := a.geom.D
	return a.bx + float64(a.top)*d + float64(t.X)*a.hw - float64(t.Z)*d, a.by + float64(a.top)*d + float64(t.Y)*a.hh - float64(t.Z)*d
}

// tileAt: the topmost tile under a device point.
func (a *App) tileAt(x, y float64) int {
	if a.pl == nil {
		return -1
	}
	b := a.pl.B
	for k := len(b.order) - 1; k >= 0; k-- {
		i := b.order[k]
		if b.T[i].Gone {
			continue
		}
		tx, ty := a.tileXY(i)
		if x >= tx && y >= ty && x < tx+a.geom.W+a.geom.D*.5 && y < ty+a.geom.H+a.geom.D*.5 {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------- drawing

func (a *App) cv(dst *ebiten.Image) *Cv { return &Cv{dst: dst, s: a.s, ox: a.ox, oy: a.oy} }

func (a *App) addHit(x, y, w, h float64, id string, fn func()) {
	a.hits = append(a.hits, hit{x, y, w, h, id, fn, a.modal != ""})
}

func (a *App) drawScene(dst *ebiten.Image) {
	key := a.sd.Settings.Scene + "/" + itoa(a.W) + "x" + itoa(a.H)
	if key != a.scKey {
		if a.sceneImg != nil {
			a.sceneImg.Deallocate()
		}
		a.sceneImg = ebiten.NewImage(a.W, a.H)
		s := math.Max(float64(a.W)/LW, float64(a.H)/LH)
		c := &Cv{dst: a.sceneImg, s: s, ox: (float64(a.W) - LW*s) / 2, oy: (float64(a.H) - LH*s) / 2}
		paintScene(c, a.sd.Settings.Scene)
		a.scKey = key
	}
	dst.DrawImage(a.sceneImg, nil)
	s := math.Max(float64(a.W)/LW, float64(a.H)/LH)
	a.amb.draw(&Cv{dst: dst, s: s, ox: (float64(a.W) - LW*s) / 2, oy: (float64(a.H) - LH*s) / 2})
}

func (a *App) Draw(dst *ebiten.Image) {
	a.prev, a.hits = a.hits, nil
	a.drawScene(dst)
	c := a.cv(dst)
	if a.screen == "game" && a.pl != nil {
		a.drawBoard(dst)
		a.drawBars(c)
	} else {
		a.drawTitle(c)
	}
	a.drawParticles(c, false)
	if a.modal != "" {
		c.Rect(-a.ox/a.s-10, -a.oy/a.s-10, float64(a.W)/a.s+20, float64(a.H)/a.s+20, rgba(0x05040a, .55))
		switch a.modal {
		case "puzzles":
			a.drawPuzzles(c)
		case "settings":
			a.drawSettings(c)
		case "rules":
			a.drawRules(c)
		case "win":
			a.drawParticles(c, true)
			a.drawWin(c)
		case "stuck":
			a.drawStuck(c)
		case "confirm":
			a.drawConfirm(c)
		case "update":
			a.drawUpdate(c)
		}
	}
	a.drawToasts(c)
}

func (a *App) drawBoard(dst *ebiten.Image) {
	b := a.pl.B
	g := a.geom
	pulse := .5 + .5*math.Sin(a.now*6)
	shadow := rgba(0x000000, .26)
	for _, i := range b.order {
		t := b.T[i]
		if t.Gone {
			continue
		}
		x, y := a.tileXY(i)
		if i == a.shakeI && a.shakeT > 0 {
			x += math.Sin(a.shakeT*60) * g.W * .06 * (a.shakeT / .35)
		}
		lift := 0.0
		if i == a.sel {
			lift = g.D * .35
		}
		// shadow onto whatever is below
		fillPath(dst, rrPath(float32(x+g.D*.9), float32(y+g.D*1.1), float32(g.W+g.D*.3), float32(g.H+g.D*.3), float32(g.W*.14)), shadow, shadow, 0, 1)
		img := a.sprites[t.Face]
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		sx := 1.0
		if a.shuffleT > 0 {
			p := 1 - a.shuffleT/.7
			sx = math.Abs(math.Cos(2 * math.Pi * p))
			if p > .25 && p < .75 {
				img = a.backImg
			}
		}
		sc := 1.0
		if v, ok := a.pops[i]; ok {
			sc = 1 + .25*(v/.3)
		}
		op.GeoM.Translate(-g.Pad-g.W/2, -g.Pad-g.H/2)
		op.GeoM.Scale(sx*sc, sc)
		op.GeoM.Translate(x+g.W/2, y+g.H/2-lift)
		free := b.Free(i)
		// lower layers sit a little in shadow, so the stacks read clearly
		sh := float32(1 - .075*float64(a.top-t.Z))
		if a.sd.Settings.Highlight && !free {
			sh *= .66
		}
		op.ColorScale.Scale(sh, sh, sh*1.02, 1)
		dst.DrawImage(img, op)
		c := &Cv{dst: dst, s: 1}
		if i == a.sel {
			c.RRect(x, y-lift, g.W, g.H, g.W*.12, rgba(0xf2c650, .28))
			c.RRectStroke(x-1, y-lift-1, g.W+2, g.H+2, g.W*.13, math.Max(2, g.W*.05), rgb(0xf2c650))
		}
		if i == a.hint[0] || i == a.hint[1] {
			c.RRectStroke(x-2, y-2, g.W+4, g.H+4, g.W*.14, math.Max(2, g.W*.06), rgba(0x6ff0c0, .55+.45*pulse))
			c.RRect(x, y, g.W, g.H, g.W*.12, rgba(0x6ff0c0, .12+.12*pulse))
		}
	}
	// matched tiles gliding together and fading
	for _, f := range a.flies {
		p := math.Min(1, f.t/.3)
		e := 1 - (1-p)*(1-p)
		x := f.x + (f.tx-f.x)*e
		y := f.y + (f.ty-f.y)*e
		alpha, sc := 1.0, 1.0
		if f.t > .3 {
			q := (f.t - .3) / .32
			alpha, sc = 1-q, 1+.25*q
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(-g.Pad-g.W/2, -g.Pad-g.H/2)
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(x+g.W/2, y+g.H/2)
		op.ColorScale.ScaleAlpha(float32(alpha))
		dst.DrawImage(a.sprites[f.face], op)
	}
}

// drawParticles: with onlyWin, just the celebration (drawn above the win dialog's veil).
func (a *App) drawParticles(c *Cv, onlyWin bool) {
	for _, p := range a.parts {
		if onlyWin && p.kind == 0 || !onlyWin && a.modal == "win" && p.kind != 0 {
			continue
		}
		fade := math.Min(1, (p.max-p.life)/math.Min(.5, p.max*.4))
		switch p.kind {
		case 0:
			c.Circle(p.x, p.y, p.size*fade, withA(p.col, fade))
		case 1:
			leaf(c, p.x, p.y, p.size*2, p.size*.8, p.rot, withA(p.col, fade))
		case 2:
			s := p.size
			c.Glow(p.x, p.y, s*4, rgb(0xffa040), .6*fade)
			c.Path().M(p.x-s*.7, p.y-s).L(p.x+s*.7, p.y-s).L(p.x+s*.5, p.y+s).L(p.x-s*.5, p.y+s).Z().Grad(withA(rgb(0xffd890), fade), withA(rgb(0xf07a2a), fade), p.y-s, p.y+s)
		}
	}
}

// button draws a lacquer pill button and registers it.
func (a *App) button(c *Cv, x, y, w, h float64, label, id string, enabled, primary bool, fn func()) {
	hov := a.hover == id && enabled
	top, bot := rgb(0x3a2a2a), rgb(0x241818)
	if primary {
		top, bot = rgb(0xb02a24), rgb(0x7a1616)
	}
	if hov {
		top, bot = mix(top, cGold, .25), mix(bot, cGold, .15)
	}
	if !enabled {
		top, bot = withA(top, .5), withA(bot, .5)
	}
	c.RRect(x, y+2, w, h, h/2, rgba(0x000000, .35))
	c.RRectGrad(x, y, w, h, h/2, top, bot)
	edge := withA(cGold, .55)
	if hov {
		edge = cGold
	}
	c.RRectStroke(x+.5, y+.5, w-1, h-1, h/2, 1.4, edge)
	col := cPaper
	if !enabled {
		col = withA(cMuted, .5)
	}
	size := math.Min(18, h*.44)
	c.Text(label, x+w/2, y+h/2+size*.36, size, FBold, col, AlignCenter)
	if enabled {
		a.addHit(x, y, w, h, id, func() { a.sfx.Click(); fn() })
	}
}

func (a *App) iconButton(c *Cv, cx, cy, r float64, id string, on bool, icon func(cx, cy, s float64, col color.NRGBA), fn func()) {
	hov := a.hover == id
	c.Circle(cx, cy+2, r, rgba(0x000000, .35))
	bg := rgb(0x2c2020)
	if hov {
		bg = mix(bg, cGold, .25)
	}
	c.Circle(cx, cy, r, bg)
	c.CircleStroke(cx, cy, r-.5, 1.4, withA(cGold, map[bool]float64{true: 1, false: .55}[hov]))
	col := cPaper
	if !on {
		col = withA(cMuted, .55)
	}
	icon(cx, cy, r*.62, col)
	if !on {
		c.Slash(cx, cy, r*.62, rgb(0xe0605a))
	}
	a.addHit(cx-r, cy-r, 2*r, 2*r, id, fn)
}

func (a *App) drawBars(c *Cv) {
	pl := a.pl
	left, right := -a.ox/a.s, LW+a.ox/a.s
	// top bar
	c.RRectGrad(left, 0, right-left, 64, 0, rgba(0x1c1418, .9), rgba(0x120c10, .9))
	c.Rect(left, 63, right-left, 2, withA(cGold, .6))
	miniTile(c, 28, 10, 34, 44)
	c.TextShadow("Mahjong", 76, 41, 30, FTitle, cGold, rgba(0, .6), 2, AlignLeft)
	c.Text("Jade Moon Garden", 76+c.Measure("Mahjong", 30, FTitle)+14, 40, 16, FItalic, cMuted, AlignLeft)

	x := LW - 24.0
	a.iconButton(c, x-20, 32, 20, "settings", true, func(cx, cy, s float64, col color.NRGBA) { c.GearIcon(cx, cy, s, col, rgb(0x2c2020)) }, func() { a.sfx.Click(); a.modal = "settings" })
	x -= 50
	a.iconButton(c, x-20, 32, 20, "music", a.sd.Settings.Music, c.NoteIcon, a.toggleMusic)
	x -= 50
	a.iconButton(c, x-20, 32, 20, "sfx", a.sd.Settings.Sfx, c.SpeakerIcon, a.toggleSfx)
	x -= 62
	type btn struct {
		label, id string
		en        bool
		fn        func()
	}
	busy := pl.Won || a.pendingWin > 0
	for _, b := range []btn{
		{"Shuffle", "shuffle", !busy, a.doShuffle},
		{"Hint", "hint", !busy, a.doHint},
		{"Undo", "undo", len(pl.Hist) > 0 && !busy, a.undo},
		{"New Deal", "deal", true, a.newDeal},
		{"Puzzles", "puzzles", true, func() { a.modal = "puzzles" }},
	} {
		w := c.Measure(b.label, 17, FBold) + 40
		x -= w
		a.button(c, x, 12, w, 40, b.label, b.id, b.en, false, b.fn)
		x -= 10
	}

	// bottom bar
	c.RRectGrad(left, 852, right-left, 48, 0, rgba(0x1c1418, .86), rgba(0x120c10, .92))
	c.Rect(left, 852, right-left, 1.5, withA(cGold, .45))
	c.Text(pl.P.Name, 28, 884, 21, FTitle, cPaper, AlignLeft)
	nx := 28 + c.Measure(pl.P.Name, 21, FTitle) + 14
	diffBadge(c, nx, 864, pl.P.Diff)
	n := pl.B.Remaining()
	open := len(pl.B.OpenPairs())
	mid := "Tiles left " + itoa(n) + "      Open pairs " + itoa(open)
	col := cPaper
	if open == 0 && !pl.Won {
		col = rgb(0xff9a8a)
	}
	c.Text(mid, LW/2, 883, 17, FUI, col, AlignCenter)
	tm := fmtTime(pl.Elapsed)
	c.Text(tm, LW-28, 884, 22, FBold, cGold, AlignRight)
	clockIcon(c, LW-38-c.Measure(tm, 22, FBold), 876, 9, cGold)
	extra := ""
	if pl.Hints > 0 || pl.Shuffles > 0 {
		extra = "Hints " + itoa(pl.Hints) + "  ·  Shuffles " + itoa(pl.Shuffles)
	}
	if r := a.sd.Records[pl.P.ID]; r != nil && r.Best > 0 {
		if extra != "" {
			extra += "  ·  "
		}
		extra += "Best " + fmtTime(r.Best)
	}
	c.Text(extra, LW-80-c.Measure(tm, 22, FBold), 883, 15, FUI, cMuted, AlignRight)
}

func clockIcon(c *Cv, cx, cy, r float64, col color.NRGBA) {
	c.CircleStroke(cx, cy, r, 2, col)
	c.Line(cx, cy, cx, cy-r*.6, 2, col)
	c.Line(cx, cy, cx+r*.45, cy, 2, col)
}

func diffColor(d string) color.NRGBA {
	switch d {
	case Easy:
		return rgb(0x3f9d6e)
	case Medium:
		return rgb(0xc8902e)
	}
	return rgb(0xb83a30)
}

func diffBadge(c *Cv, x, y float64, d string) float64 {
	w := c.Measure(d, 14, FBold) + 22
	c.RRect(x, y, w, 26, 13, diffColor(d))
	c.Text(d, x+w/2, y+18, 14, FBold, cPaper, AlignCenter)
	return w
}

// miniTile: a small Red Dragon tile used as the game's mark.
func miniTile(c *Cv, x, y, w, h float64) {
	d := w * .14
	c.RRect(x+d, y+d, w, h, w*.14, rgb(0x2f8a63))
	c.RRect(x+d*.45, y+d*.45, w, h, w*.14, rgb(0xf1e6cb))
	c.RRectGrad(x, y, w, h, w*.14, rgb(0xfbf5e4), rgb(0xeee2c4))
	glyph(c, "中", x+w/2, y+h*.52, h*.62, cRed)
}

func (a *App) drawToasts(c *Cv) {
	if a.toastT > 0 && a.toast != "" {
		al := math.Min(1, a.toastT/.4)
		w := c.Measure(a.toast, 17, FUI) + 48
		y := 80.0
		if a.screen != "game" {
			y = 30
		}
		c.RRect(LW/2-w/2, y, w, 40, 20, rgba(0x120c10, .88*al))
		c.RRectStroke(LW/2-w/2, y, w, 40, 20, 1.2, withA(cGold, .6*al))
		c.Text(a.toast, LW/2, y+26, 17, FUI, withA(cPaper, al), AlignCenter)
	}
	if a.tuneT > 0 && a.tune != "" {
		al := math.Min(1, math.Min(a.tuneT/.6, (6-a.tuneT)/.6))
		y := 812.0
		if a.screen != "game" {
			y = 850
		}
		w := c.Measure(a.tune, 15, FItalic) + 56
		c.RRect(20, y, w, 30, 15, rgba(0x120c10, .7*al))
		c.NoteIcon(40, y+15, 9, withA(cGold, al))
		c.Text(a.tune, 56, y+20, 15, FItalic, withA(cPaper, al), AlignLeft)
	}
}

func (a *App) nowPlaying(name string) { a.tune, a.tuneT = name, 6 }

// panel draws a dialog panel and returns its inner origin.
func panel(c *Cv, w, h float64) (float64, float64) {
	x, y := (LW-w)/2, (LH-h)/2
	c.RRect(x+4, y+8, w, h, 22, rgba(0x000000, .45))
	c.RRectGrad(x, y, w, h, 22, rgba(0x2a1d22, .97), rgba(0x170f13, .97))
	c.RRectStroke(x+1, y+1, w-2, h-2, 21, 2, withA(cGold, .75))
	c.RRectStroke(x+8, y+8, w-16, h-16, 16, 1, withA(cGold, .25))
	return x, y
}

func (a *App) closeX(c *Cv, x, y float64, fn func()) {
	hov := a.hover == "close"
	col := withA(cPaper, .7)
	if hov {
		col = cGold
	}
	c.Line(x-9, y-9, x+9, y+9, 2.6, col)
	c.Line(x-9, y+9, x+9, y-9, 2.6, col)
	a.addHit(x-20, y-20, 40, 40, "close", func() { a.sfx.Click(); fn() })
}

func heading(c *Cv, s string, cx, y float64) {
	c.TextShadow(s, cx, y, 40, FTitle, cGold, rgba(0, .6), 2, AlignCenter)
	w := c.Measure(s, 40, FTitle)
	c.Line(cx-w/2-60, y-13, cx-w/2-16, y-13, 1.5, withA(cGold, .6))
	c.Line(cx+w/2+16, y-13, cx+w/2+60, y-13, 1.5, withA(cGold, .6))
	c.Circle(cx-w/2-66, y-13, 3, cGold)
	c.Circle(cx+w/2+66, y-13, 3, cGold)
}

// ---------------------------------------------------------------- title

func (a *App) bigTile(k int, w float64) *ebiten.Image {
	key := itoa(int(w*a.s)) + a.sd.Settings.Tiles
	if key != a.bigKey {
		for _, im := range a.bigTiles {
			im.Deallocate()
		}
		a.bigTiles, a.bigKey = map[int]*ebiten.Image{}, key
	}
	if im, ok := a.bigTiles[k]; ok {
		return im
	}
	g := TileGeom{W: w * a.s, H: w * 1.32 * a.s, D: w * .12 * a.s, Pad: 3, Style: a.sd.Settings.Tiles}
	im := renderTile(k, g)
	a.bigTiles[k] = im
	return im
}

func (a *App) drawBigTile(c *Cv, k int, x, y, w, rot float64) {
	im := a.bigTile(k, w)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	b := im.Bounds()
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(float64(c.X(x)), float64(c.Y(y)))
	// shadow
	sh := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	sh.GeoM = op.GeoM
	sh.GeoM.Translate(8*a.s, 12*a.s)
	sh.ColorScale.Scale(0, 0, 0, .35)
	c.dst.DrawImage(im, sh)
	c.dst.DrawImage(im, op)
}

func (a *App) drawTitle(c *Cv) {
	cx := LW / 2
	bob := math.Sin(a.now*1.2) * 4
	a.drawBigTile(c, KBamboo, cx-150, 170+bob, 108, -.22)
	a.drawBigTile(c, KDots, cx+150, 170-bob, 108, .22)
	a.drawBigTile(c, KDragon, cx, 150, 118, 0)
	c.TextShadow("Mahjong", cx, 350, 104, FTitle, cGold, rgba(0x000000, .7), 4, AlignCenter)
	c.TextShadow("Jade Moon Garden", cx, 400, 30, FItalic, cPaper, rgba(0x000000, .7), 2, AlignCenter)
	if a.screen == "cover" { // the browser build's cover-art mode: just the title
		return
	}
	y := 448.0
	w := 420.0
	if a.pl != nil && !a.pl.Won {
		pl := a.pl
		a.button(c, cx-w/2, y, w, 58, "Continue: "+pl.P.Name+" ("+itoa(pl.B.Remaining())+" tiles left)", "continue", true, true, func() {
			a.screen = "game"
			a.spKey = ""
		})
		y += 72
	}
	a.button(c, cx-w/2, y, w, 54, "Choose a Puzzle", "choose", true, a.pl == nil || a.pl.Won, func() { a.modal = "puzzles" })
	y += 68
	a.button(c, cx-w/2, y, w/2-7, 50, "How to Play", "rules", true, false, func() { a.modal = "rules" })
	a.button(c, cx+7, y, w/2-7, 50, "Settings", "settings2", true, false, func() { a.modal = "settings" })
	y += 66
	// progress
	solved, stars := 0, 0
	for _, p := range puzzles {
		if r := a.sd.Records[p.ID]; r != nil && r.Wins > 0 {
			solved++
			stars += r.Stars
		}
	}
	msg := "Twelve garden puzzles, from a gentle Lotus Pad to the Great Pagoda."
	if solved > 0 {
		msg = itoa(solved) + " of " + itoa(len(puzzles)) + " puzzles solved  ·  " + itoa(stars) + " of " + itoa(3*len(puzzles)) + " stars"
	}
	c.TextShadow(msg, cx, y+20, 18, FUI, cPaper, rgba(0, .7), 1.5, AlignCenter)
	foot := "Version " + Version + "  ·  An Ironite game"
	if st := a.upd.snapshot(); st.State == "available" {
		foot += "  ·  Version " + st.Latest + " is available"
		w := c.Measure("Update", 15, FBold) + 30
		a.button(c, cx+c.Measure(foot, 15, FUI)/2+12, LH-46, w, 30, "Update", "updnow", true, true, func() { a.modal = "update" })
	}
	c.TextShadow(foot, cx, LH-26, 15, FUI, withA(cPaper, .8), rgba(0, .7), 1.5, AlignCenter)
	a.iconButton(c, LW-50, 44, 22, "music", a.sd.Settings.Music, c.NoteIcon, a.toggleMusic)
	a.iconButton(c, LW-104, 44, 22, "sfx", a.sd.Settings.Sfx, c.SpeakerIcon, a.toggleSfx)
}

// ---------------------------------------------------------------- dialogs

// layoutPreview draws a puzzle's shape small, layer by layer.
func layoutPreview(c *Cv, p *Puzzle, x, y, w, h float64) {
	pos := p.Positions()
	bw, bh, top := 0, 0, 0
	for _, q := range pos {
		bw, bh, top = max(bw, q.X+2), max(bh, q.Y+2), max(top, q.Z)
	}
	u := math.Min(w/(float64(bw)+float64(top)*.6+.6), h/(float64(bh)*1.3+float64(top)*.6+.6))
	tw, th := 2*u, 2*u*1.3
	ox := x + (w-(float64(bw)*u+float64(top)*u*.6))/2 + float64(top)*u*.6
	oy := y + (h-(float64(bh)*u*1.3+float64(top)*u*.6))/2 + float64(top)*u*.6
	for z := 0; z <= top; z++ {
		shade := 1 - .12*float64(top-z)
		for _, q := range pos {
			if q.Z != z {
				continue
			}
			px, py := ox+float64(q.X)*u-float64(z)*u*.6, oy+float64(q.Y)*u*1.3-float64(z)*u*.6
			c.RRect(px+u*.25, py+u*.25, tw-u*.15, th-u*.15, u*.3, rgb(0x1f5a40))
			c.RRect(px, py, tw-u*.15, th-u*.15, u*.3, mix(rgb(0x8a7f68), rgb(0xf6efe0), shade))
		}
	}
}

func (a *App) drawPuzzles(c *Cv) {
	x0, y0 := panel(c, 1380, 780)
	heading(c, "Choose a Puzzle", LW/2, y0+66)
	a.closeX(c, x0+1380-36, y0+36, func() { a.modal = "" })
	colW := 430.0
	for di, d := range difficulties {
		cx := x0 + 30 + float64(di)*(colW+15)
		solved, total := 0, 0
		for _, p := range puzzles {
			if p.Diff == d {
				total++
				if r := a.sd.Records[p.ID]; r != nil && r.Wins > 0 {
					solved++
				}
			}
		}
		diffBadge(c, cx, y0+100, d)
		c.Text(itoa(solved)+" of "+itoa(total)+" solved", cx+colW, y0+119, 15, FUI, cMuted, AlignRight)
		k := 0
		for i := range puzzles {
			p := &puzzles[i]
			if p.Diff != d {
				continue
			}
			yy := y0 + 140 + float64(k)*150
			k++
			id := "pz:" + p.ID
			hov := a.hover == id
			bg := rgba(0xffffff, .05)
			if hov {
				bg = rgba(0xe3b65c, .16)
			}
			c.RRect(cx, yy, colW, 138, 14, bg)
			cur := a.pl != nil && !a.pl.Won && a.pl.P.ID == p.ID
			edge := withA(cGold, .25)
			if hov || cur {
				edge = withA(cGold, .8)
			}
			c.RRectStroke(cx, yy, colW, 138, 14, 1.3, edge)
			c.RRect(cx+10, yy+10, 150, 118, 10, rgba(0x000000, .25))
			layoutPreview(c, p, cx+16, yy+14, 138, 110)
			tx := cx + 176
			c.Text(p.Name, tx, yy+36, 23, FTitle, cPaper, AlignLeft)
			pos := p.Positions()
			c.Text(itoa(len(pos))+" tiles  ·  "+itoa(len(p.Layers))+" layers", tx, yy+60, 15, FUI, cMuted, AlignLeft)
			r := a.sd.Records[p.ID]
			for s := 0; s < 3; s++ {
				col := rgba(0xffffff, .15)
				if r != nil && r.Stars > s {
					col = cGold
				}
				c.Star(tx+12+float64(s)*26, yy+86, 11, col)
			}
			info := "Not solved yet"
			if r != nil && r.Wins > 0 {
				info = "Best " + fmtTime(r.Best)
			}
			if cur {
				info = "In progress  ·  " + itoa(a.pl.B.Remaining()) + " left"
			}
			c.Text(info, tx+90, yy+92, 15, FUI, cPaper, AlignLeft)
			c.Para(p.Blurb, tx, yy+116, colW-190, 13, 16, FItalic, withA(cMuted, .9), AlignLeft)
			pid := p.ID
			a.addHit(cx, yy, colW, 138, id, func() {
				a.sfx.Click()
				if cur {
					a.modal, a.screen = "", "game"
					return
				}
				a.askNew(pid)
			})
		}
	}
}

func (a *App) chips(c *Cv, x, y float64, id string, opts [][2]string, cur string, pick func(v string)) {
	for _, o := range opts {
		w := c.Measure(o[1], 16, FBold) + 32
		on := o[0] == cur
		hid := id + ":" + o[0]
		hov := a.hover == hid
		bg := rgba(0xffffff, .06)
		if on {
			bg = rgb(0x9a2620)
		} else if hov {
			bg = rgba(0xe3b65c, .2)
		}
		c.RRect(x, y, w, 36, 18, bg)
		c.RRectStroke(x, y, w, 36, 18, 1.2, withA(cGold, map[bool]float64{true: .9, false: .35}[on || hov]))
		c.Text(o[1], x+w/2, y+24, 16, FBold, cPaper, AlignCenter)
		v := o[0]
		a.addHit(x, y, w, 36, hid, func() { a.sfx.Click(); pick(v) })
		x += w + 10
	}
}

func onOff(b bool) string { return map[bool]string{true: "on", false: "off"}[b] }

func (a *App) drawSettings(c *Cv) {
	st := &a.sd.Settings
	W, H := 820.0, 700.0
	x0, y0 := panel(c, W, H)
	heading(c, "Settings", LW/2, y0+66)
	a.closeX(c, x0+W-36, y0+36, func() { a.modal = "" })
	lx, vx := x0+50, x0+250
	y := y0 + 110
	row := func(label string) {
		c.Text(label, lx, y+25, 18, FBold, cMuted, AlignLeft)
	}
	changed := func() { a.sd.write(false) }
	row("Music")
	a.chips(c, vx, y, "music", [][2]string{{"on", "On"}, {"off", "Off"}}, onOff(st.Music), func(v string) { st.Music = v == "on"; a.applyAudio() })
	y += 56
	row("Sound effects")
	a.chips(c, vx, y, "sfx", [][2]string{{"on", "On"}, {"off", "Off"}}, onOff(st.Sfx), func(v string) { st.Sfx = v == "on"; a.applyAudio() })
	y += 56
	row("Volume")
	// slider
	sx, sw := vx+8, 300.0
	c.RRect(sx, y+15, sw, 6, 3, rgba(0xffffff, .15))
	c.RRect(sx, y+15, sw*st.Vol/100, 6, 3, cGold)
	c.Circle(sx+sw*st.Vol/100, y+18, 11, cPaper)
	c.CircleStroke(sx+sw*st.Vol/100, y+18, 11, 2, cGold)
	c.Text(itoa(int(st.Vol))+"%", sx+sw+28, y+25, 17, FUI, cPaper, AlignLeft)
	a.addHit(sx-12, y, sw+24, 36, "vol", func() { a.volDrag = true })
	if a.volDrag {
		st.Vol = math.Round(math.Max(0, math.Min(100, (a.mx-sx)/sw*100)))
		a.synth.SetLevels(st.Music, st.Sfx, st.Vol)
	}
	y += 56
	row("Tiles")
	a.chips(c, vx, y, "tiles", [][2]string{{"ivory", "Ivory & Jade"}, {"jade", "Celadon"}, {"rosewood", "Rosewood"}}, st.Tiles, func(v string) { st.Tiles = v; changed() })
	y += 56
	row("Scene")
	a.chips(c, vx, y, "scene", [][2]string{{"moon", "Moon Garden"}, {"lantern", "Lantern Festival"}, {"dawn", "Plum Blossom Dawn"}}, st.Scene, func(v string) { st.Scene = v; changed() })
	y += 56
	row("Blocked tiles")
	a.chips(c, vx, y, "hl", [][2]string{{"on", "Shade them"}, {"off", "Don't shade"}}, onOff(st.Highlight), func(v string) { st.Highlight = v == "on"; changed() })
	y += 56
	row("Corner marks")
	a.chips(c, vx, y, "marks", [][2]string{{"on", "Show"}, {"off", "Hide"}}, onOff(st.Corners), func(v string) { st.Corners = v == "on"; changed() })
	y += 56
	row("Updates")
	us := a.upd.snapshot()
	if us.Managed {
		c.Text("Ironite Launcher keeps this copy up to date.", vx, y+24, 16, FUI, cPaper, AlignLeft)
	} else {
		a.chips(c, vx, y, "upd", [][2]string{{"on", "Check automatically"}, {"off", "Off"}}, onOff(st.Updates), func(v string) { st.Updates = v == "on"; changed() })
		y += 46
		label := "Check now"
		bx := vx
		a.button(c, bx, y, 140, 34, label, "checknow", us.State != "checking" && us.State != "downloading", false, func() {
			a.updateNoticeSeen = false
			a.sd.Skipped = ""
			go a.upd.Check(true)
		})
		c.Text(us.status(), bx+156, y+23, 15, FUI, cMuted, AlignLeft)
	}
	c.Text("Mahjong "+Version+"  ·  Music and sounds are played live by the game  ·  © 2026 Ironite", LW/2, y0+H-30, 14, FUI, withA(cMuted, .8), AlignCenter)
}

func (a *App) drawRules(c *Cv) {
	W, H := 1100.0, 760.0
	x0, y0 := panel(c, W, H)
	heading(c, "How to Play", LW/2, y0+66)
	a.closeX(c, x0+W-36, y0+36, func() { a.modal = "" })
	lx := x0 + 60
	y := y0 + 112
	tw := 560.0
	sec := func(title, body string) {
		c.Text(title, lx, y, 21, FTitle, cGold, AlignLeft)
		y += 10
		y += c.Para(body, lx, y+18, tw, 16, 23, FUI, cPaper, AlignLeft) + 36
	}
	sec("Clear the garden", "Remove every tile from the board, two matching tiles at a time. Click a tile, then click its twin.")
	sec("Free tiles only", "A tile can be picked when nothing is lying on top of it and its left or right side is open. Tiles in the middle of a row have to wait until one end is cleared.")
	sec("What matches", "Tiles match when they are exactly the same. The four Flowers all match one another, and so do the four Seasons.")
	sec("Help when you need it", "Hint shows a pair you can take. Shuffle mixes the tiles that are left (it never makes the board impossible). Undo takes back your last pair as often as you like.")
	sec("Stars", "Finish without hints or shuffles for three stars, with up to three for two stars. Undo is always free. Every deal can be cleared.")
	c.Text("Keys:  H hint   ·   U or Ctrl+Z undo   ·   S shuffle   ·   N new deal   ·   M music   ·   F11 full screen", LW/2, y0+H-34, 15, FUI, cMuted, AlignCenter)

	// the tile guide
	gx, gy := x0+680, y0+104
	c.RRect(gx-20, gy-6, 400, 560, 14, rgba(0x000000, .22))
	rowT := func(label string, kinds []int) {
		c.Text(label, gx, gy+16, 15, FBold, cMuted, AlignLeft)
		for i, k := range kinds {
			a.drawBigTile(c, k, gx+30+float64(i)*62, gy+68, 50, 0)
		}
		gy += 112
	}
	rowT("Dots (1 to 9)", []int{KDots, KDots + 2, KDots + 4, KDots + 8, KDots + 6})
	rowT("Bamboo (the One is a koi)", []int{KBamboo, KBamboo + 1, KBamboo + 4, KBamboo + 6, KBamboo + 8})
	rowT("Characters, Winds, Dragons", []int{KChar + 2, KChar + 7, KWind, KDragon, KDragon + 2})
	rowT("Any Flower matches any Flower", []int{KFlower, KFlower + 1, KFlower + 2, KFlower + 3})
	rowT("Any Season matches any Season", []int{KSeason, KSeason + 1, KSeason + 2, KSeason + 3})
}

func (a *App) drawWin(c *Cv) {
	W, H := 660.0, 500.0
	x0, y0 := panel(c, W, H)
	_ = x0
	heading(c, "Garden Cleared!", LW/2, y0+72)
	c.Text(a.win.name, LW/2, y0+112, 22, FItalic, cPaper, AlignCenter)
	for s := 0; s < 3; s++ {
		col := rgba(0xffffff, .14)
		if a.win.stars > s {
			col = cGold
		}
		r := 30.0
		if s == 1 {
			r = 38
		}
		c.Star(LW/2+float64(s-1)*90, y0+180-float64(s%2)*10, r, col)
	}
	c.Text("Time  "+fmtTime(a.win.time), LW/2, y0+262, 30, FBold, cPaper, AlignCenter)
	sub := ""
	if a.win.best {
		sub = "A new best time!"
	} else if r := a.sd.Records[a.pl.P.ID]; r != nil {
		sub = "Best  " + fmtTime(r.Best)
	}
	c.Text(sub, LW/2, y0+296, 19, FUI, cGold, AlignCenter)
	c.Text("Hints "+itoa(a.pl.Hints)+"   ·   Shuffles "+itoa(a.pl.Shuffles)+"   ·   Undos "+itoa(a.pl.Undos), LW/2, y0+330, 16, FUI, cMuted, AlignCenter)
	bw := 180.0
	bx := LW/2 - (3*bw+30)/2
	next := a.nextPuzzle()
	a.button(c, bx, y0+H-100, bw, 52, "Next Puzzle", "next", true, true, func() { a.startPuzzle(next) })
	a.button(c, bx+bw+15, y0+H-100, bw, 52, "Play Again", "again", true, false, func() { a.startPuzzle(a.pl.P.ID) })
	a.button(c, bx+2*bw+30, y0+H-100, bw, 52, "All Puzzles", "all", true, false, func() { a.modal = "puzzles" })
}

func (a *App) drawStuck(c *Cv) {
	W, H := 640.0, 340.0
	_, y0 := panel(c, W, H)
	heading(c, "No More Matches", LW/2, y0+70)
	c.Para("There are no free pairs left. Shuffle the remaining tiles (the board stays clearable), take back a move, or deal again.", (LW-520)/2, y0+122, 520, 18, 26, FUI, cPaper, AlignCenter)
	bw := 170.0
	bx := LW/2 - (3*bw+30)/2
	a.button(c, bx, y0+H-90, bw, 50, "Shuffle", "s_shuffle", true, true, a.doShuffle)
	a.button(c, bx+bw+15, y0+H-90, bw, 50, "Undo", "s_undo", len(a.pl.Hist) > 0, false, a.undo)
	a.button(c, bx+2*bw+30, y0+H-90, bw, 50, "New Deal", "s_deal", true, false, func() { a.modal = ""; a.startPuzzle(a.pl.P.ID) })
}

func (a *App) drawConfirm(c *Cv) {
	W, H := 620.0, 300.0
	_, y0 := panel(c, W, H)
	heading(c, a.conf.title, LW/2, y0+70)
	c.Para(a.conf.text, (LW-500)/2, y0+120, 500, 18, 26, FUI, cPaper, AlignCenter)
	bw := 200.0
	cb := a.conf.cb
	a.button(c, LW/2-bw-8, y0+H-86, bw, 50, a.conf.yes, "c_yes", true, true, func() { a.modal = ""; cb(true) })
	a.button(c, LW/2+8, y0+H-86, bw, 50, a.conf.no, "c_no", true, false, func() { a.modal = ""; cb(false) })
}

func (a *App) drawUpdate(c *Cv) {
	us := a.upd.snapshot()
	W, H := 680.0, 440.0
	x0, y0 := panel(c, W, H)
	heading(c, "Update Available", LW/2, y0+70)
	c.Text("Mahjong "+us.Latest+" is ready  (you have "+Version+")", LW/2, y0+116, 19, FBold, cPaper, AlignCenter)
	notes := us.Notes
	if len(notes) > 420 {
		notes = notes[:420] + "…"
	}
	c.RRect(x0+50, y0+140, W-100, 150, 12, rgba(0x000000, .25))
	c.Para(notes, x0+70, y0+170, W-140, 15, 21, FUI, cMuted, AlignLeft)
	switch us.State {
	case "downloading":
		c.RRect(x0+80, y0+H-84, W-160, 14, 7, rgba(0xffffff, .12))
		c.RRect(x0+80, y0+H-84, (W-160)*us.Progress, 14, 7, cGold)
		c.Text("Downloading…  "+itoa(int(us.Progress*100))+"%", LW/2, y0+H-40, 16, FUI, cPaper, AlignCenter)
	case "ready":
		c.Text("Installing the update and restarting Mahjong…", LW/2, y0+H-60, 17, FUI, cPaper, AlignCenter)
	case "error":
		c.Para(us.Err, x0+60, y0+H-112, W-120, 15, 20, FUI, rgb(0xff9a8a), AlignCenter)
		a.button(c, LW/2-100, y0+H-72, 200, 46, "Close", "u_close", true, false, func() { a.modal = "" })
	default:
		bw := 200.0
		a.button(c, LW/2-bw-8, y0+H-86, bw, 50, "Update Now", "u_yes", true, true, func() {
			a.sd.write(true)
			go a.upd.Apply()
		})
		a.button(c, LW/2+8, y0+H-86, bw, 50, "Not Now", "u_no", true, false, func() {
			a.sd.Skipped = us.Latest
			a.sd.write(false)
			a.modal = ""
		})
	}
}
