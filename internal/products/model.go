package products

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"column:name;not null" json:"name"`
}

type Product struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID  *int64         `gorm:"column:category_id;index" json:"category_id"`
	Name        string         `gorm:"column:name;not null" json:"name"`
	Description string         `gorm:"column:description;type:text" json:"description"`
	BasePrice   int32          `gorm:"column:base_price;not null" json:"base_price"`          // Harga termurah untuk display
	Status      string         `gorm:"column:status;not null;default:'active'" json:"status"` // active, draft, archived
	CreatedAt   time.Time      `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // Enterprise standard: Soft delete

	// Relasi
	Category Category         `gorm:"foreignKey:CategoryID" json:"category"`
	Images   []ProductImage   `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;" json:"images"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;" json:"variants"`
}

func (Product) TableName() string {
	return "products"
}

type ProductImage struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID int64  `gorm:"column:product_id;index;not null" json:"product_id"`
	ImageURL  string `gorm:"column:image_url;not null" json:"image_url"`
	IsPrimary bool   `gorm:"column:is_primary;default:false" json:"is_primary"`
}

type ProductVariant struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID    int64     `gorm:"column:product_id;index;not null" json:"product_id"`
	SKU          string    `gorm:"column:sku;uniqueIndex;not null" json:"sku"` // Penting untuk pergudangan
	Name         string    `gorm:"column:name;not null" json:"name"`           // cth: "Merah - L"
	PriceInCents int32     `gorm:"column:price_in_cents;not null" json:"price_in_cents"`
	Stock        int32     `gorm:"column:stock;not null;default:0" json:"stock"`
	CreatedAt    time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null" json:"updated_at"`

	Product Product `gorm:"foreignKey:ProductID" json:"product"`
}
