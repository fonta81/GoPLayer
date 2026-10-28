package player

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRebuildFilteredIndices(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		tracks      []Track
		limit       int
		wantIndices []int
		wantSuggest []int
	}{
		{
			name:        "empty query includes all tracks",
			tracks:      []Track{{Title: "Blue Train"}, {Title: "Red Bird"}, {Title: "Green Day"}},
			limit:       2,
			wantIndices: []int{0, 1, 2},
			wantSuggest: []int{0, 1},
		},
		{
			name:        "substring ranks title prefix first",
			query:       "blue",
			tracks:      []Track{{Title: "The Blue Note"}, {Title: "Blue Train"}, {Title: "Red Bird"}},
			limit:       2,
			wantIndices: []int{1, 0},
			wantSuggest: []int{1, 0},
		},
		{
			name:        "fuzzy subsequence matches",
			query:       "blutrn",
			tracks:      []Track{{Title: "Blue Train"}, {Title: "Red Bird"}},
			limit:       2,
			wantIndices: []int{0},
			wantSuggest: []int{0},
		},
		{
			name:        "negative suggestion limit",
			tracks:      []Track{{Title: "Blue Train"}},
			limit:       -1,
			wantIndices: []int{0},
			wantSuggest: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewAppModel("")
			defer model.Audio.Close()
			for _, track := range tt.tracks {
				model.playlist.Add(track)
			}
			model.filterQuery = tt.query
			model.suggestionLimit = tt.limit

			model.rebuildFilteredIndices()

			if !reflect.DeepEqual(model.filteredIndices, tt.wantIndices) {
				t.Errorf("filtered indices = %v, want %v", model.filteredIndices, tt.wantIndices)
			}
			if !reflect.DeepEqual(model.suggestionIndices, tt.wantSuggest) {
				t.Errorf("suggestion indices = %v, want %v", model.suggestionIndices, tt.wantSuggest)
			}
		})
	}
}

func TestFilterEnterSelectsHighlightedTrack(t *testing.T) {
	model := NewAppModel("")
	defer model.Audio.Close()
	model.playlist.Add(Track{Title: "First"})
	model.playlist.Add(Track{Title: "Second"})
	model.isFiltering = true
	model.filteredIndices = []int{1}
	model.filterCursor = 0

	updated, cmd := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyEnter})

	got := updated.(AppModel)
	if cmd == nil {
		t.Fatal("expected a command to load the selected track")
	}
	if got.isFiltering {
		t.Fatal("filter mode should close after selecting a result")
	}
	if got.cursorIndex != 1 || got.playlist.current != 1 {
		t.Fatalf("selected track indices = cursor %d, playlist %d; want 1, 1", got.cursorIndex, got.playlist.current)
	}
}

func TestSearchCacheInvalidatesWhenPlaylistChanges(t *testing.T) {
	model := NewAppModel("")
	defer model.Audio.Close()
	model.playlist.Add(Track{Title: "First"})
	model.filterQuery = "first"
	model.rebuildFilteredIndices()

	model.playlist.Add(Track{Title: "Second"})
	model.filterQuery = "second"
	model.rebuildFilteredIndices()
	if !reflect.DeepEqual(model.filteredIndices, []int{1}) {
		t.Fatalf("indices after adding a track = %v, want [1]", model.filteredIndices)
	}

	model.playlist.Remove(0)
	model.filterQuery = "first"
	model.rebuildFilteredIndices()
	if len(model.filteredIndices) != 0 {
		t.Fatalf("indices after removing a track = %v, want no matches", model.filteredIndices)
	}
}

func TestPlaybackPositionIgnoresStaleSession(t *testing.T) {
	model := NewAppModel("")
	defer model.Audio.Close()
	model.state = StatePlaying
	model.elapsed = 3 * time.Second
	model.totalTime = 10 * time.Second
	sessionID, _ := model.Audio.startPlaybackSession()

	updated, _ := model.Update(playbackPositionMsg{
		sessionID: sessionID - 1,
		position:  8 * time.Second,
	})
	if got := updated.(AppModel).elapsed; got != 3*time.Second {
		t.Fatalf("elapsed for stale session = %v, want 3s", got)
	}

	updated, _ = model.Update(playbackPositionMsg{
		sessionID: sessionID,
		position:  8 * time.Second,
	})
	if got := updated.(AppModel).elapsed; got != 8*time.Second {
		t.Fatalf("elapsed for active session = %v, want 8s", got)
	}
}

func BenchmarkRebuildFilteredIndices(b *testing.B) {
	model := NewAppModel("")
	defer model.Audio.Close()
	for i := range 1000 {
		model.playlist.Add(Track{
			Title:  fmt.Sprintf("Track %04d - Example Song", i),
			Artist: fmt.Sprintf("Artist %02d", i%50),
		})
	}
	model.filterQuery = "track 042 artist"

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		model.rebuildFilteredIndices()
	}
}
