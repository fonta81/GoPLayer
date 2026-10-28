package tui

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fonta81/Reproductor-go/internal/player/model"
)

func TestRebuildFilteredIndices(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		tracks      []model.Track
		limit       int
		wantIndices []int
		wantSuggest []int
	}{
		{
			name:        "empty query includes all tracks",
			tracks:      []model.Track{{Title: "Blue Train"}, {Title: "Red Bird"}, {Title: "Green Day"}},
			limit:       2,
			wantIndices: []int{0, 1, 2},
			wantSuggest: []int{0, 1},
		},
		{
			name:        "substring ranks title prefix first",
			query:       "blue",
			tracks:      []model.Track{{Title: "The Blue Note"}, {Title: "Blue Train"}, {Title: "Red Bird"}},
			limit:       2,
			wantIndices: []int{1, 0},
			wantSuggest: []int{1, 0},
		},
		{
			name:        "fuzzy subsequence matches",
			query:       "blutrn",
			tracks:      []model.Track{{Title: "Blue Train"}, {Title: "Red Bird"}},
			limit:       2,
			wantIndices: []int{0},
			wantSuggest: []int{0},
		},
		{
			name:        "negative suggestion limit",
			tracks:      []model.Track{{Title: "Blue Train"}},
			limit:       -1,
			wantIndices: []int{0},
			wantSuggest: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewAppModel("")
			defer app.Audio.Close()
			for _, track := range tt.tracks {
				app.playlist.Add(track)
			}
			app.filterQuery = tt.query
			app.suggestionLimit = tt.limit

			app.rebuildFilteredIndices()

			if !reflect.DeepEqual(app.filteredIndices, tt.wantIndices) {
				t.Errorf("filtered indices = %v, want %v", app.filteredIndices, tt.wantIndices)
			}
			if !reflect.DeepEqual(app.suggestionIndices, tt.wantSuggest) {
				t.Errorf("suggestion indices = %v, want %v", app.suggestionIndices, tt.wantSuggest)
			}
		})
	}
}

func TestFilterEnterSelectsHighlightedTrack(t *testing.T) {
	app := NewAppModel("")
	defer app.Audio.Close()
	app.playlist.Add(model.Track{Title: "First"})
	app.playlist.Add(model.Track{Title: "Second"})
	app.isFiltering = true
	app.filteredIndices = []int{1}
	app.filterCursor = 0

	updated, cmd := app.handleFilterInput(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(AppModel)
	if cmd == nil {
		t.Fatal("expected a command to load the selected track")
	}
	if got.isFiltering {
		t.Fatal("filter mode should close after selecting a result")
	}
	if got.cursorIndex != 1 || got.playlist.CurrentIndex() != 1 {
		t.Fatalf("selected track indices = cursor %d, playlist %d; want 1, 1", got.cursorIndex, got.playlist.CurrentIndex())
	}
}

func TestSearchCacheInvalidatesWhenPlaylistChanges(t *testing.T) {
	app := NewAppModel("")
	defer app.Audio.Close()
	app.playlist.Add(model.Track{Title: "First"})
	app.filterQuery = "first"
	app.rebuildFilteredIndices()

	app.playlist.Add(model.Track{Title: "Second"})
	app.filterQuery = "second"
	app.rebuildFilteredIndices()
	if !reflect.DeepEqual(app.filteredIndices, []int{1}) {
		t.Fatalf("indices after adding a track = %v, want [1]", app.filteredIndices)
	}

	app.playlist.Remove(0)
	app.filterQuery = "first"
	app.rebuildFilteredIndices()
	if len(app.filteredIndices) != 0 {
		t.Fatalf("indices after removing a track = %v, want no matches", app.filteredIndices)
	}
}

func TestPlaybackPositionIgnoresStaleSession(t *testing.T) {
	app := NewAppModel("")
	defer app.Audio.Close()
	app.state = model.StatePlaying
	app.elapsed = 3 * time.Second
	app.totalTime = 10 * time.Second
	sessionID, _ := app.Audio.StartPlaybackSession()

	updated, _ := app.Update(playbackPositionMsg{sessionID: sessionID - 1, position: 8 * time.Second})
	if got := updated.(AppModel).elapsed; got != 3*time.Second {
		t.Fatalf("elapsed for stale session = %v, want 3s", got)
	}

	updated, _ = app.Update(playbackPositionMsg{sessionID: sessionID, position: 8 * time.Second})
	if got := updated.(AppModel).elapsed; got != 8*time.Second {
		t.Fatalf("elapsed for active session = %v, want 8s", got)
	}
}

func BenchmarkRebuildFilteredIndices(b *testing.B) {
	app := NewAppModel("")
	defer app.Audio.Close()
	for i := range 1000 {
		app.playlist.Add(model.Track{Title: fmt.Sprintf("Track %04d - Example Song", i), Artist: fmt.Sprintf("Artist %02d", i%50)})
	}
	app.filterQuery = "track 042 artist"

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		app.rebuildFilteredIndices()
	}
}
