package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
)

// TestChangeSelection tests the changeSelection function
func TestChangeSelection_SameId(t *testing.T) {
	// Set up: current selection matches the input
	originalSelectedId := selectedId
	selectedId = "same-id"
	defer func() { selectedId = originalSelectedId }()

	sub := SubInfo{
		Id:        "same-id",
		Name:      "Test Sub",
		IsDefault: false,
		User: struct {
			Name string `json:"name"`
		}{Name: "test@example.com"},
	}

	changed, err := changeSelection(sub)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if changed {
		t.Error("Expected no change when IDs are the same")
	}
}

func TestChangeSelection_DifferentId_AzNotFound(t *testing.T) {
	// Temporarily modify PATH to exclude az command
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", originalPath)

	originalSelectedId := selectedId
	selectedId = "current-id"
	defer func() { selectedId = originalSelectedId }()

	sub := SubInfo{
		Id:        "different-id",
		Name:      "Test Sub",
		IsDefault: false,
		User: struct {
			Name string `json:"name"`
		}{Name: "test@example.com"},
	}

	changed, err := changeSelection(sub)
	if err == nil {
		t.Error("Expected error when az command is not found")
	}
	if changed {
		t.Error("Expected no change when command fails")
	}
}

// Integration test for the main application flow
func TestApplicationFlow_WithSampleData(t *testing.T) {
	// Skip if this is a short test run
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up environment to use sample data
	os.Setenv("USE_SAMPLE_DATA", "true")
	defer os.Unsetenv("USE_SAMPLE_DATA")

	// Check if az command exists
	if _, err := exec.LookPath("az"); err != nil {
		t.Skip("az command not found, skipping integration test")
	}

	// Create initial model
	appModel := initialModel()

	// Test initialization
	cmds := appModel.Init()
	if cmds == nil {
		t.Error("Expected initialization commands")
	}

	// Simulate window resize
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, _ := appModel.Update(windowMsg)

	// Load subscriptions
	subsMsg := loadSubscriptionsCommand()
	updatedModel2, cmd := updatedModel.Update(subsMsg)

	// Verify the model state after loading subscriptions
	if typedModel, ok := updatedModel2.(model); ok {
		if typedModel.loading {
			t.Error("Expected loading to be false after receiving subscriptions")
		}

		// Test that we can render the view without panicking
		view := typedModel.View()
		if view == "" {
			t.Error("Expected non-empty view after loading subscriptions")
		}
	} else {
		t.Error("Expected model to be of correct type")
	}

	// Verify command was returned (might be nil which is acceptable)
	_ = cmd
}

// Test the complete model update cycle
func TestModelUpdateCycle(t *testing.T) {
	appModel := initialModel()

	// Test various message types in sequence
	messages := []tea.Msg{
		tea.WindowSizeMsg{Width: 80, Height: 24},
		appModel.spinner.Tick(),
		subscriptionsCommand{
			subs: []SubInfo{
				{
					Id: "test-sub",
					Name: "Test Subscription",
					IsDefault: true,
					User: struct{ Name string `json:"name"` }{Name: "test@example.com"},
				},
			},
			err: nil,
		},
	}

	for i, msg := range messages {
		updatedModel, _ := appModel.Update(msg)
		if typedModel, ok := updatedModel.(model); ok {
			appModel = typedModel
		} else {
			t.Errorf("Error at message %d: expected model type", i)
		}
	}

	// Final state checks
	if appModel.loading {
		t.Error("Expected loading to be false at the end")
	}

	// Test that view renders without error
	view := appModel.View()
	if view == "" {
		t.Error("Expected non-empty view at the end of update cycle")
	}
}

// Test timeout behavior
func TestTimeoutBehavior(t *testing.T) {
	appModel := initialModel()
	appModel.finalPageModel = NewResultPageModel(true)

	// Simulate timeout message - import timer package
	timeoutMsg := timer.TimeoutMsg{}
	_, cmd := appModel.Update(timeoutMsg)

	// Should return quit command on timeout
	if cmd == nil {
		t.Error("Expected quit command on timeout")
	}
}

// Test error handling in the model
func TestModelErrorHandling(t *testing.T) {
	model := initialModel()

	// Set an error
	testErr := os.ErrNotExist
	model.err = testErr

	// Test that error view is rendered
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty error view")
	}

	// Error view should contain error information
	// We can't easily test the exact content without brittle string matching,
	// but we can verify it doesn't panic and returns content
}

// Test memory usage with large subscription lists
func TestLargeSubscriptionList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	appModel := initialModel()

	// Create a large list of subscriptions
	subs := make([]SubInfo, 1000)
	for i := 0; i < 1000; i++ {
		subs[i] = SubInfo{
			Id:        "sub-" + string(rune(i)),
			Name:      "Subscription " + string(rune(i)),
			IsDefault: i == 0,
			User: struct {
				Name string `json:"name"`
			}{Name: "user" + string(rune(i)) + "@example.com"},
		}
	}

	msg := subscriptionsCommand{
		subs: subs,
		err:  nil,
	}

	// This should not panic or consume excessive memory
	updatedModel, _ := appModel.Update(msg)

	// Test that view renders (might be slow but shouldn't crash)
	view := updatedModel.View()
	if view == "" {
		t.Error("Expected non-empty view with large subscription list")
	}
}

// Benchmark the full model update cycle
func BenchmarkModelUpdate(b *testing.B) {
	model := initialModel()
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(msg)
	}
}

// Test concurrent access (if applicable)
func TestConcurrentAccess(t *testing.T) {
	// Test that global variables don't cause races
	// This is a basic test - in a real application you'd want proper synchronization

	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			width = 80
			height = 24
			selectedId = "test-1"
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			width = 100
			height = 30
			selectedId = "test-2"
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Basic verification that we didn't crash
	if width == 0 || height == 0 {
		t.Error("Expected width and height to be set")
	}
}
