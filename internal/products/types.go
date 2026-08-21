package products

type ProductParams struct {
	ID          int64                  `json:"id"`
	CategoryID  *int64                 `json:"category_id"`
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Status      *string                `json:"status"`
	Images      []ProductImageParams   `json:"images"`
	Variants    []ProductVariantParams `json:"variants"`
}

type ProductImageParams struct {
	ID        int64  `json:"id"`
	ImageURL  string `json:"image_url" binding:"required"`
	IsPrimary bool   `json:"is_primary"`
}

type ProductVariantParams struct {
	ID           int64  `json:"id"`
	SKU          string `json:"sku" binding:"required"`
	Name         string `json:"name" binding:"required"`
	PriceInCents int32  `json:"price_in_cents" binding:"required"`
	Stock        int32  `json:"stock" binding:"required"`
}
