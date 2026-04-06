package main

import (
	"fmt"
	"os"

	"nrcc/cmd"
)

func main() {
	if err := cmd.Run(os.Args[1:], frontendFS); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
