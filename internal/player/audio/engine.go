package audio

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/fonta81/Reproductor-go/internal/player/config"
	"github.com/fonta81/Reproductor-go/internal/player/eq"
	"github.com/fonta81/Reproductor-go/internal/player/model"
)

const standardSampleRate = beep.SampleRate(44100)

type Limiter struct {
	Streamer beep.Streamer
}

func (l *Limiter) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = l.Streamer.Stream(samples)
	for i := range samples[:n] {
		for ch := 0; ch < 2; ch++ {
			if samples[i][ch] > 1.0 {
				samples[i][ch] = 1.0
			} else if samples[i][ch] < -1.0 {
				samples[i][ch] = -1.0
			}
		}
	}
	return n, ok
}

func (l *Limiter) Err() error {
	if se, ok := l.Streamer.(interface{ Err() error }); ok {
		return se.Err()
	}
	return nil
}

type AudioEngine struct {
	streamer    beep.StreamSeekCloser
	ctrl        *beep.Ctrl
	volume      *effects.Volume
	eq          *eq.Equalizer
	format      beep.Format
	isInit      bool
	sessionID   int
	cancelChan  chan struct{}
	sessionMu   sync.Mutex
	lifecycleMu sync.Mutex
	playbackMu  sync.Mutex
	statusMu    sync.RWMutex
	hasEQ       bool
	eqEnabled   bool
	eqGains     [eq.BandCount]float64
	muted       bool
}

func NewAudioEngine() *AudioEngine { return &AudioEngine{} }

func (ae *AudioEngine) Load(track model.Track) (time.Duration, error) {
	ae.lifecycleMu.Lock()
	defer ae.lifecycleMu.Unlock()

	ae.Stop()

	file, err := os.Open(track.Path)
	if err != nil {
		return 0, fmt.Errorf("abrir archivo: %w", err)
	}

	var streamer beep.StreamSeekCloser
	var format beep.Format
	ext := strings.ToLower(filepath.Ext(track.Path))

	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(file)
	case ".wav":
		streamer, format, err = wav.Decode(file)
	case ".flac":
		streamer, format, err = flac.Decode(file)
	case ".ogg":
		streamer, format, err = vorbis.Decode(file)
	default:
		_ = file.Close()
		return 0, fmt.Errorf("formato no soportado: %s", ext)
	}

	if err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("error al decodificar %s: %w", track.Path, err)
	}

	realDuration := format.SampleRate.D(streamer.Len())

	if !ae.isInit {
		if err := speaker.Init(standardSampleRate, standardSampleRate.N(time.Second/10)); err != nil {
			_ = streamer.Close()
			_ = file.Close()
			return 0, fmt.Errorf("error al inicializar speaker: %w", err)
		}
		ae.isInit = true
	}

	resampled := beep.Resample(4, format.SampleRate, standardSampleRate, streamer)
	ctrl := &beep.Ctrl{Streamer: resampled}
	eqFilter := eq.NewEqualizer(ctrl, standardSampleRate)

	if enabled, gains, err := config.LoadEQ(); err == nil {
		eqFilter.Enabled = enabled
		for i := range gains {
			eqFilter.SetGain(i, gains[i])
		}
	}

	volume := &effects.Volume{Streamer: eqFilter, Base: math.Pow(10, 1.0/20.0), Volume: 0, Silent: false}

	speaker.Lock()
	ae.streamer = streamer
	ae.format = format
	ae.ctrl = ctrl
	ae.eq = eqFilter
	ae.volume = volume
	ae.statusMu.Lock()
	ae.hasEQ = true
	ae.eqEnabled = eqFilter.Enabled
	ae.eqGains = eqFilter.Gains
	ae.muted = volume.Silent
	ae.statusMu.Unlock()
	speaker.Unlock()

	return realDuration, nil
}

func (ae *AudioEngine) Play() chan struct{} {
	ae.playbackMu.Lock()
	defer ae.playbackMu.Unlock()

	done := make(chan struct{})
	speaker.Lock()
	volume := ae.volume
	speaker.Unlock()
	if volume == nil {
		close(done)
		return done
	}

	limiter := &Limiter{Streamer: volume}
	speaker.Play(beep.Seq(limiter, beep.Callback(func() { close(done) })))
	return done
}

func (ae *AudioEngine) Stop() {
	ae.playbackMu.Lock()
	defer ae.playbackMu.Unlock()

	speaker.Clear()
	ae.cancelPlaybackWait()
	speaker.Lock()
	streamer := ae.streamer
	ae.streamer = nil
	ae.ctrl = nil
	ae.volume = nil
	ae.eq = nil
	ae.statusMu.Lock()
	ae.hasEQ = false
	ae.eqEnabled = false
	ae.eqGains = [eq.BandCount]float64{}
	ae.muted = false
	ae.statusMu.Unlock()
	speaker.Unlock()

	if streamer != nil {
		_ = streamer.Close()
	}
}

func (ae *AudioEngine) Close() {
	ae.lifecycleMu.Lock()
	defer ae.lifecycleMu.Unlock()

	ae.Stop()
	if ae.isInit {
		ae.playbackMu.Lock()
		speaker.Close()
		ae.playbackMu.Unlock()
		ae.isInit = false
	}
}

func (ae *AudioEngine) startPlaybackSession() (int, <-chan struct{}) {
	ae.sessionMu.Lock()
	defer ae.sessionMu.Unlock()

	if ae.cancelChan != nil {
		close(ae.cancelChan)
	}
	ae.sessionID++
	ae.cancelChan = make(chan struct{})
	return ae.sessionID, ae.cancelChan
}

func (ae *AudioEngine) PlaybackSessionID() int {
	ae.sessionMu.Lock()
	defer ae.sessionMu.Unlock()
	return ae.sessionID
}

func (ae *AudioEngine) playbackSessionID() int { return ae.PlaybackSessionID() }

func (ae *AudioEngine) StartPlaybackSession() (int, <-chan struct{}) {
	return ae.startPlaybackSession()
}

func (ae *AudioEngine) cancelPlaybackWait() {
	ae.sessionMu.Lock()
	defer ae.sessionMu.Unlock()

	if ae.cancelChan != nil {
		close(ae.cancelChan)
		ae.cancelChan = nil
	}
}

func (ae *AudioEngine) CancelPlaybackWait() { ae.cancelPlaybackWait() }
