package orderbook

import (
	"fmt"
	"github.com/google/uuid"

)

type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

type Order struct {
	ID     string  `json:"id"`
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	Side   Side    `json:"side"`
}

func NewOrder(price float64, amount float64, side Side) (*Order, error) {

  if price <= 0 {
    return nil, fmt.Errorf("Price must be greater than 0.")
  }

  if amount <= 0 {
    return nil, fmt.Errorf("Amount must be greather than 0")
  }

  return &Order{
    ID:uuid.New().String(),
    Price: price,
    Amount: amount,
    Side: side,
  }, nil
}
