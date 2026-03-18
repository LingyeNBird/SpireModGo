package main

import (
	"fmt"
	"os"

	"slaymodgo/internal/manager"
	"slaymodgo/internal/ui"
)

func main() {
	mgr, err := manager.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

	app := ui.NewApp(mgr)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}
