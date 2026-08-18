package orders

import "time"

type OrderItemRequest struct {
	ProductID int64 `json:"productId" binding:"required"`
	Quantity  int32 `json:"quantity" binding:"required,min=1"`
}

type CreateOrderRequest struct {
	CustomerID int64              `json:"customerId"`
	Items      []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type OrderItemDetail struct {
	ProductID    int64  `json:"productId"`
	ProductName  string `json:"productName"`
	Quantity     int32  `json:"quantity"`
	PriceInCents int32  `json:"priceInCents"`
}

type OrderResponse struct {
	OrderID    int64             `json:"orderId"`
	CustomerID int64             `json:"customerId"`
	CreatedAt  time.Time         `json:"createdAt"`
	Items      []OrderItemDetail `json:"items"`
}
