package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rebeliceyang/lazypg/internal/app"
	"github.com/rebeliceyang/lazypg/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Could not load config: %v (using defaults)\n", err)
		cfg = config.GetDefaults()
	}

	// Create program with mouse support based on config
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if cfg.UI.MouseEnabled {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(app.New(cfg), opts...)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
