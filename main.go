package main

import (
	"fmt"
	"os"
)

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Err: program failed to run: %v\n", err)
		os.Exit(1)
	}
}
