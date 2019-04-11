package trading

import (
	"encoding/json"
	"fmt"
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/gocx/util"
	"strings"
	"time"
)

type (
	Balance struct {
		Total     float64
		Free      float64
		BTCEquiv  float64
		USDTEquiv float64
	}
	Snapshot struct {
		Time      time.Time
		Account   string
		Balances  map[string]Balance
		BTCEquiv  float64
		USDTEquiv float64
	}
)

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

type (
	Snapshots     []Snapshot
	BTCSnapshots  Snapshots
	USDTSnapshots Snapshots
)

func (bs BTCSnapshots) Values() (val []float64) {
	val = make([]float64, len(bs))
	for i, v := range bs {
		val[i] = v.BTCEquiv
	}
	return val
}

func (bs BTCSnapshots) XY(i int) (float64, float64) {
	return float64(bs[i].Time.Unix()), bs[i].BTCEquiv
}

func (bs BTCSnapshots) Len() int { return len(bs) }

func (bs BTCSnapshots) Range() (float64, float64, float64, float64) {
	if sz := len(bs); sz > 0 {
		y0, _ := stats.Min(bs.Values())
		yN, _ := stats.Max(bs.Values())
		return float64(bs[0].Time.Unix()), float64(bs[sz-1].Time.Unix()), y0, yN
	}
	return 0, 0, 0, 0
}

func (us USDTSnapshots) Values() (val []float64) {
	val = make([]float64, len(us))
	for i, v := range us {
		val[i] = v.USDTEquiv
	}
	return val
}

func (us USDTSnapshots) XY(i int) (float64, float64) {
	return float64(us[i].Time.Unix()), us[i].USDTEquiv
}

func (us USDTSnapshots) Len() int { return len(us) }

func (us USDTSnapshots) Range() (float64, float64, float64, float64) {
	if sz := len(us); sz > 0 {
		y0, _ := stats.Min(us.Values())
		yN, _ := stats.Max(us.Values())
		return float64(us[0].Time.Unix()), float64(us[sz-1].Time.Unix()), y0, yN
	}
	return 0, 0, 0, 0
}

type (
	AssetHistory struct {
		Balances []Balance
		Time     []time.Time
	}
	AssetHistoryBTC  AssetHistory
	AssetHistoryUSDT AssetHistory
)

func ExtractAssetsHistory(snaps []Snapshot, assets ...string) (hist []AssetHistory) {
	hist = make([]AssetHistory, len(assets))
	for _, sn := range snaps {
		for i, asset := range assets {
			asset = strings.ToUpper(asset)
			if b, ok := sn.Balances[asset]; ok {
				hist[i].Balances = append(hist[i].Balances, b)
				hist[i].Time = append(hist[i].Time, sn.Time)
			}
		}
	}
	return hist
}

func ExtractAssetHistory(snaps []Snapshot, asset string) AssetHistory {
	return ExtractAssetsHistory(snaps, asset)[0]
}

func (s AssetHistory) XY(i int) (float64, float64) {
	return float64(s.Time[i].Unix()), s.Balances[i].Total
}

func (s AssetHistory) Len() int { return len(s.Balances) }

func (s AssetHistoryBTC) Values() (val []float64) {
	val = make([]float64, len(s.Balances))
	for i, v := range s.Balances {
		val[i] = v.BTCEquiv
	}
	return val
}

func (s AssetHistoryBTC) XY(i int) (float64, float64) {
	return float64(s.Time[i].Unix()), s.Balances[i].BTCEquiv
}

func (s AssetHistoryBTC) Len() int { return len(s.Balances) }

func (s AssetHistoryBTC) Range() (float64, float64, float64, float64) {
	if sz := len(s.Balances); sz > 0 {
		y0, _ := stats.Min(s.Values())
		yN, _ := stats.Max(s.Values())
		return float64(s.Time[0].Unix()), float64(s.Time[sz-1].Unix()), y0, yN
	}
	return 0, 0, 0, 0
}

func (s AssetHistoryUSDT) XY(i int) (float64, float64) {
	return float64(s.Time[i].Unix()), s.Balances[i].USDTEquiv
}

func (s AssetHistoryUSDT) Len() int { return len(s.Balances) }
