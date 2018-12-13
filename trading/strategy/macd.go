package strategy

import (
	"fmt"
	"github.com/markcheno/go-talib"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/ts"
	"time"
)

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

	LastOHLCV                   ts.OHLCV
	OutMACD, OutSignal, OutHist []float64
}

func NewMACD(f, s, signalPeriod int) *MACD {
	return &MACD{
		MACDOpts: MACDOpts{
			Fast:         f,
			Slow:         s,
			SignalPeriod: signalPeriod,
		},
		Data: make(ts.OHLCVs, 0, s),
	}
}

func (m *MACD) AddTick(ohlcv ts.OHLCV) {
	m.LastOHLCV = ohlcv
	m.Data = append(m.Data, ohlcv)
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
	if len(m.OutHist) < 2 {
		return NoSignal
	}
	x0, x := m.OutHist[len(m.OutHist)-2], m.OutHist[len(m.OutHist)-1]
	if x0 > 0 && x < 0 {
		m.LastSignal = Signal{
			Action:   Sell,
			Time:     time.Time(m.LastOHLCV.Timestamp),
			Strength: 1,
		}
		return m.LastSignal
	} else if x0 < 0 && x > 0 {
		m.LastSignal = Signal{
			Action:   Buy,
			Time:     time.Time(m.LastOHLCV.Timestamp),
			Strength: 1,
		}
		return m.LastSignal
	}
	return NoSignal
}
