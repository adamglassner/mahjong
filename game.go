// A game in progress: matching, undo, hints, shuffles, the clock and stars.
// No graphics here. (c) 2026 Ironite
package main

import (
	"math/rand"
	"time"
)

const Version = "1.0.0"

type Play struct {
	P        *Puzzle
	B        *Board
	Seed     int64
	Hist     [][2]int
	Elapsed  float64 // seconds of play
	Hints    int
	Shuffles int
	Undos    int
	Won      bool
	rng      *rand.Rand
}

// SavedGame is a game in progress as it's kept in the save file.
type SavedGame struct {
	Puzzle   string   `json:"puzzle"`
	Seed     int64    `json:"seed"`
	Faces    []int    `json:"faces"`
	Gone     []bool   `json:"gone"`
	Hist     [][2]int `json:"hist"`
	Elapsed  float64  `json:"elapsed"`
	Hints    int      `json:"hints"`
	Shuffles int      `json:"shuffles"`
	Undos    int      `json:"undos"`
}

func NewPlay(p *Puzzle, seed int64) *Play {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	b := NewDeal(p.Positions(), seed)
	for b == nil { // never happens with the shipped layouts, but don't hang on a bad seed
		seed++
		b = NewDeal(p.Positions(), seed)
	}
	return &Play{P: p, B: b, Seed: seed, rng: rand.New(rand.NewSource(seed ^ 0x5eed))}
}

// RestorePlay rebuilds a saved game; nil if the save doesn't fit the layout.
func RestorePlay(s *SavedGame) *Play {
	if s == nil {
		return nil
	}
	p := puzzleByID(s.Puzzle)
	if p == nil {
		return nil
	}
	pos := p.Positions()
	if len(s.Faces) != len(pos) || len(s.Gone) != len(pos) {
		return nil
	}
	b := NewBoard(pos)
	for i := range b.T {
		if s.Faces[i] < 0 || s.Faces[i] >= NumKinds {
			return nil
		}
		b.T[i].Face, b.T[i].Gone = s.Faces[i], s.Gone[i]
	}
	for _, h := range s.Hist {
		if h[0] < 0 || h[1] < 0 || h[0] >= len(pos) || h[1] >= len(pos) {
			return nil
		}
	}
	if b.Remaining() == 0 {
		return nil
	}
	return &Play{P: p, B: b, Seed: s.Seed, Hist: s.Hist, Elapsed: s.Elapsed, Hints: s.Hints, Shuffles: s.Shuffles, Undos: s.Undos,
		rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (pl *Play) Save() *SavedGame {
	s := &SavedGame{Puzzle: pl.P.ID, Seed: pl.Seed, Hist: append([][2]int{}, pl.Hist...), Elapsed: pl.Elapsed,
		Hints: pl.Hints, Shuffles: pl.Shuffles, Undos: pl.Undos}
	for _, t := range pl.B.T {
		s.Faces = append(s.Faces, t.Face)
		s.Gone = append(s.Gone, t.Gone)
	}
	return s
}

// CanMatch: two different free tiles that match.
func (pl *Play) CanMatch(a, b int) bool {
	return a != b && pl.B.Free(a) && pl.B.Free(b) && tilesMatch(pl.B.T[a].Face, pl.B.T[b].Face)
}

// Match removes the pair. It reports whether the board is now clear.
func (pl *Play) Match(a, b int) bool {
	pl.B.T[a].Gone, pl.B.T[b].Gone = true, true
	pl.Hist = append(pl.Hist, [2]int{a, b})
	if pl.B.Remaining() == 0 {
		pl.Won = true
	}
	return pl.Won
}

func (pl *Play) Undo() ([2]int, bool) {
	if len(pl.Hist) == 0 || pl.Won {
		return [2]int{}, false
	}
	h := pl.Hist[len(pl.Hist)-1]
	pl.Hist = pl.Hist[:len(pl.Hist)-1]
	pl.B.T[h[0]].Gone, pl.B.T[h[1]].Gone = false, false
	pl.Undos++
	return h, true
}

func (pl *Play) Shuffle() bool {
	if pl.B.Reshuffle(pl.rng) {
		pl.Shuffles++
		return true
	}
	return false
}

func (pl *Play) Hint() ([2]int, bool) {
	h, ok := pl.B.BestHint()
	if ok {
		pl.Hints++
	}
	return h, ok
}

// Stars for a finished game: three with no hints or shuffles, two with up to
// three of them, else one. Undo is free.
func (pl *Play) Stars() int {
	n := pl.Hints + pl.Shuffles
	switch {
	case n == 0:
		return 3
	case n <= 3:
		return 2
	}
	return 1
}

func (pl *Play) Stuck() bool { return !pl.Won && len(pl.B.OpenPairs()) == 0 }

func fmtTime(sec float64) string {
	s := int(sec)
	if s >= 3600 {
		return itoa(s/3600) + ":" + pad2(s/60%60) + ":" + pad2(s%60)
	}
	return itoa(s/60) + ":" + pad2(s%60)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
