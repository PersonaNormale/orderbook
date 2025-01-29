package api

import (
	"encoding/json"
	"net/http"
	"orderbook/internal/orderbook"
	"strconv"
)

type Handler struct {
	book *orderbook.OrderBook
}

// Create a new book handler for OrderBook
func NewHandler(book *orderbook.OrderBook) *Handler {
	return &Handler{book: book}
}

// Handler for PlaceOrder function
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var order orderbook.Order

	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}

	if err := h.book.PlaceOrder(order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Handler for CancelOrder function
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	orderID := r.URL.Query().Get("id")
	if orderID == "" {
		http.Error(w, "Order ID is Required", http.StatusBadRequest)
		return
	}

	if err := h.book.CancelOrder(orderID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Handel for ModifyOrder function
func (h *Handler) ModifyOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var price, amount float64
	orderID := r.URL.Query().Get("id")
	priceString := r.URL.Query().Get("price")
	amountString := r.URL.Query().Get("price")

	if orderID == "" {
		http.Error(w, "Order ID is Required", http.StatusBadRequest)
		return
	}

	if _, err := strconv.ParseFloat(priceString, 64); err != nil {
		http.Error(w, "Price is Not a Number", http.StatusBadRequest)
		return
	}

	if _, err := strconv.ParseFloat(amountString, 64); err != nil {
		http.Error(w, "Amount is Not a Number", http.StatusBadRequest)
		return
	}

	price, _ = strconv.ParseFloat(priceString, 64)
	amount, _ = strconv.ParseFloat(amountString, 64)

	if err := h.book.ModifyOrder(orderID, price, amount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Handler for ProcessOrder function
func (h *Handler) ProcessOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming order
	var order orderbook.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}

	// Process the order and get resulting trades
	trades, err := h.book.ProcessOrder(order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If there are no trades, return an empty array instead of null
	if trades == nil {
		trades = []*orderbook.Trade{}
	}

	// Set response header
	w.Header().Set("Content-Type", "application/json")

	// Encode and return the trades
	if err := json.NewEncoder(w).Encode(trades); err != nil {
		http.Error(w, "Error Encoding Response", http.StatusInternalServerError)
		return
	}
}

// Handler for GetBestBid function
func (h *Handler) GetBestBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bestBid, err := h.book.GetBestBid()

	if err == orderbook.ErrNoOrders {
		http.Error(w, "No Orders Present", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(bestBid); err != nil {
		http.Error(w, "Error Encoding Response", http.StatusInternalServerError)
		return
	}
}

// Handler for GetBestAsk function
func (h *Handler) GetBestAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bestBid, err := h.book.GetBestAsk()

	if err == orderbook.ErrNoOrders {
		http.Error(w, "No Orders Present", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(bestBid); err != nil {
		http.Error(w, "Error Encoding Response", http.StatusInternalServerError)
		return
	}
}

// Handler for GetOrderbookSNapshot function
func (h *Handler) GetOrderbookSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.book.GetOrderBookSnapshot()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		http.Error(w, "Error Encoding Response", http.StatusInternalServerError)
		return
	}
}
