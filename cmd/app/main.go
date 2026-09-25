package main

import (
	"os"

	_ "time/tzdata"

	app "platform/backend/internal/routes"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "migrate" {
		app.RunMigrate(args[1:])
		return
	}

	app.RunApp()
}
