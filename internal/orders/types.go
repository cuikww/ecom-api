package orders

import "time"

type OrderItemRequest struct {
	VariantID int64 `json:"variantId" binding:"required"`
	Quantity  int32 `json:"quantity" binding:"required,min=1"`
}

type CreateOrderRequest struct {
	CustomerID int64              `json:"customerId" binding:"required"`
	Items      []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type OrderItemDetail struct {
	VariantID    int64  `json:"variantId"`
	SKU          string `json:"sku"`
	VariantName  string `json:"variantName"`
	Quantity     int32  `json:"quantity"`
	PriceInCents int32  `json:"priceInCents"`
}

type OrderResponse struct {
	OrderID    int64             `json:"orderId"`
	CustomerID int64             `json:"customerId"`
	Status     string            `json:"status"`
	CreatedAt  time.Time         `json:"createdAt"`
	Items      []OrderItemDetail `json:"items"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=PENDING SHIPPED COMPLETED CANCELED"`
}
