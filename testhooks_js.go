//go:build js

// Test hooks for the browser build only (used for automated screenshots).
// Not in the Windows game.
package main

import (
	"encoding/json"
	"syscall/js"
)

var testApp *App

func setTestApp(a *App) { testApp = a }

func init() {
	js.Global().Set("__mj", js.ValueOf(map[string]any{
		"start": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			a.startPuzzle(args[0].String())
			if len(args) > 1 {
				a.pl = NewPlay(a.pl.P, int64(args[1].Int()))
			}
			return nil
		}),
		"modal":  js.FuncOf(func(this js.Value, args []js.Value) any { testApp.modal = args[0].String(); return nil }),
		"screen": js.FuncOf(func(this js.Value, args []js.Value) any { testApp.screen = args[0].String(); return nil }),
		"set": js.FuncOf(func(this js.Value, args []js.Value) any {
			st := &testApp.sd.Settings
			switch args[0].String() {
			case "scene":
				st.Scene = args[1].String()
			case "tiles":
				st.Tiles = args[1].String()
			case "highlight":
				st.Highlight = args[1].Bool()
			}
			return nil
		}),
		// play n hinted pairs straight away
		"play": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			for i := 0; i < args[0].Int(); i++ {
				h, ok := a.pl.B.BestHint()
				if !ok {
					break
				}
				a.doMatch(h[0], h[1])
			}
			return nil
		}),
		// page position of a hinted pair's tiles
		"hintPos": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			h, _ := a.pl.B.BestHint()
			r := js.Global().Get("devicePixelRatio").Float()
			var out []any
			for _, i := range h {
				x, y := a.tileCenterDev(i)
				out = append(out, x/r, y/r)
			}
			return out
		}),
		"hint": js.FuncOf(func(this js.Value, args []js.Value) any { testApp.doHint(); return nil }),
		"select": js.FuncOf(func(this js.Value, args []js.Value) any {
			testApp.sel = testApp.pl.B.FreeTiles()[args[0].Int()]
			return nil
		}),
		"win": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			a.pl.Elapsed = 263
			a.finishWin()
			return nil
		}),
		"state": js.FuncOf(func(this js.Value, args []js.Value) any {
			a := testApp
			m := map[string]any{"screen": a.screen, "modal": a.modal}
			if a.pl != nil {
				m["left"] = a.pl.B.Remaining()
				m["open"] = len(a.pl.B.OpenPairs())
			}
			b, _ := json.Marshal(m)
			return string(b)
		}),
	}))
}
