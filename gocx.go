package gocx

import (
	"flag"
	"github.com/mediocregopher/radix.v2/redis"
)

var (
	Debug bool
	Chart bool

	Db *redis.Client
)

func init() {
	flag.BoolVar(&Debug, "debug", false, "enable debug log")
	flag.BoolVar(&Chart, "chart", false, "generate chart")
}
