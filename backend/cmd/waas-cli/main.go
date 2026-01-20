package main

import (
	"os"

	"webhook-platform/cmd/waas-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
