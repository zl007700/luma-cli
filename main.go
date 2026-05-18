package main

import (
	"os"

	"github.com/luma-cli/lumer-cli/internal/commands"
)

func main() {
	os.Exit(commands.Run(os.Args))
}
