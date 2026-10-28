package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
	"github.com/fonta81/Reproductor-go/internal/player/model"
)

func ExtractMetadata(path string) model.Track {
	ext := strings.ToLower(filepath.Ext(path))
	filename := strings.TrimSuffix(filepath.Base(path), ext)
	track := model.Track{Title: filename, Artist: "Desconocido", Album: "Desconocido", Path: path}

	f, err := os.Open(path)
	if err != nil {
		return track
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return track
	}

	if t := m.Title(); t != "" {
		track.Title = t
	}
	if a := m.Artist(); a != "" {
		track.Artist = a
	}
	if al := m.Album(); al != "" {
		track.Album = al
	}
	return track
}
