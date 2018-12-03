package backtest

import (
	"fmt"
	"github.com/rkjdid/gocx/scraper"
	"github.com/rkjdid/gocx/ts"
	"time"
)

type Historical struct {
	CommonConfig
	Data ts.OHLCVs
}

func (h Historical) String() string {
	const tformatH = "02-01-2006 15:04"
	var hi string
	if h.Exchange != "" {
		hi += h.Exchange + ":"
	}
	return fmt.Sprintf("%s%s%s - tf:%s %6d elements from %s to %s",
		hi, h.Base, h.Quote, h.Timeframe, h.Data.Len(),
		h.From.Format(tformatH), h.To.Format(tformatH))
}

func LoadHistorical(x, bcur, qcur string, tf Timeframe, from, to time.Time) (*Historical, error) {
	data, err := scraper.FetchHistorical(x, bcur, qcur, tf.Unit, tf.N, from, to)
	if err != nil {
		return nil, err
	}
	// cleanup input data
	data = data.Trim().Clean()

	if len(data) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	h := Historical{
		CommonConfig: CommonConfig{
			To:        data.XNT(),
			From:      data.X0T(),
			Timeframe: tf,
			Exchange:  x,
			Base:      bcur, Quote: qcur,
		},
		Data: data,
	}
	fmt.Println("loaded:", h)
	return &h, nil
}
