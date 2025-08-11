package main

import (
	"flag"
	"fmt"
	"os"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/safecast"
	"fortio.org/terminal"
	"fortio.org/terminal/ansipixels"
	"fortio.org/terminal/ansipixels/tcolor"
)

func main() {
	os.Exit(Main())
}

type mode int

const (
	mode16Colors mode = iota
	mode256Colors
	modeHSLColors
	modeRGBColors
	maxMode
)

type component int

const (
	componentRed component = iota
	componentGreen
	componentBlue
	numColorComponents
)

type State struct {
	AP          *ansipixels.AnsiPixels
	Mode        mode
	Step        int                // Step is the lightness step for HSL colors, other color for RGB. 0-255
	Component   component          // Component is the current color component missing/adjusted with arrows in RGB mode
	Dirty       bool               // Used to track if the screen needs repainting
	ColorOutput tcolor.ColorOutput // For truecolor to 256 color support
}

func Main() int {
	cli.ArgsHelp = " explore colors"
	fFps := flag.Float64("fps", 60, "Frames per second for the terminal refresh rate")
	fTrueColor := flag.Bool("truecolor", ansipixels.DetectColorMode().TrueColor,
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
		AP:          ap,
		Mode:        mode16Colors, // Start with 16 colors
		Step:        128,          // Default lightness (128/255) for HSL colors
		ColorOutput: colorOutput,
	}
	ap.OnResize = func() error {
		s.Dirty = true
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
			s.Dirty = false
			continue
		}
		c := ap.Data[0]
		s.Dirty = true
		switch c {
		case 'q', 'Q':
			log.Infof("Exiting on 'q' or 'Q'")
			return 0
		case 27: // ESC
			if len(ap.Data) >= 3 && ap.Data[1] == '[' {
				s.processArrowKey()
			}
		case ' ':
			if s.Mode == modeRGBColors {
				s.Component = (s.Component + 1) % numColorComponents // Cycle through color components
			} else {
				s.NextMode() // Cycle through modes
			}
		default:
			s.NextMode()
		}
	}
}

func (s *State) processArrowKey() {
	dir := s.AP.Data[2]
	precise := false
	if len(s.AP.Data) >= 6 { // Modifier sequence eg "\x1b[1;2A"
		dir = s.AP.Data[5]
		precise = true // Shift key pressed
	}
	// Arrow key
	switch dir {
	case 'D': // left arrow
		s.PrevMode()
	case 'A': // up arrow
		if precise {
			s.Step++
		} else {
			s.Step += 16 // Increase step by 16 for coarse adjustment
		}
		if s.Step > 255 {
			s.Step = 255 // Cap step at 255
		}
	case 'B': // down arrow
		if precise {
			s.Step--
		} else {
			s.Step -= 16 // Decrease step by 16 for coarse adjustment
		}
		if s.Step < 0 {
			s.Step = 0 // Cap step at 0
		}
	case 'C': // right arrow
		s.NextMode()
	default:
	}
}

func (s *State) PrevMode() {
	s.Mode = (s.Mode + maxMode - 1) % maxMode // Cycle to previous mode
	s.Dirty = true
}

func (s *State) NextMode() {
	s.Mode = (s.Mode + 1) % maxMode // Cycle through modes
	s.Dirty = true
}

func (s *State) Repaint() {
	if s.Dirty {
		s.AP.StartSyncMode()
		s.AP.ClearScreen()
		switch s.Mode {
		case mode16Colors:
			s.show16colors()
		case mode256Colors:
			s.show256colors()
		case modeHSLColors:
			s.showHSLColors()
		case modeRGBColors:
			s.showRGBColors()
		default:
			panic("invalid mode")
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
	lightness := float64(s.Step) / 255.0
	var hue, sat float64
	// leave bottom line for status
	available := s.AP.H - 1 - 1
	for ll := 1; ll < s.AP.H-1; ll++ {
		s.AP.WriteString(tcolor.Reset + "\r\n")
		offset := 8 // skip some of the gray-er colors (low saturation)
		sat = float64(ll+offset) / float64(available+offset)
		for hh := range s.AP.W {
			hue = float64(hh) / float64(s.AP.W)
			// Use the lightness step for HSL colors
			color := tcolor.HSLToRGB(hue, sat, lightness).Color()
			s.AP.WriteString(s.ColorOutput.Background(color) + " ")
		}
	}
	s.AP.WriteAt(0, s.AP.H-1, "%sColor: Lightness=%.3f x%X ↑ to increase ↓ to decrease (shift for precise steps) ",
		tcolor.Reset, lightness, s.Step)
}

func (s *State) makeColor(xi, yi, zi int) (tcolor.Color, string) {
	x, y, z := safecast.MustConvert[uint8](xi), safecast.MustConvert[uint8](yi), safecast.MustConvert[uint8](zi)
	switch s.Component {
	case componentRed:
		color := tcolor.RGBColor{R: z, G: x, B: y}.Color()
		return color, "Red"
	case componentGreen:
		color := tcolor.RGBColor{R: x, G: z, B: y}.Color()
		return color, "Green"
	case componentBlue:
		color := tcolor.RGBColor{R: y, G: x, B: z}.Color()
		return color, "Blue"
	default:
		panic("Invalid color component")
	}
}

func (s *State) showRGBColors() {
	s.AP.WriteString("RGB colors")
	z := s.Step
	var y, x int
	// leave bottom line for status
	available := s.AP.H - 1 - 1
	lastL := available - 1
	var label string
	var color tcolor.Color
	for l := range available {
		s.AP.WriteString(tcolor.Reset + "\r\n")
		y = 255 * l / lastL
		for hh := range s.AP.W {
			x = 255 * hh / (s.AP.W - 1)
			// Use the step value for the selected RGB component
			color, label = s.makeColor(x, y, z)
			s.AP.WriteString(s.ColorOutput.Background(color) + " ")
		}
	}
	s.AP.WriteAt(0, s.AP.H-1, "%sColor: %s=%d x%X ↑ to increase ↓ to decrease (shift for precise steps) ",
		tcolor.Reset, label, s.Step, s.Step)
}
