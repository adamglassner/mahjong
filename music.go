// The four garden tunes, the sequencer that plays them, and the sound
// effects. (c) 2026 Ironite
package main

import (
	"math"
	"strconv"
	"strings"
)

const (
	evFlute = iota
	evZither
	evPad
	evBass
	evArp
	evBlock
	evDrum
	evChime
)

type event struct {
	t, midi, dur, vel, pan float64
	kind                   int
}

type tuneDef struct {
	name   string
	tempo  float64
	mel    string // "A4:2 B4:1 r:1 ..." in beats, 16 bars of 4/4
	chords string // one chord per bar
	lead   [2]int // lead instrument for the first and second time through
	busy   bool   // eighth-note zither arpeggios and percussion
}

var tuneDefs = []tuneDef{
	{name: "Moon Over the Koi Pond", tempo: 70, lead: [2]int{evFlute, evZither},
		mel: "A4:2 B4:1 D5:1 E5:3 D5:1 B4:1.5 A4:.5 F#4:1 A4:1 E4:4 " +
			"F#4:1 A4:1 B4:1 D5:1 E5:1.5 F#5:.5 E5:1 D5:1 B4:2 A4:1 B4:1 D5:4 " +
			"F#5:2 E5:1 D5:1 E5:2 B4:2 D5:1 B4:1 A4:1 F#4:1 A4:4 " +
			"B4:1 D5:1 E5:1 F#5:1 A5:2 F#5:1 E5:1 D5:1.5 E5:.5 B4:1 A4:1 D5:4",
		chords: "D Bm G A D Bm Em A Bm G D A G Em Asus D"},
	{name: "Lantern Walk", tempo: 96, lead: [2]int{evZither, evFlute}, busy: true,
		mel: "G4:.5 A4:.5 B4:1 D5:1 B4:1 A4:.5 G4:.5 E4:1 G4:2 D5:.5 E5:.5 D5:1 B4:1 A4:1 B4:3 r:1 " +
			"E5:1 D5:.5 E5:.5 G5:1 E5:1 D5:.5 B4:.5 A4:1 B4:2 A4:.5 B4:.5 D5:1 E5:.5 D5:.5 B4:1 A4:4 " +
			"G4:.5 A4:.5 B4:1 D5:1 E5:1 G5:1.5 E5:.5 D5:2 E5:.5 D5:.5 B4:1 A4:1 B4:1 D5:3 r:1 " +
			"E5:1 G5:1 E5:1 D5:1 B4:.5 D5:.5 B4:1 A4:2 G4:.5 A4:.5 B4:.5 D5:.5 A4:1 B4:1 G4:4",
		chords: "G Em C D G Em C D Em C G D C Am D G"},
	{name: "Plum Blossom Rain", tempo: 64, lead: [2]int{evZither, evFlute},
		mel: "E5:2 D5:1 C5:1 A4:3 G4:1 A4:1 C5:1 D5:1 E5:1 D5:4 " +
			"G5:2 E5:1 D5:1 E5:1.5 D5:.5 C5:2 D5:1 C5:1 A4:1 G4:1 A4:4 " +
			"C5:1 D5:1 E5:2 G5:1 E5:1 D5:2 C5:1 A4:1 C5:1 D5:1 E5:4 " +
			"A5:2 G5:1 E5:1 D5:1.5 E5:.5 G5:2 E5:1 D5:1 C5:1 D5:1 A4:4",
		chords: "Am F C G Am F Dm Em F C Dm Em F G Dm Am"},
	{name: "Jade Pavilion", tempo: 84, lead: [2]int{evFlute, evZither}, busy: true,
		mel: "C5:1 D5:1 F5:2 D5:1 C5:1 A4:2 G4:1 A4:1 C5:1 D5:1 C5:4 " +
			"F5:1 G5:1 A5:1 G5:1 F5:1.5 D5:.5 C5:2 D5:1 F5:1 D5:1 C5:1 A4:4 " +
			"A4:1 C5:1 D5:1 F5:1 G5:2 F5:1 D5:1 C5:1 D5:.5 C5:.5 A4:1 G4:1 F4:2 G4:2 " +
			"A4:1 C5:1 D5:2 F5:1 D5:1 C5:1 A4:1 G4:1.5 A4:.5 C5:1 G4:1 F4:4",
		chords: "F Dm Bb C F Dm Gm C F Dm Bb C Dm Bb Csus F"},
}

var noteNum = map[string]int{"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3, "E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11}

// n2m: "F#4" -> 66
func n2m(n string) int {
	oct := n[len(n)-1] - '0'
	return noteNum[n[:len(n)-1]] + 12*(int(oct)+1)
}

// chordTones: root pitch class and intervals for "Bm", "Asus", "Bb"...
func chordTones(sym string) (int, []int) {
	r := 1
	if len(sym) > 1 && (sym[1] == '#' || sym[1] == 'b') {
		r = 2
	}
	root := noteNum[sym[:r]]
	switch q := sym[r:]; {
	case strings.HasPrefix(q, "sus"):
		return root, []int{0, 5, 7}
	case strings.HasPrefix(q, "m"):
		return root, []int{0, 3, 7}
	}
	return root, []int{0, 4, 7}
}

type melNote struct {
	m   int // -1 rest
	at  float64
	len float64
}

func parseMel(s string) []melNote {
	var out []melNote
	at := 0.0
	for _, tok := range strings.Fields(s) {
		p := strings.SplitN(tok, ":", 2)
		d, _ := strconv.ParseFloat(p[1], 64)
		m := -1
		if p[0] != "r" {
			m = n2m(p[0])
		}
		out = append(out, melNote{m, at, d})
		at += d
	}
	return out
}

// arrange turns a tune into timed events (seconds): twice through, then an ending.
func arrange(td tuneDef) ([]event, float64) {
	beat := 60 / td.tempo
	mel := parseMel(td.mel)
	chords := strings.Fields(td.chords)
	var ev []event
	bars := len(chords)
	for pass := 0; pass < 2; pass++ {
		off := float64(pass*bars*4) * beat
		lead := td.lead[pass]
		for _, n := range mel {
			if n.m < 0 {
				continue
			}
			e := event{t: off + n.at*beat, midi: float64(n.m), dur: n.len * beat * .92, vel: .15, pan: .1, kind: lead}
			if lead == evZither {
				e.vel, e.pan, e.dur = .9, -.15, math.Max(1.2, n.len*beat*1.5)
			}
			ev = append(ev, e)
		}
		for b, sym := range chords {
			t0 := off + float64(b*4)*beat
			root, iv := chordTones(sym)
			// pad: the chord around middle C, held for the bar
			for k, i := range iv {
				m := pcAbove(root+i, 55)
				ev = append(ev, event{t: t0, midi: float64(m), dur: 4 * beat * .96, vel: .06, pan: -.4 + .4*float64(k), kind: evPad})
			}
			// bass
			bm := pcAbove(root, 38)
			ev = append(ev, event{t: t0, midi: float64(bm), dur: 2.5, vel: .55, pan: 0, kind: evBass})
			if td.busy {
				ev = append(ev, event{t: t0 + 2*beat, midi: float64(bm + 7), dur: 1.5, vel: .35, pan: 0, kind: evBass})
			}
			// zither arpeggio
			arp := []int{0, iv[1], iv[2], 12, iv[1] + 12, 12, iv[2], iv[1]}
			step, n := beat/2, 8
			if !td.busy {
				step, n = beat, 4
				arp = []int{0, iv[2], 12, iv[1] + 12}
			}
			for k := 0; k < n; k++ {
				if !td.busy && k == 0 {
					continue
				}
				ev = append(ev, event{t: t0 + float64(k)*step, midi: float64(pcAbove(root, 55) + arp[k%len(arp)]), dur: 1.4, vel: .16, pan: .35, kind: evArp})
			}
			if td.busy {
				ev = append(ev, event{t: t0 + beat, kind: evBlock, vel: .18, pan: .5}, event{t: t0 + 3*beat, kind: evBlock, vel: .14, pan: .5})
				if b%2 == 0 {
					ev = append(ev, event{t: t0, kind: evDrum, vel: .45})
				}
			}
			if b%4 == 3 && pass == 1 {
				ev = append(ev, event{t: t0 + 3*beat, midi: float64(pcAbove(root, 79)), kind: evChime, vel: .07, pan: .6})
			}
		}
	}
	// ending: the last chord rolled, and a chime
	end := float64(2*bars*4) * beat
	root, iv := chordTones(chords[len(chords)-1])
	for k, i := range append([]int{0}, append(iv, 12, iv[1]+12)...) {
		ev = append(ev, event{t: end + float64(k)*.11, midi: float64(pcAbove(root, 50) + i), dur: 3, vel: .3, pan: -.3 + .15*float64(k), kind: evArp})
	}
	ev = append(ev, event{t: end, midi: float64(pcAbove(root, 38)), dur: 4, vel: .5, kind: evBass},
		event{t: end + .5, midi: float64(pcAbove(root, 74)), kind: evChime, vel: .08, pan: .4})
	for _, i := range iv {
		ev = append(ev, event{t: end, midi: float64(pcAbove(root+i, 55)), dur: 3, vel: .06, kind: evPad})
	}
	// sort by time (insertion sort is fine at this size)
	for i := 1; i < len(ev); i++ {
		for j := i; j > 0 && ev[j].t < ev[j-1].t; j-- {
			ev[j], ev[j-1] = ev[j-1], ev[j]
		}
	}
	return ev, end + 5
}

func pcAbove(pc, lo int) int { return lo + ((pc-lo)%12+12)%12 }

// play starts one event's voice. Caller holds mu.
func (s *Synth) play(e event, dt float64) {
	f := mtof(e.midi)
	switch e.kind {
	case evFlute:
		s.add(&flute{f: f, amp: e.vel, dur: e.dur, rng: s.rng}, dt, e.pan, true, .45)
	case evZither:
		s.add(newPluck(f, e.vel, e.dur, .75, s.rng), dt, e.pan, true, .35)
	case evArp:
		s.add(newPluck(f, e.vel, e.dur, .45, s.rng), dt, e.pan, true, .3)
	case evBass:
		s.add(newPluck(f, e.vel, e.dur, .2, s.rng), dt, e.pan, true, .15)
	case evPad:
		s.add(&pad{f: f, amp: e.vel, dur: e.dur}, dt, e.pan, true, .5)
	case evBlock:
		s.add(&wood{amp: e.vel, f: 1250, q: .985, life: .12, rng: s.rng}, dt, e.pan, true, .2)
	case evDrum:
		s.add(&drum{amp: e.vel, f: 82}, dt, e.pan, true, .25)
	case evChime:
		s.add(newBell(f, e.vel, 0), dt, e.pan, true, .6)
	}
}

// sequence advances the music by n samples, starting events that fall due.
func (s *Synth) sequence(n int) {
	dt := float64(n) / SR
	if s.tuneEvts == nil || s.tuneT >= s.tuneLen {
		s.nextTune()
	}
	t1 := s.tuneT + dt
	for s.tuneEvi < len(s.tuneEvts) && s.tuneEvts[s.tuneEvi].t < t1 {
		e := s.tuneEvts[s.tuneEvi]
		if e.t >= 0 {
			s.play(e, math.Max(0, e.t-s.tuneT))
		}
		s.tuneEvi++
	}
	if s.tuneT < 0 && t1 >= 0 && s.OnTune != nil {
		name := tuneDefs[s.tune].name
		go s.OnTune(name)
	}
	s.tuneT = t1
}

func (s *Synth) nextTune() {
	if len(s.order) == 0 {
		s.order = s.rng.Perm(len(tuneDefs))
		if s.order[0] == s.tune && len(s.order) > 1 && s.tuneEvts != nil {
			s.order[0], s.order[1] = s.order[1], s.order[0]
		}
	}
	s.tune, s.order = s.order[0], s.order[1:]
	s.tuneEvts, s.tuneLen = arrange(tuneDefs[s.tune])
	s.tuneEvi = 0
	s.tuneT = -3 // a breath between tunes
}

// StartMusic restarts with a new tune (when music is switched on).
func (s *Synth) StartMusic() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tuneEvts = nil
}

func (s *Synth) TuneName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tuneEvts == nil {
		return ""
	}
	return tuneDefs[s.tune].name
}

// ---------------------------------------------------------------- sound effects

type SFX struct{ s *Synth }

func (x SFX) do(fn func(s *Synth)) {
	x.s.mu.Lock()
	defer x.s.mu.Unlock()
	if x.s.sfxOn {
		fn(x.s)
	}
}

// clack: one tile knocking on another.
func (s *Synth) clack(t, v, pitch, pan float64) {
	s.add(&wood{amp: v, f: 2100 * pitch, q: .93, life: .07, rng: s.rng}, t, pan, false, .12)
	s.add(&wood{amp: v * .6, f: 640 * pitch, q: .96, life: .1, rng: s.rng}, t, pan, false, .1)
}

var pentaUp = []int{0, 2, 4, 7, 9, 12, 14, 16, 19, 21, 24}

func (x SFX) Select(pan float64) {
	x.do(func(s *Synth) { s.clack(0, .55, 1, pan) })
}
func (x SFX) Deselect(pan float64) {
	x.do(func(s *Synth) { s.clack(0, .3, .85, pan) })
}

// Match: two tiles click together and a chime rings, climbing with a streak.
func (x SFX) Match(streak int, pan float64) {
	x.do(func(s *Synth) {
		s.clack(0, .6, 1.1, pan)
		s.clack(.05, .45, 1.25, pan)
		m := 74 + pentaUp[min(streak, len(pentaUp)-1)]
		s.add(newBell(mtof(float64(m)), .22, 0), .06, pan, false, .5)
		s.add(newPluck(mtof(float64(m-12)), .3, 1.2, .7, s.rng), .06, -pan, false, .3)
	})
}

func (x SFX) Blocked() {
	x.do(func(s *Synth) {
		s.add(&wood{amp: .5, f: 300, q: .97, life: .15, rng: s.rng, thud: true}, 0, 0, false, .05)
	})
}

func (x SFX) Hint() {
	x.do(func(s *Synth) {
		s.add(newBell(mtof(86), .16, 2), 0, -.3, false, .6)
		s.add(newBell(mtof(93), .14, 2), .14, .3, false, .6)
	})
}

func (x SFX) Shuffle() {
	x.do(func(s *Synth) {
		for i := 0; i < 14; i++ {
			s.clack(float64(i)*.045+s.rng.Float64()*.02, .25+.2*s.rng.Float64(), .8+.5*s.rng.Float64(), s.rng.Float64()*1.4-.7)
		}
		s.add(newPluck(mtof(62), .3, 1.5, .6, s.rng), .65, 0, false, .4)
		s.add(newPluck(mtof(69), .3, 1.5, .6, s.rng), .72, 0, false, .4)
	})
}

func (x SFX) Undo() {
	x.do(func(s *Synth) {
		s.clack(0, .4, .9, -.2)
		s.add(newPluck(mtof(67), .22, .8, .5, s.rng), .03, 0, false, .3)
		s.add(newPluck(mtof(62), .22, .8, .5, s.rng), .1, 0, false, .3)
	})
}

func (x SFX) Click() {
	x.do(func(s *Synth) { s.add(&wood{amp: .25, f: 2600, q: .9, life: .04, rng: s.rng}, 0, 0, false, .05) })
}

func (x SFX) Start() {
	x.do(func(s *Synth) {
		for k, m := range []int{62, 64, 66, 69, 71, 74} {
			s.add(newPluck(mtof(float64(m)), .35, 1.6, .65, s.rng), float64(k)*.07, -.5+.2*float64(k), false, .4)
		}
	})
}

func (x SFX) Stuck() {
	x.do(func(s *Synth) {
		s.add(newPluck(mtof(69), .4, 1.4, .5, s.rng), 0, 0, false, .4)
		s.add(newPluck(mtof(64), .4, 1.6, .5, s.rng), .22, 0, false, .4)
		s.add(newPluck(mtof(57), .4, 2, .5, s.rng), .44, 0, false, .4)
	})
}

// Win: a soft gong, then a rising run on the zither and a chime on top.
func (x SFX) Win() {
	x.do(func(s *Synth) {
		s.add(newBell(mtof(45), .5, 1), 0, 0, false, .7)
		for k, i := range []int{0, 2, 4, 7, 9, 12, 14, 16, 19, 21, 24} {
			s.add(newPluck(mtof(float64(62+i)), .42, 2, .7, s.rng), .35+float64(k)*.085, -.6+.12*float64(k), false, .45)
		}
		s.add(newBell(mtof(86), .2, 0), 1.35, .3, false, .7)
		s.add(newBell(mtof(93), .18, 0), 1.55, -.3, false, .7)
	})
}
