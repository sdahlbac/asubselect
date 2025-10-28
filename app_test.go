package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSubscription_Methods(t *testing.T) {
	sub := Subscription{
		ID:        "test-id",
		Name:      "Test Subscription",
		IsDefault: true,
		User: struct {
			Name string `json:"name"`
		}{
			Name: "test@example.com",
		},
	}

	// Test Title method
	if sub.Title() != "Test Subscription" {
		t.Errorf("Expected title 'Test Subscription', got '%s'", sub.Title())
	}

	// Test Description method
	expected := "test-id (test@example.com)"
	if sub.Description() != expected {
		t.Errorf("Expected description '%s', got '%s'", expected, sub.Description())
	}

	// Test FilterValue method
	expectedFilter := "Test Subscription/test@example.com"
	if sub.FilterValue() != expectedFilter {
		t.Errorf("Expected filter value '%s', got '%s'", expectedFilter, sub.FilterValue())
	}
}

func TestNewApp(t *testing.T) {
	app := NewApp()

	// Test that app is properly initialized
	if app.state != StateLoading {
		t.Error("Expected app to be in loading state initially")
	}

	if app.list.Title != AppTitle {
		t.Errorf("Expected list title '%s', got '%s'", AppTitle, app.list.Title)
	}

	// Test that spinner is initialized
	if app.spinner.Spinner.Frames == nil {
		t.Error("Expected spinner to be initialized")
	}
}

func TestParseSubscriptions(t *testing.T) {
	testData := `[
		{
			"id": "test-id-1",
			"name": "Test Sub 1",
			"isDefault": true,
			"user": {
				"name": "user1@example.com"
			}
		},
		{
			"id": "test-id-2",
			"name": "Test Sub 2",
			"isDefault": false,
			"user": {
				"name": "user2@example.com"
			}
		}
	]`

	subs, err := parseSubscriptions([]byte(testData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(subs) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subs))
	}

	// Test first subscription
	if subs[0].ID != "test-id-1" {
		t.Errorf("Expected first sub ID 'test-id-1', got '%s'", subs[0].ID)
	}

	if subs[0].Name != "Test Sub 1" {
		t.Errorf("Expected first sub name 'Test Sub 1', got '%s'", subs[0].Name)
	}

	if !subs[0].IsDefault {
		t.Error("Expected first sub to be default")
	}

	if subs[0].User.Name != "user1@example.com" {
		t.Errorf("Expected first sub user 'user1@example.com', got '%s'", subs[0].User.Name)
	}

	// Test second subscription
	if subs[1].IsDefault {
		t.Error("Expected second sub to not be default")
	}
}

func TestParseSubscriptions_InvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid": json}`

	_, err := parseSubscriptions([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestIsAzureCLIAvailable(t *testing.T) {
	// This test depends on the system state, but we can test the function exists
	available := isAzureCLIAvailable()
	// Result could be true or false depending on system, just verify no panic
	_ = available
}

func TestFindDefaultSubscription(t *testing.T) {
	subs := []Subscription{
		{ID: "sub-1", Name: "Sub 1", IsDefault: false},
		{ID: "sub-2", Name: "Sub 2", IsDefault: true},
		{ID: "sub-3", Name: "Sub 3", IsDefault: false},
	}

	index := findDefaultSubscription(subs)
	if index != 1 {
		t.Errorf("Expected index 1, got %d", index)
	}

	// Test with no default
	subsNoDefault := []Subscription{
		{ID: "sub-1", Name: "Sub 1", IsDefault: false},
		{ID: "sub-2", Name: "Sub 2", IsDefault: false},
	}

	index = findDefaultSubscription(subsNoDefault)
	if index != -1 {
		t.Errorf("Expected index -1, got %d", index)
	}
}

func TestApp_Update_WindowSize(t *testing.T) {
	app := NewApp()

	msg := tea.WindowSizeMsg{
		Width:  800,
		Height: 600,
	}

	updatedModel, _ := app.Update(msg)

	// Check that global width and height were updated
	if width != 800 || height != 600 {
		t.Errorf("Expected width=800, height=600, got width=%d, height=%d", width, height)
	}

	// Verify the model is returned
	if _, ok := updatedModel.(*App); !ok {
		t.Error("Expected returned model to be of type *App")
	}
}

func TestApp_Update_KeyMsg_Quit(t *testing.T) {
	app := NewApp()

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'q'},
	}

	_, cmd := app.Update(msg)

	// We can't easily test tea.Quit directly, but verify it doesn't panic
	_ = cmd
}

func TestApp_View_States(t *testing.T) {
	app := NewApp()

	// Test loading view
	app.state = StateLoading
	view := app.View()
	if view == "" {
		t.Error("Expected non-empty view for loading state")
	}

	// Test error view
	app.state = StateError
	app.err = ErrAzureCLINotFound
	view = app.View()
	if view == "" {
		t.Error("Expected non-empty view for error state")
	}

	// Test selecting subscription view
	app.state = StateSelectingSubscription
	view = app.View()
	if view == "" {
		t.Error("Expected non-empty view for selecting subscription state")
	}
}

func TestNewResultPage(t *testing.T) {
	// Test with changed = true
	rp := NewResultPage(true)
	if !rp.changed {
		t.Error("Expected changed to be true")
	}

	// Test with changed = false
	rp = NewResultPage(false)
	if rp.changed {
		t.Error("Expected changed to be false")
	}
}

func TestResultPage_Init(t *testing.T) {
	rp := NewResultPage(true)
	cmd := rp.Init()

	// Verify that a command is returned (timer initialization)
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

func TestResultPage_View(t *testing.T) {
	// Set global dimensions for testing
	width = 80
	height = 20

	// Test changed result
	rp := NewResultPage(true)
	view := rp.View()

	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Test no change result
	rp = NewResultPage(false)
	view = rp.View()

	if view == "" {
		t.Error("Expected non-empty view")
	}
}

// Benchmark for parseSubscriptions with large JSON
func BenchmarkParseSubscriptions(b *testing.B) {
	testData := `[
		{
			"id": "test-id-1",
			"name": "Test Sub 1",
			"isDefault": true,
			"user": {"name": "user1@example.com"}
		}
	]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parseSubscriptions([]byte(testData))
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// Error handling tests
func TestAppError_Classification(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name        string
		err         error
		expectedType ErrorType
		retryable   bool
	}{
		{
			name:         "Network error",
			err:          errors.New("network connection failed"),
			expectedType: ErrorTypeNetwork,
			retryable:    true,
		},
		{
			name:         "Authentication error",
			err:          errors.New("authentication failed"),
			expectedType: ErrorTypeAuth,
			retryable:    false,
		},
		{
			name:         "Permission error",
			err:          errors.New("permission denied"),
			expectedType: ErrorTypePermission,
			retryable:    false,
		},
		{
			name:         "CLI not found",
			err:          errors.New("az command not found"),
			expectedType: ErrorTypeConfig,
			retryable:    false,
		},
		{
			name:         "Unknown error",
			err:          errors.New("something went wrong"),
			expectedType: ErrorTypeUnknown,
			retryable:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := app.classifyError(tt.err)

			if appErr.Type != tt.expectedType {
				t.Errorf("Expected error type %v, got %v", tt.expectedType, appErr.Type)
			}

			if appErr.Retryable != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, appErr.Retryable)
			}

			if appErr.Err != tt.err {
				t.Errorf("Expected error %v, got %v", tt.err, appErr.Err)
			}

			if appErr.Suggestion == "" {
				t.Error("Expected non-empty suggestion")
			}
		})
	}
}

func TestApp_ShouldRetry(t *testing.T) {
	app := NewApp()

	// Test retryable error within limit
	networkErr := errors.New("network timeout")
	if !app.shouldRetry(networkErr) {
		t.Error("Should retry network error when under retry limit")
	}

	// Test retryable error at limit
	app.retryCount = MaxRetries
	if app.shouldRetry(networkErr) {
		t.Error("Should not retry when at retry limit")
	}

	// Test non-retryable error
	app.retryCount = 0
	authErr := errors.New("authentication failed")
	if app.shouldRetry(authErr) {
		t.Error("Should not retry authentication error")
	}
}

func TestApp_ErrorHandling_Integration(t *testing.T) {
	app := NewApp()

	// Test handling subscription load error with retry
	msg := SubscriptionsLoadedMsg{
		Subscriptions: nil,
		Error:         errors.New("network timeout"),
	}

	model, cmd := app.handleSubscriptionsLoaded(msg)
	app = model.(*App)

	// Should be in retrying state
	if app.state != StateRetrying {
		t.Errorf("Expected state %v, got %v", StateRetrying, app.state)
	}

	// Should have incremented retry count
	if app.retryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", app.retryCount)
	}

	// Should have set last operation
	if app.lastOperation != "load" {
		t.Errorf("Expected last operation 'load', got %s", app.lastOperation)
	}

	// Should return retry command
	if cmd == nil {
		t.Error("Expected retry command to be returned")
	}
}

func TestApp_RetryView(t *testing.T) {
	app := NewApp()
	app.state = StateRetrying
	app.retryCount = 2
	app.maxRetries = MaxRetries

	view := app.retryingView()

	if view == "" {
		t.Error("Retry view should not be empty")
	}

	// Should contain retry information
	expectedText := fmt.Sprintf("Retrying... (attempt %d/%d)", app.retryCount, app.maxRetries)
	if !strings.Contains(view, expectedText) {
		t.Errorf("Retry view should contain retry count information, got: %s", view)
	}
}
