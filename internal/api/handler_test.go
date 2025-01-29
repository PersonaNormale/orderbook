package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"orderbook/internal/orderbook"
	"strings"
	"testing"
	"time"
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

func TestGetBestBid_Success(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	expectedOrder := orderbook.Order{
		ID:     "bid1",
		Price:  100.0,
		Amount: 5.0,
		Side:   orderbook.Buy,
	}

	if err := ob.PlaceOrder(expectedOrder); err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodGet, "/best-bid", nil)
	rr := httptest.NewRecorder()

	handler.GetBestBid(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want application/json", contentType)
	}

	var responseOrder orderbook.Order
	if err := json.Unmarshal(rr.Body.Bytes(), &responseOrder); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if responseOrder.ID != expectedOrder.ID {
		t.Errorf("Handler returned wrong order ID: got %v want %v", responseOrder.ID, expectedOrder.ID)
	}
	if responseOrder.Price != expectedOrder.Price {
		t.Errorf("Handler returned wrong price: got %v want %v", responseOrder.Price, expectedOrder.Price)
	}
	if responseOrder.Amount != expectedOrder.Amount {
		t.Errorf("Handler returned wrong amount: got %v want %v", responseOrder.Amount, expectedOrder.Amount)
	}
}

func TestGetBestBid_NoBids(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodGet, "/best-bid", nil)
	rr := httptest.NewRecorder()

	handler.GetBestBid(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	if !strings.Contains(rr.Body.String(), "No Orders Present") {
		t.Errorf("Handler returned wrong error message: got %v want 'No Orders Present'", rr.Body.String())
	}
}

func TestGetBestBid_WrongMethod(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodPost, "/best-bid", nil)
	rr := httptest.NewRecorder()

	handler.GetBestBid(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}

	if !strings.Contains(rr.Body.String(), "Method Not Allowed") {
		t.Errorf("Handler returned wrong error message: got %v want 'Method Not Allowed'", rr.Body.String())
	}
}

func TestGetBestBid_MultipleBids(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	expectedBestBid := orderbook.Order{
		ID:     "best",
		Price:  200.0, // Highest price should be best bid
		Amount: 3.0,
		Side:   orderbook.Buy,
	}

	bids := []orderbook.Order{
		expectedBestBid,
		{ID: "bid2", Price: 150.0, Amount: 2.0, Side: orderbook.Buy},
		{ID: "bid3", Price: 100.0, Amount: 5.0, Side: orderbook.Buy},
	}

	for _, bid := range bids {
		if err := ob.PlaceOrder(bid); err != nil {
			t.Fatalf("Failed to place order: %v", err)
		}
	}

	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodGet, "/best-bid", nil)
	rr := httptest.NewRecorder()

	handler.GetBestBid(rr, req)

	var bestBid orderbook.Order
	if err := json.Unmarshal(rr.Body.Bytes(), &bestBid); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if bestBid.ID != expectedBestBid.ID {
		t.Errorf("Handler returned wrong best bid ID: got %v want %v", bestBid.ID, expectedBestBid.ID)
	}
}

func TestGetBestAskHandler_Success(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	expectedOrder := orderbook.Order{
		ID:     "ask1",
		Price:  100.0,
		Amount: 5.0,
		Side:   orderbook.Sell,
	}

	// Add a valid ask order
	if err := ob.PlaceOrder(expectedOrder); err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodGet, "/best-ask", nil)
	rr := httptest.NewRecorder()

	handler.GetBestAsk(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want application/json", contentType)
	}

	var responseOrder orderbook.Order
	if err := json.Unmarshal(rr.Body.Bytes(), &responseOrder); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if responseOrder.ID != expectedOrder.ID {
		t.Errorf("Handler returned wrong order ID: got %v want %v", responseOrder.ID, expectedOrder.ID)
	}
	if responseOrder.Price != expectedOrder.Price {
		t.Errorf("Handler returned wrong price: got %v want %v", responseOrder.Price, expectedOrder.Price)
	}
	if responseOrder.Amount != expectedOrder.Amount {
		t.Errorf("Handler returned wrong amount: got %v want %v", responseOrder.Amount, expectedOrder.Amount)
	}
}

func TestGetBestAskHandler_NoAsks(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodGet, "/best-ask", nil)
	rr := httptest.NewRecorder()

	handler.GetBestAsk(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	if !strings.Contains(rr.Body.String(), "No Orders Present") {
		t.Errorf("Handler returned wrong error message: got %v want 'No Orders Present'", rr.Body.String())
	}
}

func TestGetBestAskHandler_WrongMethod(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodPost, "/best-ask", nil)
	rr := httptest.NewRecorder()

	handler.GetBestAsk(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}

	if !strings.Contains(rr.Body.String(), "Method Not Allowed") {
		t.Errorf("Handler returned wrong error message: got %v want 'Method Not Allowed'", rr.Body.String())
	}
}

func TestGetBestAskHandler_MultipleAsks(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	expectedBestAsk := orderbook.Order{
		ID:     "best",
		Price:  100.0, // Lowest price should be best ask
		Amount: 3.0,
		Side:   orderbook.Sell,
	}

	// Add asks in ascending order
	asks := []orderbook.Order{
		expectedBestAsk,
		{ID: "ask2", Price: 150.0, Amount: 2.0, Side: orderbook.Sell},
		{ID: "ask3", Price: 200.0, Amount: 5.0, Side: orderbook.Sell},
	}

	for _, ask := range asks {
		if err := ob.PlaceOrder(ask); err != nil {
			t.Fatalf("Failed to place order: %v", err)
		}
	}

	handler := NewHandler(ob)
	req := httptest.NewRequest(http.MethodGet, "/best-ask", nil)
	rr := httptest.NewRecorder()

	handler.GetBestAsk(rr, req)

	var bestAsk orderbook.Order
	if err := json.Unmarshal(rr.Body.Bytes(), &bestAsk); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if bestAsk.ID != expectedBestAsk.ID {
		t.Errorf("Handler returned wrong best ask ID: got %v want %v", bestAsk.ID, expectedBestAsk.ID)
	}
}

func TestGetOrderbookSnapshot_Success(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)

	// Add some test orders
	orders := []orderbook.Order{
		{ID: "ask1", Price: 100.0, Amount: 5.0, Side: orderbook.Sell},
		{ID: "ask2", Price: 101.0, Amount: 3.0, Side: orderbook.Sell},
		{ID: "bid1", Price: 99.0, Amount: 4.0, Side: orderbook.Buy},
		{ID: "bid2", Price: 98.0, Amount: 2.0, Side: orderbook.Buy},
	}

	for _, order := range orders {
		if err := ob.PlaceOrder(order); err != nil {
			t.Fatalf("Failed to place order: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/orderbook-snapshot", nil)
	rr := httptest.NewRecorder()

	handler.GetOrderbookSnapshot(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want application/json", contentType)
	}

	// Parse response
	var snapshot orderbook.OrderBookSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify asks
	if len(snapshot.Asks) != 2 {
		t.Errorf("Expected 2 ask levels, got %d", len(snapshot.Asks))
	}
	if snapshot.Asks[0].Price != 100.0 || snapshot.Asks[0].TotalAmount != 5.0 {
		t.Errorf("First ask level incorrect: got price %.2f amount %.2f, want price 100.00 amount 5.00",
			snapshot.Asks[0].Price, snapshot.Asks[0].TotalAmount)
	}

	// Verify bids
	if len(snapshot.Bids) != 2 {
		t.Errorf("Expected 2 bid levels, got %d", len(snapshot.Bids))
	}
	if snapshot.Bids[0].Price != 99.0 || snapshot.Bids[0].TotalAmount != 4.0 {
		t.Errorf("First bid level incorrect: got price %.2f amount %.2f, want price 99.00 amount 4.00",
			snapshot.Bids[0].Price, snapshot.Bids[0].TotalAmount)
	}

	// Verify timestamp is recent
	if time.Since(snapshot.Time) > 5*time.Second {
		t.Error("Snapshot timestamp is too old")
	}
}

func TestGetOrderbookSnapshot_EmptyOrderbook(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)

	req := httptest.NewRequest(http.MethodGet, "/orderbook-snapshot", nil)
	rr := httptest.NewRecorder()

	handler.GetOrderbookSnapshot(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var snapshot orderbook.OrderBookSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(snapshot.Asks) != 0 {
		t.Errorf("Expected empty asks, got %d levels", len(snapshot.Asks))
	}
	if len(snapshot.Bids) != 0 {
		t.Errorf("Expected empty bids, got %d levels", len(snapshot.Bids))
	}
}

func TestGetOrderbookSnapshot_WrongMethod(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)

	req := httptest.NewRequest(http.MethodPost, "/orderbook-snapshot", nil)
	rr := httptest.NewRecorder()

	handler.GetOrderbookSnapshot(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestGetOrderbookSnapshot_MultipleOrdersSamePrice(t *testing.T) {
	ob := orderbook.NewOrderBook("test")
	handler := NewHandler(ob)

	// Add multiple orders at the same price level
	orders := []orderbook.Order{
		{ID: "ask1", Price: 100.0, Amount: 5.0, Side: orderbook.Sell},
		{ID: "ask2", Price: 100.0, Amount: 3.0, Side: orderbook.Sell},
		{ID: "bid1", Price: 99.0, Amount: 4.0, Side: orderbook.Buy},
		{ID: "bid2", Price: 99.0, Amount: 2.0, Side: orderbook.Buy},
	}

	for _, order := range orders {
		if err := ob.PlaceOrder(order); err != nil {
			t.Fatalf("Failed to place order: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/orderbook-snapshot", nil)
	rr := httptest.NewRecorder()

	handler.GetOrderbookSnapshot(rr, req)

	var snapshot orderbook.OrderBookSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify asks are aggregated
	if len(snapshot.Asks) != 1 {
		t.Errorf("Expected 1 ask level, got %d", len(snapshot.Asks))
	}
	if snapshot.Asks[0].Price != 100.0 || snapshot.Asks[0].TotalAmount != 8.0 {
		t.Errorf("Ask level incorrect: got price %.2f amount %.2f, want price 100.00 amount 8.00",
			snapshot.Asks[0].Price, snapshot.Asks[0].TotalAmount)
	}
	if snapshot.Asks[0].OrderCount != 2 {
		t.Errorf("Expected 2 orders at ask level, got %d", snapshot.Asks[0].OrderCount)
	}

	// Verify bids are aggregated
	if len(snapshot.Bids) != 1 {
		t.Errorf("Expected 1 bid level, got %d", len(snapshot.Bids))
	}
	if snapshot.Bids[0].Price != 99.0 || snapshot.Bids[0].TotalAmount != 6.0 {
		t.Errorf("Bid level incorrect: got price %.2f amount %.2f, want price 99.00 amount 6.00",
			snapshot.Bids[0].Price, snapshot.Bids[0].TotalAmount)
	}
	if snapshot.Bids[0].OrderCount != 2 {
		t.Errorf("Expected 2 orders at bid level, got %d", snapshot.Bids[0].OrderCount)
	}
}

func TestPlaceOrder_InvalidJSON(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	tests := []struct {
		name         string
		body         string
		expectedCode int
	}{
		{
			name:         "Malformed JSON",
			body:         `{"id": "order1", "side": "BUY", "price": 100.0, "amount": 10.0,}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Missing Price Field",
			body:         `{"id": "order1", "side": "BUY", "amount": 10.0}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Negative Amount",
			body:         `{"id": "order1", "side": "BUY", "price": 100.0, "amount": -5.0}`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/place-order", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			handler.PlaceOrder(w, req)
			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestCancelOrder_InvalidIDFormat(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	// Test ID with special characters
	req := httptest.NewRequest("DELETE", "/cancel-order?id=order@123", nil)
	w := httptest.NewRecorder()
	handler.CancelOrder(w, req)

	// Should return 400 since order doesn't exist, not because of ID format
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for non-existent order, got %d", w.Code)
	}
}

func TestProcessOrder_PartialFillAndRemaining(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	// Place a sell order
	sellOrder := orderbook.Order{
		ID:     "sell1",
		Side:   orderbook.Sell,
		Price:  100.0,
		Amount: 5.0,
	}
	book.PlaceOrder(sellOrder)

	// Process a larger buy order
	buyOrder := orderbook.Order{
		ID:     "buy1",
		Side:   orderbook.Buy,
		Price:  100.0,
		Amount: 8.0,
	}
	orderJSON, _ := json.Marshal(buyOrder)
	req := httptest.NewRequest("POST", "/process-order", bytes.NewBuffer(orderJSON))
	w := httptest.NewRecorder()
	handler.ProcessOrder(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	// Verify trade and remaining order
	var trades []*orderbook.Trade
	json.NewDecoder(w.Body).Decode(&trades)
	if len(trades) != 1 || trades[0].Amount != 5.0 {
		t.Errorf("Expected 1 trade for 5.0, got %v", trades)
	}

	// Check remaining buy order in bids
	snapshot := book.GetOrderBookSnapshot()
	if len(snapshot.Bids) != 1 || snapshot.Bids[0].TotalAmount != 3.0 {
		t.Errorf("Expected remaining buy amount 3.0, got %v", snapshot.Bids)
	}
}

func TestGetBestAsk_AfterModify(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	// Place two asks
	book.PlaceOrder(orderbook.Order{ID: "ask1", Side: orderbook.Sell, Price: 105.0, Amount: 2.0})
	book.PlaceOrder(orderbook.Order{ID: "ask2", Side: orderbook.Sell, Price: 100.0, Amount: 3.0})

	// Modify ask2 to have lower price
	book.ModifyOrder("ask2", 95.0, 3.0)

	req := httptest.NewRequest("GET", "/best-ask", nil)
	w := httptest.NewRecorder()
	handler.GetBestAsk(w, req)

	var bestAsk orderbook.Order
	json.NewDecoder(w.Body).Decode(&bestAsk)
	if bestAsk.Price != 95.0 {
		t.Errorf("Expected best ask 95.0, got %.2f", bestAsk.Price)
	}
}

func TestGetOrderbookSnapshot_AfterMultipleOperations(t *testing.T) {
	book := orderbook.NewOrderBook("TEST")
	handler := NewHandler(book)

	// Add and modify orders
	book.PlaceOrder(orderbook.Order{ID: "bid1", Side: orderbook.Buy, Price: 99.0, Amount: 5.0})
	book.PlaceOrder(orderbook.Order{ID: "bid2", Side: orderbook.Buy, Price: 100.0, Amount: 3.0})
	book.CancelOrder("bid1")
	book.ModifyOrder("bid2", 101.0, 4.0)

	req := httptest.NewRequest("GET", "/orderbook-snapshot", nil)
	w := httptest.NewRecorder()
	handler.GetOrderbookSnapshot(w, req)

	var snapshot orderbook.OrderBookSnapshot
	json.NewDecoder(w.Body).Decode(&snapshot)

	// Verify bids
	if len(snapshot.Bids) != 1 || snapshot.Bids[0].Price != 101.0 || snapshot.Bids[0].TotalAmount != 4.0 {
		t.Errorf("Snapshot bids incorrect: %v", snapshot.Bids)
	}
}
