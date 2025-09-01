package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// --- STYLES ---
var (
	appStyle    = lipgloss.NewStyle().Margin(1, 2)
	docStyle    = lipgloss.NewStyle()
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// --- STATE ---
type appState int

const (
	stateInitial appState = iota
	statePromptForPath
	stateShowList
)

// --- LIST ITEM ---
type item struct {
	path     string
	noteType string // "md" or "pdf"
}

func (i item) Title() string       { return filepath.Base(i.path) }
func (i item) Description() string { return i.path }
func (i item) FilterValue() string { return filepath.Base(i.path) }

// --- MODEL & CONFIG ---
type Config struct {
	NotesPath string `toml:"notes_path"`
}

type model struct {
	state        appState
	list         list.Model
	textInput    textinput.Model
	viewport     viewport.Model
	showViewport bool
	config       Config
	err          error
}

// --- MESSAGES ---
type (
	configLoadedMsg   struct{ config Config }
	notesFoundMsg     struct{ notes []item }
	configSavedMsg    struct{}
	fileContentMsg    string
	errorMsg          struct{ err error }
)

func (e errorMsg) Error() string { return e.err.Error() }

// --- BUBBLETEA ---
func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "/path/to/your/notes"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	// Initialize list
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Your Notes"
	l.SetShowHelp(true)

	// Initialize viewport
	vp := viewport.New(80, 24) // Default size, will be resized on WindowSizeMsg

	return model{
		state:     stateInitial,
		textInput: ti,
		list:      l,
		viewport:  vp,
	}
}

func (m model) Init() tea.Cmd {
	return loadConfig
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// -- Global Messages --
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.viewport.Width = msg.Width - h
		m.viewport.Height = msg.Height - v
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	// -- State-specific updates --
	case configLoadedMsg:
		if msg.config.NotesPath == "" {
			m.state = statePromptForPath
			return m, nil
		}
		m.config = msg.config
		m.state = stateShowList
		return m, findNotes(m.config.NotesPath)

	case notesFoundMsg:
		items := make([]list.Item, len(msg.notes))
		for i, note := range msg.notes {
			items[i] = note
		}
		m.list.SetItems(items)
		return m, nil

	case configSavedMsg:
		return m, loadConfig

	case errorMsg:
		m.err = msg.err
		return m, tea.Quit
	}

	// -- Delegate to sub-models based on state --
	switch m.state {
	case statePromptForPath:
		m, cmd = updatePromptView(msg, m)
		cmds = append(cmds, cmd)

	case stateShowList:
		m, cmd = updateListView(msg, m)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func updatePromptView(msg tea.Msg, m model) (model, tea.Cmd) {
	var cmd tea.Cmd

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		path := m.textInput.Value()
		return m, saveConfig(Config{NotesPath: path})
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func updateListView(msg tea.Msg, m model) (model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showViewport {
			switch msg.String() {
			case "q", "esc":
				m.showViewport = false
			default:
				m.viewport, cmd = m.viewport.Update(msg)
			}
			return m, cmd
		}

		if m.list.FilterState() == list.Filtering {
			break
		}

		if msg.Type == tea.KeyEnter {
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			if i.noteType == "md" {
				return m, readMarkdownContent(i.path)
			} else if i.noteType == "pdf" {
				return m, openPDF(i.path)
			}
		}

	case fileContentMsg:
		m.showViewport = true
		renderer, _ := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(m.viewport.Width))
		str, err := renderer.Render(string(msg))
		if err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.viewport.SetContent(str)
		m.viewport.GotoTop()
		return m, nil
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return appStyle.Render(fmt.Sprintf("Error: %s", m.err.Error()))
	}

	switch m.state {
	case statePromptForPath:
		title := titleStyle.Render("Welcome to gopuntes!")
		prompt := promptStyle.Render("Please enter the full path to your notes folder:")
		help := helpStyle.Render("(press enter to save)")
		return appStyle.Render(fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", title, prompt, m.textInput.View(), help))

	case stateShowList:
		if m.showViewport {
			return docStyle.Render(m.viewport.View())
		}
		return appStyle.Render(m.list.View())
	default:
		return appStyle.Render("Initializing...")
	}
}

// --- IO & HELPERS ---

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gopuntes", "config.toml"), nil
}

func loadConfig() tea.Msg {
	path, err := getConfigPath()
	if err != nil {
		return errorMsg{err}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return configLoadedMsg{Config{}} // Return empty config to trigger prompt
	}

	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return errorMsg{fmt.Errorf("failed to load config file: %w", err)}
	}

	return configLoadedMsg{config}
}

func saveConfig(config Config) tea.Cmd {
	return func() tea.Msg {
		path, err := getConfigPath()
		if err != nil {
			return errorMsg{err}
		}
		configDir := filepath.Dir(path)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return errorMsg{fmt.Errorf("failed to create config directory: %w", err)}
		}
		file, err := os.Create(path)
		if err != nil {
			return errorMsg{fmt.Errorf("failed to create config file: %w", err)}
		}
		defer file.Close()

		if err := toml.NewEncoder(file).Encode(config); err != nil {
			return errorMsg{fmt.Errorf("failed to save configuration: %w", err)}
		}
		return configSavedMsg{}
	}
}

func findNotes(root string) tea.Cmd {
	return func() tea.Msg {
		var items []item
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".md" {
					items = append(items, item{path: path, noteType: "md"})
				} else if ext == ".pdf" {
					items = append(items, item{path: path, noteType: "pdf"})
				}
			}
			return nil
		})
		if err != nil {
			return errorMsg{err}
		}
		return notesFoundMsg{notes: items}
	}
}

func readMarkdownContent(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		if err != nil {
			return errorMsg{err}
		}
		return fileContentMsg(content)
	}
}

func openPDF(path string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", path)
		case "linux":
			cmd = exec.Command("xdg-open", path)
		case "windows":
			cmd = exec.Command("cmd", "/C", "start", path)
		default:
			return errorMsg{fmt.Errorf("unsupported operating system: %s", runtime.GOOS)}
		}

		if err := cmd.Run(); err != nil {
			return errorMsg{fmt.Errorf("failed to open PDF file: %w", err)}
		}
		return nil
	}
}

// --- MAIN ---

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
