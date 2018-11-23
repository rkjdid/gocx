package risk

const (
	BTC = "BTC"
	USD = "USD"
)

type Portfolio struct {
	Assets    map[string]*Asset
	Positions map[int]*Position
}

type Asset struct {
	Symbol   string
	Quantity float64

	// Repartition contains all the physical locations of asset indexed by w.Location
	Wallets map[string]Wallet
}

type Wallet struct {
	Name     string
	Quantity float64

	// Address could be:
	//   - a physical wallet address
	//   - position<ID> if asset is actively used
	//   - exchangeName:wallet
	//   - ..
	Address string
}

func NewPortfolio() *Portfolio {
	return &Portfolio{
		Assets:    make(map[string]*Asset),
		Positions: make(map[int]*Position),
	}
}
