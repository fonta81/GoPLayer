package player

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// handleBrowserInput procesa las pulsaciones de teclado cuando el explorador de carpetas está en primer plano,
// permitiendo navegar directorios, subir de nivel respetando el home del usuario y confirmar la selección.
func (m AppModel) handleBrowserInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.isPickingFolder = false
		return m, nil
	case "up", "k":
		m.browserCursor = max(0, m.browserCursor-1)
	case "down", "j":
		m.browserCursor = min(len(m.browserEntries)-1, m.browserCursor+1)
	case "left", "backspace":
		home, err := os.UserHomeDir()
		if err != nil {
			return m, nil // No es posible restringir navegación sin directorio home
		}

		absPath, err := filepath.Abs(m.browserPath)
		if err != nil {
			return m, nil
		}

		parent := filepath.Dir(absPath)

		// Asegurar que el directorio padre permanezca dentro de home para evitar salir del espacio de usuario
		if !strings.HasPrefix(parent, home) {
			return m, nil // No permitir salir de la carpeta personal
		}

		cmd := m.loadBrowserDir(parent)
		return m, cmd
	case "right", "enter":
		if len(m.browserEntries) > 0 {
			selected := m.browserEntries[m.browserCursor]
			if selected.isDir {
				newPath := filepath.Join(m.browserPath, selected.name)
				cmd := m.loadBrowserDir(newPath)
				return m, cmd
			}
		}
	case " ":
		target := m.browserPath
		if len(m.browserEntries) > 0 {
			selected := m.browserEntries[m.browserCursor]
			if selected.isDir {
				target := filepath.Join(m.browserPath, selected.name)
				home, err := os.UserHomeDir()
				if err == nil && !strings.HasPrefix(target, home) {
					return m, nil // Evitar seleccionar carpetas fuera del directorio personal
				}
				m.isPickingFolder = false
				return m, m.scanLibraryCmd(target)
			}
		}
		m.isPickingFolder = false
		return m, m.scanLibraryCmd(target)
	}
	return m, nil
}

// handleFilterInput procesa los eventos de teclado cuando el modo de búsqueda rápida está activo.
// Permite navegar por la lista de sugerencias o resultados coincidentes, selecciona y reproduce
// la pista destacada con Enter, o cancela la búsqueda restaurando la vista normal con Esc.
func (m AppModel) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.isFiltering = false
		m.filterInput.SetValue("")
		m.filterQuery = ""
		m.filteredIndices = nil
		m.filterCursor = 0
		return m, nil
	case "enter":
		// Reproducir el elemento filtrado seleccionado actualmente (si existe alguno)
		if len(m.filteredIndices) > 0 {
			m.cursorIndex = m.filteredIndices[m.filterCursor]
			m.isFiltering = false
			return m.playSelected()
		}
		return m, nil
	case "up", "k":
		// Mover hacia arriba en las sugerencias o resultados filtrados
		if len(m.suggestionIndices) > 0 {
			m.suggestionCursor = max(0, m.suggestionCursor-1)
			// Mantener filterCursor alineado con la posición de la sugerencia
			m.filterCursor = m.suggestionCursor
		} else {
			m.filterCursor = max(0, m.filterCursor-1)
		}
		return m, nil
	case "down", "j":
		// Mover hacia abajo en las sugerencias o resultados filtrados
		if len(m.suggestionIndices) > 0 {
			m.suggestionCursor = min(len(m.suggestionIndices)-1, m.suggestionCursor+1)
			m.filterCursor = m.suggestionCursor
		} else {
			m.filterCursor = min(len(m.filteredIndices)-1, m.filterCursor+1)
		}
		return m, nil
	default:
		// Delegar el procesamiento al componente de texto y recalcular coincidencias
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		oldQ := m.filterQuery
		m.filterQuery = m.filterInput.Value()
		m.rebuildFilteredIndices()

		// Reiniciar cursores si la consulta de búsqueda cambió
		if m.filterQuery != oldQ {
			m.suggestionCursor = 0
			m.filterCursor = 0
		}

		// Ajustar cursores dentro de los rangos válidos
		if m.filterCursor >= len(m.filteredIndices) {
			m.filterCursor = max(0, len(m.filteredIndices)-1)
		}
		if m.suggestionCursor >= len(m.suggestionIndices) {
			m.suggestionCursor = max(0, len(m.suggestionIndices)-1)
		}
		return m, cmd
	}
}

// handleKeyInput procesa los atajos de teclado estándar en la vista principal del reproductor
// para controlar la reproducción, volumen, modos (shuffle/repeat), búsqueda y navegación por la cola.
func (m AppModel) handleKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.Audio.HasEQ() {
			_ = SaveEQ(m.Audio.EQEnabled(), m.Audio.GetEQBands())
		}
		m.Audio.Close()
		return m, tea.Quit
	case "o", "ctrl+o":
		cmd := m.openBrowser(m.musicDir)
		return m, cmd
	case " ":
		return m.togglePlayback()
	case "n":
		return m.playNext()
	case "N":
		return m.playPrevious()
	case "enter":
		return m.playSelected()
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "d":
		return m.removeSelected()
	case "+", "=":
		m.adjustVolume(volumeStep)
	case "-":
		m.adjustVolume(-volumeStep)
	case "m":
		m.Audio.ToggleMute()
	case "0":
		return m, m.seekTo(0)
	case ".", ">":
		return m, m.seekForward(seekSeconds)
	case ",", "<":
		return m, m.seekBackward(seekSeconds)
	case "s":
		m.playlist.ToggleShuffle()
	case "r":
		m.playlist.repeat = (m.playlist.repeat + 1) % 3
	case "l":
		m.showQueue = !m.showQueue
	case "h", "?":
		m.showHelp = !m.showHelp
	case "ctrl+f":
		// Activar modo de búsqueda rápida
		m.isFiltering = true
		m.filterInput.SetValue("")
		m.filterQuery = ""
		m.filteredIndices = nil
		m.filterCursor = 0
		return m, nil
	case "e":
		// Abrir/cerrar el panel del ecualizador
		m.eqActive = !m.eqActive
	}
	return m, nil
}

// handleEQInput procesa las teclas mientras el panel del ecualizador tiene el foco:
// izquierda/derecha eligen banda, arriba/abajo ajustan dB, 0 resetea la banda,
// x activa/desactiva el EQ y Esc cierra el panel.
func (m AppModel) handleEQInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	bands := m.Audio.GetEQBands()
	switch msg.String() {
	case "esc", "e":
		m.eqActive = false
		if m.Audio.HasEQ() {
			_ = SaveEQ(m.Audio.EQEnabled(), bands)
		}
	case "left", "h":
		m.eqCursor = max(0, m.eqCursor-1)
	case "right", "l":
		m.eqCursor = min(eqBandCount-1, m.eqCursor+1)
	case "up", "k":
		m.Audio.SetEQBand(m.eqCursor, bands[m.eqCursor]+eqGainStep)
	case "down", "j":
		m.Audio.SetEQBand(m.eqCursor, bands[m.eqCursor]-eqGainStep)
	case "0":
		m.Audio.SetEQBand(m.eqCursor, 0)
	case "x":
		m.Audio.ToggleEQ()
	}
	return m, nil
}
