package trading

import "testing"

func TestTransaction_Cost(t *testing.T) {
	tx := Transaction{
		Price: 500, Quantity: 2,
	}
	if tx.Cost() != 1000 {
		t.Errorf("expecting cost 1000 got %f", tx.Cost())
	}
}

func TestPosition_Net(t *testing.T) {
	p := Position{
		Total:    10,
		Traded:   5,
		AvgEntry: 100,
		AvgExit:  500,
		FeesRate: 0,
	}
	p.Direction = Long
	if p.Net() != 2000 {
		t.Errorf("expecting 2000 gain for long position, got %f", p.Net())
	}
	p.Direction = Short
	if p.Net() != -2000 {
		t.Errorf("expecting 2000 gain for long position, got %f", p.Net())
	}
}

func TestPosition_AddTransaction(t *testing.T) {
	p := Position{
		Direction: Long,
		State:     Active,
		FeesRate:  DefaultFees,
	}

	p.PaperBuy(1, 4)
	if p.AvgEntry != 4 {
		t.Errorf("p.AvgEntry != 4")
	}
	p.PaperBuy(1, 2)
	if p.AvgEntry != 3 {
		t.Errorf("p.AvgEntry != 3")
	}

	p.PaperSell(1, 10)
	if p.AvgExit != 10 {
		t.Errorf("p.AvgEntry != 3")
	}
	if p.State == Closed {
		t.Errorf("position shouldn't be closed")
	}
	if p.Traded != 1 {
		t.Errorf("position should've traded 1")
	}

	p.PaperBuy(2, 5)
	if p.AvgEntry != 4 {
		t.Errorf("p.AvgEntry != 4")
	}
	if p.Total != 4 {
		t.Errorf("p.Total != 4")
	}

	p.PaperSell(3, 10)
	if p.State != Closed {
		t.Errorf("position should be closed")
	}
	if p.Traded != p.Total {
		t.Errorf("traded != total")
	}
	if p.Net() != 6*4-(4*p.AvgEntry*p.FeesRate+4*p.AvgExit*p.FeesRate) {
		t.Errorf("unexpected net worth, got %f", p.Net())
	}
}

func TestPosition_PaperClose(t *testing.T) {
	p := Position{Direction: Long}
	p.PaperBuy(10, 10)
	p.PaperClose(10)
}
