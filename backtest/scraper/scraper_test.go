package scraper

import (
	"github.com/rkjdid/gocx/ts"
	"testing"
	"time"
)

func TestFetchHistorical(t *testing.T) {
	t.SkipNow() // avoid pointless spam

	to, err := time.Parse("02/01/2006", "31/12/2017")
	if err != nil {
		t.Fatal(err)
	}

	for _, tf := range []string{ts.TfHour, ts.TfDay} {
		d, ok := ts.TfToDuration[tf]
		if !ok {
			t.Fatal("invalid timeframe")
		}

		for i := 1; i < 6; i++ {
			from := to.Add(-d * time.Duration(i) * 10)
			data, err := FetchHistorical("bitfinex", "BTC", "USD", tf, i, from, to)
			if err != nil {
				t.Error(err)
				continue
			}
			if !from.Equal(time.Time(data[0].Timestamp)) || !to.Equal(time.Time(data[len(data)-1].Timestamp)) {
				t.Errorf("dates differ for %s/%d\n\texpected: %s to %s\n\t     got: %s to %s",
					tf, i, from, to, data[0].Timestamp, data[len(data)-1].Timestamp)
			}
		}
	}
}
