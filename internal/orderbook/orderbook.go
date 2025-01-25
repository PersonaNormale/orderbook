package orderbook

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"
)

var (
	ErrNoOrders            = errors.New("No orders available")
	ErrOrderNotFound       = errors.New("Order not found")
	ErrInvalidModification = errors.New("Invalid modification parameters")
)

type OrderBook struct {
	mu   sync.RWMutex
	asks []Order // Sell Orders ordered by increasing price
	bids []Order // Bids Orders ordered by decreasing price
}

type Trade struct {
	BuyOrderID  string  `json:"buy_order_id"`
	SellOrderID string  `json:"sell_order_id"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"Amount"`
}

type OrderBookLevel struct {
	Price       float64
	TotalAmount float64
	OrderCount  int
}

type OrderBookSnapshot struct {
	Asks []OrderBookLevel
	Bids []OrderBookLevel
	Time time.Time
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		asks: make([]Order, 0),
		bids: make([]Order, 0),
	}
}

// CancelOrder removes an order from the orderbook.
// Returns ErrOrderNotFound if the order doesn't exist.
func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Check bids
	for i, order := range ob.bids {
		if order.ID == orderID {
			ob.bids = append(ob.bids[:i], ob.bids[i+1:]...)
			return nil
		}
	}

	// Check asks
	for i, order := range ob.asks {
		if order.ID == orderID {
			ob.asks = append(ob.asks[:i], ob.asks[i+1:]...)
			return nil
		}
	}

	return ErrOrderNotFound
}

// ModifyOrder modifies an existing order in the book.
// If the price changes, the order is repositioned to maintain correct sorting.
// Returns ErrOrderNotFound if the order doesn't exist or ErrInvalidModification
// if the new values are invalid.
func (ob *OrderBook) ModifyOrder(orderID string, newPrice, newAmount float64) error {
	// Input validation
	if newPrice <= 0 || newAmount <= 0 {
		return ErrInvalidModification
	}

	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Look for the order in bids first
	for i, order := range ob.bids {
		if order.ID == orderID {
			// If only quantity changes, update in place
			if newPrice == order.Price {
				ob.bids[i].Amount = newAmount
				return nil
			}

			// If price changes, remove and reinsert the order
			order.Price = newPrice
			order.Amount = newAmount
			ob.bids = append(ob.bids[:i], ob.bids[i+1:]...)
			ob.bids = insertSorted(ob.bids, order, false) // false for descending order
			return nil
		}
	}

	// Look for the order in asks
	for i, order := range ob.asks {
		if order.ID == orderID {
			// If only quantity changes, update in place
			if newPrice == order.Price {
				ob.asks[i].Amount = newAmount
				return nil
			}

			// If price changes, remove and reinsert the order
			order.Price = newPrice
			order.Amount = newAmount
			ob.asks = append(ob.asks[:i], ob.asks[i+1:]...)
			ob.asks = insertSorted(ob.asks, order, true) // true for ascending order
			return nil
		}
	}

	return ErrOrderNotFound
}

func (ob *OrderBook) PlaceOrder(order Order) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	switch order.Side {
	case Buy:
		ob.bids = insertSorted(ob.bids, order, false) // decreasing price
	case Sell:
		ob.asks = insertSorted(ob.asks, order, true)
	}
	return nil
}

func (ob *OrderBook) ProcessOrder(order Order) []*Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var trades []*Trade

	// Determina quale lato del book usare per il matching
	var orders *[]Order
	switch order.Side {
	case Buy:
		orders = &ob.asks
	case Sell:
		orders = &ob.bids
	}

	remainingAmount := order.Amount

	// Finché ci sono ordini da matchare e quantità rimanente
	for len(*orders) > 0 && remainingAmount > 0 {
		bestOrder := &(*orders)[0]

		// Verifica se il prezzo è matchabile
		if !isPriceMatching(&order, bestOrder) {
			break
		}

		// Calcola la quantità da eseguire
		executedAmount := math.Min(remainingAmount, bestOrder.Amount)

		// Crea il trade
		trade := createTrade(&order, bestOrder, executedAmount)
		trades = append(trades, trade)

		// Aggiorna le quantità
		remainingAmount -= executedAmount
		bestOrder.Amount -= executedAmount

		// Se l'ordine è completamente eseguito, rimuovilo
		if bestOrder.Amount == 0 {
			*orders = (*orders)[1:]
		}
	}

	// Se c'è una quantità rimanente, aggiungi un nuovo ordine al book
	if remainingAmount > 0 {
		newOrder := Order{
			ID:     order.ID,
			Price:  order.Price,
			Amount: remainingAmount,
			Side:   order.Side,
		}

		switch order.Side {
		case Buy:
			ob.bids = insertSorted(ob.bids, newOrder, false) // ordine decrescente per i bid
		case Sell:
			ob.asks = insertSorted(ob.asks, newOrder, true) // ordine crescente per gli ask
		}
	}

	return trades
}

func (ob *OrderBook) GetBestBid() (Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.bids) == 0 {
		return Order{}, ErrNoOrders
	}

	return ob.bids[0], nil
}

// GetBestAsk returns the lowest ask order in the orderbook.
// Returns ErrNoOrders if there are no ask orders.
func (ob *OrderBook) GetBestAsk() (Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.asks) == 0 {
		return Order{}, ErrNoOrders
	}

	return ob.asks[0], nil
}

// GetOrderBookSnapshot returns the current state of the orderbook
// aggregated by price levels
func (ob *OrderBook) GetOrderBookSnapshot() OrderBookSnapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	snapshot := OrderBookSnapshot{
		Time: time.Now(),
	}

	// Aggregate asks by price level
	askLevels := make(map[float64]*OrderBookLevel)
	for _, order := range ob.asks {
		level, exists := askLevels[order.Price]
		if !exists {
			level = &OrderBookLevel{Price: order.Price}
			askLevels[order.Price] = level
		}
		level.TotalAmount += order.Amount
		level.OrderCount++
	}

	// Convert ask levels to sorted slice
	for _, level := range askLevels {
		snapshot.Asks = append(snapshot.Asks, *level)
	}
	// Sort asks by increasing price
	sort.Slice(snapshot.Asks, func(i, j int) bool {
		return snapshot.Asks[i].Price < snapshot.Asks[j].Price
	})

	// Aggregate bids by price level
	bidLevels := make(map[float64]*OrderBookLevel)
	for _, order := range ob.bids {
		level, exists := bidLevels[order.Price]
		if !exists {
			level = &OrderBookLevel{Price: order.Price}
			bidLevels[order.Price] = level
		}
		level.TotalAmount += order.Amount
		level.OrderCount++
	}

	// Convert bid levels to sorted slice
	for _, level := range bidLevels {
		snapshot.Bids = append(snapshot.Bids, *level)
	}
	// Sort bids by decreasing price
	sort.Slice(snapshot.Bids, func(i, j int) bool {
		return snapshot.Bids[i].Price > snapshot.Bids[j].Price
	})

	return snapshot
}

func isPriceMatching(order *Order, matchOrder *Order) bool {
	switch order.Side {
	case Buy:
		return matchOrder.Price <= order.Price
	case Sell:
		return matchOrder.Price >= order.Price
	}
	return false
}

func createTrade(order *Order, matchOrder *Order, executedAmount float64) *Trade {
	trade := &Trade{
		Price:  matchOrder.Price,
		Amount: executedAmount,
	}
	switch order.Side {
	case Buy:
		trade.BuyOrderID = order.ID
		trade.SellOrderID = matchOrder.ID
	case Sell:
		trade.BuyOrderID = matchOrder.ID
		trade.SellOrderID = order.ID
	}
	return trade
}

func removeOrder(orders []Order, index int) []Order {
	return append(orders[:index], orders[index+1:]...)
}

func insertSorted(orders []Order, order Order, ascending bool) []Order {
	var i int
	for i = 0; i < len(orders); i++ {
		if (ascending && orders[i].Price > order.Price) || (!ascending && orders[i].Price < order.Price) {
			break
		}
	}
	orders = append(orders[:i], append([]Order{order}, orders[i:]...)...)
	return orders
}
