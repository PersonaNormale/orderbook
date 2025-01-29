# Orderbook Microservice

A simple order book implementation with HTTP API for managing orders and matching trades.

## Features
- Order placement and cancellation
- Order book snapshots
- Best bid/ask queries 
- Order matching engine
- Real-time trade execution

## Quick Start

1. Start the server:
```bash
go run cmd/api/main.go
```

2. Place an order:
```bash
curl -X POST http://localhost:8080/orders/place \
  -H "Content-Type: application/json" \
  -d '{"side": "BUY", "price": 100.0, "amount": 1.0}'
```

## API Endpoints

- `POST /orders/place` - Place new order
- `DELETE /orders/cancel` - Cancel existing order
- `PATCH /orders/modify` - Modify order
- `GET /orderbook/snapshot` - Get orderbook state
- `GET /orderbook/best-bid` - Get best bid
- `GET /orderbook/best-ask` - Get best ask
- `POST /orders/process` - Process order

## TODO List

Priority items:
- [ ] Better Performance Implementation
- [ ] Add authentication
- [ ] Add logging system

Future enhancements:
- [ ] Add persistence layer
- [ ] Implement websocket for real-time updates
- [ ] Support for different order types (market, limit)
- [ ] Trade history
