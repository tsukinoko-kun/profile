package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var teaProgram *tea.Program
var RedoTakenTests = false

func main() {
	teaProgram = tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	go func() {
		time.Sleep(time.Millisecond * 50)
		Begin()
	}()
	if _, err := teaProgram.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if quitErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", quitErr)
		os.Exit(1)
	}

	for _, category := range Categories {
		fmt.Printf("%s: %d\n", category.Name, category.Score())
		for _, subCategory := range category.SubCategories {
			fmt.Printf("  %s: %d\n", subCategory.Name, subCategory.Score)
		}
	}
}
