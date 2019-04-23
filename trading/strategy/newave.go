package strategy

import (
	"github.com/rkjdid/gocx/trading"
	"log"
)

type NewaveOpts struct {
	Slow, Fast MACDOpts
}

func (opts NewaveOpts) NewNewave() *Newave {
	return &Newave{
		Slow: opts.Slow.NewMACDCross(),
		Fast: opts.Fast.NewMACDCross(),
	}
}

type Newave struct {
	Slow, Fast *MACDCross
	LastSignal Signal
}

func (nw *Newave) AddTick(x trading.Tick) {
	if x.Timeframe.Equals(nw.Slow.Timeframe) {
		nw.Slow.AddTick(x)
	} else if x.Timeframe.Equals(nw.Fast.Timeframe) {
		nw.Fast.AddTick(x)
	} else {
		log.Printf("bad timeframe: %s", x.Timeframe)
	}

	if nw.Fast.LastSignal.Action == nw.Slow.LastSignal.Action &&
		nw.Fast.LastSignal.Action == Buy {
		nw.LastSignal = Signal{
			Action:   Buy,
			Time:     nw.Fast.LastSignal.Time,
			Strength: 1,
		}
	} else {
		nw.LastSignal = NoSignal
	}
}

func (nw *Newave) Signal() Signal {
	return nw.LastSignal
}
