package trading

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rkjdid/gocx/ts"
	"github.com/rkjdid/gocx/util"
	"time"
)

type Balance struct {
	Total     float64
	Free      float64
	BTCEquiv  float64
	USDTEquiv float64
}

type Snapshot struct {
	Time      time.Time
	Account   string
	Balances  map[string]Balance
	BTCEquiv  float64
	USDTEquiv float64
}

func (s Snapshot) Digest() (hash string, data []byte, err error) {
	b, err := json.Marshal(s)
	return fmt.Sprintf("%s:%d", s.Account, s.Time.Unix()), b, err
}

func (s Snapshot) ZScore() float64 {
	return float64(s.Time.Unix())
}

func (s Snapshot) String() string {
	return fmt.Sprintf("%s on %s - BTC: %.4f, USD: %.2f",
		s.Account, s.Time.Format(util.DefaultTimeFormat), s.BTCEquiv, s.USDTEquiv)
}

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
