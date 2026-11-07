// The synthesizer: plucked zither, bamboo flute, warm pad, bells, wood and a
// small reverb, all made live. Music and sound effects. (c) 2026 Ironite
package main

import (
	"math"
	"math/rand"
	"sync"
)

const SR = 44100

func mtof(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

// ---------------------------------------------------------------- voices

type voice interface {
	next() (float32, bool) // one sample; false when finished
}

// pluck is a Karplus-Strong string with a little body: zither and pipa sounds.
type pluck struct {
	buf    []float32
	i      int
	decay  float32
	bright float32
	amp    float32
	age    int
	life   int
	last   float32
}

func newPluck(freq, vel, dur, bright float64, rng *rand.Rand) *pluck {
	n := int(SR / freq)
	if n < 2 {
		n = 2
	}
	p := &pluck{buf: make([]float32, n), decay: float32(math.Pow(.001, 1/(dur*freq))), bright: float32(bright), amp: float32(vel), life: int(dur * SR)}
	// excite with filtered noise: softer than white for a gentler pluck
	var lp float32
	for i := range p.buf {
		w := rng.Float32()*2 - 1
		lp += (w - lp) * float32(.35+.5*bright)
		p.buf[i] = lp
	}
	return p
}

func (p *pluck) next() (float32, bool) {
	j := (p.i + 1) % len(p.buf)
	v := p.buf[p.i]
	avg := (v*p.bright + p.buf[j]*(2-p.bright)) * .5
	p.buf[p.i] = avg * p.decay
	p.i = j
	p.age++
	// gentle fade at the end of its life
	g := float32(1)
	if rem := p.life - p.age; rem < 2000 {
		g = float32(rem) / 2000
	}
	p.last += (v - p.last) * .6
	return p.last * p.amp * g, p.age < p.life
}

// flute: sine with a touch of second harmonic, breath noise and vibrato.
type flute struct {
	ph, f, amp, dur, t float64
	noise              float64
	rng                *rand.Rand
}

func (v *flute) next() (float32, bool) {
	v.t += 1.0 / SR
	att, rel := .09, .18
	env := 1.0
	switch {
	case v.t < att:
		env = v.t / att
	case v.t > v.dur:
		env = 1 - (v.t-v.dur)/rel
	}
	if env <= 0 {
		return 0, false
	}
	vib := 0.0
	if v.t > .25 {
		vib = math.Min(1, (v.t-.25)/.4) * .0045 * math.Sin(2*math.Pi*5.2*v.t)
	}
	v.ph += v.f * (1 + vib) / SR
	s := math.Sin(2*math.Pi*v.ph) + .18*math.Sin(4*math.Pi*v.ph) + .05*math.Sin(6*math.Pi*v.ph)
	v.noise += (v.rng.Float64()*2 - 1 - v.noise) * .08
	breath := v.noise * (.35 + .65*math.Max(0, 1-v.t*6))
	return float32((s*.85 + breath*.5) * env * v.amp), true
}

// pad: detuned soft triangles with a slow swell.
type pad struct {
	ph  [3]float64
	f   float64
	amp float64
	dur float64
	t   float64
}

func tri(ph float64) float64 {
	x := ph - math.Floor(ph)
	return 4*math.Abs(x-.5) - 1
}

func (v *pad) next() (float32, bool) {
	v.t += 1.0 / SR
	att, rel := .8, 1.2
	env := math.Min(1, v.t/att)
	if v.t > v.dur {
		env *= 1 - (v.t-v.dur)/rel
	}
	if env <= 0 {
		return 0, false
	}
	s := 0.0
	for k, d := range []float64{1, 1.0035, .9968} {
		v.ph[k] += v.f * d / SR
		s += tri(v.ph[k])
	}
	return float32(s / 3 * env * v.amp), true
}

// bell: a few inharmonic partials, each decaying at its own rate.
type bell struct {
	parts [][3]float64 // ratio, amp, decay time
	ph    []float64
	f     float64
	amp   float64
	t     float64
	life  float64
}

func newBell(f, amp float64, kind int) *bell {
	b := &bell{f: f, amp: amp}
	switch kind {
	case 0: // small chime
		b.parts = [][3]float64{{1, 1, 1.4}, {2.76, .45, .6}, {5.4, .25, .3}, {8.93, .1, .15}}
	case 1: // gong: low, long, shimmering
		b.parts = [][3]float64{{1, 1, 4.5}, {1.49, .5, 3.5}, {2.03, .45, 2.8}, {2.98, .3, 2}, {4.1, .2, 1.2}, {.5, .5, 5}}
	case 2: // glass ping for the hint
		b.parts = [][3]float64{{1, 1, .9}, {3, .3, .4}, {4.2, .15, .25}}
	}
	b.ph = make([]float64, len(b.parts))
	for _, p := range b.parts {
		b.life = math.Max(b.life, p[2]*5)
	}
	return b
}

func (b *bell) next() (float32, bool) {
	b.t += 1.0 / SR
	if b.t > b.life {
		return 0, false
	}
	s := 0.0
	for i, p := range b.parts {
		b.ph[i] += b.f * p[0] / SR
		s += math.Sin(2*math.Pi*b.ph[i]) * p[1] * math.Exp(-b.t/p[2])
	}
	att := math.Min(1, b.t/.003)
	return float32(s * b.amp * att * .5), true
}

// wood: the click of a tile (resonant noise burst) or a wood block.
type wood struct {
	t, amp, f, q float64
	y1, y2       float64
	life         float64
	rng          *rand.Rand
	thud         bool
}

func (w *wood) next() (float32, bool) {
	w.t += 1.0 / SR
	if w.t > w.life {
		return 0, false
	}
	// two-pole resonator excited by a short noise burst
	x := 0.0
	if w.t < .004 {
		x = (w.rng.Float64()*2 - 1) * (1 - w.t/.004)
	}
	r := w.q
	om := 2 * math.Pi * w.f / SR
	y := x + 2*r*math.Cos(om)*w.y1 - r*r*w.y2
	w.y2, w.y1 = w.y1, y
	env := math.Exp(-w.t / (w.life / 4))
	out := y * (1 - r) * 6 * env
	if w.thud {
		out += math.Sin(2*math.Pi*90*w.t*(1-w.t*3)) * math.Exp(-w.t/.05) * .8
	}
	return float32(out * w.amp), true
}

// drum: a soft low tom.
type drum struct{ t, amp, f float64 }

func (d *drum) next() (float32, bool) {
	d.t += 1.0 / SR
	if d.t > .7 {
		return 0, false
	}
	f := d.f * (1 + .6*math.Exp(-d.t/.04))
	return float32(math.Sin(2*math.Pi*f*d.t) * math.Exp(-d.t/.18) * d.amp), true
}

// ---------------------------------------------------------------- reverb

type comb struct {
	buf      []float32
	i        int
	fb, damp float32
	store    float32
}

func (c *comb) proc(x float32) float32 {
	y := c.buf[c.i]
	c.store = y*(1-c.damp) + c.store*c.damp
	c.buf[c.i] = x + c.store*c.fb
	c.i = (c.i + 1) % len(c.buf)
	return y
}

type allpass struct {
	buf []float32
	i   int
}

func (a *allpass) proc(x float32) float32 {
	b := a.buf[a.i]
	y := -x + b
	a.buf[a.i] = x + b*.5
	a.i = (a.i + 1) % len(a.buf)
	return y
}

type reverb struct {
	cl, cr []*comb
	al, ar []*allpass
}

func newReverb() *reverb {
	r := &reverb{}
	for i, n := range []int{1116, 1188, 1277, 1356} {
		r.cl = append(r.cl, &comb{buf: make([]float32, n), fb: .84, damp: .3})
		r.cr = append(r.cr, &comb{buf: make([]float32, n+23+i), fb: .84, damp: .3})
	}
	for _, n := range []int{556, 441} {
		r.al = append(r.al, &allpass{buf: make([]float32, n)})
		r.ar = append(r.ar, &allpass{buf: make([]float32, n+23)})
	}
	return r
}

func (r *reverb) proc(x float32) (float32, float32) {
	var l, rr float32
	for i := range r.cl {
		l += r.cl[i].proc(x)
		rr += r.cr[i].proc(x)
	}
	for i := range r.al {
		l = r.al[i].proc(l)
		rr = r.ar[i].proc(rr)
	}
	return l * .25, rr * .25
}

// ---------------------------------------------------------------- synth

type playing struct {
	v       voice
	panL    float32
	panR    float32
	start   int64 // sample at which it starts
	music   bool
	reverb  float32
	started bool
}

type Synth struct {
	mu      sync.Mutex
	now     int64
	voices  []*playing
	rev     *reverb
	rng     *rand.Rand
	musicOn bool
	sfxOn   bool
	musicG  float32
	sfxG    float32
	master  float32
	// sequencer
	tune     int
	tuneT    float64 // seconds into the current tune (-gap while waiting)
	tuneEvts []event
	tuneEvi  int
	tuneLen  float64
	order    []int
	OnTune   func(name string)
	lastRms  float64
}

func NewSynth() *Synth {
	s := &Synth{rev: newReverb(), rng: rand.New(rand.NewSource(7)), sfxOn: true, musicG: .55, sfxG: .9, master: .7}
	return s
}

func (s *Synth) Now() float64 { return float64(s.now) / SR }

// SetLevels applies the music/sfx switches and the volume (0-100).
func (s *Synth) SetLevels(music, sfx bool, vol float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sfxOn = sfx
	s.master = float32(math.Pow(vol/100, 1.6))
	if !music && s.musicOn {
		s.musicOn = false
		s.silenceMusic()
	} else if music && !s.musicOn {
		s.musicOn = true
		s.tuneT = -1.5
	}
}

func (s *Synth) silenceMusic() {
	out := s.voices[:0]
	for _, p := range s.voices {
		if !p.music {
			out = append(out, p)
		}
	}
	s.voices = out
}

// add schedules a voice at time t seconds from now. pan -1..1. Caller holds mu.
func (s *Synth) add(v voice, t, pan float64, music bool, rev float64) {
	a := (pan + 1) * math.Pi / 4
	s.voices = append(s.voices, &playing{v: v, start: s.now + int64(t*SR), panL: float32(math.Cos(a)), panR: float32(math.Sin(a)), music: music, reverb: float32(rev)})
}

// Mix renders interleaved stereo float32 samples.
func (s *Synth) Mix(out []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	frames := len(out) / 2
	for f := 0; f < frames; f++ {
		if f%64 == 0 && s.musicOn {
			s.sequence(64)
		}
		var l, r, send float32
		for _, p := range s.voices {
			if p.v == nil || p.start > s.now {
				continue
			}
			v, ok := p.v.next()
			if !ok {
				p.v = nil
				continue
			}
			g := s.sfxG
			if p.music {
				g = s.musicG
			}
			v *= g
			l += v * p.panL
			r += v * p.panR
			send += v * p.reverb
		}
		rl, rr := s.rev.proc(send)
		l = (l + rl) * s.master
		r = (r + rr) * s.master
		out[f*2] = softClip(l)
		out[f*2+1] = softClip(r)
		s.now++
		if f%256 == 255 {
			s.prune()
		}
	}
	s.prune()
}

func (s *Synth) prune() {
	out := s.voices[:0]
	for _, p := range s.voices {
		if p.v != nil {
			out = append(out, p)
		}
	}
	for i := len(out); i < len(s.voices); i++ {
		s.voices[i] = nil
	}
	s.voices = out
}

func softClip(x float32) float32 { return float32(math.Tanh(float64(x))) }
