package products

import "time"

type Product struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"column:name;not null" json:"name"`
	PriceInCents int32     `gorm:"column:price_in_cents;not null" json:"priceInCents"`
	Quantity     int32     `gorm:"column:quantity;not null;default:0" json:"quantity"`
	CreatedAt    time.Time `gorm:"column:created_at;not null" json:"createdAt"`
}

func (Product) TableName() string {
	return "products"
}
