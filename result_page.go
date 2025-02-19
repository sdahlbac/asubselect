package main

import (
	"io"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type resultPageModel struct {
	changed bool
	dump    io.Writer
	timer   timer.Model
}

func NewResultPageModel(changed bool) resultPageModel {
	var dump *os.File
	if _, ok := os.LookupEnv("DEBUG"); ok {
		var err error
		dump, err = os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			os.Exit(1)
		}
	}
	return resultPageModel{
		changed: changed,
		dump:    dump,
		timer:   timer.New(1 * time.Second),
	}
}

func (m resultPageModel) Init() tea.Cmd {
	return m.timer.Init()
}

func (m resultPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	t, cmd := m.timer.Update(msg)
	m.timer = t
	return m, cmd
}

func (m resultPageModel) View() string {
	var text string
	var color lipgloss.Color
	if m.changed {
		text = "Currently selected subscription switched"
		color = Success
	} else {
		text = "No change, currently selected subscription remains the same"
		color = Info
	}
	return lipgloss.
		NewStyle().
		Height(height).
		Width(width).
		Foreground(color).
		AlignVertical(lipgloss.Center).
		AlignHorizontal(lipgloss.Center).
		Render(text)
}
