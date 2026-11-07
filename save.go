// The save file: settings, puzzle records and the game in progress. (c) 2026 Ironite
package main

import (
	"encoding/json"
	"sync"
	"time"
)

type Settings struct {
	Music     bool    `json:"music"`
	Sfx       bool    `json:"sfx"`
	Vol       float64 `json:"vol"`
	Tiles     string  `json:"tiles"`     // ivory, jade, rosewood
	Scene     string  `json:"scene"`     // moon, lantern, dawn
	Highlight bool    `json:"highlight"` // dim the tiles that can't be picked
	Corners   bool    `json:"corners"`   // little numbers and letters in the tile corners
	Updates   bool    `json:"updates"`   // check GitHub for updates (standalone copy only)
}

type Record struct {
	Wins  int     `json:"wins"`
	Best  float64 `json:"best"` // seconds, 0 = never finished
	Stars int     `json:"stars"`
}

type SaveData struct {
	Settings Settings           `json:"settings"`
	Records  map[string]*Record `json:"records"`
	Game     *SavedGame         `json:"game"`
	Last     string             `json:"last"` // last puzzle played
	Played   int                `json:"played"`
	Skipped  string             `json:"skippedVersion,omitempty"` // an update the player said "not now" to
}

var tileSets = []string{"ivory", "jade", "rosewood"}
var scenes = []string{"moon", "lantern", "dawn"}

func validIn(v string, list []string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func defaultSave() *SaveData {
	return &SaveData{
		Settings: Settings{Music: true, Sfx: true, Vol: 70, Tiles: "ivory", Scene: "moon", Highlight: false, Corners: true, Updates: true},
		Records:  map[string]*Record{},
		Last:     puzzles[0].ID,
	}
}

var (
	saveMu       sync.Mutex
	saveTimer    *time.Timer
	saveDisabled bool // tests
)

func loadSave() *SaveData {
	sd := defaultSave()
	if b, err := readSaveFile(); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, sd) // fields missing from an older save keep their defaults
	}
	if sd.Records == nil {
		sd.Records = map[string]*Record{}
	}
	st := &sd.Settings
	if !validIn(st.Tiles, tileSets) {
		st.Tiles = "ivory"
	}
	if !validIn(st.Scene, scenes) {
		st.Scene = "moon"
	}
	if st.Vol < 0 || st.Vol > 100 {
		st.Vol = 70
	}
	if puzzleByID(sd.Last) == nil {
		sd.Last = puzzles[0].ID
	}
	return sd
}

// write saves soon (debounced), or right away with now=true.
func (sd *SaveData) write(now bool) {
	if saveDisabled {
		return
	}
	b, err := json.MarshalIndent(sd, "", " ")
	if err != nil {
		return
	}
	saveMu.Lock()
	defer saveMu.Unlock()
	if saveTimer != nil {
		saveTimer.Stop()
	}
	if now {
		_ = writeSaveFile(b)
		return
	}
	saveTimer = time.AfterFunc(400*time.Millisecond, func() { _ = writeSaveFile(b) })
}

// record notes a finished puzzle; it reports whether it was a new best time.
func (sd *SaveData) record(id string, sec float64, stars int) bool {
	r := sd.Records[id]
	if r == nil {
		r = &Record{}
		sd.Records[id] = r
	}
	r.Wins++
	best := r.Best == 0 || sec < r.Best
	if best {
		r.Best = sec
	}
	if stars > r.Stars {
		r.Stars = stars
	}
	return best
}
