package main

import (
	"fmt"
	"os"

	"github.com/adeleke5140/imitsu/tui/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "upgrade":
			if err := upgrade(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "--version", "-v", "version":
			fmt.Printf("itui v%s\n", version)
			return
		case "--help", "-h", "help":
			fmt.Println("itui - Interactive TUI for imitsu")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  itui              Launch the TUI")
			fmt.Println("  itui upgrade      Upgrade to the latest version")
			fmt.Println("  itui version      Show version")
			fmt.Println("  itui help         Show this help")
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			fmt.Fprintf(os.Stderr, "Run 'itui help' for usage\n")
			os.Exit(1)
		}
	}

	p := tea.NewProgram(ui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
