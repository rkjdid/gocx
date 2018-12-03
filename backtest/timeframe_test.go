package backtest

import (
	"github.com/rkjdid/gocx/scraper"
	"testing"
	"time"
)

func TestParseTf(t *testing.T) {
	tf, err := ParseTf("4h")
	if err != nil {
		t.Error(err)
	}
	if tf.Unit != scraper.TfHour || tf.N != 4 {
		t.Errorf("unexpected values 4h")
	}
	tf, err = ParseTf("m")
	if err != nil {
		t.Error(err)
	}
	if tf.Unit != scraper.TfMinute || tf.N != 1 {
		t.Errorf("unexpected values m")
	}
}

func TestTimeframe_IsValid(t *testing.T) {
	tf := Timeframe{
		N:    2,
		Unit: scraper.TfHour,
	}
	if !tf.IsValid() {
		t.Errorf("h4 should be valid")
	}
	tf.Unit = "fail"
	if tf.IsValid() {
		t.Error("tf shouldnt be valid")
	}
}

func TestDurationToTf(t *testing.T) {
	tf, err := DurationToTf(time.Second)
	if err == nil {
		t.Errorf("expected error for < 1m")
	}
	tf, err = DurationToTf(time.Minute + time.Second)
	if err == nil {
		t.Errorf("expected error for not modulo minute")
	}
	tf, err = DurationToTf(time.Hour * 48)
	if err != nil {
		t.Errorf("2 day duration should be valid")
	}
	if tf.Unit != scraper.TfDay || tf.N != 2 {
		t.Errorf("unexpected value for 48h")
	}
}
