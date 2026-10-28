// Package menu proporciona la interfaz interactiva, componentes visuales y máquina de estados
// del menú principal de GoPlayer construidos sobre Bubble Tea y Lipgloss.
package menu

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Action define los identificadores numéricos de las acciones que el usuario puede desencadenar desde el menú.
type Action int

// Enumeración de las acciones admitidas por el menú principal.
const (
	ActionTogglePlay Action = iota // Conmuta entre reproducción y pausa de la pista activa.
	ActionViewPlayer               // Cambia la vista activa hacia el reproductor completo.
	ActionNextTrack                // Salta a la siguiente canción en la lista de reproducción.
	ActionPrevTrack                // Retrocede a la canción previa en la lista de reproducción.
	ActionLibrary                  // Accede a la vista de la biblioteca y cola musical.
	ActionSettings                 // Despliega el explorador de carpetas para seleccionar el directorio de música.
	ActionQuit                     // Finaliza la ejecución de la aplicación liberando recursos.
)

// PlayerState encapsula la telemetría y el estado operativo del motor de audio necesarios
// para renderizar el widget informativo ("Now Playing") dentro del menú principal.
type PlayerState struct {
	State                string // Descripción textual del estado ("Reproduciendo", "Pausado", "Detenido")
	IsPlaying            bool   // Indica si el motor de audio está reproduciendo activamente
	IsPaused             bool   // Indica si la reproducción se encuentra en pausa
	CurrentTrackTitle    string // Título de la pista en reproducción
	CurrentTrackArtist   string // Artista o intérprete de la pista en reproducción
	CurrentTrackDuration string // Duración formateada ("MM:SS") de la pista activa
	TrackCount           int    // Cantidad total de pistas disponibles en la biblioteca
	Volume               int    // Nivel de volumen normalizado en escala porcentual (0 - 100%)
	IsMuted              bool   // Indica si el audio se encuentra actualmente silenciado
	HasTrack             bool   // Indica si existe una pista seleccionada en el reproductor
}

// SelectMsg es el mensaje emitido por el modelo de menú cuando el usuario confirma una selección,
// transportando la acción asociada y la etiqueta legible de la opción elegida.
type SelectMsg struct {
	Action Action // Acción asociada al elemento seleccionado
	Choice string // Etiqueta descriptiva del elemento seleccionado
}

// menuItem modela un ítem individual en la lista navegable de opciones del menú principal.
type menuItem struct {
	action Action // Acción que se ejecutará al seleccionar esta opción
	label  string // Texto descriptivo visible para el usuario
	icon   string // Icono o emoji representativo de la opción
	badge  string // Dígito o tecla de acceso rápido asociada a la opción
}

// Model gestiona el estado interno del componente de menú interactivo,
// incluyendo el cursor de navegación, las dimensiones del terminal y los datos del reproductor.
type Model struct {
	cursor int         // Posición del cursor en la lista de opciones (0-indexada)
	state  PlayerState // Estado sincronizado del reproductor de audio
	width  int         // Ancho del terminal en columnas
	height int         // Alto del terminal en filas
}

// New inicializa y devuelve una nueva instancia del modelo de menú con el cursor en la primera posición.
func New() Model {
	return Model{
		cursor: 0,
	}
}

// SetState actualiza directamente la información del reproductor de audio reflejada en la tarjeta del menú.
func (m *Model) SetState(s PlayerState) {
	m.state = s
}

// Init inicializa el componente de menú conforme a la especificación de Bubble Tea.
func (m Model) Init() tea.Cmd {
	return nil
}

// getMenuItems construye la lista de elementos del menú adaptando dinámicamente
// la etiqueta y el icono de reproducción según el estado actual del reproductor (reproducir, pausar o reanudar).
func (m Model) getMenuItems() []menuItem {
	var playLabel, playIcon string
	if m.state.IsPlaying {
		playLabel = "Pausar música"
		playIcon = "⏸"
	} else if m.state.IsPaused {
		playLabel = "Reanudar música"
		playIcon = "▶"
	} else {
		playLabel = "Reproducir pista"
		playIcon = "▶"
	}

	return []menuItem{
		{action: ActionTogglePlay, label: playLabel, icon: playIcon, badge: "1"},
		{action: ActionViewPlayer, label: "Ir al reproductor (Vista completa)", icon: "📺", badge: "2"},
		{action: ActionNextTrack, label: "Siguiente canción", icon: "⏭", badge: "3"},
		{action: ActionPrevTrack, label: "Anterior canción", icon: "⏮", badge: "4"},
		{action: ActionLibrary, label: "Biblioteca musical", icon: "📁", badge: "5"},
		{action: ActionSettings, label: "Configuración (Cambiar carpeta)", icon: "⚙", badge: "6"},
		{action: ActionQuit, label: "Salir", icon: "🚪", badge: "7"},
	}
}

// Update procesa los eventos y mensajes del ciclo de vida de Bubble Tea, incluyendo
// redimensionamientos del terminal, actualización de telemetría y navegación mediante teclado.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Actualizar dimensiones para centrado y cálculo de layout
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		items := m.getMenuItems()

		switch msg.String() {
		case "ctrl+c", "q":
			// Salida directa de la aplicación
			return m, tea.Quit

		case "p", "esc":
			// Atajo directo para navegar a la vista completa del reproductor
			return m, func() tea.Msg {
				return SelectMsg{
					Action: ActionViewPlayer,
					Choice: "Ir al reproductor (Vista completa)",
				}
			}

		case "up", "k":
			// Navegación vertical ascendente con rotación cíclica
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(items) - 1
			}

		case "down", "j":
			// Navegación vertical descendente con rotación cíclica
			if m.cursor < len(items)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}

		case "1", "2", "3", "4", "5", "6", "7":
			// Selección directa inmediata mediante tecla numérica rápida
			idx := int(msg.Runes[0] - '1')
			if idx >= 0 && idx < len(items) {
				m.cursor = idx
				return m.handleSelection(items[idx])
			}

		case "enter", " ":
			// Confirmación de la opción actualmente destacada por el cursor
			if m.cursor >= 0 && m.cursor < len(items) {
				return m.handleSelection(items[m.cursor])
			}
		}
	}
	return m, nil
}

// handleSelection centraliza la lógica de ejecución al confirmar una opción del menú,
// emitiendo el comando de finalización en caso de salida o despachando el mensaje SelectMsg correspondiente.
func (m Model) handleSelection(item menuItem) (tea.Model, tea.Cmd) {
	if item.action == ActionQuit {
		return m, tea.Quit
	}
	return m, func() tea.Msg {
		return SelectMsg{
			Action: item.action,
			Choice: item.label,
		}
	}
}
