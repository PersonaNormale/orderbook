package orderbook

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNoOrders            = errors.New("No orders available")
	ErrOrderNotFound       = errors.New("Order not found")
	ErrInvalidModification = errors.New("Invalid modification parameters")
	ErrInvalidOrder        = errors.New("Invalid order's values")
)

// OrderBook represents a collection of buy (bids) and sell (asks) orders.
type OrderBook struct {
	Tag  string `json:"Tag"`
	ID   string `json:"ID"`
	mu   sync.RWMutex
	asks []Order // Sell Orders ordered by increasing price
	bids []Order // Bids Orders ordered by decreasing price
}

// Trade represents a completed transaction between a buy and a sell order.
type Trade struct {
	BuyOrderID  string  `json:"buy_order_id"`
	SellOrderID string  `json:"sell_order_id"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
}

// OrderBookLevel represents an aggregated price level in the orderbook.
type OrderBookLevel struct {
	Price       float64
	TotalAmount float64
	OrderCount  int
}

// OrderBookSnapshot represents a snapshot of the orderbook at a specific time.
type OrderBookSnapshot struct {
	Asks []OrderBookLevel
	Bids []OrderBookLevel
	Time time.Time
}

// NewOrderBook creates and returns a new, empty orderbook.
func NewOrderBook(tag string) *OrderBook {
	return &OrderBook{
		Tag:  tag,
		ID:   uuid.New().String(),
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
func (ob *OrderBook) ModifyOrder(orderID string, newPrice float64, newAmount float64) error {
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

// PlaceOrder adds a new order to the orderbook.
// Orders are sorted by price: descending for bids and ascending for asks.
func (ob *OrderBook) PlaceOrder(order Order) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if order.Price <= 0 || order.Amount <= 0 {
		return ErrInvalidOrder
	}

	switch order.Side {
	case Buy:
		ob.bids = insertSorted(ob.bids, order, false) // decreasing price
	case Sell:
		ob.asks = insertSorted(ob.asks, order, true)
	}
	return nil
}

// ProcessOrder matches an incoming order against existing orders in the book.
// It creates trades for fully or partially matched orders. Any unmatched portion
// of the incoming order is added to the orderbook.
func (ob *OrderBook) ProcessOrder(order Order) ([]*Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var err error
	var trades []*Trade
	remainingAmount := order.Amount

	// Determine which side of the book to match against
	var matchingSide *[]Order
	switch order.Side {
	case Buy:
		matchingSide = &ob.asks // Match against asks (sell orders)
	case Sell:
		matchingSide = &ob.bids // Match against bids (buy orders)
	default:
		return nil, ErrInvalidOrder // Invalid order side, return empty trades
	}

	// Iterate through the matching side to find matches
	for len(*matchingSide) > 0 && remainingAmount > 0 {
		bestOrder := &(*matchingSide)[0] // Get the best order (first in the list)

		// Check if the prices match
		if !isPriceMatching(&order, bestOrder) {
			break // No more matches possible
		}

		// Calculate the amount to execute
		executedAmount := math.Min(remainingAmount, bestOrder.Amount)

		// Create a trade
		trade := createTrade(&order, bestOrder, executedAmount)
		trades = append(trades, trade)

		// Update remaining amounts
		remainingAmount -= executedAmount
		bestOrder.Amount -= executedAmount

		// Remove the best order if it's fully executed
		if bestOrder.Amount == 0 {
			*matchingSide = (*matchingSide)[1:] // Remove the first order
		}
	}

	// If there's any remaining amount, add it to the order book
	if remainingAmount > 0 {
		newOrder := Order{
			ID:     order.ID,
			Price:  order.Price,
			Amount: remainingAmount,
			Side:   order.Side,
		}

		ob.mu.Unlock()
		err = ob.PlaceOrder(newOrder)
		ob.mu.Lock()
	}

	return trades, err
}

// GetBestBid returns the highest bid order.
// Returns ErrNoOrders if no bids are available.
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

// Helper function to check if the price of two orders match.
func isPriceMatching(order *Order, matchOrder *Order) bool {
	switch order.Side {
	case Buy:
		return matchOrder.Price <= order.Price
	case Sell:
		return matchOrder.Price >= order.Price
	}
	return false
}

// Helper function to create a trade from two orders and the executed amount.
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

// Helper function to insert an order into a sorted slice.
func insertSorted(orders []Order, order Order, ascending bool) []Order {
	i := sort.Search(len(orders), func(i int) bool {
		if ascending {
			return orders[i].Price > order.Price
		}
		return orders[i].Price < order.Price
	})
	orders = append(orders, Order{}) // Add a dummy element to extend the slice
	copy(orders[i+1:], orders[i:])   // Shift elements to the right
	orders[i] = order                // Insert the new order
	return orders
}
