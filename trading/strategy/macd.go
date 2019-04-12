package strategy

import (
	"fmt"
	"github.com/markcheno/go-talib"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/trading"
	"github.com/rkjdid/gocx/ts"
	"time"
)

type MACDOpts struct {
	Fast, Slow, SignalPeriod int
	Timeframe                ts.Timeframe
}

func (opts MACDOpts) NewMACD() *MACD {
	return NewMACD(opts.Fast, opts.Slow, opts.SignalPeriod, opts.Timeframe)
}

func (opts MACDOpts) NewMACDCross() *MACDCross {
	return NewMACDCross(opts.Fast, opts.Slow, opts.SignalPeriod, opts.Timeframe)
}

func (opts MACDOpts) String() string {
	return fmt.Sprintf("%d, %d, %d", opts.Fast, opts.Slow, opts.SignalPeriod)
}

type MACD struct {
	MACDOpts
	Data                        ts.OHLCVs
	LastOHLCV                   ts.OHLCV
	OutMACD, OutSignal, OutHist []float64
}

func NewMACD(f, s, signalPeriod int, tf ts.Timeframe) *MACD {
	return &MACD{
		MACDOpts: MACDOpts{
			Fast:         f,
			Slow:         s,
			SignalPeriod: signalPeriod,
			Timeframe:    tf,
		},
		Data: make(ts.OHLCVs, 0, s),
	}
}

func (m *MACD) AddTick(x trading.Tick) {
	m.LastOHLCV = x.OHLCV
	m.Data = append(m.Data, x.OHLCV)
	if len(m.Data) <= m.Slow {
		return
		// this value of 5*m.Slow was chosen because it
		// it yields the same results on the same input data.
		// Really not sure what to think of it
	} else if len(m.Data) > 5*m.Slow {
		m.Data = m.Data[1:]
	}
	v, s, h := m.ComputeLast()
	m.OutMACD = append(m.OutMACD, v)
	m.OutSignal = append(m.OutSignal, s)
	m.OutHist = append(m.OutHist, h)
}

func (m *MACD) Compute() ([]float64, []float64, []float64) {
	return talib.Macd(m.Data.Close(), m.Fast, m.Slow, m.SignalPeriod)
}

func (m *MACD) ComputeLast() (float64, float64, float64) {
	a, b, c := m.Compute()
	if len(a) == 0 || len(b) == 0 || len(c) == 0 {
		panic("bad length")
	}
	return a[len(a)-1], b[len(b)-1], c[len(c)-1]
}

func (m *MACD) Draw() error {
	chart.AddLines([]string{"macd", "signal"}, m.Data.ToXYerSlice(m.OutMACD, m.OutSignal)...)
	return nil
}

type MACDCross struct {
	*MACD
	LastSignal Signal
}

func NewMACDCross(f, s, signal int, tf ts.Timeframe) *MACDCross {
	mc := MACDCross{
		NewMACD(f, s, signal, tf),
		NoSignal,
	}
	return &mc
}

func (m *MACDCross) AddTick(t trading.Tick) {
	m.MACD.AddTick(t)
	if len(m.OutHist) < 2 {
		return
	}
	x0, x := m.OutHist[len(m.OutHist)-2], m.OutHist[len(m.OutHist)-1]
	if x0 > 0 && x < 0 {
		m.LastSignal = Signal{
			Action:   Sell,
			Time:     time.Time(m.LastOHLCV.Timestamp),
			Strength: 1,
		}
	} else if x0 < 0 && x > 0 {
		m.LastSignal = Signal{
			Action:   Buy,
			Time:     time.Time(m.LastOHLCV.Timestamp),
			Strength: 1,
		}
	}
}

func (m *MACDCross) Signal() Signal {
	return m.LastSignal
}
