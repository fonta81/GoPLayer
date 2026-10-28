package player

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Update procesa los eventos y mensajes del bucle de Bubble Tea (redimensionamiento, escaneos de biblioteca,
// carga de pistas, temporizadores de refresco, fin de reproducción y entrada de teclado en diferentes modos).
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
		if !m.playlist.IsEmpty() && m.state == StateStopped {
			m.playlist.current = 0
		}
		if msg.dir != "" {
			// Recordar el directorio para que LoadConfig() lo recupere en el
			// próximo arranque. Antes SaveMusicDir nunca se invocaba, así
			// que la carpeta elegida con el explorador ('o') se perdía al
			// cerrar la aplicación.
			_ = SaveMusicDir(msg.dir)
		}
		return m, nil

	case noMusicFoundMsg:
		// Abrir el navegador para que el usuario seleccione una carpeta con pistas
		m.lastError = nil
		cmd := m.openBrowser(msg.dir)
		return m, cmd

	case trackLoadedMsg:
		return m.handleTrackLoaded(msg)

	case tickMsg:
		if m.state == StatePlaying {
			return m, tea.Batch(m.tick(), m.playbackPositionCmd())
		}
		return m, nil

	case playbackPositionMsg:
		if msg.sessionID == m.Audio.playbackSessionID() && m.state == StatePlaying {
			m.elapsed = min(msg.position, m.totalTime)
		}
		return m, nil

	case playbackEndedMsg:
		if msg.sessionID != m.Audio.playbackSessionID() {
			return m, nil
		}
		if m.playlist.isLast() && m.playlist.repeat == RepeatOff {
			m.state, m.elapsed = StateStopped, 0
			m.Audio.Stop()
			return m, nil
		}
		return m.playNext()

	case errorMsg:
		m.lastError = msg.err
		if msg.err != nil {
			// Solo se programa el borrado automático cuando llega un error
			// real. De lo contrario, el propio mensaje de limpieza
			// (errorMsg{nil}) volvía a programar otro temporizador cada 5
			// segundos indefinidamente.
			return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return errorMsg{nil} })
		}
		return m, nil
	default:
		return m, nil
	}
}

// handleTrackLoaded inicia la reproducción al recibir trackLoadedMsg, incrementa el sessionID
// para invalidar eventos de pistas anteriores y programa una goroutine que aguarda el final de la pista.
func (m AppModel) handleTrackLoaded(msg trackLoadedMsg) (tea.Model, tea.Cmd) {
	sessionID, cancelChan := m.Audio.startPlaybackSession()

	done := m.Audio.Play()
	m.state = StatePlaying
	m.totalTime = max(msg.duration, defaultDuration)

	m.playlist.tracks[m.playlist.current].Duration = msg.duration
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

// togglePlayback conmuta entre reproducir y pausar según el estado actual del reproductor.
func (m AppModel) togglePlayback() (tea.Model, tea.Cmd) {
	switch m.state {
	case StatePlaying:
		m.Audio.Pause()
		m.state = StatePaused
	case StatePaused:
		m.Audio.Resume()
		m.state = StatePlaying
		return m, m.tick()
	case StateStopped:
		return m.playCurrent()
	}
	return m, nil
}

// playCurrent carga y reproduce la pista actualmente activa en la lista.
func (m AppModel) playCurrent() (tea.Model, tea.Cmd) {
	if track, ok := m.playlist.Current(); ok {
		return m, m.loadTrackCmd(track)
	}
	return m, nil
}

// playNext avanza a la siguiente canción de la lista y la reproduce.
func (m AppModel) playNext() (tea.Model, tea.Cmd) {
	if track, ok := m.playlist.Next(); ok {
		m.cursorIndex = m.playlist.current
		return m, m.loadTrackCmd(track)
	}
	m.resetPlayback()
	return m, nil
}

// playPrevious reinicia la pista si han transcurrido más de 3 segundos, o retrocede a la canción previa.
func (m AppModel) playPrevious() (tea.Model, tea.Cmd) {
	if m.elapsed > 3*time.Second {
		cmd := m.seekTo(0)
		return m, cmd
	}
	if track, ok := m.playlist.Previous(); ok {
		m.cursorIndex = m.playlist.current
		return m, m.loadTrackCmd(track)
	}
	return m, nil
}

// playSelected reproduce la pista sobre la que está posicionado el cursor.
func (m AppModel) playSelected() (tea.Model, tea.Cmd) {
	if m.playlist.JumpTo(m.cursorIndex) {
		return m.playCurrent()
	}
	return m, nil
}

// removeSelected elimina de la lista la pista bajo el cursor, gestionando la continuidad
// de la reproducción si correspondía a la canción en curso.
func (m AppModel) removeSelected() (tea.Model, tea.Cmd) {
	if !m.playlist.isValidIndex(m.cursorIndex) {
		return m, nil
	}

	wasPlaying := m.cursorIndex == m.playlist.current
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

// moveCursor desplaza el cursor de navegación vertical dentro de los límites válidos de la lista.
func (m *AppModel) moveCursor(delta int) {
	m.cursorIndex = max(0, min(m.cursorIndex+delta, m.playlist.Length()-1))
}

// adjustVolume modifica la ganancia de salida en pasos de decibelios respetando los topes configurados.
func (m *AppModel) adjustVolume(delta float64) {
	m.volumeLevel = max(minVolume, min(maxVolume, m.volumeLevel+delta))
	m.Audio.SetVolume(m.volumeLevel)
}

// seekTo desplaza la reproducción a un punto temporal concreto, limitándolo entre 0 y la duración total
// para evitar errores críticos de decodificación al rebasar los extremos del archivo.
func (m *AppModel) seekTo(position time.Duration) tea.Cmd {
	position = clampDuration(position, 0, m.totalTime)
	if err := m.Audio.Seek(position); err != nil {
		return func() tea.Msg { return errorMsg{err} }
	}
	m.elapsed = position
	return nil
}

// seekForward adelanta la reproducción en el número de segundos especificado.
func (m *AppModel) seekForward(seconds int) tea.Cmd {
	return m.seekTo(m.elapsed + time.Duration(seconds)*time.Second)
}

// seekBackward retrocede la reproducción en el número de segundos especificado.
func (m *AppModel) seekBackward(seconds int) tea.Cmd {
	return m.seekTo(m.elapsed - time.Duration(seconds)*time.Second)
}

// resetPlayback detiene la salida de audio y restablece a cero los tiempos y estados de reproducción.
func (m *AppModel) resetPlayback() {
	m.Audio.Stop()
	m.state, m.elapsed, m.totalTime = StateStopped, 0, 0
}
