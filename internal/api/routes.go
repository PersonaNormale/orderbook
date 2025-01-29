package api

import (
	"net/http"
)

type Router struct {
	handler *Handler
}

func NewRouter(handler *Handler) *Router {
	return &Router{
		handler: handler,
	}
}

func (r *Router) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	prefix := "" // API versioning maybe?

	// Order management endpoints
	mux.HandleFunc(prefix+"/orders/place", r.handler.PlaceOrder)
	mux.HandleFunc(prefix+"/orders/cancel", r.handler.CancelOrder)
	mux.HandleFunc(prefix+"/orders/modify", r.handler.ModifyOrder)
	mux.HandleFunc(prefix+"/orders/process", r.handler.ProcessOrder)

	// Order book query endpoints
	mux.HandleFunc(prefix+"/orderbook/best-bid", r.handler.GetBestBid)
	mux.HandleFunc(prefix+"/orderbook/best-ask", r.handler.GetBestAsk)
	mux.HandleFunc(prefix+"/orderbook/snapshot", r.handler.GetOrderbookSnapshot)

	return mux
}
