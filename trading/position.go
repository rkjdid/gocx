package trading

import (
	"fmt"
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
	Id        int
	Time      time.Time
	Direction Direction
	Quantity  float64
	Price     float64
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

	Transactions []*Transaction
}

func NewPosition(t time.Time, base, quote string, direction Direction) *Position {
	return &Position{
		Base: base, Quote: quote, Direction: direction, FeesRate: DefaultFees, OpenTime: t,
	}
}

func (p Position) String() string {
	format := "%.2f"
	if p.AvgEntry < 1e-2 {
		format = "%.1e"
	}

	return fmt.Sprintf("  %s @ "+format+" -> %s @ "+format+": %10.2f",
		p.OpenTime.Format("2006-01-02 15:04 -0700"), p.AvgEntry,
		p.CloseTime.Format("2006-01-02 15:04 -0700"), p.AvgExit,
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

// NetOnClose will PaperClose and return Net on a copy of p, caller Position is unchanged.
func (p Position) NetOnClose(pr float64) float64 {
	p.PaperClose(pr)
	return p.Net()
}

func (p *Position) AddTransaction(t *Transaction) {
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

func (p *Position) PaperBuy(q, pr float64) {
	p.PaperBuyAt(q, pr, time.Now())
}

func (p *Position) PaperBuyAt(q, pr float64, t time.Time) {
	p.TotalFees += p.FeesRate * (q * pr)
	p.AddTransaction(&Transaction{
		Direction: Buy,
		Quantity:  q,
		Price:     pr,
		Time:      t,
	})
}

func (p *Position) PaperSell(q, pr float64) {
	p.PaperSellAt(q, pr, time.Now())
}

func (p *Position) PaperSellAt(q, pr float64, t time.Time) {
	p.TotalFees += p.FeesRate * (q * pr)
	p.AddTransaction(&Transaction{
		Direction: Sell,
		Quantity:  q,
		Price:     pr,
		Time:      t,
	})
}

func (p *Position) PaperClose(pr float64) {
	p.PaperCloseAt(pr, time.Now())
}

func (p *Position) PaperCloseAt(pr float64, t time.Time) {
	fn := p.PaperBuyAt
	if p.Direction == Long {
		fn = p.PaperSellAt
	}
	fn(p.Total-p.Traded, pr, t)
}
