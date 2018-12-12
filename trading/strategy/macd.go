package strategy

import (
	"fmt"
	"github.com/markcheno/go-talib"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/ts"
	"time"
)

const MaxLookback = 16000

type MACDOpts struct {
	Fast, Slow, SignalPeriod int
}

func (opts MACDOpts) NewMACD() *MACD {
	return NewMACD(opts.Fast, opts.Slow, opts.SignalPeriod)
}

func (opts MACDOpts) NewMACDCross() *MACDCross {
	return NewMACDCross(opts.Fast, opts.Slow, opts.SignalPeriod)
}

func (opts MACDOpts) String() string {
	return fmt.Sprintf("%d, %d, %d", opts.Fast, opts.Slow, opts.SignalPeriod)
}

type MACD struct {
	MACDOpts
	Data       ts.OHLCVs
	LastSignal Signal

	cacheOHLCV                        ts.OHLCV
	cacheMACD, cacheSignal, cacheHist []float64
}

func NewMACD(f, s, signalPeriod int) *MACD {
	return &MACD{
		MACDOpts: MACDOpts{
			Fast:         f,
			Slow:         s,
			SignalPeriod: signalPeriod,
		},
		Data: make(ts.OHLCVs, 0, MaxLookback),
	}
}

func (m *MACD) AddTick(ohlcv ts.OHLCV) {
	m.cacheOHLCV = ohlcv
	if len(m.Data) <= m.Slow {
		m.Data = append(m.Data, ohlcv)
		return
	}
	if len(m.Data) > MaxLookback {
		m.Data = m.Data[1:]
	}
	m.Data = append(m.Data, ohlcv)
	m.cacheMACD, m.cacheSignal, m.cacheHist = m.Compute()
}

func (m *MACD) Compute() ([]float64, []float64, []float64) {
	return talib.Macd(m.Data.Close(), m.Fast, m.Slow, m.SignalPeriod)
}

func (m *MACD) Draw() error {
	chart.AddLines([]string{"macd", "signal"}, m.Data.ToXYerSlice(m.cacheMACD, m.cacheSignal)...)
	return nil
}

type MACDCross struct {
	*MACD
}

func NewMACDCross(f, s, signal int) *MACDCross {
	mc := MACDCross{
		NewMACD(f, s, signal),
	}
	return &mc
}

func (m *MACDCross) AddTick(ohlcv ts.OHLCV) {
	m.MACD.AddTick(ohlcv)
}

func (m *MACDCross) Signal() Signal {
	if len(m.cacheHist) < 2 {
		return NoSignal
	}
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
