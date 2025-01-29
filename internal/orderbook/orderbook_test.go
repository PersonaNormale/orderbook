package orderbook

import (
	"testing"
	"time"
)

func TestNewOrder(t *testing.T) {
	tests := []struct {
		name      string
		price     float64
		amount    float64
		side      Side
		expectErr bool
	}{
		{"Valid Buy Order", 100.0, 10.0, Buy, false},
		{"Negative Price", -100.0, 10.0, Buy, true},
		{"Zero Quantity", 100.0, 0.0, Buy, true},
		{"Negative Quantity", 100.0, -10.0, Buy, true},
	}

	assertOrderFields := func(t *testing.T, got *Order, price float64, amount float64, side Side) {
		if got.Price != price {
			t.Errorf("Expected price %v, got %v", price, got.Price)
		}
		if got.Amount != amount {
			t.Errorf("Expected amount %v, got %v", amount, got.Amount)
		}
		if got.Side != side {
			t.Errorf("Expected side %v, got %v", side, got.Side)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := NewOrder(tt.price, tt.amount, tt.side)
			if (err != nil) != tt.expectErr {
				t.Errorf("NewOrder() error = %v, expected error = %v", err, tt.expectErr)
				return
			}
			if err == nil {
				assertOrderFields(t, order, tt.price, tt.amount, tt.side)
			}
		})
	}
}

func TestCancelOrder(t *testing.T) {
	tests := []struct {
		name            string
		ordersToAdd     []Order
		orderIDToCancel string
		expectedError   bool
		checkBookFunc   func(*testing.T, *OrderBook)
	}{
		{
			name:            "Cancel non-existent order",
			ordersToAdd:     []Order{},
			orderIDToCancel: "non-existent",
			expectedError:   true,
			checkBookFunc:   nil,
		},
		{
			name: "Cancel bid order successfully",
			ordersToAdd: []Order{
				{ID: "bid-1", Price: 100.0, Amount: 1.0, Side: Buy},
				{ID: "bid-2", Price: 101.0, Amount: 2.0, Side: Buy},
			},
			orderIDToCancel: "bid-1",
			expectedError:   false,
			checkBookFunc: func(t *testing.T, ob *OrderBook) {
				if len(ob.bids) != 1 {
					t.Errorf("Expected 1 bid order, got %d", len(ob.bids))
				}
				if ob.bids[0].ID != "bid-2" {
					t.Errorf("Expected remaining order bid-2, got %s", ob.bids[0].ID)
				}
			},
		},
		{
			name: "Cancel ask order successfully",
			ordersToAdd: []Order{
				{ID: "ask-1", Price: 100.0, Amount: 1.0, Side: Sell},
				{ID: "ask-2", Price: 101.0, Amount: 2.0, Side: Sell},
			},
			orderIDToCancel: "ask-1",
			expectedError:   false,
			checkBookFunc: func(t *testing.T, ob *OrderBook) {
				if len(ob.asks) != 1 {
					t.Errorf("Expected 1 ask order, got %d", len(ob.asks))
				}
				if ob.asks[0].ID != "ask-2" {
					t.Errorf("Expected remaining order ask-2, got %s", ob.asks[0].ID)
				}
			},
		},
		{
			name: "Cancel last order in book",
			ordersToAdd: []Order{
				{ID: "ask-1", Price: 100.0, Amount: 1.0, Side: Sell},
			},
			orderIDToCancel: "ask-1",
			expectedError:   false,
			checkBookFunc: func(t *testing.T, ob *OrderBook) {
				if len(ob.asks) != 0 {
					t.Errorf("Expected empty asks, got %d orders", len(ob.asks))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")

			// Place test orders
			for _, order := range tt.ordersToAdd {
				err := ob.PlaceOrder(order)
				if err != nil {
					t.Fatalf("Failed to place order: %v", err)
				}
			}

			// Try to cancel order
			err := ob.CancelOrder(tt.orderIDToCancel)

			// Check error
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got error: %v", tt.expectedError, err != nil)
			}

			// Run additional book checks if provided
			if tt.checkBookFunc != nil {
				tt.checkBookFunc(t, ob)
			}
		})
	}
}

func TestModifyOrder(t *testing.T) {
	tests := []struct {
		name          string
		ordersToAdd   []Order
		orderToModify string
		newPrice      float64
		newAmount     float64
		expectedError bool
		checkBookFunc func(*testing.T, *OrderBook)
	}{
		{
			name:          "Modify non-existent order",
			ordersToAdd:   []Order{},
			orderToModify: "non-existent",
			newPrice:      100.0,
			newAmount:     1.0,
			expectedError: true,
			checkBookFunc: nil,
		},
		{
			name: "Modify bid order price",
			ordersToAdd: []Order{
				{ID: "bid-1", Price: 100.0, Amount: 1.0, Side: Buy},
				{ID: "bid-2", Price: 101.0, Amount: 2.0, Side: Buy},
			},
			orderToModify: "bid-1",
			newPrice:      102.0,
			newAmount:     1.0,
			expectedError: false,
			checkBookFunc: func(t *testing.T, ob *OrderBook) {
				if len(ob.bids) != 2 {
					t.Errorf("Expected 2 bid orders, got %d", len(ob.bids))
				}
				// Should be first due to higher price
				if ob.bids[0].ID != "bid-1" || ob.bids[0].Price != 102.0 {
					t.Errorf("Expected modified order at top with price 102.0, got order %s with price %f",
						ob.bids[0].ID, ob.bids[0].Price)
				}
			},
		},
		{
			name: "Modify ask order amount",
			ordersToAdd: []Order{
				{ID: "ask-1", Price: 100.0, Amount: 1.0, Side: Sell},
				{ID: "ask-2", Price: 101.0, Amount: 2.0, Side: Sell},
			},
			orderToModify: "ask-1",
			newPrice:      100.0,
			newAmount:     3.0,
			expectedError: false,
			checkBookFunc: func(t *testing.T, ob *OrderBook) {
				if len(ob.asks) != 2 {
					t.Errorf("Expected 2 ask orders, got %d", len(ob.asks))
				}
				// Should maintain position due to same price
				if ob.asks[0].ID != "ask-1" || ob.asks[0].Amount != 3.0 {
					t.Errorf("Expected modified order with amount 3.0, got amount %f",
						ob.asks[0].Amount)
				}
			},
		},
		{
			name: "Invalid modification - zero price",
			ordersToAdd: []Order{
				{ID: "bid-1", Price: 100.0, Amount: 1.0, Side: Buy},
			},
			orderToModify: "bid-1",
			newPrice:      0.0,
			newAmount:     1.0,
			expectedError: true,
			checkBookFunc: func(t *testing.T, ob *OrderBook) {
				// Order should remain unchanged
				if ob.bids[0].Price != 100.0 {
					t.Errorf("Expected order price to remain 100.0, got %f", ob.bids[0].Price)
				}
			},
		},
		{
			name: "Invalid modification - zero amount",
			ordersToAdd: []Order{
				{ID: "ask-1", Price: 100.0, Amount: 1.0, Side: Sell},
			},
			orderToModify: "ask-1",
			newPrice:      100.0,
			newAmount:     0.0,
			expectedError: true,
			checkBookFunc: func(t *testing.T, ob *OrderBook) {
				// Order should remain unchanged
				if ob.asks[0].Amount != 1.0 {
					t.Errorf("Expected order amount to remain 1.0, got %f", ob.asks[0].Amount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")

			// Place test orders
			for _, order := range tt.ordersToAdd {
				err := ob.PlaceOrder(order)
				if err != nil {
					t.Fatalf("Failed to place order: %v", err)
				}
			}

			// Try to modify order
			err := ob.ModifyOrder(tt.orderToModify, tt.newPrice, tt.newAmount)

			// Check error
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got error: %v", tt.expectedError, err != nil)
			}

			// Run additional book checks if provided
			if tt.checkBookFunc != nil {
				tt.checkBookFunc(t, ob)
			}
		})
	}
}

func TestPlaceOrder(t *testing.T) {
	tests := []struct {
		name              string
		orders            []Order
		expectedBidsOrder []float64
		expectedAsksOrder []float64
	}{
		{"Increasing Order Bids",
			[]Order{
				{Price: 100.0, Amount: 1.0, Side: Buy},
				{Price: 102.0, Amount: 1.0, Side: Buy},
				{Price: 101.0, Amount: 1.0, Side: Buy},
			},
			[]float64{102.0, 101.0, 100.0},
			[]float64{},
		},
		{"Decreasing Order Bids",
			[]Order{
				{Price: 100.0, Amount: 1.0, Side: Sell},
				{Price: 102.0, Amount: 1.0, Side: Sell},
				{Price: 101.0, Amount: 1.0, Side: Sell},
			},
			[]float64{},
			[]float64{100.0, 101.0, 102.0},
		},
	}

	assertPriceOrder := func(t *testing.T, got []Order, expected []float64, orderType string) {
		if len(got) != len(expected) {
			t.Errorf("Expected %d %s orders, got %d", len(expected), orderType, len(got))
			return
		}
		for i, price := range expected {
			if got[i].Price != price {
				t.Errorf("%s order at position %d: expected price %v, got %v", orderType, i, price, got[i].Price)
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")
			for _, order := range tt.orders {
				if err := ob.PlaceOrder(order); err != nil {
					t.Fatalf("Failed to place order: %v", err)
				}
			}
			assertPriceOrder(t, ob.bids, tt.expectedBidsOrder, "BID")
			assertPriceOrder(t, ob.asks, tt.expectedAsksOrder, "ASK")
		})
	}
}

func TestProcessOrder_CompleteMatch(t *testing.T) {
	tests := []struct {
		name          string
		existingOrder Order
		newOrder      Order
		expectedTrade *Trade
	}{
		{
			"Complete Match: Buy and Sell",
			Order{
				Price:  100.0,
				Amount: 1.0,
				Side:   Sell,
				ID:     "sell-1",
			},
			Order{
				Price:  100.0,
				Amount: 1.0,
				Side:   Buy,
				ID:     "buy-1",
			},
			&Trade{
				BuyOrderID:  "buy-1",
				SellOrderID: "sell-1",
				Price:       100.0,
				Amount:      1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")
			if err := ob.PlaceOrder(tt.existingOrder); err != nil {
				t.Fatalf("Failed to place existing order: %v", err)
			}

			trades, err := ob.ProcessOrder(tt.newOrder)
			if err != nil {
				t.Fatalf("Failed to process order: %v", err)
			}
			assertTradeCount(t, trades, 1)
			assertTradeDetails(t, trades[0], tt.expectedTrade)
			assertEmptyOrderBook(t, ob)
		})
	}
}

func TestProcessOrder_PartialMatch(t *testing.T) {
	tests := []struct {
		name           string
		existingOrder  Order
		newOrder       Order
		expectedTrade  *Trade
		remainingOrder Order
	}{
		{
			"Partial March: Buy greater than Sell",
			Order{
				Price:  100.0,
				Amount: 1.0,
				Side:   Sell,
				ID:     "sell-1",
			},
			Order{
				Price:  100.0,
				Amount: 2.0,
				Side:   Buy,
				ID:     "buy-1",
			},
			&Trade{
				BuyOrderID:  "buy-1",
				SellOrderID: "sell-1",
				Price:       100.0,
				Amount:      1.0,
			},
			Order{
				Price:  100.0,
				Amount: 1.0,
				Side:   Buy,
				ID:     "buy-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")
			if err := ob.PlaceOrder(tt.existingOrder); err != nil {
				t.Fatalf("Failed to place existing order: %v", err)
			}

			trades, err := ob.ProcessOrder(tt.newOrder)
			if err != nil {
				t.Fatalf("Failed to process order: %v", err)
			}
			assertTradeCount(t, trades, 1)
			assertTradeDetails(t, trades[0], tt.expectedTrade)
			assertRemainingOrder(t, ob, tt.remainingOrder)
		})
	}
}

func TestProcessOrder_PriceTimePriority(t *testing.T) {
	tests := []struct {
		name           string
		existingOrders []Order
		newOrder       Order
		expectedTrades []*Trade
	}{
		{
			name: "Buy order matches best ask first",
			existingOrders: []Order{
				{ID: "sell-1", Price: 102.0, Amount: 1.0, Side: Sell},
				{ID: "sell-2", Price: 100.0, Amount: 1.0, Side: Sell}, // Should match first
				{ID: "sell-3", Price: 101.0, Amount: 1.0, Side: Sell},
			},
			newOrder: Order{
				ID:     "buy-1",
				Price:  102.0,
				Amount: 1.0,
				Side:   Buy,
			},
			expectedTrades: []*Trade{
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-2",
					Price:       100.0,
					Amount:      1.0,
				},
			},
		},
		{
			name: "Sell order matches best bid first",
			existingOrders: []Order{
				{ID: "buy-1", Price: 98.0, Amount: 1.0, Side: Buy},
				{ID: "buy-2", Price: 100.0, Amount: 1.0, Side: Buy}, // Should match first
				{ID: "buy-3", Price: 99.0, Amount: 1.0, Side: Buy},
			},
			newOrder: Order{
				ID:     "sell-1",
				Price:  98.0,
				Amount: 1.0,
				Side:   Sell,
			},
			expectedTrades: []*Trade{
				{
					BuyOrderID:  "buy-2",
					SellOrderID: "sell-1",
					Price:       100.0,
					Amount:      1.0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")

			// Place existing orders
			for _, order := range tt.existingOrders {
				if err := ob.PlaceOrder(order); err != nil {
					t.Fatalf("Failed to place existing order: %v", err)
				}
			}

			// Process new order
			trades, err := ob.ProcessOrder(tt.newOrder)
			if err != nil {
				t.Fatalf("Failed to process order: %v", err)
			}

			// Verify number of trades
			if len(trades) != len(tt.expectedTrades) {
				t.Fatalf("Expected %d trades, got %d", len(tt.expectedTrades), len(trades))
			}

			// Verify trade details
			for i, trade := range trades {
				expectedTrade := tt.expectedTrades[i]
				assertTradeDetails(t, trade, expectedTrade)
			}
		})
	}
}

func TestProcessOrder_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		existingOrders []Order
		newOrder       Order
		expectedTrades []*Trade
		checkBook      func(*testing.T, *OrderBook)
	}{
		{
			name: "Zero remaining amount after partial fill",
			existingOrders: []Order{
				{ID: "sell-1", Price: 100.0, Amount: 1.5, Side: Sell},
			},
			newOrder: Order{
				ID:     "buy-1",
				Price:  100.0,
				Amount: 1.5,
				Side:   Buy,
			},
			expectedTrades: []*Trade{
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-1",
					Price:       100.0,
					Amount:      1.5,
				},
			},
			checkBook: func(t *testing.T, ob *OrderBook) {
				if len(ob.asks) != 0 || len(ob.bids) != 0 {
					t.Error("Expected empty orderbook after exact match")
				}
			},
		},
		{
			name: "Multiple orders same price level",
			existingOrders: []Order{
				{ID: "sell-1", Price: 100.0, Amount: 1.0, Side: Sell},
				{ID: "sell-2", Price: 100.0, Amount: 1.0, Side: Sell},
			},
			newOrder: Order{
				ID:     "buy-1",
				Price:  100.0,
				Amount: 1.5,
				Side:   Buy,
			},
			expectedTrades: []*Trade{
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-1",
					Price:       100.0,
					Amount:      1.0,
				},
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-2",
					Price:       100.0,
					Amount:      0.5,
				},
			},
			checkBook: func(t *testing.T, ob *OrderBook) {
				if len(ob.asks) != 1 {
					t.Errorf("Expected 1 ask remaining, got %d", len(ob.asks))
				}
				if ob.asks[0].Amount != 0.5 {
					t.Errorf("Expected remaining amount 0.5, got %f", ob.asks[0].Amount)
				}
			},
		},
		{
			name: "Minimum price increment handling",
			existingOrders: []Order{
				{ID: "sell-1", Price: 100.001, Amount: 1.0, Side: Sell},
				{ID: "sell-2", Price: 100.002, Amount: 1.0, Side: Sell},
			},
			newOrder: Order{
				ID:     "buy-1",
				Price:  100.002,
				Amount: 1.0,
				Side:   Buy,
			},
			expectedTrades: []*Trade{
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-1",
					Price:       100.001,
					Amount:      1.0,
				},
			},
			checkBook: func(t *testing.T, ob *OrderBook) {
				if len(ob.asks) != 1 {
					t.Errorf("Expected 1 ask remaining, got %d", len(ob.asks))
				}
			},
		},
		{
			name: "Large order matching multiple price levels",
			existingOrders: []Order{
				{ID: "sell-1", Price: 100.0, Amount: 1.0, Side: Sell},
				{ID: "sell-2", Price: 101.0, Amount: 2.0, Side: Sell},
				{ID: "sell-3", Price: 102.0, Amount: 3.0, Side: Sell},
			},
			newOrder: Order{
				ID:     "buy-1",
				Price:  102.0,
				Amount: 4.0,
				Side:   Buy,
			},
			expectedTrades: []*Trade{
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-1",
					Price:       100.0,
					Amount:      1.0,
				},
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-2",
					Price:       101.0,
					Amount:      2.0,
				},
				{
					BuyOrderID:  "buy-1",
					SellOrderID: "sell-3",
					Price:       102.0,
					Amount:      1.0,
				},
			},
			checkBook: func(t *testing.T, ob *OrderBook) {
				if len(ob.asks) != 1 {
					t.Errorf("Expected 1 ask remaining, got %d", len(ob.asks))
				}
				if ob.asks[0].Amount != 2.0 {
					t.Errorf("Expected remaining amount 2.0, got %f", ob.asks[0].Amount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")

			// Place existing orders
			for _, order := range tt.existingOrders {
				if err := ob.PlaceOrder(order); err != nil {
					t.Fatalf("Failed to place existing order: %v", err)
				}
			}

			// Process new order
			trades, err := ob.ProcessOrder(tt.newOrder)
			if err != nil {
				t.Fatalf("Failed to process order: %v", err)
			}

			// Verify number of trades
			if len(trades) != len(tt.expectedTrades) {
				t.Fatalf("Expected %d trades, got %d", len(tt.expectedTrades), len(trades))
			}

			// Verify trade details
			for i, trade := range trades {
				expectedTrade := tt.expectedTrades[i]
				assertTradeDetails(t, trade, expectedTrade)
			}

			// Run additional book checks if provided
			if tt.checkBook != nil {
				tt.checkBook(t, ob)
			}
		})
	}
}

func TestGetBestBid(t *testing.T) {
	tests := []struct {
		name          string
		ordersToAdd   []Order
		expectedPrice float64
		expectError   bool
	}{
		{
			"Empty Orderbook",
			[]Order{},
			0,
			true,
		},
		{
			"Single bid order",
			[]Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Buy},
			},
			100.0,
			false,
		},
		{
			"Multiple bid orders",
			[]Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Buy},
				{ID: "2", Price: 101.0, Amount: 1.0, Side: Buy},
				{ID: "3", Price: 99.0, Amount: 1.0, Side: Buy},
			},
			101.0,
			false,
		},
		{
			"Only ask orders",
			[]Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Sell},
			},
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")

			// Add the test orders
			for _, order := range tt.ordersToAdd {
				err := ob.PlaceOrder(order)
				if err != nil {
					t.Fatalf("Failed to place order: %v", err)
				}
			}

			// Get best bid
			bestBid, err := ob.GetBestBid()

			// Check error case
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			// Check success case
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if bestBid.Price != tt.expectedPrice {
				t.Errorf("Expected best bid price %v, got %v",
					tt.expectedPrice, bestBid.Price)
			}
		})
	}
}

func TestGetBestAsk(t *testing.T) {
	tests := []struct {
		name          string
		ordersToAdd   []Order
		expectedPrice float64
		expectError   bool
	}{
		{
			name:        "Empty orderbook",
			ordersToAdd: []Order{},
			expectError: true,
		},
		{
			name: "Single ask order",
			ordersToAdd: []Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Sell},
			},
			expectedPrice: 100.0,
			expectError:   false,
		},
		{
			name: "Multiple ask orders",
			ordersToAdd: []Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Sell},
				{ID: "2", Price: 98.0, Amount: 1.0, Side: Sell},
				{ID: "3", Price: 99.0, Amount: 1.0, Side: Sell},
			},
			expectedPrice: 98.0,
			expectError:   false,
		},
		{
			name: "Only bid orders",
			ordersToAdd: []Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Buy},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")

			// Add the test orders
			for _, order := range tt.ordersToAdd {
				err := ob.PlaceOrder(order)
				if err != nil {
					t.Fatalf("Failed to place order: %v", err)
				}
			}

			// Get best ask
			bestAsk, err := ob.GetBestAsk()

			// Check error case
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			// Check success case
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if bestAsk.Price != tt.expectedPrice {
				t.Errorf("Expected best ask price %v, got %v",
					tt.expectedPrice, bestAsk.Price)
			}
		})
	}
}

func TestGetOrderBookSnapshot(t *testing.T) {
	tests := []struct {
		name         string
		ordersToAdd  []Order
		expectedAsks []OrderBookLevel
		expectedBids []OrderBookLevel
	}{
		{
			name:         "Empty orderbook",
			ordersToAdd:  []Order{},
			expectedAsks: []OrderBookLevel{},
			expectedBids: []OrderBookLevel{},
		},
		{
			name: "Single price level",
			ordersToAdd: []Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Sell},
				{ID: "2", Price: 100.0, Amount: 2.0, Side: Sell},
				{ID: "3", Price: 90.0, Amount: 3.0, Side: Buy},
				{ID: "4", Price: 90.0, Amount: 1.0, Side: Buy},
			},
			expectedAsks: []OrderBookLevel{
				{Price: 100.0, TotalAmount: 3.0, OrderCount: 2},
			},
			expectedBids: []OrderBookLevel{
				{Price: 90.0, TotalAmount: 4.0, OrderCount: 2},
			},
		},
		{
			name: "Multiple price levels",
			ordersToAdd: []Order{
				{ID: "1", Price: 100.0, Amount: 1.0, Side: Sell},
				{ID: "2", Price: 101.0, Amount: 2.0, Side: Sell},
				{ID: "3", Price: 99.0, Amount: 3.0, Side: Buy},
				{ID: "4", Price: 98.0, Amount: 1.0, Side: Buy},
			},
			expectedAsks: []OrderBookLevel{
				{Price: 100.0, TotalAmount: 1.0, OrderCount: 1},
				{Price: 101.0, TotalAmount: 2.0, OrderCount: 1},
			},
			expectedBids: []OrderBookLevel{
				{Price: 99.0, TotalAmount: 3.0, OrderCount: 1},
				{Price: 98.0, TotalAmount: 1.0, OrderCount: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := NewOrderBook("TEST")

			// Add the test orders
			for _, order := range tt.ordersToAdd {
				err := ob.PlaceOrder(order)
				if err != nil {
					t.Fatalf("Failed to place order: %v", err)
				}
			}

			// Get snapshot
			snapshot := ob.GetOrderBookSnapshot()

			// Verify asks
			if len(snapshot.Asks) != len(tt.expectedAsks) {
				t.Errorf("Expected %d ask levels, got %d",
					len(tt.expectedAsks), len(snapshot.Asks))
			}

			for i, ask := range snapshot.Asks {
				expected := tt.expectedAsks[i]
				if !compareOrderBookLevel(ask, expected) {
					t.Errorf("Ask level %d mismatch: expected %+v, got %+v",
						i, expected, ask)
				}
			}

			// Verify bids
			if len(snapshot.Bids) != len(tt.expectedBids) {
				t.Errorf("Expected %d bid levels, got %d",
					len(tt.expectedBids), len(snapshot.Bids))
			}

			for i, bid := range snapshot.Bids {
				expected := tt.expectedBids[i]
				if !compareOrderBookLevel(bid, expected) {
					t.Errorf("Bid level %d mismatch: expected %+v, got %+v",
						i, expected, bid)
				}
			}

			// Verify time is recent
			if time.Since(snapshot.Time) > time.Second {
				t.Error("Snapshot time is too old")
			}
		})
	}
}

func compareOrderBookLevel(a, b OrderBookLevel) bool {
	return a.Price == b.Price &&
		a.TotalAmount == b.TotalAmount &&
		a.OrderCount == b.OrderCount
}

func assertTradeCount(t *testing.T, trades []*Trade, expected int) {
	if len(trades) != expected {
		t.Fatalf("Expected %d trades, got %d", expected, len(trades))
	}
}

func assertTradeDetails(t *testing.T, trade *Trade, expected *Trade) {
	if trade.Price != expected.Price {
		t.Errorf("Expected trade price %v, got %v", expected.Price, trade.Price)
	}
	if trade.Amount != expected.Amount {
		t.Errorf("Expected trade amount %v, got %v", expected.Amount, trade.Amount)
	}
	if trade.BuyOrderID != expected.BuyOrderID {
		t.Errorf("Expected buy order ID %v, got %v", expected.BuyOrderID, trade.BuyOrderID)
	}
	if trade.SellOrderID != expected.SellOrderID {
		t.Errorf("Expected sell order ID %v, got %v", expected.SellOrderID, trade.SellOrderID)
	}
}

func assertEmptyOrderBook(t *testing.T, ob *OrderBook) {
	if len(ob.bids) != 0 || len(ob.asks) != 0 {
		t.Error("Expected empty orderbook after complete match")
	}
}

func assertRemainingOrder(t *testing.T, ob *OrderBook, expected Order) {
	var orders []Order
	switch expected.Side {
	case Buy:
		orders = ob.bids
	case Sell:
		orders = ob.asks
	}

	if len(orders) != 1 {
		t.Fatalf("Expected 1 remaining order, got %d", len(orders))
	}

	remaining := orders[0]
	if remaining.Price != expected.Price {
		t.Errorf("Expected remaining order price %v, got %v",
			expected.Price, remaining.Price)
	}
	if remaining.Amount != expected.Amount {
		t.Errorf("Expected remaining order amount %v, got %v",
			expected.Amount, remaining.Amount)
	}
	if remaining.Side != expected.Side {
		t.Errorf("Expected remaining order side %v, got %v",
			expected.Side, remaining.Side)
	}
	if remaining.ID != expected.ID {
		t.Errorf("Expected remaining order ID %v, got %v",
			expected.ID, remaining.ID)
	}
}
