// Package main implements an Azure subscription selector TUI application.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed sampledata.json
var sampleData []byte

// Application constants
const (
	// ExitCodeSuccess indicates successful program execution
	ExitCodeSuccess = 0
	// ExitCodeError indicates an error occurred during execution
	ExitCodeError = 1

	// Environment variables
	EnvUseSampleData = "USE_SAMPLE_DATA"

	// Key bindings
	KeyQuit  = "q"
	KeyCtrlC = "ctrl+c"
	KeyEnter = "enter"

	// Azure CLI configuration
	AzureCommand = "az"

	// UI text
	AppTitle        = "Select Azure Subscription"
	LoadingMessage  = " Loading subscriptions..."
	ErrorPrefix     = "An error occurred: %v\nPress q to quit."

	// Result messages
	SuccessMessage = "Azure subscription successfully changed!"
	NoChangeMessage = "No change needed - subscription is already active"
)

// Application errors
var (
	ErrAzureCLINotFound = errors.New("azure CLI not found in PATH")
)

// Global styles
var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	// Dimensions are stored globally for simplicity in this small app
	width, height int
)

// AppState represents the current state of the application
type AppState int

const (
	StateLoading AppState = iota
	StateSelectingSubscription
	StateShowingResult
	StateError
)

// App represents the main application state
type App struct {
	state      AppState
	spinner    spinner.Model
	list       list.Model
	resultPage *ResultPage
	selectedID string
	err        error
}

// Subscription represents an Azure subscription
type Subscription struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
	User      struct {
		Name string `json:"name"`
	} `json:"user"`
}

// Title implements list.Item interface
func (s Subscription) Title() string {
	return s.Name
}

// Description implements list.Item interface
func (s Subscription) Description() string {
	return fmt.Sprintf("%s (%s)", s.ID, s.User.Name)
}

// FilterValue implements list.Item interface
func (s Subscription) FilterValue() string {
	return s.Name + "/" + s.User.Name
}

// ResultPage represents the result display after subscription change
type ResultPage struct {
	changed bool
	timer   timer.Model
}

// NewResultPage creates a new result page instance
func NewResultPage(changed bool) *ResultPage {
	return &ResultPage{
		changed: changed,
		timer:   timer.New(1 * time.Second),
	}
}

// Init initializes the result page
func (rp *ResultPage) Init() tea.Cmd {
	return rp.timer.Init()
}

// Update handles messages for the result page
func (rp *ResultPage) Update(msg tea.Msg) (*ResultPage, tea.Cmd) {
	var cmd tea.Cmd
	rp.timer, cmd = rp.timer.Update(msg)
	return rp, cmd
}

// View renders the result page
func (rp *ResultPage) View() string {
	var text string
	var color lipgloss.Color

	if rp.changed {
		text = SuccessMessage
		color = Success
	} else {
		text = NoChangeMessage
		color = Info
	}

	return lipgloss.NewStyle().
		Height(height).
		Width(width).
		Foreground(color).
		AlignVertical(lipgloss.Center).
		AlignHorizontal(lipgloss.Center).
		Render(text)
}

// Message types for tea application

// SubscriptionsLoadedMsg is sent when subscriptions have been loaded
type SubscriptionsLoadedMsg struct {
	Subscriptions []Subscription
	Error         error
}

// SubscriptionChangedMsg is sent when a subscription change has been attempted
type SubscriptionChangedMsg struct {
	Changed bool
	Error   error
	Subscription Subscription
}

// NewApp creates a new application instance
func NewApp() *App {
	app := &App{
		state: StateLoading,
	}

	app.initializeSpinner()
	app.initializeList()

	return app
}

// initializeSpinner sets up the loading spinner
func (app *App) initializeSpinner() {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(Text)
	app.spinner = s
}

// initializeList sets up the subscription list
func (app *App) initializeList() {
	delegate := app.createListDelegate()
	subscriptionList := list.New([]list.Item{}, delegate, 0, 0)

	app.styleList(&subscriptionList)
	subscriptionList.Title = AppTitle

	app.list = subscriptionList
}

// createListDelegate creates a styled list delegate
func (app *App) createListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(Rosewater).
		BorderLeftForeground(ActiveBorder)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(Text)
	d.Styles.NormalDesc = d.Styles.NormalTitle
	return d
}

// styleList applies styling to the subscription list
func (app *App) styleList(l *list.Model) {
	l.Styles.Title = l.Styles.Title.
		Foreground(Text).
		Background(Mantle).
		BorderBottomBackground(ActiveBorder).
		Margin(0).
		Padding(1, 0, 0, 0)

	l.Styles.FilterCursor = l.Styles.FilterCursor.Foreground(Text)
	l.Styles.FilterPrompt = l.Styles.FilterPrompt.Foreground(Rosewater)

	// Apply workaround styles for filter input
	l.FilterInput.PromptStyle = l.Styles.FilterPrompt.
		Foreground(Text).
		MarginLeft(1)
	l.FilterInput.Cursor.Style = l.Styles.FilterCursor.Foreground(Rosewater)
}

// Init implements tea.Model interface
func (app *App) Init() tea.Cmd {
	return tea.Batch(
		app.spinner.Tick,
		app.loadSubscriptions,
	)
}

// Update implements tea.Model interface
func (app *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return app.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return app.handleWindowSizeMsg(msg)
	case spinner.TickMsg:
		return app.handleSpinnerMsg(msg)
	case timer.TimeoutMsg:
		return app, tea.Quit
	case SubscriptionsLoadedMsg:
		return app.handleSubscriptionsLoaded(msg)
	case SubscriptionChangedMsg:
		return app.handleSubscriptionChanged(msg)
	}

	return app.updateSubComponents(msg)
}

// handleKeyMsg processes keyboard input
func (app *App) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == KeyQuit || key == KeyCtrlC {
		return app, tea.Quit
	}

	if key == KeyEnter && app.state == StateSelectingSubscription {
		if selectedSub, ok := app.list.SelectedItem().(Subscription); ok {
			return app, app.changeSubscription(selectedSub)
		}
	}

	return app, nil
}

// handleWindowSizeMsg processes window resize events
func (app *App) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	width, height = msg.Width, msg.Height

	// Update list size accounting for document style margins
	h, v := docStyle.GetFrameSize()
	app.list.SetSize(msg.Width-h, msg.Height-v)

	return app, nil
}

// handleSpinnerMsg processes spinner tick messages
func (app *App) handleSpinnerMsg(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if app.state != StateLoading {
		return app, nil
	}

	var cmd tea.Cmd
	app.spinner, cmd = app.spinner.Update(msg)
	return app, cmd
}

// handleSubscriptionsLoaded processes loaded subscriptions
func (app *App) handleSubscriptionsLoaded(msg SubscriptionsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		app.err = msg.Error
		app.state = StateError
		return app, nil
	}

	app.state = StateSelectingSubscription

	// Convert subscriptions to list items
	items := make([]list.Item, len(msg.Subscriptions))
	for i, sub := range msg.Subscriptions {
		items[i] = sub
	}

	// Find and select the default subscription
	defaultIndex := findDefaultSubscription(msg.Subscriptions)
	if defaultIndex >= 0 {
		app.selectedID = msg.Subscriptions[defaultIndex].ID
		app.list.Select(defaultIndex)
	}

	cmd := app.list.SetItems(items)
	return app, cmd
}

// handleSubscriptionChanged processes subscription change results
func (app *App) handleSubscriptionChanged(msg SubscriptionChangedMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		app.err = msg.Error
		app.state = StateError
		return app, nil
	}
	app.selectedID = msg.Subscription.ID
	app.resultPage = NewResultPage(msg.Changed)
	app.state = StateShowingResult

	return app, app.resultPage.Init()
}

// updateSubComponents updates child components
func (app *App) updateSubComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update list
	if app.state == StateSelectingSubscription {
		var cmd tea.Cmd
		app.list, cmd = app.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update result page
	if app.resultPage != nil {
		var cmd tea.Cmd
		app.resultPage, cmd = app.resultPage.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return app, tea.Batch(cmds...)
}

// findDefaultSubscription returns the index of the default subscription
func findDefaultSubscription(subscriptions []Subscription) int {
	return slices.IndexFunc(subscriptions, func(s Subscription) bool {
		return s.IsDefault
	})
}

// View implements tea.Model interface
func (app *App) View() string {
	switch app.state {
	case StateLoading:
		return app.loadingView()
	case StateSelectingSubscription:
		return app.subscriptionListView()
	case StateShowingResult:
		return app.resultView()
	case StateError:
		return app.errorView()
	default:
		return "Unknown state"
	}
}

// loadingView renders the loading screen
func (app *App) loadingView() string {
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		app.spinner.View(),
		LoadingMessage,
	)

	return app.centeredView(content, Text)
}

// subscriptionListView renders the subscription selection screen
func (app *App) subscriptionListView() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, "  ", app.list.View())
}

// resultView renders the result screen
func (app *App) resultView() string {
	if app.resultPage != nil {
		return app.resultPage.View()
	}
	return ""
}

// errorView renders the error screen
func (app *App) errorView() string {
	content := fmt.Sprintf(ErrorPrefix, app.err)
	return app.centeredView(content, Error)
}

// centeredView creates a centered view with the given content and color
func (app *App) centeredView(content string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Height(height).
		Width(width).
		Foreground(color).
		AlignVertical(lipgloss.Center).
		AlignHorizontal(lipgloss.Center).
		Render(content)
}

// Azure service functions

// isAzureCLIAvailable checks if the Azure CLI is available
func isAzureCLIAvailable() bool {
	_, err := exec.LookPath(AzureCommand)
	return err == nil
}

// loadSubscriptions loads subscriptions asynchronously
func (app *App) loadSubscriptions() tea.Msg {
	if !isAzureCLIAvailable() {
		return SubscriptionsLoadedMsg{
			Subscriptions: nil,
			Error:         ErrAzureCLINotFound,
		}
	}

	data, err := fetchSubscriptionData()
	if err != nil {
		return SubscriptionsLoadedMsg{
			Subscriptions: nil,
			Error:         fmt.Errorf("failed to fetch subscription data: %w", err),
		}
	}

	subscriptions, err := parseSubscriptions(data)
	if err != nil {
		return SubscriptionsLoadedMsg{
			Subscriptions: nil,
			Error:         fmt.Errorf("failed to parse subscription data: %w", err),
		}
	}

	return SubscriptionsLoadedMsg{
		Subscriptions: subscriptions,
		Error:         nil,
	}
}

// fetchSubscriptionData retrieves raw subscription data
func fetchSubscriptionData() ([]byte, error) {
	if os.Getenv(EnvUseSampleData) == "true" {
		return sampleData, nil
	}

	cmd := exec.Command(AzureCommand, "account", "list")
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("azure CLI command failed: %w", err)
	}

	return data, nil
}

// parseSubscriptions parses JSON data into Subscription structs
func parseSubscriptions(data []byte) ([]Subscription, error) {
	var subscriptions []Subscription
	if err := json.Unmarshal(data, &subscriptions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return subscriptions, nil
}

// changeSubscription changes the active subscription
func (app *App) changeSubscription(subscription Subscription) tea.Cmd {
	return func() tea.Msg {
		// If it's already the selected subscription, no change needed
		if subscription.ID == app.selectedID {
			return SubscriptionChangedMsg{Changed: false, Error: nil}
		}

		if !isAzureCLIAvailable() {
			return SubscriptionChangedMsg{
				Changed: false,
				Error:   ErrAzureCLINotFound,
			}
		}

		cmd := exec.Command(AzureCommand, "account", "set", "--subscription", subscription.ID)
		if err := cmd.Run(); err != nil {
			return SubscriptionChangedMsg{
				Changed: false,
				Error:   fmt.Errorf("failed to change subscription: %w", err),
			}
		}

		return SubscriptionChangedMsg{Changed: true, Error: nil, Subscription: subscription}
	}
}

// main is the entry point of the application
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitCodeError)
	}
}

// run executes the main application logic
func run() error {
	app := NewApp()

	program := tea.NewProgram(
		app,
		tea.WithAltScreen(),
	)

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI application: %w", err)
	}

	return nil
}
