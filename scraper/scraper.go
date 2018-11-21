package scraper

import (
	"encoding/json"
	"fmt"
	"github.com/ccxt/ccxt/go/util"
	"github.com/rkjdid/gocx"
	"github.com/rkjdid/gocx/ts"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var client = http.DefaultClient

const (
	CryptoCompareAPI = "https://min-api.cryptocompare.com/data/histo"
	TfMinute         = "minute"
	TfHour           = "hour"
	TfDay            = "day"
)

var TfToDuration = map[string]time.Duration{
	TfMinute: time.Minute,
	TfHour:   time.Hour,
	TfDay:    time.Hour * 24,
}

type CryptoCompareResponse struct {
	Response          string        `json:"Response"`
	Message           string        `json:"Message"`
	Type              int           `json:"Type"`
	Aggregated        bool          `json:"Aggregated"`
	Data              ts.OHLCVs     `json:"Data"`
	TimeTo            util.JSONTime `json:"TimeTo"`
	TimeFrom          util.JSONTime `json:"TimeFrom"`
	FirstValueInArray bool          `json:"FirstValueInArray"`
}

func (cc CryptoCompareResponse) String() string {
	s := fmt.Sprintf("%s (%d)", cc.Response, cc.Type)
	if cc.Message != "" {
		s += " " + cc.Message
	}
	if len(cc.Data) > 1 {
		s += fmt.Sprintf(" %d candles from %s to %s (interval: %s)",
			len(cc.Data), cc.TimeFrom, cc.TimeTo, time.Time(cc.Data[1].Timestamp).Sub(time.Time(cc.Data[0].Timestamp)))
	}
	return s
}

func FetchHistorical(exchange string, base, quote string, tf string, aggregate int, from, to time.Time,
) (data ts.OHLCVs, err error) {
	if aggregate < 1 {
		aggregate = 1
	}
	if from.After(to) {
		return nil, fmt.Errorf("from is after to date")
	}
	if to.After(time.Now()) {
		to = time.Now()
	}

	d, ok := TfToDuration[tf]
	if !ok {
		return nil, fmt.Errorf("timeframe \"%s\" invalid or not supported", tf)
	}
	d *= time.Duration(aggregate)
	u, _ := url.Parse(CryptoCompareAPI)
	u.Path += tf
	q := url.Values{}
	q.Set("fsym", base)
	q.Set("tsym", quote)
	if exchange != "" {
		q.Set("e", exchange)
	}

	q.Set("aggregate", fmt.Sprint(aggregate))
	i := 0
	for {
		var ccResp CryptoCompareResponse
		i++
		q.Set("toTs", fmt.Sprint(to.Unix()))
		//q.Set("limit", fmt.Sprintf("%d", to.Sub(from)/d+1))

		u.RawQuery = q.Encode()
		if gocx.Debug {
			log.Printf("GET %s", u.String())
		}
		resp, err := client.Get(u.String())
		if err != nil {
			return nil, fmt.Errorf("couldn't retreive http data: %s", err)
		}
		if gocx.Debug {
			buf, err := httputil.DumpResponse(resp, true)
			if err == nil {
				log.Println(string(buf))
			}
		}
		err = json.NewDecoder(resp.Body).Decode(&ccResp)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode body: %s", err)
		}
		if gocx.Debug {
			log.Printf("%d: %s", resp.StatusCode, ccResp)
		}
		if ccResp.Response != "Success" {
			return nil, fmt.Errorf("%d %s", ccResp.Type, ccResp.Message)
		}
		// stop querying if we only have 0 values
		if len(ts.OHLCVs(ccResp.Data).Trim()) == 0 {
			return data, nil
		}

		if time.Time(ccResp.TimeFrom).After(from) {
			// fetch more data from api after shifting "to" date
			to = time.Time(ccResp.TimeFrom)
			data = append(ccResp.Data, data...)
			continue
		} else {
			// find the closest candle before or equal to "from"
			if time.Time(ccResp.TimeFrom).Equal(from) {
				return append(ccResp.Data, data...), nil
			}
			var i int
			for ; i < len(ccResp.Data) && from.After(time.Time(ccResp.Data[i].Timestamp)); i++ {
			}
			if i == len(ccResp.Data) {
				return data, nil
			}
			if i > 0 && !from.Equal(time.Time(ccResp.Data[i].Timestamp)) {
				// no alignment between timeframe & from date, take previous candle
				i -= 1
			}
			return append(ccResp.Data[i:], data...), nil
		}
	}
}
