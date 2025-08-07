[![GoDoc](https://godoc.org/fortio.org/tcolor?status.svg)](https://pkg.go.dev/fortio.org/tcolor)
[![Go Report Card](https://goreportcard.com/badge/fortio.org/tcolor)](https://goreportcard.com/report/fortio.org/tcolor)
[![CI Checks](https://github.com/fortio/tcolor/actions/workflows/include.yml/badge.svg)](https://github.com/fortio/tcolor/actions/workflows/include.yml)
# tcolor
Terminal Color chooser using Ansipixels library

`tcolor` is a simple terminal/TUI color picker/chooser/explorer.

## Install
You can get the binary from [releases](https://github.com/fortio/tcolor/releases)

Or just run
```
CGO_ENABLED=0 go install fortio.org/tcolor@latest  # to install (in ~/go/bin typically) or just
CGO_ENABLED=0 go run fortio.org/tcolor@latest  # to run without install
```

or even
```
docker run -ti fortio/tcolor # but that's obviously slower
```

or
```
brew install fortio/tap/tcolor
```

## Run

Currently 4 screens:
- Basic 16 colors
- 256 Colors
- 24 bits Hue Saturation Luminance (HSL)
- 24 bits RGB where space bar change which component is set with arrows.

Up and down arrows to increase luminance on the HSL screen, the third color component on the RGB screen.

```sh
tcolor help
```
```
flags:
  -true-color
        Use true color (24-bit RGB) instead of 8-bit ANSI colors (default is true if
COLORTERM is set)
```
