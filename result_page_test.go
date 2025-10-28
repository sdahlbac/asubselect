package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/timer"
)

func TestNewResultPageModel(t *testing.T) {
	// Test with changed = true
	model := NewResultPageModel(true)
	if !model.changed {
		t.Error("Expected changed to be true")
	}

	// Test with changed = false
	model = NewResultPageModel(false)
	if model.changed {
		t.Error("Expected changed to be false")
	}

	// Test that timer is initialized
	if model.timer == (timer.Model{}) {
		t.Error("Expected timer to be initialized")
	}
}

func TestResultPageModel_Init(t *testing.T) {
	model := NewResultPageModel(true)
	cmd := model.Init()

	// Verify that a command is returned (timer initialization)
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

func TestResultPageModel_Update(t *testing.T) {
	model := NewResultPageModel(true)

	// Test with a generic message
	updatedModel, cmd := model.Update(tea.KeyMsg{})

	// Verify the model is returned correctly
	if resultModel, ok := updatedModel.(resultPageModel); ok {
		if resultModel.changed != model.changed {
			t.Error("Expected changed property to remain the same")
		}
	} else {
		t.Error("Expected returned model to be of type resultPageModel")
	}

	// A command might be returned from timer update
	_ = cmd // We can't easily test the specific timer command
}

func TestResultPageModel_View_Changed(t *testing.T) {
	// Set global dimensions for testing
	width = 80
	height = 20

	model := NewResultPageModel(true)
	view := model.View()

	// Check that view is not empty
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Check that the success message is included
	if !strings.Contains(view, "switched") {
		t.Error("Expected view to contain 'switched' when changed is true")
	}
}

func TestResultPageModel_View_NotChanged(t *testing.T) {
	// Set global dimensions for testing
	width = 80
	height = 20

	model := NewResultPageModel(false)
	view := model.View()

	// Check that view is not empty
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Check that the info message is included
	if !strings.Contains(view, "No change") {
		t.Error("Expected view to contain 'No change' when changed is false")
	}
}

func TestResultPageModel_View_Dimensions(t *testing.T) {
	// Test with different dimensions
	testCases := []struct {
		width, height int
	}{
		{80, 20},
		{120, 30},
		{40, 10},
	}

	for _, tc := range testCases {
		width = tc.width
		height = tc.height

		model := NewResultPageModel(true)
		view := model.View()

		if view == "" {
			t.Errorf("Expected non-empty view for dimensions %dx%d", tc.width, tc.height)
		}

		// The view should handle different dimensions gracefully
		// We can't easily test exact dimensions without lipgloss internals,
		// but we can verify it doesn't panic and returns content
	}
}

// Benchmark the View method
func BenchmarkResultPageModel_View(b *testing.B) {
	width = 80
	height = 20
	model := NewResultPageModel(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

func TestResultPageModel_Update_TimerMessage(t *testing.T) {
	model := NewResultPageModel(true)

	// Initialize the model to get the timer started
	_ = model.Init()

	// Create a timer timeout message
	timerMsg := timer.TimeoutMsg{}

	updatedModel, cmd := model.Update(timerMsg)

	// Verify the model type is preserved
	if _, ok := updatedModel.(resultPageModel); !ok {
		t.Error("Expected returned model to be of type resultPageModel")
	}

	// Timer update might return a command
	_ = cmd
}

func TestResultPageModel_StatePreservation(t *testing.T) {
	// Test that the changed state is preserved through updates
	testCases := []bool{true, false}

	for _, changed := range testCases {
		model := NewResultPageModel(changed)

		// Perform several updates
		for i := 0; i < 5; i++ {
			updatedModel, _ := model.Update(tea.KeyMsg{})
			if resultModel, ok := updatedModel.(resultPageModel); ok {
				if resultModel.changed != changed {
					t.Errorf("Expected changed=%v to be preserved after update %d", changed, i)
				}
				model = resultModel
			} else {
				t.Errorf("Expected model type to be preserved after update %d", i)
				break
			}
		}
	}
}
