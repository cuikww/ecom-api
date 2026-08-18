package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ecom-api/internal/products"
	"ecom-api/internal/utils" // <-- Sesuaikan path

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrorProductNotFound = errors.New("product not found")
	ErrorProductNoStock  = errors.New("product no stock")
)

type Service interface {
	PlaceOrder(ctx context.Context, req CreateOrderRequest) (OrderResponse, error)
	// Update interface untuk menerima struct Pagination
	ListOrdersByCustomer(ctx context.Context, customerID int64, p utils.Pagination) ([]OrderResponse, error)
}

type service struct {
	db    *gorm.DB
	redis *redis.Client // Tambahkan redis client
}

// Update constructor untuk menerima redis client
func NewService(db *gorm.DB, rdb *redis.Client) Service {
	return &service{db: db, redis: rdb}
}

func (s *service) PlaceOrder(ctx context.Context, req CreateOrderRequest) (OrderResponse, error) {
	var order Order

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order = Order{CustomerID: req.CustomerID}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		// Ambil semua product sekaligus + lock (1 query, bukan 1 per item)
		productIDs := make([]int64, len(req.Items))
		qtyByProduct := make(map[int64]int32, len(req.Items))
		for i, item := range req.Items {
			productIDs[i] = item.ProductID
			qtyByProduct[item.ProductID] = item.Quantity
		}

		var lockedProducts []products.Product
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", productIDs).
			Find(&lockedProducts).Error
		if err != nil {
			return err
		}
		if len(lockedProducts) != len(productIDs) {
			return ErrorProductNotFound
		}

		items := make([]OrderItem, 0, len(lockedProducts))
		for _, p := range lockedProducts {
			qty := qtyByProduct[p.ID]
			if p.Quantity < qty {
				return ErrorProductNoStock
			}
			items = append(items, OrderItem{
				OrderID:      order.ID,
				ProductID:    p.ID,
				Quantity:     qty,
				PriceInCents: p.PriceInCents,
				Product:      p, // dipakai untuk build response tanpa query ulang
			})

			// Guard stok negatif tetap di level SQL (aman dari race condition)
			res := tx.Model(&products.Product{}).
				Where("id = ? AND quantity >= ?", p.ID, qty).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", qty))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrorProductNoStock
			}
		}

		// Insert semua order item dalam 1 query (bulk insert)
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		order.Items = items
		return nil
	})
	if err != nil {
		return OrderResponse{}, err
	}

	// ==========================================
	// INVALIDASI CACHE PAGINATION (REDIS)
	// Karena kita menggunakan pagination, cache key-nya dinamis (contoh: orders:customer:1:page:1, dst).
	// Kita harus menghapus semua key yang diawali dengan "orders:customer:{ID}:*"
	// ==========================================
	pattern := fmt.Sprintf("orders:customer:%d:*", req.CustomerID)
	iter := s.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		s.redis.Del(ctx, iter.Val())
	}

	return toOrderResponse(order), nil
}

func (s *service) ListOrdersByCustomer(ctx context.Context, customerID int64, p utils.Pagination) ([]OrderResponse, error) {
	// Buat cache key yang spesifik untuk customer, page, dan limit tertentu
	cacheKey := fmt.Sprintf("orders:customer:%d:page:%d:limit:%d", customerID, p.Page, p.Limit)
	var result []OrderResponse

	// 1. Cek di Redis terlebih dahulu
	cachedData, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return result, nil // Cache Hit
		}
	}

	// 2. Jika tidak ada di cache, ambil dari Database menggunakan GORM Scope
	var orders []Order
	err = s.db.WithContext(ctx).
		Scopes(utils.Paginate(p)). // <-- AJAIBNYA DI SINI
		Preload("Items.Product").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	// Parsing ke bentuk response
	result = make([]OrderResponse, 0, len(orders))
	for _, o := range orders {
		result = append(result, toOrderResponse(o))
	}

	// 3. Simpan ke Redis dengan waktu TTL yang masuk akal (misal 5 menit)
	if resultJSON, err := json.Marshal(result); err == nil {
		s.redis.Set(ctx, cacheKey, resultJSON, 5*time.Minute)
	}

	return result, nil
}

// helper: menghindari duplikasi mapping Order -> OrderResponse
func toOrderResponse(o Order) OrderResponse {
	items := make([]OrderItemDetail, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, OrderItemDetail{
			ProductID:    item.ProductID,
			ProductName:  item.Product.Name,
			Quantity:     item.Quantity,
			PriceInCents: item.PriceInCents,
		})
	}
	return OrderResponse{
		OrderID:    o.ID,
		CustomerID: o.CustomerID,
		CreatedAt:  o.CreatedAt,
		Items:      items,
	}
}
