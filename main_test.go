package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestItem(t *testing.T) {
	i := item{
		path:     "/tmp/test.md",
		noteType: "md",
	}

	expectedTitle := "test.md"
	if i.Title() != expectedTitle {
		t.Errorf("expected title %s, got %s", expectedTitle, i.Title())
	}

	expectedDescription := "/tmp/test.md"
	if i.Description() != expectedDescription {
		t.Errorf("expected description %s, got %s", expectedDescription, i.Description())
	}

	expectedFilterValue := "test.md"
	if i.FilterValue() != expectedFilterValue {
		t.Errorf("expected filter value %s, got %s", expectedFilterValue, i.FilterValue())
	}
}

func TestConfigIO(t *testing.T) {
	tmpDir := t.TempDir()
	configPathOverride = filepath.Join(tmpDir, "config.toml")
	defer func() { configPathOverride = "" }()

	expectedConfig := Config{NotesPath: "/path/to/notes"}

	// Test saveConfig
	cmd := saveConfig(expectedConfig)
	msg := cmd()
	if _, ok := msg.(configSavedMsg); !ok {
		t.Errorf("expected configSavedMsg, got %T", msg)
	}

	// Test loadConfig
	msg = loadConfig()
	loaded, ok := msg.(configLoadedMsg)
	if !ok {
		t.Fatalf("expected configLoadedMsg, got %T", msg)
	}

	if loaded.config.NotesPath != expectedConfig.NotesPath {
		t.Errorf("expected notes path %s, got %s", expectedConfig.NotesPath, loaded.config.NotesPath)
	}
}

func TestFindNotes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some mock notes
	notes := []string{"note1.md", "note2.pdf", "other.txt"}
	for _, n := range notes {
		if err := os.WriteFile(filepath.Join(tmpDir, n), []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create mock note: %v", err)
		}
	}

	cmd := findNotes(tmpDir)
	msg := cmd()

	found, ok := msg.(notesFoundMsg)
	if !ok {
		t.Fatalf("expected notesFoundMsg, got %T", msg)
	}

	if len(found.notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(found.notes))
	}

	var mdCount, pdfCount int
	for _, n := range found.notes {
		if filepath.Ext(n.path) == ".md" {
			mdCount++
		} else if filepath.Ext(n.path) == ".pdf" {
			pdfCount++
		}
	}

	if mdCount != 1 || pdfCount != 1 {
		t.Errorf("expected 1 md and 1 pdf, got %d md and %d pdf", mdCount, pdfCount)
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()
	if m.state != stateInitial {
		t.Errorf("expected state stateInitial, got %v", m.state)
	}
	if m.textInput.Placeholder != "/path/to/your/notes" {
		t.Errorf("expected placeholder /path/to/your/notes, got %s", m.textInput.Placeholder)
	}
}

func TestModelUpdate(t *testing.T) {
	m := initialModel()

	// Test configLoadedMsg with empty path
	m2, cmd := m.Update(configLoadedMsg{config: Config{NotesPath: ""}})
	m = m2.(model)
	if m.state != statePromptForPath {
		t.Errorf("expected state statePromptForPath, got %v", m.state)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd, got %v", cmd)
	}

	// Test configLoadedMsg with path
	m2, _ = m.Update(configLoadedMsg{config: Config{NotesPath: "/some/path"}})
	m = m2.(model)
	if m.state != stateShowList {
		t.Errorf("expected state stateShowList, got %v", m.state)
	}
	if m.config.NotesPath != "/some/path" {
		t.Errorf("expected path /some/path, got %s", m.config.NotesPath)
	}

	// Test notesFoundMsg
	notes := []item{{path: "test.md", noteType: "md"}}
	m2, _ = m.Update(notesFoundMsg{notes: notes})
	m = m2.(model)
	if len(m.list.Items()) != 1 {
		t.Errorf("expected 1 item, got %d", len(m.list.Items()))
	}

	// Test WindowSizeMsg
	m2, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = m2.(model)
	if m.list.Width() != 100-4 { // appStyle margin is 2 (left+right=4)
		t.Errorf("expected list width 96, got %d", m.list.Width())
	}
}

func TestModelView(t *testing.T) {
	m := initialModel()

	// Initial/Loading state
	view := m.View()
	if !strings.Contains(view, "Initializing...") {
		t.Errorf("expected view to contain 'Initializing...', got %s", view)
	}

	// Prompt state
	m.state = statePromptForPath
	view = m.View()
	if !strings.Contains(view, "Welcome to gopuntes!") {
		t.Errorf("expected view to contain 'Welcome to gopuntes!', got %s", view)
	}

	// ShowList state
	m.state = stateShowList
	// The list view might not contain "Your Notes" title if it's empty or style isn't rendered exactly as string
	// But let's check for "quit" which is in the help
	view = m.View()
	if !strings.Contains(view, "quit") {
		t.Errorf("expected view to contain 'quit', got %s", view)
	}

	// Error state
	m.err = fmt.Errorf("test error")
	view = m.View()
	if !strings.Contains(view, "Error: test error") {
		t.Errorf("expected view to contain 'Error: test error', got %s", view)
	}
}
