package gocx

import "flag"

var (
	Debug bool
)

func init() {
	flag.BoolVar(&Debug, "debug", false, "enable debug log")
}
