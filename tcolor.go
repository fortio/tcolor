package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/safecast"
	"fortio.org/sets"
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
	modeOKLCHColors
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
	AP           *ansipixels.AnsiPixels
	Mode         mode
	Step         int                     // Step is the lightness step for HSL colors, other color for RGB. 0-255
	Component    component               // Component is the current color component missing/adjusted with arrows in RGB mode
	Dirty        bool                    // Used to track if the screen needs repainting
	MouseAt      map[[2]int]tcolor.Color // MouseAt tracks mouse positions and colors at those positions
	Title        string                  // Title is the current title of the screen/mode.
	SavedColors  sets.Set[string]        // SavedColors is a list of colors strings/info saved by the user.
	ShowHelp     bool                    // ShowHelp is a flag to indicate if help should be shown.
	LastX, LastY int                     // LastX and LastY are the last mouse coordinates for the mouse event.
	Rounding     int                     // Rounding is the number of digits to round HSL/OKLCH values for WebHSL/OKLCH output.
}

func Main() int {
	cli.MaxArgs = -1
	cli.ArgsHelp = "[colors] decode or explore colors"
	fFps := flag.Float64("fps", 60, "Frames per second for the terminal refresh rate")
	fTrueColor := flag.Bool("truecolor", ansipixels.DetectColorMode().TrueColor,
		"Use true color (24-bit RGB) instead of 8-bit ANSI colors (default is true if COLORTERM is set)")
	fRounding := flag.Int("rounding", -1,
		"Number of digits to round HSL/OKLCH values for WebHSL/OKLCH output/copy paste (negative for default full precision)")
	cli.Main()
	ap := ansipixels.NewAnsiPixels(*fFps)
	ap.TrueColor = *fTrueColor
	if ap.TrueColor {
		log.Infof("Using 24 bits true color")
	} else {
		log.Infof("Using 256 colors")
	}
	if flag.NArg() > 0 {
		return decodeColors(ap.ColorOutput, *fRounding, flag.Args())
	}
	if err := ap.Open(); err != nil {
		return log.FErrf("Error opening terminal: %v", err)
	}
	defer func() {
		ap.MouseTrackingOff()
		// ap.ShowCursor()
		ap.Restore()
	}()
	ap.MouseTrackingOn()
	// ap.HideCursor()
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
		MouseAt:     make(map[[2]int]tcolor.Color),
		SavedColors: sets.New[string](),
		ShowHelp:    true,
		Rounding:    *fRounding,
	}
	ap.OnResize = func() error {
		s.Dirty = true
		s.ShowHelp = true // (re)Show help on resize
		s.Repaint()
		return nil
	}
	ap.OnMouse = s.OnMouse
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
		case 'q', 'Q', 3: // q Q or ctrl-c
			if s.Mode == mode16Colors {
				// clear the help text so we don't have to require more than 24 lines.
				s.Dirty = true
				s.Repaint()
			}
			s.AP.MoveCursor(0, 22)
			s.AP.EndSyncMode()
			if len(s.SavedColors) == 0 {
				log.Infof("Exiting, no colors saved.")
				return 0
			}
			fmt.Fprintf(crlfWriter, "Exiting. Saved colors: \n")
			for _, color := range sets.Sort(s.SavedColors) {
				fmt.Fprintf(crlfWriter, "  %s \n", color)
			}
			fmt.Fprintf(crlfWriter, "\n")
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

func (s *State) OnMouse() {
	x, y := s.AP.Mx, s.AP.My
	color, ok := s.MouseAt[[2]int{x, y}]
	click := s.AP.MouseRelease()
	doUpdate := click || s.LastX != x || s.LastY != y
	s.LastX, s.LastY = x, y
	if !doUpdate {
		s.AP.MoveCursor(s.LastX-1, s.LastY-1)
		return
	}
	if !ok {
		return
	}
	s.AP.WriteAt(0, 0, s.Title)
	s.AP.ClearEndOfLine()
	colorString, colorExtra, ctype := color.Extra()
	clipBoardColor := colorString
	webHSL := tcolor.WebHSL(color, s.Rounding)
	webOKLCH := tcolor.WebOklch(color, s.Rounding)
	if ctype == tcolor.ColorTypeHSL || ctype == tcolor.ColorTypeRGB {
		switch {
		case s.AP.RightClick():
			clipBoardColor = webHSL
		case s.AP.AnyModifier():
			clipBoardColor = webOKLCH
		default:
			clipBoardColor = colorExtra
		}
	}
	if colorExtra != "" {
		colorExtra = " (" + colorExtra + ")"
	}
	if webHSL != "" {
		colorExtra += " " + webHSL
	}
	if webOKLCH != "" {
		colorExtra += " " + webOKLCH
	}
	extra := ""
	if click {
		extra = "Copied "
	}
	s.AP.WriteRight(0, "%s%s   %d,%d   %s %s%s", extra, color.Background(), x, y, tcolor.Reset, colorString, colorExtra)
	if click {
		s.AP.CopyToClipboard(clipBoardColor)
		s.SavedColors.Add(fmt.Sprintf("%s    %s : %s%s",
			color.Background(), tcolor.Reset, colorString, colorExtra))
	}
	s.AP.MoveCursor(x-1, y-1)
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
			s.Show16colors()
		case mode256Colors:
			s.Show256colors()
		case modeHSLColors:
			s.ShowHSLColors()
		case modeOKLCHColors:
			s.ShowOKLCHColors()
		case modeRGBColors:
			s.ShowRGBColors()
		default:
			panic("invalid mode")
		}
		s.OnMouse()
		s.Dirty = false
	}
}

func (s *State) writeBasicColor(line int, i tcolor.BasicColor) int {
	s.AP.WriteString(fmt.Sprintf("%15s: %s   %s\r\n", i.String(), i.Background(), tcolor.Reset))
	s.MouseAt[[2]int{18, line}] = tcolor.Basic(i)
	s.MouseAt[[2]int{19, line}] = tcolor.Basic(i)
	s.MouseAt[[2]int{20, line}] = tcolor.Basic(i)
	return line + 1
}

func (s *State) Show16colors() {
	s.NewPage("        16 Basic Colors")
	s.AP.WriteString("\r\n\n")
	line := 3 // in mouse coords 1,1 start
	for i := tcolor.Black; i <= tcolor.Gray; i++ {
		line = s.writeBasicColor(line, i)
	}
	for i := tcolor.DarkGray; i <= tcolor.White; i++ {
		line = s.writeBasicColor(line, i)
	}
	// Also show our extra orange
	s.AP.WriteString("\r\n Extra ansipixel named color\r\n")
	line += 2
	_ = s.writeBasicColor(line, tcolor.Orange)
	if s.ShowHelp {
		s.AP.WriteString("\r\n Use space and arrows to navigate, mouse to see and select colors,\r\n" +
			" click to copy to clipboard and save for showing at the end (Q to exit) ")
		s.ShowHelp = false // Only show help once
	}
}

func (s *State) write256color(i tcolor.Uint8, x, line int) {
	c := tcolor.Color256(i).Color()
	s.MouseAt[[2]int{x + 1, line}] = c
	s.MouseAt[[2]int{x + 2, line}] = c
	s.AP.WriteString(c.Background() + "  ")
}

func (s *State) Show256colors() {
	s.NewPage("      256 colors")
	s.AP.WriteString("\r\n\n 16 basic colors\r\n\n ")
	line := 5 // in mouse coords 1,1 start
	for i := range 16 {
		s.write256color(tcolor.Uint8(i), 2*i+1, line) //nolint:gosec // duh 0-16 overflows, right...
	}
	s.AP.WriteString("\033[0m\r\n\r\n 216 cube\r\n")
	line += 3
	x := 1
	for i := 16; i < 232; i++ {
		if (i-16)%36 == 0 {
			s.AP.WriteString("\033[0m\r\n ")
			line++
			x = 1
		}
		s.write256color(tcolor.Uint8(i), x, line) //nolint:gosec // duh 16-231 overflows, right...
		x += 2
	}
	s.AP.WriteString("\033[0m\r\n\r\n Grayscale\r\n\r\n ")
	line += 4
	x = 1
	for i := 232; i < 256; i++ {
		s.write256color(tcolor.Uint8(i), x, line) //nolint:gosec // duh 232-255 overflows, right...
		x += 2
	}
	s.AP.WriteString(tcolor.Reset + "\r\n\n")
}

func (s *State) NewPage(title string) {
	clear(s.MouseAt)
	s.LastX, s.LastY = -1, -1 // Reset last mouse position
	s.Title = title
	s.AP.WriteString(s.Title)
}

func (s *State) ShowHSLColors() {
	s.NewPage("HSL colors")
	lightness := tcolor.Uint10(s.Step << 2) //nolint:gosec // gosec not smart enough to see that 0-255<<2 is ok.
	var sat tcolor.Uint8
	var hue tcolor.Uint12
	// leave bottom line for status
	available := s.AP.H - 1 - 1
	for ll := 1; ll < s.AP.H-1; ll++ {
		s.AP.WriteString(tcolor.Reset + "\r\n")
		offset := 8 // skip some of the gray-er colors (low saturation)
		sat = tcolor.Uint8(math.Round(255. * float64(ll+offset) / float64(available+offset)))
		for hh := range s.AP.W {
			hue = tcolor.Uint12(math.Round(4095 * float64(hh) / float64(s.AP.W)))
			// Use the lightness step for HSL colors
			color := tcolor.HSLColor{H: hue, S: sat, L: lightness}.Color()
			s.MouseAt[[2]int{hh + 1, ll + 1}] = color
			s.AP.WriteBg(color)
			s.AP.WriteRune(' ')
		}
	}
	s.AP.WriteAt(0, s.AP.H-1, "%sColor: Lightness=%d x%X ↑ to increase ↓ to decrease (shift for precise steps) ",
		tcolor.Reset, lightness, s.Step)
}

func (s *State) ShowOKLCHColors() {
	s.NewPage("OKLCH colors")
	l := float64(s.Step) / 255.
	var c float64
	var h float64
	// leave bottom line for status
	available := s.AP.H - 1 - 1
	for ll := 1; ll < s.AP.H-1; ll++ {
		s.AP.WriteString(tcolor.Reset + "\r\n")
		c = float64(ll) / float64(available)
		for hh := range s.AP.W {
			h = float64(hh) / float64(s.AP.W)
			// Use the lightness step for OKLCH colors
			color := tcolor.Oklchf(l, c, h)
			s.MouseAt[[2]int{hh + 1, ll + 1}] = color
			s.AP.WriteBg(color)
			s.AP.WriteRune(' ')
		}
	}
	s.AP.WriteAt(0, s.AP.H-1, "%sColor: L=%.3f x%X ↑ to increase ↓ to decrease (shift for precise steps) ",
		tcolor.Reset, l, s.Step)
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

func (s *State) ShowRGBColors() {
	s.NewPage("RGB colors")
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
			s.MouseAt[[2]int{hh + 1, l + 2}] = color
			s.AP.WriteBg(color)
			s.AP.WriteRune(' ')
		}
	}
	s.AP.WriteAt(0, s.AP.H-1, "%sColor: %s=%d x%X ↑ to increase ↓ to decrease (shift for precise steps) ",
		tcolor.Reset, label, s.Step, s.Step)
}
