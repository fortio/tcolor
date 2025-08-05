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

Move the mouse wheel to zoom, arrows to increase luminance

```sh
tcolor help
```
```
flags:
  -true-color
        Use true color (24-bit RGB) instead of 8-bit ANSI colors (default is true if
COLORTERM is set)
```
