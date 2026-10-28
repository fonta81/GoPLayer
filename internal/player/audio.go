// Package player proporciona la abstracción y control del motor de audio de bajo nivel
// sobre la biblioteca beep y el backend de sonido del sistema operativo.
package player

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
)

// Limiter es un streamer envoltorio que acota las muestras de audio en el rango [-1.0, 1.0].
// Su propósito es prevenir la saturación acústica y distorsión por recorte (clipping)
// cuando se aplican ganancias altas de volumen.
type Limiter struct {
	Streamer beep.Streamer
}

// Stream procesa las muestras del streamer subyacente y restringe la amplitud de cada canal
// (estéreo: canal 0 izquierdo, canal 1 derecho) al intervalo [-1.0, 1.0].
func (l *Limiter) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = l.Streamer.Stream(samples)
	for i := range samples[:n] {
		for ch := 0; ch < 2; ch++ { // Canales Izquierdo (0) y Derecho (1)
			if samples[i][ch] > 1.0 {
				samples[i][ch] = 1.0
			} else if samples[i][ch] < -1.0 {
				samples[i][ch] = -1.0
			}
		}
	}
	return n, ok
}

// Err propaga cualquier error generado por el Streamer subyacente si este implementa
// la interfaz de comprobación de errores.
func (l *Limiter) Err() error {
	if se, ok := l.Streamer.(interface{ Err() error }); ok {
		return se.Err()
	}
	return nil
}

// AudioEngine centraliza la carga, decodificación, efectos y reproducción de archivos de audio.
// Controla el ciclo de vida del altavoz (speaker) y la sincronización segura frente a accesos concurrentes.
type AudioEngine struct {
	streamer    beep.StreamSeekCloser // Decodificador activo del archivo de audio con capacidad de seek y cierre
	ctrl        *beep.Ctrl            // Controlador para pausar y reanudar el flujo de datos
	volume      *effects.Volume       // Efecto de ganancia para control de volumen logarítmico
	eq          *Equalizer            // Ecualizador paramétrico de 5 bandas
	format      beep.Format           // Formato original del archivo (frecuencia de muestreo y precisión)
	isInit      bool                  // Bandera para asegurar inicialización única del altavoz
	sessionID   int                   // Identificador de la sesión de reproducción actual (invalida eventos de pistas pasadas)
	cancelChan  chan struct{}         // Canal para abortar de forma limpia la espera de término si se salta de pista
	sessionMu   sync.Mutex
	lifecycleMu sync.Mutex
	playbackMu  sync.Mutex
	statusMu    sync.RWMutex
	hasEQ       bool
	eqEnabled   bool
	eqGains     [eqBandCount]float64
	muted       bool
}

// NewAudioEngine inicializa y devuelve una nueva instancia del motor de audio.
func NewAudioEngine() *AudioEngine {
	return &AudioEngine{}
}

// Load detiene cualquier reproducción en curso, abre el archivo en disco, detecta su formato
// por extensión (.mp3, .wav, .flac, .ogg), decodifica el stream, lo remuestrea a la frecuencia
// estándar de 44.1 kHz y configura la cadena de efectos (control de pausa y volumen).
// Devuelve la duración real del archivo o un error en caso de fallo.
func (ae *AudioEngine) Load(track Track) (time.Duration, error) {
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

	// Inicializar el subsistema del speaker una sola vez con la frecuencia estándar y un buffer de 100ms
	if !ae.isInit {
		if err := speaker.Init(standardSampleRate, standardSampleRate.N(time.Second/10)); err != nil {
			_ = streamer.Close()
			_ = file.Close()
			return 0, fmt.Errorf("error al inicializar speaker: %w", err)
		}
		ae.isInit = true
	}

	// Remuestreo de alta calidad a 44100 Hz para evitar desajustes de velocidad en archivos con diferente frecuencia
	resampled := beep.Resample(4, format.SampleRate, standardSampleRate, streamer)

	ctrl := &beep.Ctrl{Streamer: resampled}
	eq := NewEqualizer(ctrl, standardSampleRate)

	// Restaurar los ajustes de EQ guardados en la configuración de usuario,
	// si existen, para conservarlos entre sesiones.
	if enabled, gains, err := LoadEQ(); err == nil {
		eq.Enabled = enabled
		for i := range gains {
			eq.SetGain(i, gains[i])
		}
	}

	volume := &effects.Volume{
		Streamer: eq,
		Base:     math.Pow(10, 1.0/20.0), // Base para atenuación/ganancia logarítmica en decibelios (dB)
		Volume:   0,
		Silent:   false,
	}

	speaker.Lock()
	ae.streamer = streamer
	ae.format = format
	ae.ctrl = ctrl
	ae.eq = eq
	ae.volume = volume
	ae.statusMu.Lock()
	ae.hasEQ = true
	ae.eqEnabled = eq.Enabled
	ae.eqGains = eq.Gains
	ae.muted = volume.Silent
	ae.statusMu.Unlock()
	speaker.Unlock()

	return realDuration, nil
}

// Play envía la cadena de audio (Limiter -> Volume -> Ctrl -> Streamer) al subsistema de altavoces
// y devuelve un canal que se cierra automáticamente cuando la reproducción concluye con éxito.
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

	speaker.Play(beep.Seq(limiter, beep.Callback(func() {
		close(done)
	})))
	return done
}

// Stop cancela inmediatamente la reproducción activa, limpia la cola de muestras del altavoz
// y cierra el archivo de audio liberando los descriptores de archivo asociados.
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
	ae.eqGains = [eqBandCount]float64{}
	ae.muted = false
	ae.statusMu.Unlock()
	speaker.Unlock()

	if streamer != nil {
		_ = streamer.Close()
	}
}

// Close detiene la reproducción y libera de manera definitiva los recursos del subsistema de audio.
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

func (ae *AudioEngine) playbackSessionID() int {
	ae.sessionMu.Lock()
	defer ae.sessionMu.Unlock()
	return ae.sessionID
}

func (ae *AudioEngine) cancelPlaybackWait() {
	ae.sessionMu.Lock()
	defer ae.sessionMu.Unlock()

	if ae.cancelChan != nil {
		close(ae.cancelChan)
		ae.cancelChan = nil
	}
}
