package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fonta81/Reproductor-go/internal/player/config"
	"github.com/fonta81/Reproductor-go/internal/player/model"
)

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.progressBar.Width = min(progressWidth, max(1, msg.Width-36))
		return m, nil
	case tea.KeyMsg:
		if m.isPickingFolder {
			return m.handleBrowserInput(msg)
		}
		if m.isFiltering {
			return m.handleFilterInput(msg)
		}
		if m.eqActive {
			return m.handleEQInput(msg)
		}
		return m.handleKeyInput(msg)
	case libraryScannedMsg:
		m.playlist.Clear()
		m.resetPlayback()
		m.musicDir = msg.dir
		m.cursorIndex = 0

		for _, track := range msg.tracks {
			m.playlist.Add(track)
		}
		if !m.playlist.IsEmpty() && m.state == model.StateStopped {
			m.playlist.JumpTo(0)
		}
		if msg.dir != "" {
			_ = config.SaveMusicDir(msg.dir)
		}
		return m, nil
	case noMusicFoundMsg:
		m.lastError = nil
		return m, m.openBrowser(msg.dir)
	case trackLoadedMsg:
		return m.handleTrackLoaded(msg)
	case tickMsg:
		if m.state == model.StatePlaying {
			return m, tea.Batch(m.tick(), m.playbackPositionCmd())
		}
		return m, nil
	case playbackPositionMsg:
		if msg.sessionID == m.Audio.PlaybackSessionID() && m.state == model.StatePlaying {
			m.elapsed = min(msg.position, m.totalTime)
		}
		return m, nil
	case playbackEndedMsg:
		if msg.sessionID != m.Audio.PlaybackSessionID() {
			return m, nil
		}
		if m.playlist.IsLast() && m.playlist.Repeat() == model.RepeatOff {
			m.state, m.elapsed = model.StateStopped, 0
			m.Audio.Stop()
			return m, nil
		}
		return m.playNext()
	case errorMsg:
		m.lastError = msg.err
		if msg.err != nil {
			return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return errorMsg{nil} })
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m AppModel) handleTrackLoaded(msg trackLoadedMsg) (tea.Model, tea.Cmd) {
	sessionID, cancelChan := m.Audio.StartPlaybackSession()
	done := m.Audio.Play()
	m.state = model.StatePlaying
	m.totalTime = max(msg.duration, defaultDuration)
	m.playlist.UpdateCurrentDuration(msg.duration)
	m.elapsed = 0

	waitCmd := func() tea.Msg {
		select {
		case <-done:
			return playbackEndedMsg{sessionID: sessionID}
		case <-cancelChan:
			return nil
		}
	}
	return m, tea.Batch(m.tick(), waitCmd)
}

func (m AppModel) togglePlayback() (tea.Model, tea.Cmd) {
	switch m.state {
	case model.StatePlaying:
		m.Audio.Pause()
		m.state = model.StatePaused
	case model.StatePaused:
		m.Audio.Resume()
		m.state = model.StatePlaying
		return m, m.tick()
	case model.StateStopped:
		return m.playCurrent()
	}
	return m, nil
}

func (m AppModel) playCurrent() (tea.Model, tea.Cmd) {
	if track, ok := m.playlist.Current(); ok {
		return m, m.loadTrackCmd(track)
	}
	return m, nil
}

func (m AppModel) playNext() (tea.Model, tea.Cmd) {
	if track, ok := m.playlist.Next(); ok {
		m.cursorIndex = m.playlist.CurrentIndex()
		return m, m.loadTrackCmd(track)
	}
	m.resetPlayback()
	return m, nil
}

func (m AppModel) playPrevious() (tea.Model, tea.Cmd) {
	if m.elapsed > 3*time.Second {
		return m, m.seekTo(0)
	}
	if track, ok := m.playlist.Previous(); ok {
		m.cursorIndex = m.playlist.CurrentIndex()
		return m, m.loadTrackCmd(track)
	}
	return m, nil
}

func (m AppModel) playSelected() (tea.Model, tea.Cmd) {
	if m.playlist.JumpTo(m.cursorIndex) {
		return m.playCurrent()
	}
	return m, nil
}

func (m AppModel) removeSelected() (tea.Model, tea.Cmd) {
	if !m.playlist.IsValidIndex(m.cursorIndex) {
		return m, nil
	}

	wasPlaying := m.cursorIndex == m.playlist.CurrentIndex()
	m.playlist.Remove(m.cursorIndex)

	if m.cursorIndex >= m.playlist.Length() && m.cursorIndex > 0 {
		m.cursorIndex--
	}
	if wasPlaying {
		m.resetPlayback()
		if !m.playlist.IsEmpty() {
			return m.playCurrent()
		}
	}
	return m, nil
}

func (m *AppModel) moveCursor(delta int) {
	m.cursorIndex = max(0, min(m.cursorIndex+delta, m.playlist.Length()-1))
}

func (m *AppModel) adjustVolume(delta float64) {
	m.volumeLevel = max(minVolume, min(maxVolume, m.volumeLevel+delta))
	m.Audio.SetVolume(m.volumeLevel)
}

func (m *AppModel) seekTo(position time.Duration) tea.Cmd {
	position = max(0, min(position, m.totalTime))
	if err := m.Audio.Seek(position); err != nil {
		return func() tea.Msg { return errorMsg{err} }
	}
	m.elapsed = position
	return nil
}

func (m *AppModel) seekForward(seconds int) tea.Cmd {
	return m.seekTo(m.elapsed + time.Duration(seconds)*time.Second)
}

func (m *AppModel) seekBackward(seconds int) tea.Cmd {
	return m.seekTo(m.elapsed - time.Duration(seconds)*time.Second)
}

func (m *AppModel) resetPlayback() {
	m.Audio.Stop()
	m.state, m.elapsed, m.totalTime = model.StateStopped, 0, 0
}
