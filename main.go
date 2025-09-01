package main

import (
	"bufio"
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
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// --- STYLES ---
var (
	appStyle = lipgloss.NewStyle().Margin(1, 2)
	docStyle = lipgloss.NewStyle()
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
	list         list.Model
	viewport     viewport.Model
	showViewport bool
	config       Config
	err          error
}

// --- MESSAGES ---
type fileContentMsg string
type errorMsg struct{ err error }

func (e errorMsg) Error() string { return e.err.Error() }

// --- BUBBLETEA ---
func initialModel(notes []item, config Config) model {
	items := make([]list.Item, len(notes))
	for i, note := range notes {
		items[i] = note
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Twoje notatki"
	l.SetShowHelp(true)

	m := model{
		list:   l,
		config: config,
	}

	if len(notes) == 0 {
		m.err = fmt.Errorf("nie znaleziono żadnych plików .md ani .pdf w '%s'", config.NotesPath)
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.viewport.Width = msg.Width - h
		m.viewport.Height = msg.Height - v
		return m, nil

	case tea.KeyMsg:
		// If in viewport, handle viewport keys and return immediately
		if m.showViewport {
			switch msg.String() {
			case "q", "esc":
				m.showViewport = false
				return m, nil
			default:
				var viewportCmd tea.Cmd
				m.viewport, viewportCmd = m.viewport.Update(msg)
				return m, viewportCmd
			}
		}

		// If in list view, handle list keys
		if m.list.FilterState() == list.Filtering {
			break // Let the list handle filtering keys
		}

		switch msg.String() {
		case "enter":
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
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithStylePath("dark"),
			glamour.WithWordWrap(m.viewport.Width),
		)
		str, err := renderer.Render(string(msg))
		if err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.viewport.SetContent(str)
		m.viewport.GotoTop()
		return m, nil

	case errorMsg:
		m.err = msg.err
		return m, tea.Quit
	}

	// Pass all other messages to the list component
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return appStyle.Render(fmt.Sprintf("Błąd: %s", m.err.Error()))
	}

	if m.showViewport {
		return docStyle.Render(m.viewport.View())
	}

	return appStyle.Render(m.list.View())
}

// --- HELPERS ---

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
			return errorMsg{fmt.Errorf("nieobsługiwany system operacyjny: %s", runtime.GOOS)}
		}

		if err := cmd.Run(); err != nil {
			return errorMsg{fmt.Errorf("nie udało się otworzyć pliku PDF: %w", err)}
		}
		return nil
	}
}

// --- MAIN ---

func main() {
	config, err := getConfig()
	if err != nil {
		fmt.Printf("Błąd inicjalizacji: %v\n", err)
		os.Exit(1)
	}

	notes, err := findNotes(config.NotesPath)
	if err != nil {
		log.Fatalf("Błąd krytyczny: nie udało się przeszukać folderu '%s': %v", config.NotesPath, err)
	}

	m := initialModel(notes, config)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}

func getConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	configPath := filepath.Join(home, ".config", "gopuntes", "config.toml")

	var config Config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("Witaj w gopuntes! Wygląda na to, że to pierwsze uruchomienie.")
		fmt.Print("Podaj pełną ścieżkę do folderu z notatkami: ")

		reader := bufio.NewReader(os.Stdin)
		notesPath, err := reader.ReadString('\n')
		if err != nil {
			return config, fmt.Errorf("nie udało się odczytać ścieżki: %w", err)
		}
		notesPath = strings.TrimSpace(notesPath)

		config.NotesPath = notesPath

		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return config, fmt.Errorf("nie udało się stworzyć folderu konfiguracyjnego: %w", err)
		}

		file, err := os.Create(configPath)
		if err != nil {
			return config, fmt.Errorf("nie udało się stworzyć pliku konfiguracyjnego: %w", err)
		}
		defer file.Close()

		if err := toml.NewEncoder(file).Encode(config); err != nil {
			return config, fmt.Errorf("nie udało się zapisać konfiguracji: %w", err)
		}
		fmt.Printf("Ścieżka zapisana w %s\n", configPath)
		fmt.Println("Uruchamianie interfejsu...")
	}

	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return config, fmt.Errorf("nie udało się wczytać pliku konfiguracyjnego: %w", err)
	}

	if config.NotesPath == "" {
		return config, fmt.Errorf("plik konfiguracyjny %s nie zawiera ścieżki 'notes_path'", configPath)
	}

	return config, nil
}

func findNotes(root string) ([]item, error) {
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
	return items, err
}
