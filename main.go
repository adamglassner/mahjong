//go:build !headless

// Mahjong - Jade Moon Garden. An Ironite game.
// Window, input and the frame loop. (c) 2026 Ironite
package main

import (
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	a    *App
	last time.Time
	hand bool
}

func inRect(x, y float64, h hit) bool { return x >= h.x && y >= h.y && x < h.x+h.w && y < h.y+h.h }

func (g *Game) Update() error {
	a := g.a
	if a.quit {
		return ebiten.Termination
	}
	now := time.Now()
	dt := math.Min(.05, now.Sub(g.last).Seconds())
	g.last = now
	a.step(dt)

	cx, cy := ebiten.CursorPosition()
	// with LayoutF in device pixels, the cursor is already in device pixels
	dx, dy := float64(cx), float64(cy)
	if a.s > 0 {
		a.mx, a.my = (dx-a.ox)/a.s, (dy-a.oy)/a.s
	}
	// hover over last frame's buttons
	hov := ""
	for i := len(a.prev) - 1; i >= 0; i-- {
		h := a.prev[i]
		if (a.modal == "" || h.modal) && inRect(a.mx, a.my, h) {
			hov = h.id
			break
		}
	}
	a.hover = hov
	onTile := -1
	if a.modal == "" && a.screen == "game" && hov == "" {
		onTile = a.tileAt(dx, dy)
	}
	hand := hov != "" || (onTile >= 0 && a.pl.B.Free(onTile))
	if hand != g.hand {
		g.hand = hand
		if hand {
			ebiten.SetCursorShape(ebiten.CursorShapePointer)
		} else {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		handled := false
		for i := len(a.prev) - 1; i >= 0; i-- {
			h := a.prev[i]
			if (a.modal == "" || h.modal) && inRect(a.mx, a.my, h) && h.fn != nil {
				h.fn()
				handled = true
				break
			}
		}
		if !handled && onTile >= 0 {
			a.clickTile(onTile)
		} else if !handled && a.modal == "" && a.screen == "game" && a.sel >= 0 {
			a.sel = -1
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && a.sel >= 0 {
		a.sel = -1
	}
	if a.volDrag && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		a.volDrag = false
		a.sd.write(false)
	}
	g.keys()
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || (inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	return nil
}

func (g *Game) keys() {
	a := g.a
	jp := inpututil.IsKeyJustPressed
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if jp(ebiten.KeyM) {
		a.toggleMusic()
	}
	if jp(ebiten.KeyEscape) {
		switch a.modal {
		case "settings", "rules", "puzzles", "stuck":
			a.modal = ""
			return
		case "confirm":
			a.modal = ""
			a.conf.cb(false)
			return
		case "":
			if a.sel >= 0 {
				a.sel = -1
			} else if a.screen == "game" {
				a.screen = "title"
			}
		}
	}
	if a.screen != "game" || (a.modal != "" && a.modal != "stuck") {
		return
	}
	switch {
	case jp(ebiten.KeyH):
		a.doHint()
	case jp(ebiten.KeyU), ctrl && jp(ebiten.KeyZ):
		a.undo()
	case jp(ebiten.KeyS):
		a.doShuffle()
	case jp(ebiten.KeyN):
		a.newDeal()
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	b := screen.Bounds()
	g.a.layout(b.Dx(), b.Dy())
	g.a.Draw(screen)
}

// LayoutF works in real device pixels so the game is drawn at full resolution.
func (g *Game) LayoutF(w, h float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return math.Ceil(w * s), math.Ceil(h * s)
}
func (g *Game) Layout(w, h int) (int, int) { panic("LayoutF is used") }

func main() {
	loadFonts()
	sd := loadSave()
	synth := NewSynth()
	app := NewApp(sd, synth)
	setTestApp(app)
	synth.OnTune = app.nowPlaying
	synth.SetLevels(sd.Settings.Music, sd.Settings.Sfx, sd.Settings.Vol)
	startAudio(synth)
	app.upd.Quit = func() { app.quit = true }
	if sd.Settings.Updates {
		go func() {
			time.Sleep(3 * time.Second)
			app.upd.Check(false)
		}()
	}
	cleanupOldUpdate()

	ebiten.SetWindowTitle("Mahjong - Jade Moon Garden")
	ebiten.SetWindowIcon(windowIcons())
	mw, mh := ebiten.Monitor().Size()
	w := int(math.Min(float64(mw)*.85, float64(mh)*.85*16/9))
	if w < 1024 {
		w = 1024
	}
	ebiten.SetWindowSize(w, w*9/16)
	ebiten.SetWindowSizeLimits(800, 450, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	runGame(&Game{a: app, last: time.Now()})
	if app.pl != nil && !app.pl.Won {
		sd.Game = app.pl.Save()
	}
	sd.write(true)
}

// windowIcons: the game icon at the sizes Windows asks for.
func windowIcons() []image.Image {
	src := assetPNG("icon-256.png")
	var out []image.Image
	for _, s := range []int{16, 24, 32, 48, 64, 128, 256} {
		out = append(out, scaleImg(src, s))
	}
	return out
}

func scaleImg(src image.Image, s int) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, s, s))
	// box filter: average the source pixels under each destination pixel
	fx := float64(b.Dx()) / float64(s)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			var r, g, bb, al, n float64
			for sy := int(float64(y) * fx); sy < int(float64(y+1)*fx); sy++ {
				for sx := int(float64(x) * fx); sx < int(float64(x+1)*fx); sx++ {
					cr, cg, cb, ca := src.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					r, g, bb, al, n = r+float64(cr), g+float64(cg), bb+float64(cb), al+float64(ca), n+1
				}
			}
			if n == 0 {
				continue
			}
			i := dst.PixOffset(x, y)
			a := al / n
			if a > 0 {
				dst.Pix[i] = uint8(r / n / a * 255)
				dst.Pix[i+1] = uint8(g / n / a * 255)
				dst.Pix[i+2] = uint8(bb / n / a * 255)
			}
			dst.Pix[i+3] = uint8(a / 257)
		}
	}
	return dst
}

func runGame(g *Game) {
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{}); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
