package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"orderbook/internal/orderbook"
	"testing"
)

func TestPlaceOrder(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	tests := []struct {
		name         string
		order        orderbook.Order
		method       string
		expectedCode int
	}{
		{
			name: "Valid Buy Order",
			order: orderbook.Order{
				ID:     "order1",
				Side:   orderbook.Buy,
				Price:  100.0,
				Amount: 10.0,
			},
			method:       "POST",
			expectedCode: http.StatusCreated,
		},
		{
			name: "Invalid Price",
			order: orderbook.Order{
				ID:     "order2",
				Side:   orderbook.Buy,
				Price:  -100.0,
				Amount: 10.0,
			},
			method:       "POST",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Wrong Method",
			order: orderbook.Order{
				ID:     "order3",
				Side:   orderbook.Buy,
				Price:  100.0,
				Amount: 10.0,
			},
			method:       "GET",
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderJSON, _ := json.Marshal(tt.order)
			req := httptest.NewRequest(tt.method, "/place-order", bytes.NewBuffer(orderJSON))
			w := httptest.NewRecorder()

			handler.PlaceOrder(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestCancelOrder(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	// Place an order first
	order := orderbook.Order{
		ID:     "order-to-cancel",
		Side:   orderbook.Buy,
		Price:  100.0,
		Amount: 10.0,
	}
	book.PlaceOrder(order)

	tests := []struct {
		name         string
		orderID      string
		method       string
		expectedCode int
	}{
		{
			name:         "Valid Cancel",
			orderID:      "order-to-cancel",
			method:       "DELETE",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Non-existent Order",
			orderID:      "non-existent",
			method:       "DELETE",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Wrong Method",
			orderID:      "order-to-cancel",
			method:       "GET",
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/cancel-order?id="+tt.orderID, nil)
			w := httptest.NewRecorder()

			handler.CancelOrder(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestModifyOrder(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	// Place an order first
	order := orderbook.Order{
		ID:     "order-to-modify",
		Side:   orderbook.Buy,
		Price:  100.0,
		Amount: 10.0,
	}
	book.PlaceOrder(order)

	tests := []struct {
		name         string
		orderID      string
		price        string
		amount       string
		method       string
		expectedCode int
	}{
		{
			name:         "Valid Modification",
			orderID:      "order-to-modify",
			price:        "110.0",
			amount:       "15.0",
			method:       "PATCH",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid Price",
			orderID:      "order-to-modify",
			price:        "-110.0",
			amount:       "15.0",
			method:       "PATCH",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Wrong Method",
			orderID:      "order-to-modify",
			price:        "110.0",
			amount:       "15.0",
			method:       "GET",
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/modify-order?id=" + tt.orderID + "&price=" + tt.price + "&amount=" + tt.amount
			req := httptest.NewRequest(tt.method, url, nil)
			w := httptest.NewRecorder()

			handler.ModifyOrder(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestProcessOrder(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	// Place a sell order first
	sellOrder := orderbook.Order{
		ID:     "sell-order",
		Side:   orderbook.Sell,
		Price:  100.0,
		Amount: 10.0,
	}
	book.PlaceOrder(sellOrder)

	tests := []struct {
		name         string
		order        orderbook.Order
		method       string
		expectedCode int
		checkTrades  bool
	}{
		{
			name: "Matching Buy Order",
			order: orderbook.Order{
				ID:     "buy-order",
				Side:   orderbook.Buy,
				Price:  100.0,
				Amount: 5.0,
			},
			method:       "POST",
			expectedCode: http.StatusOK,
			checkTrades:  true,
		},
		{
			name: "Non-matching Buy Order",
			order: orderbook.Order{
				ID:     "non-matching-buy",
				Side:   orderbook.Buy,
				Price:  90.0,
				Amount: 5.0,
			},
			method:       "POST",
			expectedCode: http.StatusOK,
			checkTrades:  false,
		},
		{
			name: "Invalid Order",
			order: orderbook.Order{
				ID:     "invalid-order",
				Side:   orderbook.Buy,
				Price:  -100.0,
				Amount: 5.0,
			},
			method:       "POST",
			expectedCode: http.StatusBadRequest,
			checkTrades:  false,
		},
		{
			name: "Wrong Method",
			order: orderbook.Order{
				ID:     "wrong-method",
				Side:   orderbook.Buy,
				Price:  100.0,
				Amount: 5.0,
			},
			method:       "GET",
			expectedCode: http.StatusMethodNotAllowed,
			checkTrades:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderJSON, _ := json.Marshal(tt.order)
			req := httptest.NewRequest(tt.method, "/process-order", bytes.NewBuffer(orderJSON))
			w := httptest.NewRecorder()

			handler.ProcessOrder(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.checkTrades && w.Code == http.StatusOK {
				var trades []*orderbook.Trade
				if err := json.NewDecoder(w.Body).Decode(&trades); err != nil {
					t.Errorf("Failed to decode response body: %v", err)
				}
				if len(trades) == 0 {
					t.Error("Expected trades but got none")
				}
				// Verify trade details
				for _, trade := range trades {
					if trade.Amount <= 0 || trade.Price <= 0 {
						t.Error("Invalid trade values")
					}
					if trade.BuyOrderID == "" || trade.SellOrderID == "" {
						t.Error("Missing order IDs in trade")
					}
				}
			}
		})
	}
}
