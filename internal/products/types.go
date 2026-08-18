package products

type ProductParams struct {
	ID           int64   `json:"id"`
	ProductName  *string `json:"product_name"`
	PriceInCents *int32  `json:"price_in_cents"`
	Quantity     *int32  `json:"quantity"`
}
