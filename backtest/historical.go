package backtest

import (
	"fmt"
	"github.com/rkjdid/gocx/db"
	"github.com/rkjdid/gocx/scraper"
	"github.com/rkjdid/gocx/ts"
	"time"
)

type Source struct {
	Exchange    string
	Base, Quote string
	From, To    time.Time
	Timeframe   Timeframe
}

func (s Source) String() string {
	return fmt.Sprintf("%8s - %s - %s to %s", fmt.Sprint(s.Base, s.Quote), s.Timeframe,
		s.From.Format("02/01/06"), s.To.Format("02/01/2006"))
}

type Historical struct {
	Source
	Data ts.OHLCVs
}

func LoadHistorical(db *db.RedisDriver, x, bcur, qcur string, tf Timeframe, from, to time.Time) (*Historical, error) {
	meta := Source{
		Exchange: x, Base: bcur, Quote: qcur, Timeframe: tf, From: from, To: to,
	}
	h := Historical{
		Source: meta,
	}
	err := h.Load(db)
	//log.Printf("loaded: %s", h)
	return &h, err
}

func (h *Historical) Load(db *db.RedisDriver) error {
	hash, _, err := h.Digest()
	if err != nil {
		return fmt.Errorf("h.digest error: %s", err)
	}
	if len(hash) > 0 {
		err := db.LoadJSON(hash, h)
		if err == nil {
			//log.Printf("loaded %s", hash)
			return nil
		}
	}

	h.Data, err = scraper.FetchHistorical(
		h.Exchange, h.Base, h.Quote, h.Timeframe.Unit, h.Timeframe.N, h.From, h.To)
	if err != nil {
		return fmt.Errorf("scraper: %s", err)
	}
	// cleanup input data
	h.Data = h.Data.Trim().Clean()
	if len(h.Data) == 0 {
		return fmt.Errorf("no data available")
	}

	// fix from/to values
	h.From, h.To = h.Data.X0T(), h.Data.XNT()

	// reencode complete data
	_, data, err := h.Digest()
	if err != nil {
		return fmt.Errorf("h.Digest error: %s", err)
	}

	// manual save at previously calculated hash (input params)
	err = db.SET(hash, data)
	if err != nil {
		return fmt.Errorf("db.SET: %s", err)
	}
	//log.Printf("cached %s", hash)
	return nil
}

func (h *Historical) Digest() (hash string, data []byte, err error) {
	hash, _, err = db.JSONDigest("cache", h.Source)
	if err != nil {
		return
	}
	if len(h.Data) == 0 {
		return
	}
	_, data, err = db.JSONDigest("", h)
	return
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
