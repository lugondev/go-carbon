package main

import (
	"os"

	"github.com/lugondev/go-carbon/cmd/carbon/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
