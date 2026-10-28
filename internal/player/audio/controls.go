package audio

import (
	"fmt"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/fonta81/Reproductor-go/internal/player/eq"
)

func (ae *AudioEngine) Pause() {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.ctrl != nil {
		ae.ctrl.Paused = true
	}
}

func (ae *AudioEngine) Resume() {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.ctrl != nil {
		ae.ctrl.Paused = false
	}
}

func (ae *AudioEngine) SetVolume(level float64) {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.volume != nil {
		ae.volume.Volume = level
	}
}

func (ae *AudioEngine) ToggleMute() {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.volume != nil {
		ae.volume.Silent = !ae.volume.Silent
		ae.statusMu.Lock()
		ae.muted = ae.volume.Silent
		ae.statusMu.Unlock()
	}
}

func (ae *AudioEngine) SetEQBand(band int, gainDB float64) {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.eq != nil {
		ae.eq.SetGain(band, gainDB)
		ae.statusMu.Lock()
		ae.eqGains = ae.eq.Gains
		ae.statusMu.Unlock()
	}
}

func (ae *AudioEngine) GetEQBands() [eq.BandCount]float64 {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.eqGains
}

func (ae *AudioEngine) ToggleEQ() bool {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.eq == nil {
		return false
	}
	ae.eq.Enabled = !ae.eq.Enabled
	ae.statusMu.Lock()
	ae.eqEnabled = ae.eq.Enabled
	ae.statusMu.Unlock()
	return ae.eq.Enabled
}

func (ae *AudioEngine) HasEQ() bool {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.hasEQ
}

func (ae *AudioEngine) EQEnabled() bool {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.eqEnabled
}

func (ae *AudioEngine) IsMuted() bool {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.muted
}

func (ae *AudioEngine) Position() time.Duration {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.streamer != nil {
		return ae.format.SampleRate.D(ae.streamer.Position())
	}
	return 0
}

func (ae *AudioEngine) Seek(position time.Duration) error {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.streamer == nil {
		return fmt.Errorf("no hay stream activo")
	}

	seeker, ok := ae.streamer.(beep.StreamSeeker)
	if !ok {
		return fmt.Errorf("formato no permite seek")
	}

	samples := ae.format.SampleRate.N(position)
	if err := seeker.Seek(samples); err != nil {
		return fmt.Errorf("seek fallido: %w", err)
	}
	return nil
}
