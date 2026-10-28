package eq

import (
	"math"
	"testing"

	"github.com/faiface/beep"
)

type staticStreamer struct {
	samples [][2]float64
	pos     int
}

func (s *staticStreamer) Stream(samples [][2]float64) (int, bool) {
	n := 0
	for n < len(samples) && s.pos < len(s.samples) {
		samples[n] = s.samples[s.pos]
		n++
		s.pos++
	}
	return n, s.pos < len(s.samples)
}

func (s *staticStreamer) Err() error { return nil }

func sine(freq, sampleRate float64, n int) [][2]float64 {
	out := make([][2]float64, n)
	for i := range out {
		v := 0.5 * math.Sin(2*math.Pi*freq*float64(i)/sampleRate)
		out[i] = [2]float64{v, v}
	}
	return out
}

func TestEqualizerBypassAtZeroGain(t *testing.T) {
	const testSampleRate = beep.SampleRate(44100)
	sr := 44100.0
	src := sine(1000, sr, 4096)
	s := &staticStreamer{samples: src}
	eq := NewEqualizer(s, testSampleRate)

	out := make([][2]float64, len(src))
	n, _ := eq.Stream(out)
	for i := 0; i < n; i++ {
		if math.Abs(out[i][0]-src[i][0]) > 1e-9 || math.Abs(out[i][1]-src[i][1]) > 1e-9 {
			t.Fatalf("muestra %d difiere en bypass: got %v want %v", i, out[i], src[i])
		}
	}
}

func TestEqualizerAttenuatesBand(t *testing.T) {
	const testSampleRate = beep.SampleRate(44100)
	sr := 44100.0
	src := sine(60, sr, 8192)
	s := &staticStreamer{samples: src}
	eq := NewEqualizer(s, testSampleRate)
	eq.SetGain(0, GainMin)

	out := make([][2]float64, len(src))
	eq.Stream(out)

	var inAmp, outAmp float64
	for i := 2048; i < len(src); i++ {
		inAmp = max(inAmp, math.Abs(src[i][0]))
		outAmp = max(outAmp, math.Abs(out[i][0]))
	}
	if outAmp >= inAmp*0.5 {
		t.Fatalf("se esperaba atenuación fuerte: in=%f out=%f", inAmp, outAmp)
	}
}

func TestEqualizerDisabledPassthrough(t *testing.T) {
	const testSampleRate = beep.SampleRate(44100)
	sr := 44100.0
	src := sine(60, sr, 2048)
	s := &staticStreamer{samples: src}
	eq := NewEqualizer(s, testSampleRate)
	eq.SetGain(0, GainMax)
	eq.Enabled = false

	out := make([][2]float64, len(src))
	eq.Stream(out)
	for i := range src {
		if out[i] != src[i] {
			t.Fatalf("con Enabled=false debe ser bypass exacto")
		}
	}
}

func TestSetGainClamps(t *testing.T) {
	eq := NewEqualizer(&staticStreamer{}, beep.SampleRate(44100))
	eq.SetGain(2, 100)
	if eq.Gains[2] != GainMax {
		t.Fatalf("esperaba clamp a %v, obtuve %v", GainMax, eq.Gains[2])
	}
	eq.SetGain(2, -100)
	if eq.Gains[2] != GainMin {
		t.Fatalf("esperaba clamp a %v, obtuve %v", GainMin, eq.Gains[2])
	}
	eq.SetGain(-1, 5)
	eq.SetGain(BandCount, 5)
}
