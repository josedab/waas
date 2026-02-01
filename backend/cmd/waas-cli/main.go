package main

import (
	"os"

	"github.com/josedab/waas/cmd/waas-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
