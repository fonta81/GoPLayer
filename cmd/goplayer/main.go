package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fonta81/Reproductor-go/internal/player/config"
	"github.com/fonta81/Reproductor-go/internal/player/tui"
)

func main() {
	dirFlag := flag.String("dir", "", "Directorio inicial de música a escanear")
	flag.Parse()

	if *dirFlag == "" {
		if saved, err := config.LoadMusicDir(); err == nil && saved != "" {
			*dirFlag = saved
		}
	}

	model := tui.NewRootModel(*dirFlag)
	defer model.Close()

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Printf("Error al iniciar el reproductor: %v\n", err)
		os.Exit(1)
	}
}
