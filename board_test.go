package main

import (
	"math/rand"
	"testing"
)

func TestTileSet(t *testing.T) {
	pool := pairPool()
	if len(pool) != 72 {
		t.Fatalf("pairs %d, want 72", len(pool))
	}
	count := map[int]int{}
	for _, p := range pool {
		if !tilesMatch(p[0], p[1]) {
			t.Fatalf("pair %v doesn't match", p)
		}
		count[p[0]]++
		count[p[1]]++
	}
	for k := 0; k < NumKinds; k++ {
		want := 4
		if k >= KFlower {
			want = 1
		}
		if count[k] != want {
			t.Errorf("%s: %d tiles, want %d", tileName(k), count[k], want)
		}
	}
	if tilesMatch(KDots, KBamboo) || !tilesMatch(KFlower, KFlower+3) || tilesMatch(KFlower, KSeason) {
		t.Fatal("match rules")
	}
}

func TestLayouts(t *testing.T) {
	per := map[string]int{}
	ids := map[string]bool{}
	for _, p := range puzzles {
		if ids[p.ID] {
			t.Errorf("duplicate id %s", p.ID)
		}
		ids[p.ID] = true
		per[p.Diff]++
		pos := p.Positions()
		if len(pos)%2 != 0 || len(pos) > 144 || len(pos) < 20 {
			t.Errorf("%s: %d tiles (must be even, 20-144)", p.Name, len(pos))
		}
		for i, a := range pos {
			for j, c := range pos {
				if i < j && a.Z == c.Z && overlaps(a, c) {
					t.Errorf("%s: tiles %v and %v overlap", p.Name, a, c)
				}
			}
			if a.Z == 0 {
				continue
			}
			// every quarter of a raised tile must rest on a tile below
			for _, q := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				ok := false
				for _, c := range pos {
					if c.Z == a.Z-1 && a.X+q[0] >= c.X && a.X+q[0] < c.X+2 && a.Y+q[1] >= c.Y && a.Y+q[1] < c.Y+2 {
						ok = true
					}
				}
				if !ok {
					t.Errorf("%s: tile %v isn't supported at quarter %v", p.Name, a, q)
				}
			}
		}
		t.Logf("%-16s %-6s %3d tiles, %d layers", p.Name, p.Diff, len(pos), len(p.Layers))
	}
	for _, d := range difficulties {
		if per[d] < 3 {
			t.Errorf("%s has %d puzzles", d, per[d])
		}
	}
	if len(puzzles) < 10 {
		t.Errorf("only %d puzzles", len(puzzles))
	}
}

// Every deal must be clearable: follow the solver to the end.
func TestDealsSolvable(t *testing.T) {
	for _, p := range puzzles {
		for seed := int64(1); seed <= 6; seed++ {
			b := NewDeal(p.Positions(), seed)
			if b == nil {
				t.Fatalf("%s seed %d: no deal", p.Name, seed)
			}
			groups := map[int]int{}
			for _, tl := range b.T {
				if tl.Face < 0 {
					t.Fatalf("%s: tile without a face", p.Name)
				}
				groups[matchGroup(tl.Face)]++
			}
			for g, n := range groups {
				if n%2 != 0 {
					t.Fatalf("%s: group %d has %d tiles", p.Name, g, n)
				}
			}
			replay(t, b, p.Name)
		}
	}
}

func TestReshuffle(t *testing.T) {
	p := puzzleByID("turtle")
	b := NewDeal(p.Positions(), 7)
	rng := rand.New(rand.NewSource(3))
	// play ten hinted moves, then reshuffle
	for i := 0; i < 10; i++ {
		h, ok := b.BestHint()
		if !ok {
			t.Fatal("no hint")
		}
		b.T[h[0]].Gone, b.T[h[1]].Gone = true, true
	}
	before := map[int]int{}
	for _, tl := range b.T {
		if !tl.Gone {
			before[tl.Face]++
		}
	}
	if !b.Reshuffle(rng) {
		t.Fatal("reshuffle failed")
	}
	after := map[int]int{}
	for _, tl := range b.T {
		if !tl.Gone {
			after[tl.Face]++
		}
	}
	for k, n := range before {
		if after[k] != n {
			t.Fatalf("face %d: %d before, %d after", k, n, after[k])
		}
	}
	replay(t, b, "reshuffled turtle")
}

func TestFreeRules(t *testing.T) {
	// a row of three, with one tile on top of the middle
	b := NewBoard([]Pos{{0, 0, 0}, {2, 0, 0}, {4, 0, 0}, {2, 0, 1}})
	want := []bool{true, false, true, true}
	for i, w := range want {
		if b.Free(i) != w {
			t.Errorf("tile %d free=%v want %v", i, b.Free(i), w)
		}
	}
	b.T[3].Gone = true
	if b.Free(1) {
		t.Error("middle of a row should be blocked")
	}
	b.T[0].Gone = true
	if !b.Free(1) {
		t.Error("middle tile with left side open should be free")
	}
}

// replay clears the board along its known solution, checking each step.
func replay(t *testing.T, b *Board, name string) {
	t.Helper()
	saved := make([]bool, len(b.T))
	for i := range b.T {
		saved[i] = b.T[i].Gone
	}
	for k, pr := range b.Solution {
		if !b.Free(pr[0]) || !b.Free(pr[1]) || !tilesMatch(b.T[pr[0]].Face, b.T[pr[1]].Face) {
			t.Fatalf("%s: step %d %v isn't a legal match", name, k, pr)
		}
		b.T[pr[0]].Gone, b.T[pr[1]].Gone = true, true
	}
	if b.Remaining() != 0 {
		t.Fatalf("%s: %d tiles left after the solution", name, b.Remaining())
	}
	for i := range b.T {
		b.T[i].Gone = saved[i]
	}
}

// Playing only hints should usually clear the easy boards; and never offer an illegal pair.
func TestHints(t *testing.T) {
	cleared := 0
	for _, p := range puzzles {
		for seed := int64(1); seed <= 4; seed++ {
			b := NewDeal(p.Positions(), seed)
			for {
				h, ok := b.BestHint()
				if !ok {
					break
				}
				if !b.Free(h[0]) || !b.Free(h[1]) || !tilesMatch(b.T[h[0]].Face, b.T[h[1]].Face) {
					t.Fatal("bad hint")
				}
				b.T[h[0]].Gone, b.T[h[1]].Gone = true, true
			}
			if b.Remaining() == 0 {
				cleared++
			}
		}
	}
	t.Logf("hint-only play cleared %d of %d boards", cleared, len(puzzles)*4)
}

func TestPlaySaveRestore(t *testing.T) {
	saveDisabled = true
	pl := NewPlay(puzzleByID("lotus"), 42)
	h, _ := pl.Hint()
	pl.Match(h[0], h[1])
	s := pl.Save()
	r := RestorePlay(s)
	if r == nil || r.B.Remaining() != pl.B.Remaining() || len(r.Hist) != 1 || r.Hints != 1 {
		t.Fatal("restore mismatch")
	}
	if _, ok := r.Undo(); !ok || r.B.Remaining() != len(r.B.T) {
		t.Fatal("undo after restore")
	}
	// clear the whole board with hints and shuffles
	for !pl.Won {
		if pl.Stuck() {
			if !pl.Shuffle() {
				t.Fatal("shuffle failed")
			}
			continue
		}
		h, _ := pl.Hint()
		pl.Match(h[0], h[1])
	}
	if pl.Stars() < 1 || fmtTime(65) != "1:05" || fmtTime(3725) != "1:02:05" {
		t.Fatal("stars/time")
	}
	sd := defaultSave()
	if !sd.record("lotus", 100, 2) || sd.record("lotus", 120, 3) || sd.Records["lotus"].Stars != 3 || sd.Records["lotus"].Best != 100 {
		t.Fatal("records")
	}
}
