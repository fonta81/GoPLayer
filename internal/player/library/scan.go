package library

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fonta81/Reproductor-go/internal/player/model"
)

func Scan(targetDir string, defaultDirs []string) []model.Track {
	dirs := defaultDirs
	if targetDir != "" {
		dirs = []string{targetDir}
	}
	if len(dirs) == 0 {
		home, err := os.UserHomeDir()
		if err == nil {
			dirs = []string{"./music", "./songs", filepath.Join(home, "Music"), filepath.Join(home, "Música")}
		} else {
			dirs = []string{"./music", "./songs"}
		}
	}

	found := make([]model.Track, 0, 50)
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !IsSupportedAudio(d.Name()) {
				return nil
			}
			track := ExtractMetadata(path)
			track.ID = fmt.Sprintf("track-%d", len(found))
			found = append(found, track)
			return nil
		})
		if err != nil {
			continue
		}
	}
	return found
}
