package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSubInfo_Methods(t *testing.T) {
	sub := SubInfo{
		Id:        "test-id",
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

func TestInitialModel(t *testing.T) {
	model := initialModel()

	// Test that model is properly initialized
	if !model.loading {
		t.Error("Expected model to be in loading state initially")
	}

	if model.list.Title != "Select Azure Subscription" {
		t.Errorf("Expected list title 'Select Azure Subscription', got '%s'", model.list.Title)
	}

	// Test that spinner is initialized
	if model.spinner.Spinner.Frames == nil {
		t.Error("Expected spinner to be initialized")
	}
}

func TestConvertToSubInfo(t *testing.T) {
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

	subs, err := convertToSubInfo([]byte(testData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(subs) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subs))
	}

	// Test first subscription
	if subs[0].Id != "test-id-1" {
		t.Errorf("Expected first sub ID 'test-id-1', got '%s'", subs[0].Id)
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

func TestConvertToSubInfo_InvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid": json}`

	_, err := convertToSubInfo([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestFetchSubscriptionData_WithSampleData(t *testing.T) {
	// Set environment variable to use sample data
	os.Setenv("USE_SAMPLE_DATA", "true")
	defer os.Unsetenv("USE_SAMPLE_DATA")

	data, err := fetchSubscriptionData()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify it's valid JSON
	var subs []SubInfo
	err = json.Unmarshal(data, &subs)
	if err != nil {
		t.Fatalf("Expected valid JSON from sample data, got error: %v", err)
	}

	if len(subs) == 0 {
		t.Error("Expected sample data to contain subscriptions")
	}
}

func TestFetchSubscriptionData_AzNotFound(t *testing.T) {
	// Temporarily modify PATH to exclude az command
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", originalPath)

	_, err := fetchSubscriptionData()
	if err == nil {
		t.Error("Expected error when az command is not found")
	}

	if err.Error() != "az not found in PATH" {
		t.Errorf("Expected 'az not found in PATH' error, got: %v", err)
	}
}

func TestLoadSubscriptionsCommand_AzNotFound(t *testing.T) {
	// Temporarily modify PATH to exclude az command
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", originalPath)

	msg := loadSubscriptionsCommand()

	if subsCmd, ok := msg.(subscriptionsCommand); ok {
		if subsCmd.err == nil {
			t.Error("Expected error when az command is not found")
		}
		if subsCmd.subs != nil {
			t.Error("Expected subs to be nil when error occurs")
		}
	} else {
		t.Error("Expected subscriptionsCommand type")
	}
}

func TestLoadSubscriptionsCommand_WithSampleData(t *testing.T) {
	// Set environment variable to use sample data
	os.Setenv("USE_SAMPLE_DATA", "true")
	defer os.Unsetenv("USE_SAMPLE_DATA")

	// Check if az command exists (needed for the initial check)
	if _, err := exec.LookPath("az"); err != nil {
		t.Skip("az command not found, skipping test")
	}

	msg := loadSubscriptionsCommand()

	if subsCmd, ok := msg.(subscriptionsCommand); ok {
		if subsCmd.err != nil {
			t.Errorf("Expected no error with sample data, got: %v", subsCmd.err)
		}
		if len(subsCmd.subs) == 0 {
			t.Error("Expected subscriptions from sample data")
		}
	} else {
		t.Error("Expected subscriptionsCommand type")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	model := initialModel()

	msg := tea.WindowSizeMsg{
		Width:  800,
		Height: 600,
	}

	updatedModel, _ := model.Update(msg)

	// Check that global width and height were updated
	if width != 800 || height != 600 {
		t.Errorf("Expected width=800, height=600, got width=%d, height=%d", width, height)
	}

	// Verify the model is returned (basic check)
	_ = updatedModel // We can't easily check the exact type without reflection
}

func TestModel_Update_KeyMsg_Quit(t *testing.T) {
	model := initialModel()

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'q'},
	}

	updatedModel, cmd := model.Update(msg)

	// Verify model type is preserved
	_ = updatedModel // We can't easily check the exact type without reflection

	// We can't easily test tea.Quit directly, but we can verify a command might be returned
	_ = cmd
}

func TestModel_Update_SpinnerTick_NotLoading(t *testing.T) {
	model := initialModel()
	model.loading = false // Set to not loading

	// Create a spinner tick message
	msg := model.spinner.Tick()

	updatedModel, cmd := model.Update(msg)

	// When not loading, spinner tick should return nil command
	if cmd != nil {
		t.Error("Expected nil command when not loading")
	}

	// Basic check that update returned something
	_ = updatedModel
}

func TestModel_Update_SubscriptionsCommand(t *testing.T) {
	model := initialModel()

	testSubs := []SubInfo{
		{
			Id:        "sub-1",
			Name:      "Test Sub 1",
			IsDefault: true,
			User: struct {
				Name string `json:"name"`
			}{Name: "test@example.com"},
		},
		{
			Id:        "sub-2",
			Name:      "Test Sub 2",
			IsDefault: false,
			User: struct {
				Name string `json:"name"`
			}{Name: "test2@example.com"},
		},
	}

	msg := subscriptionsCommand{
		subs: testSubs,
		err:  nil,
	}

	updatedModel, cmd := model.Update(msg)

	// Check that selected ID was set from default subscription
	if selectedId != "sub-1" {
		t.Errorf("Expected selectedId to be 'sub-1', got '%s'", selectedId)
	}

	// Basic check that update completed
	_ = updatedModel

	// Command might be nil which is acceptable
	_ = cmd
}

func TestModel_Update_SubscriptionsCommand_WithError(t *testing.T) {
	model := initialModel()

	msg := subscriptionsCommand{
		subs: nil,
		err:  os.ErrNotExist,
	}

	updatedModel, _ := model.Update(msg)

	// Basic check that update completed
	_ = updatedModel
}

func TestModel_View_States(t *testing.T) {
	// Test loading view
	model := initialModel()
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view for loading state")
	}

	// Test error view
	model.err = os.ErrNotExist
	view = model.View()
	if view == "" {
		t.Error("Expected non-empty view for error state")
	}

	// Test final page view
	model.err = nil
	model.finalPageModel = NewResultPageModel(true)
	view = model.View()
	if view == "" {
		t.Error("Expected non-empty view for final page state")
	}
}

// Benchmark for convertToSubInfo with large JSON
func BenchmarkConvertToSubInfo(b *testing.B) {
	// Create test data with multiple subscriptions
	testData := make([]SubInfo, 100)
	for i := 0; i < 100; i++ {
		testData[i] = SubInfo{
			Id:        "test-id-" + string(rune(i)),
			Name:      "Test Subscription " + string(rune(i)),
			IsDefault: i == 0,
			User: struct {
				Name string `json:"name"`
			}{Name: "user" + string(rune(i)) + "@example.com"},
		}
	}

	jsonData, _ := json.Marshal(testData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := convertToSubInfo(jsonData)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
