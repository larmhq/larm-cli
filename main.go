package main

import (
	"os"

	"github.com/larmhq/larm-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
