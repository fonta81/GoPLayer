package player

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Init inicializa la aplicación escaneando la biblioteca y iniciando el temporizador de refresco.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.scanLibraryCmd(m.musicDir), m.tick())
}

// tick genera un comando para actualizar la interfaz periódicamente.
func (m AppModel) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// scanLibraryCmd escanea un directorio en busca de archivos de audio compatibles de forma recursiva.
func (m AppModel) scanLibraryCmd(targetDir string) tea.Cmd {
	return func() tea.Msg {
		var dirs []string

		if targetDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				home = ""
			}
			dirs = []string{defaultDir, "./songs"}
			if home != "" {
				dirs = append(dirs, filepath.Join(home, "Music"), filepath.Join(home, "Música"))
			}
		} else {
			dirs = []string{targetDir}
		}

		found := make([]Track, 0, 50)
		for _, dir := range dirs {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				continue
			}

			err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil // Ignorar errores de acceso a directorios
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

		if len(found) == 0 {
			return noMusicFoundMsg{dir: targetDir}
		}

		return libraryScannedMsg{tracks: found, dir: targetDir}
	}
}

// loadTrackCmd ejecuta un comando asíncrono que delega en el motor de audio la apertura
// y decodificación de la pista, retornando un mensaje con su duración calculada.
func (m AppModel) loadTrackCmd(track Track) tea.Cmd {
	return func() tea.Msg {
		duration, err := m.Audio.Load(track)
		if err != nil {
			return errorMsg{err}
		}
		track.Duration = duration
		return trackLoadedMsg{track: track, duration: duration}
	}
}

// loadBrowserDir lee el contenido de una ruta en disco, resuelve enlaces simbólicos
// y agrupa las carpetas y archivos de audio compatibles ordenados alfabéticamente.
func (m *AppModel) loadBrowserDir(target string) tea.Cmd {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return func() tea.Msg { return errorMsg{err} }
	}

	entries, err := os.ReadDir(absTarget)
	if err != nil {
		return func() tea.Msg { return errorMsg{err} }
	}

	m.browserPath = absTarget

	var dirs, files []browserEntry
	for _, e := range entries {
		name := e.Name()
		isDir := e.IsDir()

		if e.Type()&os.ModeSymlink != 0 {
			if info, statErr := os.Stat(filepath.Join(target, name)); statErr == nil {
				isDir = info.IsDir()
			} else {
				continue
			}
		}

		if isDir {
			dirs = append(dirs, browserEntry{name: name, isDir: true})
			continue
		}

		if IsSupportedAudio(name) {
			files = append(files, browserEntry{name: name, isDir: false})
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].name) < strings.ToLower(files[j].name)
	})

	m.browserEntries = append(dirs, files...)
	m.browserCursor = 0
	return nil
}

// openBrowser activa el explorador de carpetas apuntando a dir (o "." si está vacío).
func (m *AppModel) openBrowser(dir string) tea.Cmd {

	m.isPickingFolder = true
	if dir == "" {
		dir = "."
	}
	return m.loadBrowserDir(dir)
}
