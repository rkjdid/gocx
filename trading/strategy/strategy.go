package strategy

import (
	"fmt"
	"github.com/rkjdid/gocx/ts"
	"github.com/rkjdid/util"
	"time"
)

type Strategy interface {
	AddTick(ts.OHLCV)
	Signal() Signal
}

type Signal struct {
	Time     time.Time
	Action   Action
	Strength float64
	Data     interface{}
}

var NoSignal Signal

func (sig Signal) String() string {
	return fmt.Sprintf("%4s %s (%f)",
		sig.Action, util.ParisTime(sig.Time), sig.Strength)
}

type Action int

const (
	None = Action(iota)
	Buy
	Sell
)

func (a Action) String() string {
	switch a {
	case None:
		return "NONE"
	case Buy:
		return "BUY"
	case Sell:
		return "SELL"
	}
	return ""
}
