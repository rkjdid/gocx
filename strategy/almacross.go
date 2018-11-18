package strategy

import (
	"fmt"
	"github.com/markcheno/go-talib"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/ts"
	"math"
	"time"
)

type ALMACross struct {
	Short     int
	Long      int
	Buffer    []float64
	Ready     bool
	PrevState bool
}

func NewALMACross(s, l int) *ALMACross {
	if l <= s {
		panic("alma long <= sort")
	}
	return &ALMACross{
		s, l, nil, false, false,
	}
}

func (s *ALMACross) AddTick(ohlcv ts.OHLCV) Signal {
	s.Buffer = append(s.Buffer, ohlcv.Close)
	if len(s.Buffer) <= s.Long {
		return NoSignal
	}
	salma := talib.Alma(s.Buffer, s.Short, 6, 0.6)
	lalma := talib.Alma(s.Buffer, s.Long, 6, 0.6)
	i := len(s.Buffer) - 1
	s.Buffer = s.Buffer[1:]
	state := salma[i] >= lalma[i]
	if !s.Ready {
		s.Ready = true
		s.PrevState = state
		return NoSignal
	}
	if state != s.PrevState {
		s.PrevState = state
		action := Sell
		if state {
			action = Buy
		}
		return Signal{
			time.Time(ohlcv.Timestamp),
			action,
			math.Abs(lalma[i]-salma[i]) / ohlcv.Close * 100,
		}
	}
	return NoSignal
}

func (s *ALMACross) Draw(data ts.OHLCVs) error {
	chart.AddLine(data.ToXYer(talib.Alma(data.Close(), s.Short, 6, 0.6)[s.Short:])[0],
		fmt.Sprintf("alma%d", s.Short))
	chart.AddLine(data.ToXYer(talib.Alma(data.Close(), s.Long, 6, 0.6)[s.Long:])[0],
		fmt.Sprintf("alma%d", s.Long))
	return nil
}
