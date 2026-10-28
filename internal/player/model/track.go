package model

import (
	"fmt"
	"time"

	"github.com/fonta81/Reproductor-go/internal/player/format"
)

type Track struct {
	ID       string
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
	Path     string
}

func (t Track) DisplayName() string {
	if t.Artist == "" {
		return t.Title
	}
	return fmt.Sprintf("%s — %s", t.Artist, t.Title)
}

func (t Track) FormattedDuration() string {
	if t.Duration <= 0 {
		return "?:??"
	}
	return format.Duration(t.Duration)
}
