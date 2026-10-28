package player

import tea "github.com/charmbracelet/bubbletea"

func (m AppModel) playbackPositionCmd() tea.Cmd {
	sessionID := m.Audio.playbackSessionID()
	audio := m.Audio
	return func() tea.Msg {
		return playbackPositionMsg{
			sessionID: sessionID,
			position:  audio.Position(),
		}
	}
}
