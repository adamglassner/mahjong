// Mahjong - Jade Moon Garden. An Ironite game.
// The 144-tile set: kinds, names and which tiles match. (c) 2026 Ironite
package main

// Tile kinds (faces). 0-8 Dots 1-9, 9-17 Bamboo 1-9, 18-26 Characters 1-9,
// 27-30 Winds (East, South, West, North), 31-33 Dragons (Red, Green, White),
// 34-37 Flowers (Plum, Orchid, Chrysanthemum, Bamboo),
// 38-41 Seasons (Spring, Summer, Autumn, Winter).
const (
	KDots    = 0
	KBamboo  = 9
	KChar    = 18
	KWind    = 27
	KDragon  = 31
	KFlower  = 34
	KSeason  = 38
	NumKinds = 42
)

// Suit of a kind: 0 dots, 1 bamboo, 2 characters, 3 winds, 4 dragons, 5 flowers, 6 seasons.
func suitOf(k int) int {
	switch {
	case k < KBamboo:
		return 0
	case k < KChar:
		return 1
	case k < KWind:
		return 2
	case k < KDragon:
		return 3
	case k < KFlower:
		return 4
	case k < KSeason:
		return 5
	}
	return 6
}

// rankOf is 1-9 for suited tiles, 1-4 for winds, flowers and seasons, 1-3 for dragons.
func rankOf(k int) int {
	switch suitOf(k) {
	case 0:
		return k - KDots + 1
	case 1:
		return k - KBamboo + 1
	case 2:
		return k - KChar + 1
	case 3:
		return k - KWind + 1
	case 4:
		return k - KDragon + 1
	case 5:
		return k - KFlower + 1
	}
	return k - KSeason + 1
}

// matchGroup: two tiles match when their groups are equal. Any flower matches
// any flower, and any season any season.
func matchGroup(k int) int {
	switch suitOf(k) {
	case 5:
		return 100
	case 6:
		return 101
	}
	return k
}

func tilesMatch(a, b int) bool { return matchGroup(a) == matchGroup(b) }

var windNames = []string{"East", "South", "West", "North"}
var dragonNames = []string{"Red", "Green", "White"}
var flowerNames = []string{"Plum", "Orchid", "Chrysanthemum", "Bamboo"}
var seasonNames = []string{"Spring", "Summer", "Autumn", "Winter"}
var numWords = []string{"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine"}

// tileName: "Three of Dots", "East Wind", "Red Dragon", "Plum (Flower)"...
func tileName(k int) string {
	r := rankOf(k)
	switch suitOf(k) {
	case 0:
		return numWords[r-1] + " of Dots"
	case 1:
		return numWords[r-1] + " of Bamboo"
	case 2:
		return numWords[r-1] + " of Characters"
	case 3:
		return windNames[r-1] + " Wind"
	case 4:
		return dragonNames[r-1] + " Dragon"
	case 5:
		return flowerNames[r-1] + " (Flower)"
	}
	return seasonNames[r-1] + " (Season)"
}

// pairPool is all 72 matching pairs of the full 144-tile set: two pairs of
// each of the 34 regular tiles, plus the flowers and seasons paired up.
func pairPool() [][2]int {
	var out [][2]int
	for k := 0; k < KFlower; k++ {
		out = append(out, [2]int{k, k}, [2]int{k, k})
	}
	out = append(out, [2]int{KFlower, KFlower + 1}, [2]int{KFlower + 2, KFlower + 3})
	out = append(out, [2]int{KSeason, KSeason + 1}, [2]int{KSeason + 2, KSeason + 3})
	return out
}
