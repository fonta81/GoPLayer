package eq

import (
	"math"

	"github.com/faiface/beep"
)

const (
	BandCount   = 5
	Q           = 1.0
	GainMin     = -12.0
	GainMax     = 12.0
	GainStep    = 1.0
	DefaultGain = 0.0
)

var Frequencies = [BandCount]float64{60, 250, 1000, 4000, 10000}

type biquadCoeffs struct {
	b0, b1, b2, a1, a2 float64
}

func peakingCoeffs(freq, sampleRate, q, gainDB float64) biquadCoeffs {
	A := math.Pow(10, gainDB/40.0)
	w0 := 2 * math.Pi * freq / sampleRate
	alpha := math.Sin(w0) / (2 * q)
	cosW0 := math.Cos(w0)

	a0 := 1 + alpha/A
	return biquadCoeffs{
		b0: (1 + alpha*A) / a0,
		b1: (-2 * cosW0) / a0,
		b2: (1 - alpha*A) / a0,
		a1: (-2 * cosW0) / a0,
		a2: (1 - alpha/A) / a0,
	}
}

type biquadState struct {
	x1, x2, y1, y2 float64
}

func (s *biquadState) process(c biquadCoeffs, x float64) float64 {
	y := c.b0*x + c.b1*s.x1 + c.b2*s.x2 - c.a1*s.y1 - c.a2*s.y2
	s.x2, s.x1 = s.x1, x
	s.y2, s.y1 = s.y1, y
	return y
}

type Equalizer struct {
	Streamer   beep.Streamer
	SampleRate float64
	Gains      [BandCount]float64
	Enabled    bool

	coeffs [BandCount]biquadCoeffs
	states [BandCount][2]biquadState
}

func NewEqualizer(s beep.Streamer, sampleRate beep.SampleRate) *Equalizer {
	eq := &Equalizer{Streamer: s, SampleRate: float64(sampleRate), Enabled: true}
	for i := range eq.Gains {
		eq.Gains[i] = DefaultGain
	}
	eq.recalc()
	return eq
}

func (eq *Equalizer) recalc() {
	for i := range eq.coeffs {
		eq.coeffs[i] = peakingCoeffs(Frequencies[i], eq.SampleRate, Q, eq.Gains[i])
	}
}

func (eq *Equalizer) SetGain(band int, gainDB float64) {
	if band < 0 || band >= BandCount {
		return
	}
	eq.Gains[band] = max(GainMin, min(GainMax, gainDB))
	eq.coeffs[band] = peakingCoeffs(Frequencies[band], eq.SampleRate, Q, eq.Gains[band])
}

func (eq *Equalizer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = eq.Streamer.Stream(samples)
	if !eq.Enabled {
		return n, ok
	}
	for i := range samples[:n] {
		for ch := 0; ch < 2; ch++ {
			v := samples[i][ch]
			for b := range eq.coeffs {
				v = eq.states[b][ch].process(eq.coeffs[b], v)
			}
			samples[i][ch] = v
		}
	}
	return n, ok
}

func (eq *Equalizer) Err() error {
	if se, ok := eq.Streamer.(interface{ Err() error }); ok {
		return se.Err()
	}
	return nil
}
