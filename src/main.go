package main

import (
	"os"

	"github.com/cappuch/cuardach/src/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
