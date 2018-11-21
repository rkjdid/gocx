package binance

import (
	"encoding/json"
	"fmt"
	"github.com/rkjdid/gocx"
	"github.com/rkjdid/gocx/scraper"
	"log"
	"net/http/httputil"
	"sort"
	"strings"
	"time"
)

const (
	API            = "https://binance.com"
	TickerEndpoint = "/api/v1/ticker/24hr"
)

type Ticker struct {
	Base, Quote        string
	Symbol             string  `json:"symbol"`
	PriceChange        string  `json:"priceChange"`
	PriceChangePercent string  `json:"priceChangePercent"`
	WeightedAvgPrice   string  `json:"weightedAvgPrice"`
	PrevClosePrice     string  `json:"prevClosePrice"`
	LastPrice          string  `json:"lastPrice"`
	LastQty            string  `json:"lastQty"`
	BidPrice           string  `json:"bidPrice"`
	BidQty             string  `json:"bidQty"`
	AskPrice           string  `json:"askPrice"`
	AskQty             string  `json:"askQty"`
	OpenPrice          string  `json:"openPrice"`
	HighPrice          string  `json:"highPrice"`
	LowPrice           string  `json:"lowPrice"`
	Volume             float64 `json:"volume,string"`
	QuoteVolume        float64 `json:"quoteVolume,string"`
	OpenTime           int64   `json:"openTime"`
	CloseTime          int64   `json:"closeTime"`
	FirstID            int     `json:"firstId"`
	LastID             int     `json:"lastId"`
	Count              int     `json:"count"`
}

func ParseSymbol(sym string) (base, quote string) {
	for _, q := range []string{"BTC", "USDT"} {
		if i0 := strings.Index(sym, q); i0 == 0 {
			return sym[0:len(q)], sym[len(q):]
		} else {
			sz := len(sym) - len(q)
			return sym[0:sz], sym[sz:]
		}
	}
	return "", ""
}

type TickersByVolumeDesc []Ticker

func (tvt TickersByVolumeDesc) Len() int {
	return len(tvt)
}

func (tvt TickersByVolumeDesc) Less(i, j int) bool {
	return tvt[i].QuoteVolume > tvt[j].QuoteVolume
}

func (tvt TickersByVolumeDesc) Swap(i, j int) {
	tvt[i], tvt[j] = tvt[j], tvt[i]
}

func FetchTopTickers(baseFilter, quoteFilter string) ([]Ticker, error) {
	client := scraper.CacheClient(time.Hour * 12)
	resp, err := client.Get(API + TickerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("couldn't retreive http data: %s", err)
	}
	if gocx.Debug {
		buf, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println(string(buf))
		}
	}
	var tickers []Ticker
	err = json.NewDecoder(resp.Body).Decode(&tickers)
	if err != nil {
		return nil, fmt.Errorf("couldn't decode body: %s", err)
	}
	if gocx.Debug {
		log.Printf("%d: %v", resp.StatusCode, tickers)
	}

	var filtered []Ticker
	for _, t := range tickers {
		t.Base, t.Quote = ParseSymbol(t.Symbol)
		if (baseFilter != "" && t.Base != baseFilter) || (quoteFilter != "" && t.Quote != quoteFilter) {
			continue
		}
		filtered = append(filtered, t)
	}
	sort.Sort(TickersByVolumeDesc(filtered))
	return filtered, nil
}
