package ts

import (
	"fmt"
	"time"
)

const (
	TfMinute = "minute"
	TfHour   = "hour"
	TfDay    = "day"
)

var TfToDuration = map[string]time.Duration{
	TfMinute: time.Minute,
	TfHour:   time.Hour,
	TfDay:    time.Hour * 24,
}

type Timeframe struct {
	N    int
	Unit string
}

func (tf Timeframe) String() string {
	return fmt.Sprintf("%d%s", tf.N, tf.Unit)
}

func (tf Timeframe) IsValid() bool {
	_, ok := TfToDuration[tf.Unit]
	return ok
}

func (tf Timeframe) ToDuration() time.Duration {
	v := TfToDuration[tf.Unit]
	return v * time.Duration(tf.N)
}

func (tf Timeframe) Diff(tf2 Timeframe) time.Duration {
	return tf.ToDuration() - tf2.ToDuration()
}

func (tf Timeframe) Equals(tf2 Timeframe) bool {
	return tf.N == tf2.N && tf.Unit == tf2.Unit
}

func (tf Timeframe) Gt(tf2 Timeframe) bool {
	return tf.Diff(tf2) > 0
}

func (tf Timeframe) Lt(tf2 Timeframe) bool {
	return tf.Diff(tf2) < 0
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
		ttf.Unit = TfHour
	case "m", "M":
		ttf.Unit = TfMinute
	case "d", "D":
		ttf.Unit = TfDay
	}
	if !ttf.IsValid() {
		return ttf, fmt.Errorf("invalid duration unit: %s", ttf.Unit)
	}
	return ttf, nil
}

func DurationToTf(d time.Duration) (tf Timeframe, err error) {
	for _, unit := range []string{
		TfDay,
		TfHour,
		TfMinute,
	} {
		dunit := TfToDuration[unit]
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
