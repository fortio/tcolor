package main

import (
	"fmt"

	"fortio.org/log"
	"fortio.org/terminal/ansipixels/tcolor"
)

func decodeColors(colorOutput tcolor.ColorOutput, rounding int, args []string) int {
	log.Infof("Decoding %d colors mode (pass no argument for interactive)", len(args))
	for _, arg := range args {
		color, err := tcolor.FromString(arg)
		if err != nil {
			log.Warnf("Invalid color '%s': %v", arg, err)
			continue
		}
		name, rgb, _ := color.Extra()
		hsl := tcolor.WebHSL(color, rounding)
		oklch := tcolor.WebOklch(color, rounding)
		fmt.Printf(" %s    %s %s %s %s %s\n", colorOutput.Background(color), tcolor.Reset, name, rgb, hsl, oklch)
	}
	return 0
}
