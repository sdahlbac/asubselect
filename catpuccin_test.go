package main

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestColorConstants(t *testing.T) {
	// Test that all color constants are defined and not empty
	colorTests := map[string]lipgloss.Color{
		"Rosewater":    Rosewater,
		"Flamingo":     Flamingo,
		"Pink":         Pink,
		"Mauve":        Mauve,
		"Red":          Red,
		"Maroon":       Maroon,
		"Peach":        Peach,
		"Yellow":       Yellow,
		"Green":        Green,
		"Teal":         Teal,
		"Sky":          Sky,
		"Sapphire":     Sapphire,
		"Blue":         Blue,
		"Lavender":     Lavender,
		"Text":         Text,
		"Subtext1":     Subtext1,
		"Subtext0":     Subtext0,
		"Overlay2":     Overlay2,
		"Overlay1":     Overlay1,
		"Overlay0":     Overlay0,
		"Surface2":     Surface2,
		"Surface1":     Surface1,
		"Surface0":     Surface0,
		"Base":         Base,
		"Mantle":       Mantle,
		"Crust":        Crust,
		"ActiveBorder": ActiveBorder,
		"InactiveBorder": InactiveBorder,
		"Success":      Success,
		"Error":        Error,
		"Info":         Info,
	}

	for name, color := range colorTests {
		if string(color) == "" {
			t.Errorf("Color %s should not be empty", name)
		}

		// Test that colors are valid hex codes (start with #)
		if len(string(color)) > 0 && string(color)[0] != '#' {
			t.Errorf("Color %s should be a hex color code starting with #, got: %s", name, string(color))
		}

		// Test that hex colors have correct length (should be 7 characters: # + 6 hex digits)
		if len(string(color)) != 7 {
			t.Errorf("Color %s should be 7 characters long (# + 6 hex digits), got: %s (%d chars)",
				name, string(color), len(string(color)))
		}
	}
}

func TestSpecialColorAssignments(t *testing.T) {
	// Test that special color assignments are correct
	if ActiveBorder != Lavender {
		t.Errorf("Expected ActiveBorder to be Lavender, got %s", string(ActiveBorder))
	}

	if InactiveBorder != Overlay0 {
		t.Errorf("Expected InactiveBorder to be Overlay0, got %s", string(InactiveBorder))
	}

	if Success != Green {
		t.Errorf("Expected Success to be Green, got %s", string(Success))
	}

	if Error != Red {
		t.Errorf("Expected Error to be Red, got %s", string(Error))
	}

	if Info != Teal {
		t.Errorf("Expected Info to be Teal, got %s", string(Info))
	}
}

func TestColorHexValues(t *testing.T) {
	// Test specific hex values to ensure they match Catppuccin Mocha theme
	expectedColors := map[lipgloss.Color]string{
		Rosewater: "#f5e0dc",
		Text:      "#cdd6f4",
		Green:     "#a6e3a1",
		Red:       "#f38ba8",
		Teal:      "#94e2d5",
		Lavender:  "#b4befe",
		Base:      "#1e1e2e",
		Mantle:    "#181825",
		Crust:     "#11111b",
	}

	for color, expected := range expectedColors {
		if string(color) != expected {
			t.Errorf("Expected color to be %s, got %s", expected, string(color))
		}
	}
}

func TestColorConstants_NotNil(t *testing.T) {
	// Ensure that creating styles with these colors doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Creating style with colors panicked: %v", r)
		}
	}()

	testStyle := lipgloss.NewStyle().
		Foreground(Text).
		Background(Base).
		BorderForeground(ActiveBorder)

	// Basic test that the style was created without panicking
	// We can't compare styles directly, so just check it's not nil-like
	testString := testStyle.Render("test")
	if testString == "" {
		t.Error("Expected style to render non-empty string")
	}
}

// Benchmark color constant access (should be very fast)
func BenchmarkColorAccess(b *testing.B) {
	var color lipgloss.Color
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		color = Text
		color = Success
		color = Error
		color = ActiveBorder
	}
	_ = color // Prevent optimization
}

func TestDocStyle(t *testing.T) {
	// Test that docStyle is properly initialized and can be used
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Using docStyle panicked: %v", r)
		}
	}()

	// Test that it can render content
	testOutput := docStyle.Render("test content")
	if testOutput == "" {
		t.Error("Expected docStyle to render non-empty content")
	}
}

func TestGlobalVariables(t *testing.T) {
	// Test that global variables can be set and read
	originalWidth := width
	originalHeight := height
	originalSelectedId := selectedId

	// Set test values
	width = 100
	height = 50
	selectedId = "test-id"

	// Verify they were set
	if width != 100 {
		t.Errorf("Expected width to be 100, got %d", width)
	}
	if height != 50 {
		t.Errorf("Expected height to be 50, got %d", height)
	}
	if selectedId != "test-id" {
		t.Errorf("Expected selectedId to be 'test-id', got '%s'", selectedId)
	}

	// Restore original values
	width = originalWidth
	height = originalHeight
	selectedId = originalSelectedId
}
