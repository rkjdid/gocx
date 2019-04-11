package trading

import (
	"errors"
	"fmt"
	"github.com/rkjdid/gocx/ts"
	"time"
)



type Broker interface {
	MarketBuy(sym string, q float64) ([]*Transaction, error)
	MarketSell(sym string, q float64) ([]*Transaction, error)

	Snapshot() (*Snapshot, error)
	Name() string
	Symbol(base, quote string) string
}

type PaperTrading struct {
	FeesRate float64
	Time     time.Time
	Price    float64
}

func (p PaperTrading) MarketBuy(sym string, q float64) ([]*Transaction, error) {
	return []*Transaction{
		{
			Direction:  Buy,
			Quantity:   q,
			Price:      p.Price,
			Time:       p.Time,
			Commission: p.FeesRate * (q * p.Price),
		},
	}, nil
}

func (p PaperTrading) MarketSell(sym string, q float64) ([]*Transaction, error) {
	return []*Transaction{
		{
			Direction:  Sell,
			Quantity:   q,
			Price:      p.Price,
			Time:       p.Time,
			Commission: p.FeesRate * (q * p.Price),
		},
	}, nil
}

func (p PaperTrading) Symbol(base, quote string) string {
	return fmt.Sprintf("%s%s", base, quote)
}

func (p PaperTrading) Snapshot() (*Snapshot, error) {
	return nil, errors.New("not implemented for PaperTrading")
}

func (p PaperTrading) Name() string {
	return "paper"
}

func (p *PaperTrading) Update(o ts.OHLCV) {
	p.Time = o.Timestamp.T()
	p.Price = o.Close
}
