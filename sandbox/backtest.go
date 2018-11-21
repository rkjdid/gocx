package main

import (
	"fmt"
	"github.com/rkjdid/gocx/scraper"
	"github.com/rkjdid/gocx/ts"
	"time"
)

type Historical struct {
	Data      ts.OHLCVs
	From      time.Time
	To        time.Time
	Timeframe time.Duration

	Exchange, Base, Quote string
}

func (h Historical) String() string {
	var hi string
	if h.Exchange != "" {
		hi += h.Exchange + ":"
	}
	return fmt.Sprintf("%s%s%s - tf:%s %6d elements from %s to %s",
		hi, h.Base, h.Quote, h.Timeframe, h.Data.Len(),
		tfrom.Format(tformatH), tto.Format(tformatH))
}

func LoadHistorical(x, bcur, qcur string, tf string, agg int, from, to time.Time) (*Historical, error) {
	data, err := scraper.FetchHistorical(x, bcur, qcur, tf, agg, from, to)
	if err != nil {
		return nil, err
	}
	// cleanup input data
	data = data.Trim().Clean()

	if len(data) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	h := Historical{
		Data:      data,
		To:        time.Time(data[0].Timestamp),
		From:      time.Time(data[len(data)-1].Timestamp),
		Timeframe: time.Duration(agg) * scraper.TfToDuration[tf],
		Exchange:  x, Base: bcur, Quote: qcur,
	}
	fmt.Println("loaded:", h)
	return &h, nil
}
