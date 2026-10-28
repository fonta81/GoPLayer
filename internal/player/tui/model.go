package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/fonta81/Reproductor-go/internal/player/audio"
	"github.com/fonta81/Reproductor-go/internal/player/model"
)

var defaultScanDirs = []string{defaultDir, "./songs"}

type browserEntry struct {
	name  string
	isDir bool
}

type AppModel struct {
	playlist *model.Playlist
	Audio    *audio.AudioEngine

	state       model.PlaybackState
	elapsed     time.Duration
	totalTime   time.Duration
	cursorIndex int
	volumeLevel float64
	lastError   error

	width       int
	height      int
	progressBar progress.Model
	showHelp    bool
	showQueue   bool

	musicDir        string
	isPickingFolder bool
	browserPath     string
	browserEntries  []browserEntry
	browserCursor   int

	isFiltering      bool
	filterInput      textinput.Model
	filterQuery      string
	filteredIndices  []int
	filterCursor     int
	searchCandidates []searchCandidate
	searchRevision   uint64

	eqActive bool
	eqCursor int

	suggestionLimit   int
	suggestionIndices []int
	suggestionCursor  int
}

func NewAppModel(initialDir string) AppModel {
	bar := progress.New(progress.WithDefaultGradient())
	bar.Width = progressWidth
	bar.ShowPercentage = false

	ti := textinput.New()
	ti.Placeholder = "Buscar título o artista..."
	ti.CharLimit = 200
	ti.Width = 40

	return AppModel{
		playlist:        model.NewPlaylist(),
		Audio:           audio.NewAudioEngine(),
		state:           model.StateStopped,
		progressBar:     bar,
		volumeLevel:     0,
		showHelp:        true,
		showQueue:       true,
		musicDir:        initialDir,
		filterInput:     ti,
		isFiltering:     false,
		filterQuery:     "",
		filteredIndices: nil,
		filterCursor:    0,
		suggestionLimit: 6,
	}
}
