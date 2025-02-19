package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle      = lipgloss.NewStyle().Margin(1, 2)
	selectedId    string
	width, height int
)

type model struct {
	loading        bool
	spinner        spinner.Model
	list           list.Model
	finalPageModel resultPageModel
	err            error
}

type SubInfo struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
	User      struct {
		Name string `json:"name"`
	}
}

func (si SubInfo) Title() string       { return si.Name }
func (si SubInfo) Description() string { return fmt.Sprintf("%s (%s)", si.Id, si.User.Name) }
func (si SubInfo) FilterValue() string { return si.Name + "/" + si.User.Name }

func initialModel() model {
	items := make([]list.Item, 0)

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(Rosewater).BorderLeftForeground(ActiveBorder)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle // reuse the title style here
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(Text)
	d.Styles.NormalDesc = d.Styles.NormalTitle

	subscriptionList := list.New(items, d, 0, 0)
	subscriptionList.Styles.Title = subscriptionList.Styles.Title.Foreground(Text).Background(Mantle).BorderBottomBackground(ActiveBorder).Margin(0).Padding(1, 0, 0, 0)
	subscriptionList.Styles.FilterCursor = subscriptionList.Styles.FilterCursor.Foreground(Text)
	subscriptionList.Styles.FilterPrompt = subscriptionList.Styles.FilterPrompt.Foreground(Rosewater)
	subscriptionList.Title = "Select Azure Subscription"

	// Workaround so that we can set the styles
	subscriptionList.FilterInput.PromptStyle = subscriptionList.Styles.FilterPrompt.Foreground(Text).MarginLeft(1)
	subscriptionList.FilterInput.Cursor.Style = subscriptionList.Styles.FilterCursor.Foreground(Rosewater)

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(Text)
	m := model{
		loading: true,
		spinner: s,
		list:    subscriptionList,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadSubscriptionsCommand,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q,ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			if selectedSub, ok := m.list.SelectedItem().(SubInfo); ok {
				didChange, err := changeSelection(selectedSub)
				if err != nil {
					m.err = err
					return m, nil
				}
				fpm := NewResultPageModel(didChange)
				m.finalPageModel = fpm
				cmd := m.finalPageModel.Init()
				return m, cmd
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		width, height = msg.Width, msg.Height
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case timer.TimeoutMsg:
		return m, tea.Quit
	case subscriptionsCommand:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		}
		subs := msg.subs

		items := make([]list.Item, 0)
		for _, sub := range subs {
			items = append(items, sub)
		}
		currentSelection := slices.IndexFunc(subs, func(s SubInfo) bool {
			return s.IsDefault
		})
		if currentSelection != -1 {
			selectedId = subs[currentSelection].Id
		}
		cmd := m.list.SetItems(items)
		m.list.Select(currentSelection)
		return m, cmd
	}

	var cmd, fpmCmd, timerCmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	fpm, fpmCmd := m.finalPageModel.Update(msg)
	m.finalPageModel = fpm.(resultPageModel)

	return m, tea.Batch(cmd, fpmCmd, timerCmd)
}

// TODO: convert this to a command?
func changeSelection(selectedSub SubInfo) (bool, error) {
	if selectedSub.Id == selectedId {
		return false, nil
	}
	err := exec.Command("az", "account", "set", "--subscription", selectedSub.Id).Err

	if err != nil {
		return false, err
	}
	return true, nil
}

func (m model) View() string {
	if m.finalPageModel != (resultPageModel{}) {
		return m.finalPageModel.View()
	}
	if m.err != nil {
		return m.errorView()
	} else if m.loading {
		return m.loadingView()
	} else {
		return lipgloss.JoinHorizontal(lipgloss.Top, "  ", m.list.View())
	}
}

func (m model) loadingView() string {
	return lipgloss.
		NewStyle().
		Height(height).
		Width(width).
		Foreground(Text).
		AlignVertical(lipgloss.Center).
		AlignHorizontal(lipgloss.Center).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, m.spinner.View(), " Loading data..."))
}

func (m model) errorView() string {
	return lipgloss.
		NewStyle().
		Height(height).
		Width(width).
		Foreground(Error).
		AlignVertical(lipgloss.Center).
		AlignHorizontal(lipgloss.Center).
		Render(fmt.Sprintf("An error occured: %v\nPress q to quit.", m.err))
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type subscriptionsCommand struct {
	subs []SubInfo
	err  error
}

func loadSubscriptionsCommand() tea.Msg {
	subsData, err := exec.Command("az", "account", "list").Output()
	if err != nil {
		return subscriptionsCommand{subs: nil, err: err}
	}
	var subs []SubInfo
	err = json.Unmarshal(subsData, &subs)
	if err != nil {
		return subscriptionsCommand{subs: nil, err: err}
	}
	return subscriptionsCommand{subs: subs, err: nil}
}
