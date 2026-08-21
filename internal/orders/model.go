package orders

import (
	"ecom-api/internal/products"
	"time"
)

type Order struct {
	ID         int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	CustomerID int64       `gorm:"column:customer_id;not null" json:"customerId"`
	Status     string      `gorm:"column:status;not null;default:'PENDING'" json:"status"`
	CreatedAt  time.Time   `gorm:"column:created_at;not null" json:"createdAt"`
	Items      []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID           int64                   `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID      int64                   `gorm:"column:order_id;not null" json:"orderId"`
	ProductID    int64                   `gorm:"column:product_id;not null" json:"productId"`
	VariantID    int64                   `gorm:"column:variant_id;not null" json:"variantId"`
	ProductName  string                  `gorm:"column:product_name;not null" json:"productName"` // <-- Tambahkan ini
	VariantName  string                  `gorm:"column:variant_name;not null" json:"variantName"`
	SKU          string                  `gorm:"column:sku;not null" json:"sku"`
	Quantity     int32                   `gorm:"column:quantity;not null" json:"quantity"`
	PriceInCents int32                   `gorm:"column:price_in_cents;not null" json:"priceInCents"`
	Variant      products.ProductVariant `gorm:"foreignKey:VariantID" json:"-"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
