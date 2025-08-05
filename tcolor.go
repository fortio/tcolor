package main

import (
	"flag"
	"fmt"
	"os"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/terminal"
	"fortio.org/terminal/ansipixels"
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

	ap := ansipixels.NewAnsiPixels(60)
	if err := ap.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening terminal: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		ap.ShowCursor()
		ap.MouseTrackingOff()
		ap.Restore()
	}()
	ap.HideCursor()
	ap.MouseTrackingOn()
	crlfWriter := &terminal.CRLFWriter{Out: os.Stdout}
	terminal.LoggerSetup(crlfWriter)
	mode := 0
	ap.OnResize = func() error {
		Repaint(ap, mode)
		return nil
	}
	for {
		Repaint(ap, mode)
		if err := ap.ReadOrResizeOrSignal(); err != nil {
			return log.FErrf("Error reading terminal: %v", err)
		}
		if len(ap.Data) == 0 {
			// No data, just a resize or signal, continue to next iteration.
			continue
		}
		switch ap.Data[0] {
		case 'q', 'Q':
			log.Infof("Exiting on 'q' or 'Q'")
			return 0
		default:
			mode = (mode + 1) % 3
			log.Infof("Received input: %q", ap.Data)
		}
	}
}
func Repaint(ap *ansipixels.AnsiPixels, mode int) {
	ap.StartSyncMode()
	ap.ClearScreen()
	switch mode {
	case 0:
		show16colors(ap)
	case 1:
		show256colors(ap)
	case 2:
		showHSLColors(ap)
	}
}

func show16colors(ap *ansipixels.AnsiPixels) {
	ap.WriteString("       Basic 16 colors\r\n\n")
	for i := tcolor.Black; i <= tcolor.Gray; i++ {
		ap.WriteString(fmt.Sprintf("%15s: %s   %s\r\n", i.String(), i.Background(), tcolor.Reset))
	}
	for i := tcolor.DarkGray; i <= tcolor.White; i++ {
		ap.WriteString(fmt.Sprintf("%15s: %s   %s\r\n", i.String(), i.Background(), tcolor.Reset))
	}
}

func show256colors(ap *ansipixels.AnsiPixels) {
	ap.WriteString("       256 colors\r\n\n16 basic colors\r\n\n")
	for i := 0; i < 16; i++ {
		ap.WriteString(fmt.Sprintf("\033[48;5;%dm  ", i))
	}
	ap.WriteString("\033[0m\r\n\r\n216 cube\r\n")
	for i := 16; i < 232; i++ {
		if (i-16)%36 == 0 {
			ap.WriteString("\033[0m\r\n")
		}
		ap.WriteString(fmt.Sprintf("\033[48;5;%dm  ", i))
	}
	ap.WriteString("\033[0m\r\n\r\nGrayscale\r\n\r\n")
	for i := 232; i < 256; i++ {
		ap.WriteString(fmt.Sprintf("\033[48;5;%dm  ", i))
	}
	ap.WriteString(tcolor.Reset)
}

func showHSLColors(ap *ansipixels.AnsiPixels) {
	ap.WriteString("HSL colors")
	var h, s, l float64
	l = 0.5 // lightness
	// leave bottom line for status
	available := ap.H - 1 - 1
	for ll := 1; ll < ap.H-1; ll++ {
		ap.WriteString(tcolor.Reset + "\r\n")
		offset := 8
		s = float64(ll+offset) / float64(available+offset)
		/*if s > 1.0 {
			s = 1.0
		}*/
		for hh := range ap.W / 2 {
			h = float64(hh) / float64(ap.W/2)
			color := tcolor.HSLToRGB(h, s, l)
			ap.WriteString(color.Background() + "  ")
		}
	}
	ap.WriteString(tcolor.Reset + "\r\nColor: HSL(hue, saturation, lightness)")
}
