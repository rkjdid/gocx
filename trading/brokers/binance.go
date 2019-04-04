package brokers

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance"
	"github.com/rkjdid/gocx/trading"
	"github.com/rkjdid/gocx/util"
	"log"
	"strconv"
	"time"
)

type Binance struct {
	*binance.Client
	Account string
}

func NewBinanceBroker(account, key, secret string) *Binance {
	return &Binance{binance.NewClient(key, secret), account}
}

func (b Binance) MarketBuy(sym string, q float64) ([]*trading.Transaction, error) {
	resp, err := b.Client.NewCreateOrderService().
		Symbol(sym).
		Type(binance.OrderTypeMarket).
		Side(binance.SideTypeBuy).
		Quantity(fmt.Sprintf("%f", q)).Do(context.Background())
	if err != nil {
		return nil, err
	}
	var ts []*trading.Transaction
	for _, fill := range resp.Fills {
		p, err := strconv.ParseFloat(fill.Price, 64)
		if err != nil {
			log.Printf("bad price from binance response: %s", err)
			continue
		}
		q, err := strconv.ParseFloat(fill.Quantity, 64)
		if err != nil {
			log.Printf("bad quantity from binance response: %s", err)
			continue
		}
		fee, err := strconv.ParseFloat(fill.Commission, 64)
		if err != nil {
			log.Printf("bad commission from binance response: %s", err)
		}
		ts = append(ts, &trading.Transaction{
			Id:         int(resp.OrderID),
			Time:       util.UnixToTime(resp.TransactTime),
			Direction:  trading.Buy,
			Quantity:   q,
			Price:      p,
			Commission: fee,
		})
	}
	return ts, nil
}

func (b Binance) MarketSell(sym string, q float64) ([]*trading.Transaction, error) {
	resp, err := b.Client.NewCreateOrderService().
		Symbol(sym).
		Type(binance.OrderTypeMarket).
		Side(binance.SideTypeSell).
		Quantity(fmt.Sprintf("%f", q)).Do(context.Background())
	if err != nil {
		return nil, err
	}
	var ts []*trading.Transaction
	for _, fill := range resp.Fills {
		p, err := strconv.ParseFloat(fill.Price, 64)
		if err != nil {
			log.Printf("bad price from binance response: %s", err)
			continue
		}
		q, err := strconv.ParseFloat(fill.Quantity, 64)
		if err != nil {
			log.Printf("bad quantity from binance response: %s", err)
			continue
		}
		fee, err := strconv.ParseFloat(fill.Commission, 64)
		if err != nil {
			log.Printf("bad commission from binance response: %s", err)
		}
		ts = append(ts, &trading.Transaction{
			Id:         int(resp.OrderID),
			Time:       util.UnixToTime(resp.TransactTime),
			Direction:  trading.Sell,
			Quantity:   q,
			Price:      p,
			Commission: fee,
		})
	}
	return ts, nil
}

func (b Binance) Name() string {
	if prefix := "binance"; b.Account == "" {
		return prefix
	} else {
		return fmt.Sprintf("%s.%s", prefix, b.Account)
	}
}

func (b Binance) Symbol(base, quote string) string {
	return fmt.Sprintf("%s%s", base, quote)
}

func (b Binance) GetFees() (maker, taker float64, err error) {
	acc, err := b.NewGetAccountService().Do(context.Background())
	if err != nil {
		return 0, 0, fmt.Errorf("binance api: %s", err)
	}
	return float64(acc.MakerCommission), float64(acc.TakerCommission), nil
}

func (b Binance) BTCUSDTicker() (float64, error) {
	tname := "BTCUSDT"
	ticker, err := b.NewBookTickerService().Symbol(tname).Do(context.Background())
	if err != nil {
		return 0, fmt.Errorf("error getting binance ticker %s: %s", tname, err)
	}
	bid, _ := strconv.ParseFloat(ticker.BidPrice, 64)
	ask, _ := strconv.ParseFloat(ticker.AskPrice, 64)
	return (bid + ask) / 2, nil
}

func (b Binance) Snapshot() (*trading.Snapshot, error) {
	acc, err := b.NewGetAccountService().Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("binance api: %s", err)
	}
	s := &trading.Snapshot{
		Time:     time.Now(),
		Account:  b.Name(),
		Balances: map[string]trading.Balance{},
	}
	btcusdt, errBtcUsdt := b.BTCUSDTicker()
	if err != nil {
		log.Println("Snapshot:", err)
	}
	for _, asset := range acc.Balances {
		free, _ := strconv.ParseFloat(asset.Free, 64)
		locked, _ := strconv.ParseFloat(asset.Locked, 64)
		total := free + locked
		if total == 0 {
			continue
		}
		bal := trading.Balance{
			Total: total,
			Free:  free,
		}

		switch asset.Asset {
		case "BTC":
			bal.BTCEquiv = total
			if errBtcUsdt == nil {
				bal.USDTEquiv = total * btcusdt
			}
		case "USDT":
			bal.USDTEquiv = total
			if errBtcUsdt == nil {
				bal.BTCEquiv = total / btcusdt
			}
		default:
			tname := fmt.Sprintf("%sBTC", asset.Asset)
			ticker, err := b.NewBookTickerService().Symbol(tname).Do(context.Background())
			if err != nil {
				log.Printf("error getting binance ticker %s: %s", tname, err)
			} else {
				bid, _ := strconv.ParseFloat(ticker.BidPrice, 64)
				ask, _ := strconv.ParseFloat(ticker.AskPrice, 64)
				price := (bid + ask) / 2
				bal.BTCEquiv = total * price
				if errBtcUsdt == nil {
					bal.USDTEquiv = bal.BTCEquiv * btcusdt
				}
			}
		}
		// ignore dust
		if bal.BTCEquiv < 0.0001 {
			continue
		}
		s.BTCEquiv += bal.BTCEquiv
		s.USDTEquiv += bal.USDTEquiv
		s.Balances[asset.Asset] = bal
	}
	return s, nil
}
