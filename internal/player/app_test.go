package player

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestAppViewAdaptsToTerminalWidth(t *testing.T) {
	model := NewAppModel("")
	defer model.Audio.Close()
	model.playlist.Add(Track{
		Title:    "Una canción de prueba",
		Artist:   "Artista de prueba",
		Duration: 4 * time.Minute,
	})
	model.playlist.Add(Track{
		Title:    "Otra canción",
		Artist:   "Otro artista",
		Duration: 3 * time.Minute,
	})
	model.state = StatePlaying
	model.elapsed = 45 * time.Second
	model.totalTime = 4 * time.Minute

	for _, width := range []int{40, 80, 140} {
		t.Run(fmt.Sprintf("width-%d", width), func(t *testing.T) {
			updated, _ := model.Update(tea.WindowSizeMsg{Width: width, Height: 40})
			view := updated.(AppModel).View()

			for _, text := range []string{"G O P L A Y E R", "Próximamente", "Otra"} {
				if !strings.Contains(view, text) {
					t.Errorf("la vista de %d columnas debería incluir %q:\n%s", width, text, view)
				}
			}
			if !strings.Contains(view, "Una canción") || !strings.Contains(view, "Artista") {
				t.Errorf("la vista de %d columnas debería mostrar título y artista", width)
			}

			for i, line := range strings.Split(view, "\n") {
				if got := lipgloss.Width(line); got > width {
					t.Errorf("línea %d excede el ancho del terminal: %d > %d", i+1, got, width)
				}
			}
		})
	}
}

func TestAppProgressBarFitsNarrowTerminal(t *testing.T) {
	model := NewAppModel("")
	defer model.Audio.Close()

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	model = updated.(AppModel)
	if model.progressBar.Width < 1 || model.progressBar.Width > 40 {
		t.Errorf("ancho de progreso no adaptado al terminal: %d", model.progressBar.Width)
	}
}
