package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fonta81/Reproductor-go/internal/player/model"
)

const (
	appName         = "GoPlayer"
	appSubtitle     = "Reproductor de música TUI"
	defaultDir      = "./music"
	seekSeconds     = 10
	progressWidth   = 50
	defaultDuration = 3 * time.Minute
	volumeStep      = 3.0
	maxVolume       = 0.0
	minVolume       = -30.0
)

const (
	iconPlay      = "󰐊"
	iconPause     = "󰏤"
	iconStop      = "󰓛"
	iconRepeatOne = "󰑘"
	iconRepeatAll = "󰑗"
	iconRepeatOff = "󰁔"
	iconShuffle   = "󰒟"
	iconVolume    = "󰕾"
	iconMute      = "󰖁"
	iconQueue     = "󰉖"
	iconUp        = "󰅂"
	iconDown      = "󰅀"
	iconNav       = "󰍉"
	iconAudio     = "󰕾"
	iconSystem    = "󰒓"
	iconFolder    = "󰉋"
)

var (
	pink       = lipgloss.Color("#F5C2E7")
	cyan       = lipgloss.Color("#94E2D5")
	green      = lipgloss.Color("#A6E3A1")
	yellow     = lipgloss.Color("#F9E2AF")
	orange     = lipgloss.Color("#FAB387")
	red        = lipgloss.Color("#F38BA8")
	purple     = lipgloss.Color("#CBA6F7")
	foreground = lipgloss.Color("#CDD6F4")
	comment    = lipgloss.Color("#7F849C")
	selection  = lipgloss.Color("#313244")

	helpContainerStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(selection).Padding(1, 2).MarginLeft(2).MarginTop(1)
	helpCategoryStyle  = lipgloss.NewStyle().Foreground(purple).Bold(true).MarginBottom(1)
	helpKeyStyle       = lipgloss.NewStyle().Background(selection).Foreground(cyan).Bold(true).Align(lipgloss.Center).Width(10).MarginRight(1).MarginBottom(1)
	helpDescStyle      = lipgloss.NewStyle().Foreground(comment).MarginBottom(1)
)

func repeatIcon(r model.RepeatMode) string {
	switch r {
	case model.RepeatOne:
		return iconRepeatOne
	case model.RepeatAll:
		return iconRepeatAll
	default:
		return iconRepeatOff
	}
}
