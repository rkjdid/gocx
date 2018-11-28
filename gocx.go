package gocx

import "flag"

var (
	Debug bool
	Chart bool
)

func init() {
	flag.BoolVar(&Debug, "debug", false, "enable debug log")
	flag.BoolVar(&Chart, "chart", false, "generate chart")
}
