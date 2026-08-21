package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ecom-api/internal/products"
	"ecom-api/internal/utils"
	"ecom-api/internal/worker"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrorVariantNotFound = errors.New("product variant not found")
	ErrorProductNoStock  = errors.New("product variant out of stock")
	ErrorOrderNotFound   = errors.New("order not found")
	ErrorProductNotFound = errors.New("product not found")
)

type Service interface {
	PlaceOrder(ctx context.Context, req CreateOrderRequest) (OrderResponse, error)
	ListOrdersByCustomer(ctx context.Context, customerID int64, p utils.Pagination) ([]OrderResponse, error)
	ListAllOrders(ctx context.Context, p utils.Pagination) ([]OrderResponse, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, status string) (OrderResponse, error)
}

type service struct {
	db         *gorm.DB
	redis      *redis.Client
	workerPool worker.Pool
}

func NewService(db *gorm.DB, rdb *redis.Client, wp worker.Pool) Service {
	return &service{db: db, redis: rdb, workerPool: wp}
}

func (s *service) getCustomerCacheKey(ctx context.Context, customerID int64, p utils.Pagination) string {
	versionKey := fmt.Sprintf("order_version:user:%d", customerID)
	version, err := s.redis.Get(ctx, versionKey).Result()
	if err != nil || version == "" {
		version = "1"
	}
	return fmt.Sprintf("orders:customer:%d:v:%s:page:%d:limit:%d", customerID, version, p.Page, p.Limit)
}

func (s *service) getGlobalCacheKey(ctx context.Context, p utils.Pagination) string {
	versionKey := "order_version:global"
	version, err := s.redis.Get(ctx, versionKey).Result()
	if err != nil || version == "" {
		version = "1"
	}
	return fmt.Sprintf("orders:all:v:%s:page:%d:limit:%d", version, p.Page, p.Limit)
}

func (s *service) invalidateCaches(ctx context.Context, customerID int64) {
	s.redis.Incr(ctx, fmt.Sprintf("order_version:user:%d", customerID))
	s.redis.Incr(ctx, "order_version:global")
}

func (s *service) PlaceOrder(ctx context.Context, req CreateOrderRequest) (OrderResponse, error) {
	var order Order

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order = Order{CustomerID: req.CustomerID, Status: "PENDING"}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		variantIDs := make([]int64, len(req.Items))
		qtyByVariant := make(map[int64]int32, len(req.Items))
		for i, item := range req.Items {
			variantIDs[i] = item.VariantID
			qtyByVariant[item.VariantID] = item.Quantity
		}

		var lockedVariants []products.ProductVariant
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Product"). // <-- Ambil data produk induk
			Where("id IN ?", variantIDs).
			Find(&lockedVariants).Error
		if err != nil {
			return err
		}
		if len(lockedVariants) != len(variantIDs) {
			return ErrorVariantNotFound
		}

		items := make([]OrderItem, 0, len(lockedVariants))
		for _, v := range lockedVariants {
			qty := qtyByVariant[v.ID]
			if v.Stock < qty {
				return ErrorProductNoStock
			}

			items = append(items, OrderItem{
				OrderID:      order.ID,
				ProductID:    v.ProductID,
				ProductName:  v.Product.Name, // <-- Masukkan nama produk di sini
				VariantID:    v.ID,
				VariantName:  v.Name,
				SKU:          v.SKU,
				Quantity:     qty,
				PriceInCents: v.PriceInCents,
				Variant:      v,
			})

			// Kurangi stok di tabel product_variants secara aman
			res := tx.Model(&products.ProductVariant{}).
				Where("id = ? AND stock >= ?", v.ID, qty).
				UpdateColumn("stock", gorm.Expr("stock - ?", qty))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrorProductNoStock
			}
		}

		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		order.Items = items
		return nil
	})

	if err != nil {
		return OrderResponse{}, err
	}

	s.workerPool.Enqueue(func(bgCtx context.Context) error {
		s.invalidateCaches(bgCtx, req.CustomerID)
		return nil
	})

	return toOrderResponse(order), nil
}

func (s *service) ListOrdersByCustomer(ctx context.Context, customerID int64, p utils.Pagination) ([]OrderResponse, error) {
	cacheKey := s.getCustomerCacheKey(ctx, customerID, p)
	return s.fetchOrdersFromCacheOrDB(ctx, cacheKey, p, func(db *gorm.DB) *gorm.DB {
		return db.Where("customer_id = ?", customerID)
	})
}

func (s *service) ListAllOrders(ctx context.Context, p utils.Pagination) ([]OrderResponse, error) {
	cacheKey := s.getGlobalCacheKey(ctx, p)
	return s.fetchOrdersFromCacheOrDB(ctx, cacheKey, p, func(db *gorm.DB) *gorm.DB {
		return db
	})
}

func (s *service) UpdateOrderStatus(ctx context.Context, orderID int64, status string) (OrderResponse, error) {
	var order Order
	err := s.db.WithContext(ctx).Preload("Items").First(&order, orderID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return OrderResponse{}, ErrorOrderNotFound
		}
		return OrderResponse{}, err
	}
	order.Status = status
	if err := s.db.WithContext(ctx).Save(&order).Error; err != nil {
		return OrderResponse{}, err
	}

	s.invalidateCaches(ctx, order.CustomerID)
	return toOrderResponse(order), nil
}

func (s *service) fetchOrdersFromCacheOrDB(ctx context.Context, cacheKey string, p utils.Pagination, applyFilter func(*gorm.DB) *gorm.DB) ([]OrderResponse, error) {
	var result []OrderResponse

	cachedData, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return result, nil
		}
	}
	var orders []Order
	query := s.db.WithContext(ctx).Scopes(utils.Paginate(p)).Preload("Items").Order("created_at DESC")
	query = applyFilter(query)

	if err := query.Find(&orders).Error; err != nil {
		return nil, err
	}

	result = make([]OrderResponse, 0, len(orders))
	for _, o := range orders {
		result = append(result, toOrderResponse(o))
	}
	if resultJSON, err := json.Marshal(result); err == nil {
		s.redis.Set(ctx, cacheKey, resultJSON, 5*time.Minute)
	}

	return result, nil
}

func toOrderResponse(o Order) OrderResponse {
	items := make([]OrderItemDetail, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, OrderItemDetail{
			VariantID:    item.VariantID,
			SKU:          item.SKU,
			VariantName:  item.VariantName,
			Quantity:     item.Quantity,
			PriceInCents: item.PriceInCents,
		})
	}
	return OrderResponse{
		OrderID:    o.ID,
		CustomerID: o.CustomerID,
		Status:     o.Status,
		CreatedAt:  o.CreatedAt,
		Items:      items,
	}
}
