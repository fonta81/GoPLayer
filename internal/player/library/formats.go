package library

import (
	"path/filepath"
	"strings"
)

var SupportedAudioExtensions = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".flac": true,
	".ogg":  true,
}

func IsSupportedAudio(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return SupportedAudioExtensions[ext]
}
