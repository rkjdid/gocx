package strategy

import (
	"github.com/markcheno/go-talib"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/ts"
	"time"
)

const MaxLookback = 16000

type MACD struct {
	Data                     ts.OHLCVs
	Fast, Slow, SignalPeriod int
	LastSignal               Signal

	cacheOHLCV                        ts.OHLCV
	cacheMACD, cacheSignal, cacheHist []float64
}

func NewMACD(f, s, signalPeriod int) *MACD {
	return &MACD{
		Fast: f, Slow: s, SignalPeriod: signalPeriod,
		Data: make(ts.OHLCVs, 0, MaxLookback),
	}
}

func (m *MACD) AddTick(ohlcv ts.OHLCV) Signal {
	m.cacheOHLCV = ohlcv
	if len(m.Data) <= m.Slow {
		m.Data = append(m.Data, ohlcv)
		return NoSignal
	}
	if len(m.Data) > MaxLookback {
		m.Data = m.Data[1:]
	}
	m.Data = append(m.Data, ohlcv)
	m.cacheMACD, m.cacheSignal, m.cacheHist = m.Compute()
	return m.Signal()
}

func (m *MACD) Signal() Signal {
	x0, x := m.cacheHist[len(m.cacheHist)-2], m.cacheHist[len(m.cacheHist)-1]
	if x0 > 0 && x < 0 {
		m.LastSignal = Signal{
			Action:   Sell,
			Time:     time.Time(m.cacheOHLCV.Timestamp),
			Strength: 1,
		}
		return m.LastSignal
	} else if x0 < 0 && x > 0 {
		m.LastSignal = Signal{
			Action:   Buy,
			Time:     time.Time(m.cacheOHLCV.Timestamp),
			Strength: 1,
		}
		return m.LastSignal
	}
	return NoSignal
}

func (m *MACD) Compute() ([]float64, []float64, []float64) {
	return talib.Macd(m.Data.Close(), m.Fast, m.Slow, m.SignalPeriod)
}

func (m *MACD) Draw() error {
	chart.AddLines([]string{"macd", "signal"}, m.Data.ToXYer(m.cacheMACD, m.cacheSignal)...)
	return nil
}
