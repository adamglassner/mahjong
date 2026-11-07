// The board: which tiles are free, open pairs, and dealing boards that can
// always be cleared. No graphics here. (c) 2026 Ironite
package main

import (
	"math/rand"
	"sort"
)

type Tile struct {
	Pos
	Face int
	Gone bool
}

type Board struct {
	T     []Tile
	above [][]int // tiles resting on (overlapping, higher) each tile
	left  [][]int // same-layer tiles touching the left edge
	right [][]int
	below [][]int // tiles directly underneath (z-1, overlapping)
	order []int   // draw order: back to front
	// Solution is the order the last Deal or Reshuffle was built from: a known
	// way to clear the board from that point.
	Solution [][2]int
}

func iabs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func overlaps(a, b Pos) bool { return iabs(a.X-b.X) < 2 && iabs(a.Y-b.Y) < 2 }

// NewBoard builds the neighbour tables for a set of positions.
func NewBoard(pos []Pos) *Board {
	n := len(pos)
	b := &Board{T: make([]Tile, n), above: make([][]int, n), left: make([][]int, n), right: make([][]int, n), below: make([][]int, n)}
	for i, p := range pos {
		b.T[i] = Tile{Pos: p, Face: -1}
	}
	for i, a := range pos {
		for j, c := range pos {
			if i == j {
				continue
			}
			if c.Z > a.Z && overlaps(a, c) {
				b.above[i] = append(b.above[i], j)
			}
			if c.Z == a.Z-1 && overlaps(a, c) {
				b.below[i] = append(b.below[i], j)
			}
			if c.Z == a.Z && iabs(c.Y-a.Y) < 2 {
				if c.X == a.X-2 {
					b.left[i] = append(b.left[i], j)
				}
				if c.X == a.X+2 {
					b.right[i] = append(b.right[i], j)
				}
			}
		}
	}
	b.order = make([]int, n)
	for i := range b.order {
		b.order[i] = i
	}
	sort.SliceStable(b.order, func(x, y int) bool {
		p, q := pos[b.order[x]], pos[b.order[y]]
		if p.Z != q.Z {
			return p.Z < q.Z
		}
		if p.X != q.X {
			return p.X < q.X
		}
		return p.Y < q.Y
	})
	return b
}

func anyPresent(t []Tile, idx []int) bool {
	for _, j := range idx {
		if !t[j].Gone {
			return true
		}
	}
	return false
}

// Free: nothing on top, and the left or the right side open.
func (b *Board) Free(i int) bool {
	if b.T[i].Gone || anyPresent(b.T, b.above[i]) {
		return false
	}
	return !anyPresent(b.T, b.left[i]) || !anyPresent(b.T, b.right[i])
}

func (b *Board) Remaining() int {
	n := 0
	for _, t := range b.T {
		if !t.Gone {
			n++
		}
	}
	return n
}

func (b *Board) FreeTiles() []int {
	var out []int
	for i := range b.T {
		if b.Free(i) {
			out = append(out, i)
		}
	}
	return out
}

// OpenPairs lists every pair of free tiles that match.
func (b *Board) OpenPairs() [][2]int {
	free := b.FreeTiles()
	var out [][2]int
	for x := 0; x < len(free); x++ {
		for y := x + 1; y < len(free); y++ {
			if tilesMatch(b.T[free[x]].Face, b.T[free[y]].Face) {
				out = append(out, [2]int{free[x], free[y]})
			}
		}
	}
	return out
}

// removalOrder plays the board backwards at random: it keeps taking two free
// tiles (ignoring faces) until none are left. Any faces assigned in pairs along
// that order give a board that can be cleared. Returns nil if it got stuck.
func (b *Board) removalOrder(rng *rand.Rand) [][2]int {
	saved := make([]bool, len(b.T))
	for i := range b.T {
		saved[i] = b.T[i].Gone
	}
	defer func() {
		for i := range b.T {
			b.T[i].Gone = saved[i]
		}
	}()
	var order [][2]int
	weight := func(i int) float64 {
		w := 1 + 2.5*float64(b.T[i].Z)
		for _, j := range b.below[i] {
			if !b.T[j].Gone {
				w += 1.5
			}
		}
		// tiles holding a long row shut are worth freeing early
		if anyPresent(b.T, b.left[i]) || anyPresent(b.T, b.right[i]) {
			w += 1
		}
		return w
	}
	pick := func(cands []int) int {
		tot := 0.0
		for _, i := range cands {
			tot += weight(i)
		}
		r := rng.Float64() * tot
		for k, i := range cands {
			r -= weight(i)
			if r <= 0 {
				return k
			}
		}
		return len(cands) - 1
	}
	for {
		left := b.Remaining()
		if left == 0 {
			return order
		}
		free := b.FreeTiles()
		if len(free) < 2 {
			return nil
		}
		k := pick(free)
		a := free[k]
		free = append(free[:k:k], free[k+1:]...)
		c := free[pick(free)]
		b.T[a].Gone, b.T[c].Gone = true, true
		order = append(order, [2]int{a, c})
	}
}

// Deal gives every present tile a face so the board can be cleared, using the
// given pairs of faces (len(pairs) must be Remaining()/2).
func (b *Board) Deal(rng *rand.Rand, pairs [][2]int) bool {
	for try := 0; try < 4000; try++ {
		order := b.removalOrder(rng)
		if order == nil {
			continue
		}
		rng.Shuffle(len(pairs), func(i, j int) { pairs[i], pairs[j] = pairs[j], pairs[i] })
		for k, pr := range order {
			f := pairs[k]
			if rng.Intn(2) == 0 {
				f[0], f[1] = f[1], f[0]
			}
			b.T[pr[0]].Face, b.T[pr[1]].Face = f[0], f[1]
		}
		b.Solution = order
		return true
	}
	return false
}

// NewDeal deals a fresh board for a layout from the full tile set.
func NewDeal(pos []Pos, seed int64) *Board {
	b := NewBoard(pos)
	rng := rand.New(rand.NewSource(seed))
	pool := pairPool()
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	pairs := append([][2]int(nil), pool[:len(pos)/2]...)
	if !b.Deal(rng, pairs) {
		return nil
	}
	return b
}

// Reshuffle moves the faces of the remaining tiles around so the rest of the
// board can still be cleared. The same tiles stay in play.
func (b *Board) Reshuffle(rng *rand.Rand) bool {
	groups := map[int][]int{}
	var keys []int
	for _, t := range b.T {
		if t.Gone {
			continue
		}
		g := matchGroup(t.Face)
		if _, ok := groups[g]; !ok {
			keys = append(keys, g)
		}
		groups[g] = append(groups[g], t.Face)
	}
	sort.Ints(keys)
	var pairs [][2]int
	for _, g := range keys {
		fs := groups[g]
		sort.Ints(fs)
		for i := 0; i+1 < len(fs); i += 2 {
			pairs = append(pairs, [2]int{fs[i], fs[i+1]})
		}
	}
	return b.Deal(rng, pairs)
}

// Bounds of the layout in half-tile units (max x+2, max y+2, top layer).
func (b *Board) Bounds() (w, h, top int) {
	for _, t := range b.T {
		if t.X+2 > w {
			w = t.X + 2
		}
		if t.Y+2 > h {
			h = t.Y + 2
		}
		if t.Z > top {
			top = t.Z
		}
	}
	return
}

// BestHint returns an open pair. It prefers pairs that can't cost anything
// (all the remaining tiles of that kind are free), then tiles that are
// holding others down: high stacks and the ends of long rows.
func (b *Board) BestHint() ([2]int, bool) {
	pairs := b.OpenPairs()
	if len(pairs) == 0 {
		return [2]int{}, false
	}
	left := map[int]int{}
	for _, t := range b.T {
		if !t.Gone {
			left[matchGroup(t.Face)]++
		}
	}
	freeOf := map[int]int{}
	for _, i := range b.FreeTiles() {
		freeOf[matchGroup(b.T[i].Face)]++
	}
	score := func(p [2]int) float64 {
		s := 0.0
		g := matchGroup(b.T[p[0]].Face)
		if left[g] == 2 || (left[g] == 4 && freeOf[g] == 4) {
			s += 100
		}
		for _, i := range p {
			s += 10 * float64(b.T[i].Z)
			for _, j := range b.below[i] {
				if !b.T[j].Gone {
					s += 6
				}
			}
			// how many tiles are still in its row: freeing a long row helps
			n := 0
			for k := i; ; {
				nx := -1
				for _, j := range b.right[k] {
					if !b.T[j].Gone {
						nx = j
					}
				}
				for _, j := range b.left[k] {
					if !b.T[j].Gone && nx < 0 {
						nx = j
					}
				}
				if nx < 0 || n > 20 {
					break
				}
				n++
				k = nx
			}
			s += float64(n)
		}
		return s
	}
	best := pairs[0]
	for _, p := range pairs[1:] {
		if score(p) > score(best) {
			best = p
		}
	}
	return best, true
}
