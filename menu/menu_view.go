package menu

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Paleta de colores para los elementos del menú inspirada en el tema Catppuccin Mocha.
var (
	colorAccent    = lipgloss.Color("#CBA6F7") // Lavanda / Acento principal para bordes y títulos destacados
	colorSecondary = lipgloss.Color("#94E2D5") // Cyan / Teal para texto secundario e insignias numéricas
	colorGreen     = lipgloss.Color("#A6E3A1") // Verde para indicar estado de reproducción activa
	colorYellow    = lipgloss.Color("#F9E2AF") // Amarillo para indicar estado de reproducción en pausa
	colorRed       = lipgloss.Color("#F38BA8") // Rojo para indicar silenciamiento (Mute) y errores
	colorText      = lipgloss.Color("#CDD6F4") // Color de texto principal de alto contraste
	colorSubtext   = lipgloss.Color("#A6ADC8") // Color de texto secundario para subtítulos y artistas
	colorMuted     = lipgloss.Color("#6C7086") // Color atenuado para metadatos complementarios y atajos
	colorSurface   = lipgloss.Color("#313244") // Fondo para resaltar la opción actualmente seleccionada

	// menuBoxStyle define el contenedor exterior que engloba todo el contenido del menú.
	menuBoxStyle = lipgloss.NewStyle().
			Width(60).
			Padding(0, 1)

	// headerBoxStyle define el contenedor con borde redondeado para el encabezado del menú.
	headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Width(56).
			Align(lipgloss.Center).
			Padding(0, 1).
			MarginBottom(1)

	// headerTitleStyle define la tipografía en negrita y color lavanda para el título principal.
	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	// headerSubStyle define el estilo en cursiva y color atenuado para el subtítulo del menú.
	headerSubStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Italic(true)

	// cardStyle define el marco de la tarjeta informativa "Now Playing" con bordes redondeados.
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Width(56).
			Padding(0, 1).
			MarginBottom(1)

	// cardHeaderStyle define la apariencia del indicador de estado en la cabecera de la tarjeta.
	cardHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(0)

	// trackTitleStyle define el estilo en negrita y alto contraste para el título de la canción activa.
	trackTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)

	// trackArtistStyle define el color secundario para el artista o intérprete de la pista activa.
	trackArtistStyle = lipgloss.NewStyle().
				Foreground(colorSecondary)

	// metaInfoStyle define el estilo para la información complementaria (volumen, total de pistas).
	metaInfoStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// itemBadgeNormal define el formato visual de los atajos numéricos [1]-[7] para opciones no seleccionadas.
	itemBadgeNormal = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// itemBadgeSelected define el formato visual de los atajos numéricos [1]-[7] para la opción bajo el cursor.
	itemBadgeSelected = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	// selectedItemStyle define el fondo y resaltado visual para la opción actualmente destacada por el cursor.
	selectedItemStyle = lipgloss.NewStyle().
				Background(colorSurface).
				Foreground(colorAccent).
				Bold(true).
				Padding(0, 1).
				Width(56)

	// unselectedItemStyle define la apariencia estándar para las opciones no seleccionadas del menú.
	unselectedItemStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Padding(0, 1).
				Width(56)

	// footerStyle define la presentación del pie de página con los atajos de teclado y ayuda rápida.
	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Align(lipgloss.Center).
			Width(56).
			MarginTop(1)
)

// renderNowPlayingCard genera la tarjeta visual "Now Playing" con información sobre el estado
// de la reproducción, la pista activa, el volumen actual y el total de canciones cargadas en la biblioteca.
func (m Model) renderNowPlayingCard() string {
	var borderColor lipgloss.Color
	var statusBadge string

	// Definir color de borde e insignia según el estado del motor de sonido
	if m.state.IsPlaying {
		borderColor = colorGreen
		statusBadge = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("▶ REPRODUCIENDO")
	} else if m.state.IsPaused {
		borderColor = colorYellow
		statusBadge = lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render("⏸ PAUSADO")
	} else {
		borderColor = colorMuted
		statusBadge = lipgloss.NewStyle().Foreground(colorSubtext).Bold(true).Render("⏹ DETENIDO")
	}

	// Renderizado del indicador de volumen y silencio
	var volText string
	if m.state.IsMuted {
		volText = lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("󰖁 MUTE")
	} else {
		filled := m.state.Volume / 10
		if filled > 10 {
			filled = 10
		} else if filled < 0 {
			filled = 0
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
		volText = fmt.Sprintf("🔊 %s %d%%", bar, m.state.Volume)
	}

	trackCountText := fmt.Sprintf("📁 %d pistas", m.state.TrackCount)
	footerMeta := metaInfoStyle.Render(fmt.Sprintf("%s   •   %s", volText, trackCountText))

	var b strings.Builder
	b.WriteString(cardHeaderStyle.Render(statusBadge) + "\n")

	// Renderizado de metadatos de la pista en reproducción
	if m.state.HasTrack && m.state.CurrentTrackTitle != "" {
		b.WriteString(trackTitleStyle.Render("🎵 "+truncateText(m.state.CurrentTrackTitle, 45)) + "\n")

		artist := m.state.CurrentTrackArtist
		if artist == "" {
			artist = "Artista desconocido"
		}
		b.WriteString(trackArtistStyle.Render("👤 "+truncateText(artist, 45)) + "\n")
	} else {
		if m.state.TrackCount == 0 {
			b.WriteString(metaInfoStyle.Render("Sin pistas en biblioteca. Pulsa [5] o [6] para explorar carpetas.") + "\n")
		} else {
			b.WriteString(metaInfoStyle.Render("Reproducción en espera. Selecciona [1] para iniciar.") + "\n")
		}
	}

	b.WriteString(footerMeta)

	return cardStyle.Copy().BorderForeground(borderColor).Render(b.String())
}

// truncateText acota una cadena de texto a un número máximo de runas agregando puntos suspensivos ("...")
// si se sobrepasa dicho límite, preservando caracteres multibyte y evitando desbordamientos visuales.
func truncateText(str string, maxLen int) string {
	runes := []rune(str)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return str
}

// View renderiza la interfaz visual completa del menú, reuniendo el encabezado,
// la tarjeta informativa de reproducción, las opciones seleccionables y los atajos de teclado,
// aplicando centrado geométrico si se conocen las dimensiones del terminal.
func (m Model) View() string {
	var b strings.Builder

	// 1. Encabezado principal estilizado
	headerContent := fmt.Sprintf("%s\n%s",
		headerTitleStyle.Render("🎵  G O P L A Y E R  •  M E N Ú"),
		headerSubStyle.Render("Reproductor de Audio TUI"),
	)
	b.WriteString(headerBoxStyle.Render(headerContent))
	b.WriteString("\n")

	// 2. Tarjeta widget "Now Playing" con telemetría en tiempo real
	b.WriteString(m.renderNowPlayingCard())
	b.WriteString("\n")

	// 3. Renderizado de las opciones de menú interactivas
	items := m.getMenuItems()
	for i, item := range items {
		if m.cursor == i {
			line := fmt.Sprintf("▸ %s  %s %s",
				itemBadgeSelected.Render("["+item.badge+"]"),
				item.icon,
				item.label,
			)
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			line := fmt.Sprintf("  %s  %s %s",
				itemBadgeNormal.Render("["+item.badge+"]"),
				item.icon,
				item.label,
			)
			b.WriteString(unselectedItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// 4. Pie de página informativo con atajos rápidos de ayuda
	helpText := "1-7: selección • ↑/↓: navegar • enter: elegir\nesc/p: reproductor • q: salir"
	b.WriteString(footerStyle.Render(helpText))

	content := menuBoxStyle.Render(b.String())

	// Centrado adaptable en el terminal según ancho y alto disponibles
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return content
}
