package main

import (
	"flag"
	"os"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/terminal/ansipixels/tcolor"
)

func main() {
	os.Exit(Main())
}

func Main() int {
	cli.ArgsHelp = " explore colors"
	defaultTrueColor := false
	if os.Getenv("COLORTERM") != "" {
		defaultTrueColor = true
	}
	fTrueColor := flag.Bool("true-color", defaultTrueColor,
		"Use true color (24-bit RGB) instead of 8-bit ANSI colors (default is true if COLORTERM is set)")
	cli.Main()
	colorOutput := tcolor.ColorOutput{TrueColor: *fTrueColor}
	if colorOutput.TrueColor {
		log.Infof("Using 24 bits true color")
	} else {
		log.Infof("Using 256 colors")
	}
	return 0
}
