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

type State struct {
	AP        *ansipixels.AnsiPixels
	Mode      int
	Lightness float64
	Dirty     bool // Used to track if the screen needs repainting
}

func Main() int {
	cli.ArgsHelp = " explore colors"
	fFps := flag.Float64("fps", 60, "Frames per second for the terminal refresh rate")
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
	ap := ansipixels.NewAnsiPixels(*fFps)
	if err := ap.Open(); err != nil {
		return log.FErrf("Error opening terminal: %v", err)
	}
	defer func() {
		ap.MouseTrackingOff()
		ap.Restore()
	}()
	ap.MouseTrackingOn()
	// Cursor works best with ghostty:
	//   shell-integration-features= no-cursor
	//   cursor-style = block_hollow
	// Or we could do blinking block:
	//	 ap.WriteString("\033[1 q")
	crlfWriter := &terminal.CRLFWriter{Out: os.Stdout}
	terminal.LoggerSetup(crlfWriter)
	s := &State{
		AP:        ap,
		Mode:      0,
		Lightness: 0.5, // Default lightness for HSL colors
	}
	ap.OnResize = func() error {
		s.Repaint()
		return nil
	}
	s.Dirty = true
	for {
		s.Repaint()
		if err := ap.ReadOrResizeOrSignal(); err != nil {
			return log.FErrf("Error reading terminal: %v", err)
		}
		if len(ap.Data) == 0 {
			// No data, just a resize or signal, continue to next iteration.
			continue
		}
		c := ap.Data[0]
		switch c {
		case 'q', 'Q':
			log.Infof("Exiting on 'q' or 'Q'")
			return 0
		case 27: // ESC
			s.Dirty = true
			if len(ap.Data) >= 3 && ap.Data[1] == '[' {
				// Arrow key
				switch ap.Data[2] {
				case 'D': // left arrow
					s.Mode = (s.Mode + 3 - 1) % 3
				case 'A': // up arrow
					s.Lightness *= 1.05
					if s.Lightness > 1.0 {
						s.Lightness = 1.0 // Cap lightness at 1.0
					} else if s.Lightness <= 1.0/255.0 {
						s.Lightness = 1.0 / 255.0 // if we got lower than resolution and are going back up, set to 1/255
					}
				case 'B': // down arrow
					s.Lightness /= 1.05
				case 'C': // right arrow
					s.Mode = (s.Mode + 1) % 3
				default:
				}
			}
		default:
			s.Mode = (s.Mode + 1) % 3
			s.Dirty = true
			log.Infof("Received input: %q", ap.Data)
		}
	}
}

func (s *State) Repaint() {
	if s.Dirty || s.Mode == 2 {
		s.AP.StartSyncMode()
		s.AP.ClearScreen()
		switch s.Mode {
		case 0:
			s.show16colors()
		case 1:
			s.show256colors()
		case 2:
			s.showHSLColors()
		}
		s.Dirty = false
	}
	s.AP.MoveCursor(s.AP.Mx-1, s.AP.My-1)
}

func (s *State) show16colors() {
	s.AP.WriteString("       Basic 16 colors\r\n\n")
	for i := tcolor.Black; i <= tcolor.Gray; i++ {
		s.AP.WriteString(fmt.Sprintf("%15s: %s   %s\r\n", i.String(), i.Background(), tcolor.Reset))
	}
	for i := tcolor.DarkGray; i <= tcolor.White; i++ {
		s.AP.WriteString(fmt.Sprintf("%15s: %s   %s\r\n", i.String(), i.Background(), tcolor.Reset))
	}
}

func (s *State) show256colors() {
	s.AP.WriteString("      256 colors\r\n\n 16 basic colors\r\n\n ")
	for i := range 16 {
		s.AP.WriteString(fmt.Sprintf("\033[48;5;%dm  ", i))
	}
	s.AP.WriteString("\033[0m\r\n\r\n 216 cube\r\n")
	for i := 16; i < 232; i++ {
		if (i-16)%36 == 0 {
			s.AP.WriteString("\033[0m\r\n ")
		}
		s.AP.WriteString(fmt.Sprintf("\033[48;5;%dm  ", i))
	}
	s.AP.WriteString("\033[0m\r\n\r\n Grayscale\r\n\r\n ")
	for i := 232; i < 256; i++ {
		s.AP.WriteString(fmt.Sprintf("\033[48;5;%dm  ", i))
	}
	s.AP.WriteString(tcolor.Reset)
}

func (s *State) showHSLColors() {
	s.AP.WriteString("HSL colors")
	var hue, sat float64
	// leave bottom line for status
	available := s.AP.H - 1 - 1
	for ll := 1; ll < s.AP.H-1; ll++ {
		s.AP.WriteString(tcolor.Reset + "\r\n")
		offset := 8
		sat = float64(ll+offset) / float64(available+offset)
		for hh := range s.AP.W {
			hue = float64(hh) / float64(s.AP.W)
			color := tcolor.HSLToRGB(hue, sat, s.Lightness)
			s.AP.WriteString(color.Background() + " ")
		}
	}
	s.AP.WriteAt(0, s.AP.H-1, "%sColor: HSL(%.3f, %.3f, %.3f) ↑ to increase ↓ to decrease Lightness ",
		tcolor.Reset, hue, sat, s.Lightness)
}
