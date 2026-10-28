package player

import (
	"fmt"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

// Pause pausa la reproducción actual de forma segura frente al hilo de audio.
func (ae *AudioEngine) Pause() {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.ctrl != nil {
		ae.ctrl.Paused = true
	}
}

// Resume reanuda la reproducción pausada.
func (ae *AudioEngine) Resume() {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.ctrl != nil {
		ae.ctrl.Paused = false
	}
}

// SetVolume ajusta la ganancia en decibelios del stream activo.
func (ae *AudioEngine) SetVolume(level float64) {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.volume != nil {
		ae.volume.Volume = level
	}
}

// ToggleMute conmuta el silencio sin alterar el nivel configurado.
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

// SetEQBand ajusta la ganancia de una banda del ecualizador.
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

// GetEQBands devuelve una instantánea de las ganancias actuales por banda.
func (ae *AudioEngine) GetEQBands() [eqBandCount]float64 {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.eqGains
}

// ToggleEQ activa o desactiva el ecualizador y devuelve el nuevo estado.
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

// HasEQ indica si hay un ecualizador inicializado.
func (ae *AudioEngine) HasEQ() bool {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.hasEQ
}

// EQEnabled indica si el ecualizador está activo.
func (ae *AudioEngine) EQEnabled() bool {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.eqEnabled
}

// IsMuted indica si el stream se encuentra silenciado.
func (ae *AudioEngine) IsMuted() bool {
	ae.statusMu.RLock()
	defer ae.statusMu.RUnlock()
	return ae.muted
}

// Position devuelve el tiempo transcurrido de reproducción.
func (ae *AudioEngine) Position() time.Duration {
	speaker.Lock()
	defer speaker.Unlock()
	if ae.streamer != nil {
		return ae.format.SampleRate.D(ae.streamer.Position())
	}
	return 0
}

// Seek desplaza la lectura del stream a la posición indicada.
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
