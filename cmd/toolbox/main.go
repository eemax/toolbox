package main

import (
	"os"

	"toolbox/internal/cli"
)

var version = "dev"

func main() {
	app := cli.New(version, os.Stdout, os.Stderr, nil)
	os.Exit(app.Execute())
}
