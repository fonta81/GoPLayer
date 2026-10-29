package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fonta81/Reproductor-go/internal/player/config"
	"github.com/fonta81/Reproductor-go/internal/player/format"
	"github.com/fonta81/Reproductor-go/internal/player/model"
	"github.com/fonta81/Reproductor-go/menu"
)

type Screen int

const (
	ScreenMenu Screen = iota
	ScreenPlayer
)

type RootModel struct {
	currentScreen Screen
	menuModel     menu.Model
	playerModel   AppModel
}

func NewRootModel(initialDir string) RootModel {
	r := RootModel{currentScreen: ScreenMenu, menuModel: menu.New(), playerModel: NewAppModel(initialDir)}
	r.syncMenuState()
	return r
}

func (r *RootModel) syncMenuState() {
	var trackCount int
	if r.playerModel.playlist != nil {
		trackCount = r.playerModel.playlist.Length()
	}

	volPct := format.VolumePercentageInt(r.playerModel.volumeLevel, minVolume, maxVolume)
	isMuted := false
	if r.playerModel.Audio != nil {
		isMuted = r.playerModel.Audio.IsMuted()
	}

	state := menu.PlayerState{TrackCount: trackCount, Volume: volPct, IsMuted: isMuted}
	switch r.playerModel.state {
	case model.StatePlaying:
		state.State = "Reproduciendo"
		state.IsPlaying = true
	case model.StatePaused:
		state.State = "Pausado"
		state.IsPaused = true
	default:
		state.State = "Detenido"
	}

	if r.playerModel.playlist != nil {
		if track, ok := r.playerModel.playlist.Current(); ok {
			state.HasTrack = true
			state.CurrentTrackTitle = track.Title
			state.CurrentTrackArtist = track.Artist
			if track.Duration > 0 {
				state.CurrentTrackDuration = format.Duration(track.Duration)
			}
		}
	}

	r.menuModel.SetState(state)
}

func (r RootModel) Init() tea.Cmd {
	r.syncMenuState()
	return tea.Batch(r.menuModel.Init(), r.playerModel.Init())
}

func (r RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var cmdMenu, cmdPlayer tea.Cmd
		var mm, pm tea.Model

		mm, cmdMenu = r.menuModel.Update(msg)
		r.menuModel = mm.(menu.Model)

		pm, cmdPlayer = r.playerModel.Update(msg)
		r.playerModel = pm.(AppModel)

		r.syncMenuState()
		return r, tea.Batch(cmdMenu, cmdPlayer)

	case menu.SelectMsg:
		switch msg.Action {
		case menu.ActionTogglePlay:
			if r.playerModel.state == model.StateStopped {
				if r.playerModel.playlist.Length() > 0 {
					var pm tea.Model
					var cmd tea.Cmd
					pm, cmd = r.playerModel.playCurrent()
					r.playerModel = pm.(AppModel)
					r.currentScreen = ScreenPlayer
					r.syncMenuState()
					return r, cmd
				}
				r.syncMenuState()
				return r, nil
			}
			var pm tea.Model
			var cmd tea.Cmd
			pm, cmd = r.playerModel.togglePlayback()
			r.playerModel = pm.(AppModel)
			r.syncMenuState()
			return r, cmd

		case menu.ActionViewPlayer:
			r.currentScreen = ScreenPlayer
			r.syncMenuState()
			return r, nil

		case menu.ActionNextTrack:
			var pm tea.Model
			var cmd tea.Cmd
			pm, cmd = r.playerModel.playNext()
			r.playerModel = pm.(AppModel)
			r.syncMenuState()
			return r, cmd

		case menu.ActionPrevTrack:
			var pm tea.Model
			var cmd tea.Cmd
			pm, cmd = r.playerModel.playPrevious()
			r.playerModel = pm.(AppModel)
			r.syncMenuState()
			return r, cmd

		case menu.ActionLibrary:
			r.currentScreen = ScreenPlayer
			r.playerModel.isPickingFolder = false
			r.playerModel.isFiltering = false
			r.syncMenuState()
			return r, nil

		case menu.ActionSettings:
			r.currentScreen = ScreenPlayer
			r.playerModel.isFiltering = false
			cmd := r.playerModel.openBrowser(r.playerModel.musicDir)
			r.syncMenuState()
			return r, cmd

		case menu.ActionQuit:
			if r.playerModel.Audio.HasEQ() {
				_ = config.SaveEQ(r.playerModel.Audio.EQEnabled(), r.playerModel.Audio.GetEQBands())
			}
			r.playerModel.Audio.Close()
			return r, tea.Quit
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if r.playerModel.Audio.HasEQ() {
				_ = config.SaveEQ(r.playerModel.Audio.EQEnabled(), r.playerModel.Audio.GetEQBands())
			}
			r.playerModel.Audio.Close()
			return r, tea.Quit
		}

		if r.currentScreen == ScreenMenu {
			if msg.String() == "esc" || msg.String() == "p" {
				r.currentScreen = ScreenPlayer
				r.syncMenuState()
				return r, nil
			}
			var menuModel tea.Model
			var cmd tea.Cmd
			menuModel, cmd = r.menuModel.Update(msg)
			r.menuModel = menuModel.(menu.Model)
			r.syncMenuState()
			cmds = append(cmds, cmd)
			return r, tea.Batch(cmds...)
		}

		if msg.String() == "esc" && !r.playerModel.isFiltering && !r.playerModel.isPickingFolder && !r.playerModel.eqActive {
			r.currentScreen = ScreenMenu
			r.syncMenuState()
			return r, nil
		}

		var playerModel tea.Model
		var cmd tea.Cmd
		playerModel, cmd = r.playerModel.Update(msg)
		r.playerModel = playerModel.(AppModel)
		r.syncMenuState()
		cmds = append(cmds, cmd)
		return r, tea.Batch(cmds...)
	}

	var menuModel tea.Model
	var playerModel tea.Model
	var cmdMenu, cmdPlayer tea.Cmd

	menuModel, cmdMenu = r.menuModel.Update(msg)
	r.menuModel = menuModel.(menu.Model)

	playerModel, cmdPlayer = r.playerModel.Update(msg)
	r.playerModel = playerModel.(AppModel)

	r.syncMenuState()

	cmds = append(cmds, cmdMenu, cmdPlayer)
	return r, tea.Batch(cmds...)
}

func (r RootModel) View() string {
	if r.currentScreen == ScreenMenu {
		return r.menuModel.View()
	}
	return r.playerModel.View() + "\n"
}

func (r RootModel) Close() {
	r.playerModel.Audio.Close()
}
