package audio

import (
	"sync"
	"testing"

	"github.com/fonta81/Reproductor-go/internal/player/eq"
)

func TestAudioEngineStopCancelsPlaybackWait(t *testing.T) {
	engine := NewAudioEngine()
	_, cancel := engine.startPlaybackSession()

	engine.Stop()

	select {
	case <-cancel:
	default:
		t.Fatal("Stop should cancel the playback wait")
	}
}

func TestAudioEngineConcurrentIdleOperations(t *testing.T) {
	engine := NewAudioEngine()
	var workers sync.WaitGroup
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 20 {
				engine.startPlaybackSession()
				engine.Play()
				engine.Stop()
				engine.Position()
			}
		}()
	}
	workers.Wait()
	engine.Close()
}

func TestAudioEngineIdleStatusSnapshot(t *testing.T) {
	engine := NewAudioEngine()

	if engine.HasEQ() || engine.EQEnabled() || engine.IsMuted() {
		t.Fatal("idle engine should not report enabled audio state")
	}
	if got := engine.GetEQBands(); got != ([eq.BandCount]float64{}) {
		t.Fatalf("idle EQ bands = %v, want zero values", got)
	}
}
