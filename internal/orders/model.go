package orders

import (
	"time"

	"ecom-api/internal/products"
)

type Order struct {
	ID         int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	CustomerID int64       `gorm:"column:customer_id;not null" json:"customerId"`
	Status     string      `gorm:"status;not null;default:'PENDING'" json:"status"`
	CreatedAt  time.Time   `gorm:"column:created_at;not null" json:"createdAt"`
	Items      []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID           int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID      int64            `gorm:"column:order_id;not null" json:"orderId"`
	ProductID    int64            `gorm:"column:product_id;not null" json:"productId"`
	ProductName  string           `gorm:"column:product_name;not null" json:"productName"`
	Quantity     int32            `gorm:"column:quantity;not null" json:"quantity"`
	PriceInCents int32            `gorm:"column:price_in_cents;not null" json:"priceInCents"`
	Product      products.Product `gorm:"foreignKey:ProductID" json:"-"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
