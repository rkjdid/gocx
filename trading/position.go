package trading

import (
	"fmt"
	"github.com/rkjdid/gocx/ts"
	"github.com/rkjdid/gocx/util"
	"log"
	"time"
)

type Direction bool

const (
	Short = Direction(false)
	Long  = Direction(true)
	Sell  = Short
	Buy   = Long

	DefaultFees = 0.0015
)

type State string

const (
	New    = State("new")
	Active = State("active")
	Closed = State("closed")
)

type Transaction struct {
	Id         int
	Time       time.Time
	Direction  Direction
	Quantity   float64
	Price      float64
	Commission float64
}

func (t Transaction) Cost() float64 {
	return t.Price * t.Quantity
}

type Position struct {
	State       State
	Base, Quote string
	Direction   Direction

	FeesRate  float64
	TotalFees float64
	Total     float64
	Traded    float64
	AvgEntry  float64
	AvgExit   float64
	OpenTime  time.Time
	CloseTime time.Time

	Broker       Broker
	Transactions []*Transaction

	tick ts.OHLCV
}

func NewPosition(b Broker, base, quote string, direction Direction) *Position {
	return &Position{
		Broker: b, Base: base, Quote: quote, Direction: direction, FeesRate: DefaultFees,
	}
}

func (p Position) String() string {
	format := "%.2f"
	if p.AvgEntry < 1e-2 {
		format = "%.1e"
	}

	return fmt.Sprintf("  %s @ "+format+" -> %s @ "+format+": %10.2f",
		p.OpenTime.Format(util.DefaultTimeFormat), p.AvgEntry,
		p.CloseTime.Format(util.DefaultTimeFormat), p.AvgExit,
		p.Net())
}

func (p Position) Active() bool {
	return p.State == Active
}

func (p Position) Cost() float64 {
	return p.Total * p.AvgEntry
}

func (p Position) Net() float64 {
	if p.AvgEntry == 0 || p.AvgExit == 0 {
		return 0
	}
	net := p.Traded*p.AvgExit - p.Traded*p.AvgEntry
	net -= p.TotalFees
	if p.Direction == Long {
		return net
	} else {
		return -net
	}
}

func (p Position) NetRatio() float64 {
	if p.AvgEntry == 0 || p.AvgExit == 0 {
		return 0
	}
	netRatio := p.AvgExit / p.AvgEntry
	if p.Direction == Long {
		return netRatio
	} else {
		return -netRatio
	}
}

func (p *Position) AddTransactions(ts ...*Transaction) {
	if len(ts) == 0 {
		return
	}
	if len(p.Transactions) == 0 {
		p.OpenTime = ts[0].Time
	}
	for _, t := range ts {
		p.TotalFees += t.Commission
		p.Transactions = append(p.Transactions, t)
		if p.Direction == t.Direction {
			p.AvgEntry = ((p.AvgEntry * p.Total) + (t.Price * t.Quantity)) /
				(p.Total + t.Quantity)
			p.Total += t.Quantity
			p.State = Active
		} else {
			p.AvgExit = ((p.AvgExit * p.Traded) + (t.Price * t.Quantity)) /
				(p.Traded + t.Quantity)
			p.Traded += t.Quantity
			if p.Traded >= p.Total {
				p.State = Closed
				p.CloseTime = t.Time
			}
		}
	}
}

func (p *Position) SetTick(o ts.OHLCV) {
	p.tick = o
	if pb, ok := p.Broker.(*PaperTrading); ok {
		pb.Update(o)
	}
}

func (p *Position) marketOrder(fn func(string, float64) ([]*Transaction, error), q float64, direction string) error {
	var ts []*Transaction
	var err error
	for i := 0; i < 3; i++ {
		ts, err = fn(p.Broker.Symbol(p.Base, p.Quote), q)
		if err != nil {
			log.Printf("marketOrder.%s: %s", direction, err)
			time.Sleep(time.Second * 2)
			continue
		}
	}
	if err != nil {
		return err
	}
	p.AddTransactions(ts...)
	return nil
}

func (p *Position) MarketBuy(q float64) error {
	return p.marketOrder(p.Broker.MarketBuy, q, "Buy")
}

func (p *Position) MarketSell(q float64) error {
	return p.marketOrder(p.Broker.MarketSell, q, "Sell")
}

func (p *Position) Close() error {
	fn := p.MarketBuy
	if p.Direction == Long {
		fn = p.MarketSell
	}
	return fn(p.Total - p.Traded)
}

// NetOnClose will PaperClose and return Net on a copy of p, caller Position is unchanged.
func (p Position) NetOnClose() float64 {
	if _, ok := p.Broker.(*PaperTrading); !ok {
		pt := &PaperTrading{
			FeesRate: p.FeesRate,
		}
		pt.Update(p.tick)
		p.Broker = pt
	}
	// PaperTrading broker does not error
	_ = p.Close()
	return p.Net()
}
