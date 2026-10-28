package player

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderFilterBar genera el componente visual de la barra de búsqueda rápida, mostrando
// el campo de texto interactivo junto al conteo de canciones coincidentes.
func (m AppModel) renderFilterBar() string {
	if !m.isFiltering {
		return ""
	}
	count := len(m.filteredIndices)
	summary := fmt.Sprintf(" [%d matches]", count)
	bar := lipgloss.NewStyle().Foreground(cyan).Render("/ ") + lipgloss.NewStyle().Foreground(foreground).Render(m.filterInput.View())
	return lipgloss.JoinHorizontal(lipgloss.Left, bar, lipgloss.NewStyle().Foreground(comment).Render(summary))
}

// renderTrackRow construye y pinta una fila de pista con cursor, número, título truncado,
// duración y el estilo adecuado según si es la pista actual o la destacada.
func renderTrackRow(cursor string, index int, track Track, isCurrent, isHighlighted bool, titleWidth int) string {
	titleWidth = max(3, titleWidth)
	displayName := track.DisplayName()
	if titleWidth < 20 && track.Title != "" {
		displayName = track.Title
	}
	style := lipgloss.NewStyle()
	if isCurrent {
		style = style.Bold(true).Foreground(pink)
	} else if isHighlighted {
		style = style.Foreground(cyan)
	} else {
		style = style.Foreground(foreground)
	}
	row := fmt.Sprintf("%s%d. %-*s [%s]", cursor, index+1, titleWidth, truncate(displayName, titleWidth), track.FormattedDuration())
	return "  " + style.Render(row) + "\n"
}

// renderSuggestions renderiza la caja emergente con la lista compacta de mejores sugerencias
// calculadas a partir del filtro de búsqueda actual.
func (m AppModel) renderSuggestions() string {

	if !m.isFiltering || len(m.suggestionIndices) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(comment).Render(" Sugerencias:"))
	b.WriteString("\n")
	limit := len(m.suggestionIndices)
	for i := 0; i < limit; i++ {
		idx := m.suggestionIndices[i]
		t := m.playlist.tracks[idx]
		cursor := "   "
		if i == m.suggestionCursor {
			cursor = "→  "
		}
		b.WriteString(renderTrackRow(cursor, idx, t, idx == m.playlist.current, i == m.suggestionCursor, max(12, m.queueTitleWidth()-2)))
	}
	return b.String()
}

// View compone y renderiza la interfaz visual completa de la pantalla del reproductor
// reuniendo el encabezado, tarjeta de estado, explorador o cola de canciones y el panel de ayuda.
func (m AppModel) View() string {
	if m.lastError != nil {
		return m.renderErrorScreen()
	}

	sections := []string{m.renderHeader()}
	if m.isPickingFolder {
		sections = append(sections, "", m.renderNowPlayingPanel(), m.renderBrowserPanel())
	} else {
		if m.usesWideLayout() {
			sections = append(sections, "", lipgloss.JoinHorizontal(
				lipgloss.Top,
				m.renderNowPlayingPanel(),
				" ",
				m.renderQueueCard(),
			))
		} else {
			sections = append(sections, "", m.renderNowPlayingPanel())
		}

		if m.eqActive {
			sections = append(sections, m.renderEQPanel())
		}
		if m.showQueue && !m.usesWideLayout() {
			sections = append(sections, m.renderQueueCard())
		}
		if m.showHelp {
			sections = append(sections, "", m.renderHelpPanel())
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader construye el título de la aplicación y su subtítulo con estilos de Lipgloss.
func (m AppModel) renderHeader() string {
	contentWidth := 70
	if m.width > 0 {
		contentWidth = min(contentWidth, max(24, m.width-10))
	}
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Bold(true).Foreground(pink).Render("🎵  G O P L A Y E R"),
		lipgloss.NewStyle().Foreground(comment).Italic(true).Render(appSubtitle),
	)
	header := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Width(contentWidth).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(content)
	if m.width > 0 {
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, header)
	}
	return header
}

// renderNowPlayingPanel construye la tarjeta central que muestra la pista activa,
// la barra de progreso, los metadatos de audio y la ruta del directorio actual.
func (m AppModel) renderNowPlayingPanel() string {
	track, hasTrack := m.playlist.Current()
	var content strings.Builder

	if hasTrack {
		content.WriteString(m.renderStatusLine(track))
		content.WriteString("\n")
		content.WriteString(m.renderProgressBar())
		content.WriteString("\n")
		content.WriteString(m.renderMetadataLine())
	} else {
		content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(red).Render(iconStop + " Sin canciones\n"))
		prompt := truncate("Presiona 'o' para explorar carpetas y agregar música.", max(12, m.nowPlayingContentWidth()-4))
		content.WriteString(lipgloss.NewStyle().Foreground(comment).Render(prompt))
	}

	dir := m.musicDir
	if dir == "" {
		dir = "Sin directorio seleccionado"
	}
	dirWidth := max(12, m.nowPlayingContentWidth()-18)
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(comment).Render(iconFolder + " " + truncate(dir, dirWidth)))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.playbackBorderColor()).
		Width(m.nowPlayingContentWidth()).
		Padding(1, 2).
		Render(content.String())
}

// renderStatusLine devuelve el estado de reproducción junto al título y artista de la pista activa.
func (m AppModel) renderStatusLine(track Track) string {
	var icon string
	var style lipgloss.Style

	switch m.state {
	case StatePlaying:
		icon, style = iconPlay, lipgloss.NewStyle().Bold(true).Foreground(green)
	case StatePaused:
		icon, style = iconPause, lipgloss.NewStyle().Bold(true).Foreground(yellow)
	default:
		icon, style = iconStop, lipgloss.NewStyle().Bold(true).Foreground(red)
	}
	titleWidth := max(12, m.nowPlayingContentWidth()-4)
	title := lipgloss.NewStyle().Bold(true).Foreground(foreground).Render(truncate(track.Title, titleWidth))
	artist := track.Artist
	if artist == "" {
		artist = "Artista desconocido"
	}
	artist = lipgloss.NewStyle().Foreground(cyan).Render(truncate(artist, titleWidth))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		style.Render(icon+" "+m.state.Label()),
		title,
		artist,
	)
}

// renderProgressBar dibuja la barra de progreso proporcional al tiempo transcurrido
// junto a los indicadores numéricos de tiempo ("MM:SS / MM:SS").
func (m AppModel) renderProgressBar() string {
	percent := 0.0
	if m.totalTime > 0 {
		percent = float64(m.elapsed) / float64(m.totalTime)
	}

	// Asignar colores de tema a la barra de progreso
	m.progressBar.FullColor = string(pink)
	m.progressBar.EmptyColor = string(selection)

	progressBar := m.progressBar
	progressBar.Width = min(progressBar.Width, max(1, m.nowPlayingContentWidth()-18))
	bar := progressBar.ViewAs(percent)
	timeInfo := fmt.Sprintf(" %s / %s", formatDuration(m.elapsed), formatDuration(m.totalTime))
	return bar + lipgloss.NewStyle().Foreground(cyan).Render(timeInfo)
}

// renderMetadataLine genera la fila de información técnica: barra de volumen,
// número de pista en la cola, indicador de orden aleatorio e icono de repetición.
func (m AppModel) renderMetadataLine() string {
	queueDisplay := "0/0"
	if m.playlist.Length() > 0 {
		queueDisplay = fmt.Sprintf("%d/%d", m.playlist.current+1, m.playlist.Length())
	}

	shuffleIcon := ""
	if m.playlist.shuffle {
		shuffleIcon = lipgloss.NewStyle().Foreground(purple).Render(" " + iconShuffle + " ")
	}

	volBar := renderVolumeBar(m.volumeLevel, minVolume, maxVolume, m.Audio.IsMuted())
	if m.nowPlayingContentWidth() < 45 {
		modes := fmt.Sprintf("%s %s  |  %s", iconQueue, queueDisplay, m.playlist.repeat.Icon())
		if m.playlist.shuffle {
			modes += "  " + iconShuffle
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Foreground(comment).Render(volBar),
			lipgloss.NewStyle().Foreground(comment).Render(modes),
		)
	}
	return lipgloss.NewStyle().Foreground(comment).Render(
		fmt.Sprintf("%s  |  %s %s  |%s| %s ", volBar, iconQueue, queueDisplay, shuffleIcon, m.playlist.repeat.Icon()),
	)
}

func (m AppModel) playbackBorderColor() lipgloss.Color {
	switch m.state {
	case StatePlaying:
		return green
	case StatePaused:
		return yellow
	default:
		return purple
	}
}

func (m AppModel) usesWideLayout() bool {
	return m.width >= 120 && m.showQueue && !m.isPickingFolder && !m.eqActive
}

func (m AppModel) nowPlayingContentWidth() int {
	if m.usesWideLayout() {
		return max(24, (m.width-3)/2-6)
	}
	if m.width > 0 {
		return max(24, min(100, m.width-14))
	}
	return 70
}

func (m AppModel) queueTitleWidth() int {
	indexWidth := len(fmt.Sprint(max(1, m.playlist.Length())))
	return max(3, min(45, m.nowPlayingContentWidth()-15-indexWidth))
}

func (m AppModel) renderQueueCard() string {
	var content strings.Builder
	if m.isFiltering {
		content.WriteString(m.renderFilterBar())
		content.WriteString("\n")
		content.WriteString(m.renderSuggestions())
	}
	content.WriteString(m.renderPlaylistPanel())
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Width(m.nowPlayingContentWidth()).
		Padding(1, 2).
		Render(content.String())
}

// renderEQPanel dibuja el panel del ecualizador con una columna por banda,
// barras proporcionales a la ganancia en dB y la banda activa resaltada.
func (m AppModel) renderEQPanel() string {
	bands := m.Audio.GetEQBands()
	labels := []string{" 60Hz", "250Hz", "  1kHz", "  4kHz", " 10kHz"}
	levels := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	var b strings.Builder
	status := lipgloss.NewStyle().Foreground(green).Render("ON")
	if !m.Audio.EQEnabled() {
		status = lipgloss.NewStyle().Foreground(red).Render("OFF (bypass)")
	}
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(orange).Render("  Ecualizador ") + status + "\n\n")

	for i := 0; i < eqBandCount; i++ {
		g := bands[i]
		filled := int((g - eqGainMin) / (eqGainMax - eqGainMin) * float64(len(levels)-1))
		label := labels[i]
		style := lipgloss.NewStyle().Foreground(foreground)
		barStyle := lipgloss.NewStyle().Foreground(cyan)
		if i == m.eqCursor {
			label = ">" + label[1:]
			style = lipgloss.NewStyle().Bold(true).Foreground(pink)
			barStyle = lipgloss.NewStyle().Foreground(pink)
		}
		bar := ""
		for j := 0; j < len(levels); j++ {
			if j <= filled {
				bar += levels[j]
			} else {
				bar += " "
			}
		}
		b.WriteString(style.Render(label) + " " + barStyle.Render(bar) + style.Render(fmt.Sprintf(" %+5.0f dB\n", g)))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(comment).Render("\n←/→ banda | ↑/↓ ±1dB | 0 reset | x on/off | Esc cerrar"))

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(purple).Padding(0, 2).MarginLeft(2).Render(b.String())
}

// renderPlaylistPanel devuelve una representación en lista de la cola de canciones.
// Si el filtro de búsqueda está activo, muestra una ventana centrada en el cursor de búsqueda;
// en caso contrario, muestra una ventana alrededor del cursor de la lista de reproducción.
func (m AppModel) renderPlaylistPanel() string {
	var builder strings.Builder
	builder.WriteString(lipgloss.NewStyle().Bold(true).Foreground(orange).Render("  " + iconQueue + " Próximamente:"))
	builder.WriteString("\n")
	if m.playlist.IsEmpty() {
		builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("  Aún no hay pistas en la cola."))
		return builder.String()
	}

	titleWidth := m.queueTitleWidth()

	// Vista con filtro de búsqueda activo
	if m.isFiltering {
		if m.filteredIndices == nil || len(m.filteredIndices) == 0 {
			builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("  (No hay coincidencias)") + "\n")
			return builder.String()
		}

		start := max(0, m.filterCursor-3)
		end := min(len(m.filteredIndices), start+7)

		if start > 0 {
			builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("    " + iconUp + " ...\n"))
		}

		for idx := start; idx < end; idx++ {
			i := m.filteredIndices[idx]
			t := m.playlist.tracks[i]
			cursor := "   "
			if idx == m.filterCursor {
				cursor = "→  "
			}
			builder.WriteString(renderTrackRow(cursor, i, t, i == m.playlist.current, idx == m.filterCursor, titleWidth))
		}

		if end < len(m.filteredIndices) {
			builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("    " + iconDown + " ...\n"))
		}

		return builder.String()
	}

	// Vista secuencial estándar de la cola de pistas
	start := max(0, m.cursorIndex-3)
	end := min(m.playlist.Length(), start+7)

	if start > 0 {
		builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("    " + iconUp + " ...\n"))
	}

	for i := start; i < end; i++ {
		t := m.playlist.tracks[i]
		cursor := "   "
		if i == m.cursorIndex {
			cursor = "→  "
		}
		builder.WriteString(renderTrackRow(cursor, i, t, i == m.playlist.current, i == m.cursorIndex, titleWidth))
	}

	if end < m.playlist.Length() {
		builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("    " + iconDown + " ...\n"))
	}

	return builder.String()
}

// renderBrowserPanel dibuja el panel interactivo del explorador de carpetas,
// mostrando los archivos de audio y subdirectorios para su selección.
func (m AppModel) renderBrowserPanel() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(cyan).Render(iconFolder + " Explorador de Carpetas: " + m.browserPath)

	var builder strings.Builder
	builder.WriteString(header)
	builder.WriteString("\n")
	builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("  ←/Retroceso: Subir | →/Enter: Entrar | Espacio: Confirmar Directorio | Esc: Cancelar"))
	builder.WriteString("\n\n")

	if len(m.browserEntries) == 0 {
		builder.WriteString(lipgloss.NewStyle().Foreground(comment).Render("  (Directorio vacío, sin subcarpetas ni pistas de audio)\n"))
	}

	start := max(0, m.browserCursor-5)
	end := min(len(m.browserEntries), start+10)

	for i := start; i < end; i++ {
		entry := m.browserEntries[i]
		cursor := "  "

		icon := iconFolder
		style := lipgloss.NewStyle().Foreground(foreground)
		if !entry.isDir {
			icon = iconAudio
			style = style.Foreground(comment)
		}

		if i == m.browserCursor {
			cursor = "→ "
			style = style.Foreground(pink).Bold(true)
		}

		builder.WriteString(style.Render(cursor + icon + " " + entry.name))
		builder.WriteString("\n")
	}

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cyan).Padding(1, 2).MarginLeft(2).Render(builder.String())
}

// renderHelpPanel dibuja el catálogo de atajos de teclado agrupados temáticamente
// (Reproducción, Navegación, Audio y Sistema), adaptando el número de columnas al ancho del terminal.
func (m AppModel) renderHelpPanel() string {
	type binding struct{ key, desc string }
	type category struct {
		title    string
		bindings []binding
	}

	categories := []category{
		{iconPlay + " Reproducción", []binding{{"espacio", "Play / Pausa"}, {"n / N", "Sig / Anterior"}, {"> / <", "Adel. / Atrasar"}, {"0", "Reiniciar"}}},
		{iconNav + " Navegación", []binding{{"↑↓ / jk", "Mover cursor"}, {"enter", "Reproducir"}, {"d", "Eliminar de cola"}, {"l", "Ocultar cola"}, {"o", "Explorar carpetas"}, {"ctrl+f", "Buscar / Filtrar"}}},
		{iconAudio + " Audio & Modos", []binding{{"+ / -", "Volumen"}, {"m", "Silenciar"}, {"r", "Repetir"}, {"s", "Aleatorio"}, {"e", "Ecualizador"}}},
		{iconSystem + " Sistema", []binding{{"h / ?", "Ocultar ayuda"}, {"q", "Salir"}}},
	}

	var blocks []string
	for _, cat := range categories {
		var lines []string
		lines = append(lines, helpCategoryStyle.Render(cat.title))
		for _, b := range cat.bindings {
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, helpKeyStyle.Render(b.key), helpDescStyle.Render(b.desc)))
		}
		blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, lines...))
	}

	var rows []string
	colsPerRow := 1
	if m.width > 150 {
		colsPerRow = 4
	} else if m.width > 75 {
		colsPerRow = 2
	}

	for i := 0; i < len(blocks); i += colsPerRow {
		end := min(i+colsPerRow, len(blocks))
		var rowBlocks []string
		for _, block := range blocks[i:end] {
			rowBlocks = append(rowBlocks, lipgloss.NewStyle().Width(32).Render(block))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowBlocks...))
		if end < len(blocks) {
			rows = append(rows, "")
		}
	}

	return helpContainerStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// renderErrorScreen construye la vista de alerta crítica con fondo rojo en caso de errores graves.
func (m AppModel) renderErrorScreen() string {
	if m.lastError == nil {
		return ""
	}
	return lipgloss.NewStyle().Bold(true).Foreground(foreground).Background(red).Padding(1, 2).Render(
		fmt.Sprintf("CRITICAL ERROR:\n\n%v\n\nEl sistema intentará recuperarse en 5 segundos o presiona 'q' para salir.", m.lastError),
	)
}
