package main

import (
	"math"
	"testing"
	"time"
)

func TestSynth(t *testing.T) {
	for i := range tuneDefs {
		td := tuneDefs[i]
		mel := parseMel(td.mel)
		last := mel[len(mel)-1]
		if beats := last.at + last.len; beats != 64 {
			t.Errorf("%s: melody is %.1f beats, want 64", td.name, beats)
		}
		if n := len(splitFields(td.chords)); n != 16 {
			t.Errorf("%s: %d chords", td.name, n)
		}
	}
	s := NewSynth()
	s.SetLevels(true, true, 100)
	x := SFX{s}
	buf := make([]float32, 2*1024)
	start := time.Now()
	peak, sum, n := 0.0, 0.0, 0
	for blk := 0; blk < 30*SR/1024; blk++ {
		if blk%40 == 0 {
			x.Match(blk/40, .3)
			x.Select(0)
		}
		if blk == 200 {
			x.Win()
			x.Shuffle()
		}
		s.Mix(buf)
		for _, v := range buf {
			f := float64(v)
			if math.IsNaN(f) {
				t.Fatal("NaN")
			}
			peak = math.Max(peak, math.Abs(f))
			sum += f * f
			n++
		}
	}
	el := time.Since(start)
	rms := math.Sqrt(sum / float64(n))
	t.Logf("30s rendered in %v, peak %.2f rms %.3f, tune %q", el, peak, rms, s.TuneName())
	if rms < .01 || peak > 1 {
		t.Fatalf("levels: rms %.3f peak %.2f", rms, peak)
	}
	if el > 10*time.Second {
		t.Fatalf("too slow: %v for 30s", el)
	}
}

func splitFields(s string) []string {
	var out []string
	cur := ""
	for _, r := range s + " " {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
		} else {
			cur += string(r)
		}
	}
	return out
}
