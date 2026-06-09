package main

import (
	"fmt"
	"os"

	"github.com/zzokki81/eventmesh/notifier/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}
