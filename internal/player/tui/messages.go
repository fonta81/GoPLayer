package tui

import (
	"time"

	"github.com/fonta81/Reproductor-go/internal/player/model"
)

type (
	trackLoadedMsg struct {
		track    model.Track
		duration time.Duration
	}
	playbackEndedMsg    struct{ sessionID int }
	playbackPositionMsg struct {
		sessionID int
		position  time.Duration
	}
	tickMsg           time.Time
	libraryScannedMsg struct {
		tracks []model.Track
		dir    string
	}
	noMusicFoundMsg struct{ dir string }
	errorMsg        struct{ err error }
)
