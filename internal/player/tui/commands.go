package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fonta81/Reproductor-go/internal/player/library"
	"github.com/fonta81/Reproductor-go/internal/player/model"
)

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.scanLibraryCmd(m.musicDir), m.tick())
}

func (m AppModel) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m AppModel) scanLibraryCmd(targetDir string) tea.Cmd {
	return func() tea.Msg {
		if targetDir == "" {
			dirs := append([]string(nil), defaultScanDirs...)
			if home, err := os.UserHomeDir(); err == nil {
				dirs = append(dirs, filepath.Join(home, "Music"), filepath.Join(home, "Música"))
			}
			tracks := library.Scan("", dirs)
			if len(tracks) == 0 {
				return noMusicFoundMsg{dir: targetDir}
			}
			return libraryScannedMsg{tracks: tracks, dir: targetDir}
		}
		tracks := library.Scan(targetDir, nil)
		if len(tracks) == 0 {
			return noMusicFoundMsg{dir: targetDir}
		}
		return libraryScannedMsg{tracks: tracks, dir: targetDir}
	}
}

func (m AppModel) loadTrackCmd(track model.Track) tea.Cmd {
	return func() tea.Msg {
		duration, err := m.Audio.Load(track)
		if err != nil {
			return errorMsg{err}
		}
		track.Duration = duration
		return trackLoadedMsg{track: track, duration: duration}
	}
}

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
		if library.IsSupportedAudio(name) {
			files = append(files, browserEntry{name: name, isDir: false})
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name) })
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].name) < strings.ToLower(files[j].name) })

	m.browserEntries = append(dirs, files...)
	m.browserCursor = 0
	return nil
}

func (m *AppModel) openBrowser(dir string) tea.Cmd {
	m.isPickingFolder = true
	if dir == "" {
		dir = "."
	}
	return m.loadBrowserDir(dir)
}
