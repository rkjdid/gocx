package backtest

import (
	"fmt"
	"github.com/rkjdid/gocx/scraper"
	"time"
)

type Timeframe struct {
	N    int
	Unit string
}

func (tf Timeframe) String() string {
	return fmt.Sprintf("%d%s", tf.N, tf.Unit)
}

func (tf Timeframe) IsValid() bool {
	_, ok := scraper.TfToDuration[tf.Unit]
	return ok
}

func (tf Timeframe) ToDuration() time.Duration {
	v := scraper.TfToDuration[tf.Unit]
	return v * time.Duration(tf.N)
}

func ParseTf(tf string) (Timeframe, error) {
	ttf := Timeframe{
		N: 1, // default
	}
	_, err := fmt.Sscanf(tf, "%d", &ttf.N)
	if err != nil {
		_, err2 := fmt.Sscanf(tf, "%s", &ttf.Unit)
		if err2 != nil {
			return ttf, err
		}
	} else {
		_, err = fmt.Sscanf(tf[len(fmt.Sprint(ttf.N)):], "%s", &ttf.Unit)
		if err != nil {
			return ttf, err
		}
	}
	switch ttf.Unit {
	case "h", "H":
		ttf.Unit = scraper.TfHour
	case "m", "M":
		ttf.Unit = scraper.TfMinute
	case "d", "D":
		ttf.Unit = scraper.TfDay
	}
	if !ttf.IsValid() {
		return ttf, fmt.Errorf("invalid duration unit: %s", ttf.Unit)
	}
	return ttf, nil
}

func DurationToTf(d time.Duration) (tf Timeframe, err error) {
	for _, unit := range []string{
		scraper.TfDay,
		scraper.TfHour,
		scraper.TfMinute,
	} {
		dunit := scraper.TfToDuration[unit]
		if d >= dunit {
			if d%dunit != 0 {
				return tf, fmt.Errorf("not multiple of %s", dunit)
			}
			return Timeframe{
				N:    int(d / dunit),
				Unit: unit,
			}, nil
		}
	}
	return tf, fmt.Errorf("timeframe too low, minimum is %s", time.Minute)
}
