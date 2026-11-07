// The twelve puzzles (tile layouts). (c) 2026 Ironite
package main

// Positions are in half-tile units: a tile at (x, y, z) covers x..x+2 and
// y..y+2 on layer z. Each layer is drawn as text where '#' is a tile; the
// layer's dx, dy (half-tile units) shift it, so a layer can sit half a tile
// across or down from the one below.
type Layer struct {
	DX, DY int
	Rows   []string
}

type Pos struct{ X, Y, Z int }

type Puzzle struct {
	ID, Name, Diff, Blurb string
	Layers                []Layer
	Extra                 []Pos // single tiles that don't fit the grid
}

const (
	Easy   = "Easy"
	Medium = "Medium"
	Hard   = "Hard"
)

var difficulties = []string{Easy, Medium, Hard}

// Positions expands the layers into tile positions.
func (p *Puzzle) Positions() []Pos {
	var out []Pos
	for z, l := range p.Layers {
		for r, row := range l.Rows {
			for c, ch := range row {
				if ch == '#' {
					out = append(out, Pos{l.DX + 2*c, l.DY + 2*r, z})
				}
			}
		}
	}
	return append(out, p.Extra...)
}

func puzzleByID(id string) *Puzzle {
	for i := range puzzles {
		if puzzles[i].ID == id {
			return &puzzles[i]
		}
	}
	return nil
}

func puzzleIndex(id string) int {
	for i := range puzzles {
		if puzzles[i].ID == id {
			return i
		}
	}
	return -1
}

var puzzles = []Puzzle{
	// ---------------- Easy ----------------
	{ID: "lotus", Name: "Lotus Pad", Diff: Easy, Blurb: "A gentle start: a single lily pad with a blossom on top.",
		Layers: []Layer{
			{0, 0, []string{
				"  ####  ",
				" ###### ",
				"########",
				"########",
				" ###### ",
				"  ####  ",
			}},
			{4, 4, []string{
				"####",
				"####",
			}},
			{6, 5, []string{
				"##",
			}},
		}},
	{ID: "stones", Name: "Stepping Stones", Diff: Easy, Blurb: "Hop across the pond, one mossy stone at a time.",
		Layers: []Layer{
			{0, 0, []string{
				"##  ##  ##",
				"##  ##  ##",
				"  ##  ##  ",
				"  ##  ##  ",
				"##  ##  ##",
				"##  ##  ##",
			}},
			{1, 1, []string{
				"#   #   #",
				"         ",
				"  #   #  ",
				"         ",
				"#   #   #",
			}},
		}},
	{ID: "lantern", Name: "Little Lantern", Diff: Easy, Blurb: "A paper lantern glowing by the garden path.",
		Layers: []Layer{
			{0, 0, []string{
				"  ####  ",
				" ###### ",
				"########",
				"########",
				"########",
				" ###### ",
				"  ####  ",
				"   ##   ",
			}},
			{4, 4, []string{
				"####",
				"####",
			}},
			{6, 5, []string{
				"##",
			}},
		}},
	{ID: "koi", Name: "Koi", Diff: Easy, Blurb: "A golden koi swimming under the moon.",
		Layers: []Layer{
			{0, 0, []string{
				"    ####      ",
				"  ########  ##",
				" ########## ##",
				"##############",
				" ########## ##",
				"  ########  ##",
				"    ####      ",
			}},
			{6, 3, []string{
				"####",
				"####",
			}},
		}},

	// ---------------- Medium ----------------
	{ID: "moongate", Name: "Moon Gate", Diff: Medium, Blurb: "The round stone gateway into the garden.",
		Layers: []Layer{
			{0, 0, []string{
				"   ####   ",
				" ######## ",
				"####  ####",
				"###    ###",
				"###    ###",
				"###    ###",
				"###    ###",
				"####  ####",
				" ######## ",
				"   ####   ",
			}},
			{2, 2, []string{
				" ###### ",
				"##    ##",
				"#      #",
				"#      #",
				"#      #",
				"#      #",
				"##    ##",
				" ###### ",
			}},
			{2, 2, []string{
				"   ##   ",
				"        ",
				"        ",
				"#      #",
				"#      #",
				"        ",
				"        ",
				"   ##   ",
			}},
		}},
	{ID: "bridge", Name: "Arched Bridge", Diff: Medium, Blurb: "A red bridge arching over the koi pond.",
		Layers: []Layer{
			{0, 0, []string{
				"##############",
				"##############",
				"##############",
				"##############",
			}},
			{4, 1, []string{
				"##########",
				"##########",
				"##########",
			}},
			{8, 2, []string{
				"######",
				"######",
			}},
			{12, 3, []string{
				"##",
			}},
		}},
	{ID: "fan", Name: "Paper Fan", Diff: Medium, Blurb: "A painted silk fan, opened wide.",
		Layers: []Layer{
			{0, 0, []string{
				"   ########   ",
				" ############ ",
				"##############",
				" ############ ",
				"   ########   ",
				"     ####     ",
				"      ##      ",
				"      ##      ",
			}},
			{2, 2, []string{
				"  ########  ",
				" ########## ",
				"  ########  ",
			}},
			{6, 3, []string{
				"######",
				"######",
			}},
			{10, 4, []string{
				"##",
			}},
		}},
	{ID: "teahouse", Name: "Tea House", Diff: Medium, Blurb: "The little tea house with its curved tile roof.",
		Layers: []Layer{
			{0, 0, []string{
				"      ##      ",
				"    ######    ",
				"  ##########  ",
				"##############",
				"  ##########  ",
				"  ##########  ",
				"  ##########  ",
				"  ###    ###  ",
			}},
			{4, 6, []string{
				"##########",
				" ######## ",
				" ##    ## ",
				" ##    ## ",
			}},
			{6, 6, []string{
				"########",
				" ###### ",
			}},
			{8, 7, []string{
				"####",
			}},
		}},

	// ---------------- Hard ----------------
	{ID: "turtle", Name: "Stone Turtle", Diff: Hard, Blurb: "The classic: all 144 tiles, stacked five high.",
		Layers: []Layer{
			{0, 0, []string{
				" ############ ",
				"   ########   ",
				"  ##########  ",
				" ############ ",
				" ############ ",
				"  ##########  ",
				"   ########   ",
				" ############ ",
			}},
			{8, 2, []string{
				"######",
				"######",
				"######",
				"######",
				"######",
				"######",
			}},
			{10, 4, []string{
				"####",
				"####",
				"####",
				"####",
			}},
			{12, 6, []string{
				"##",
				"##",
			}},
			{13, 7, []string{
				"#",
			}},
		},
		Extra: []Pos{{0, 7, 0}, {26, 7, 0}, {28, 7, 0}}},
	{ID: "dragon", Name: "Jade Dragon", Diff: Hard, Blurb: "A dragon winding back and forth, its spine stacked high.",
		Layers: []Layer{
			{0, 0, []string{
				"############    ",
				"############    ",
				"          ##    ",
				"    ############",
				"    ############",
				"    ##          ",
				"############    ",
				"############    ",
			}},
			{1, 1, []string{
				"###########",
				"",
				"",
				"    ###########",
				"",
				"",
				"###########",
			}},
			{3, 1, []string{
				"#########",
				"",
				"",
				"    #########",
				"",
				"",
				"#########",
			}},
			{3, 1, []string{
				"   ##",
				"",
				"",
				"       ##",
				"",
				"",
				"   ##",
			}},
		}},
	{ID: "pagoda", Name: "Great Pagoda", Diff: Hard, Blurb: "Six tiers rising above the garden wall.",
		Layers: []Layer{
			{0, 0, []string{
				"##########",
				"##########",
				"##########",
				"##########",
				"##########",
				"##########",
			}},
			{2, 1, []string{
				"########",
				"########",
				"########",
				"########",
				"########",
			}},
			{4, 2, []string{
				"######",
				"######",
				"######",
				"######",
			}},
			{6, 3, []string{
				"####",
				"####",
				"####",
			}},
			{8, 4, []string{
				"##",
				"##",
			}},
			{8, 5, []string{
				"##",
			}},
		}},
	{ID: "fortress", Name: "Garden Fortress", Diff: Hard, Blurb: "High walls, four watchtowers and a keep in the middle.",
		Layers: []Layer{
			{0, 0, []string{
				"##############",
				"##############",
				"##############",
				"##############",
				"##############",
				"##############",
				"##############",
				"##############",
			}},
			{0, 0, []string{
				"##          ##",
				"##          ##",
				"              ",
				"     ####     ",
				"     ####     ",
				"              ",
				"##          ##",
				"##          ##",
			}},
			{1, 1, []string{
				"#           #",
				"             ",
				"             ",
				"     # #     ",
				"             ",
				"             ",
				"#           #",
			}},
		}},
}
