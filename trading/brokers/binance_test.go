package brokers

import (
	"context"
	"flag"
	"testing"
)

var (
	testBinanceKey    = flag.String("testBinanceKey", "", "binance api key used in tests")
	testBinanceSecret = flag.String("testBinanceSecret", "", "binance api secret used in tests")
)

func TestBinancePing(t *testing.T) {
	err := NewBinanceBroker("", "a", "b").NewPingService().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestBinance_Snapshot(t *testing.T) {
	cl := NewBinanceBroker("test", *testBinanceKey, *testBinanceSecret)
	s, err := cl.Snapshot()
	if err != nil {
		t.Errorf("snapshot: %s", err)
	}
	t.Logf("%s", s)
}
