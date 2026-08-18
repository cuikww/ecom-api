package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ecom-api/internal/products"
	"ecom-api/internal/utils"

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
	ListOrdersByCustomer(ctx context.Context, customerID int64, p utils.Pagination) ([]OrderResponse, error)
}

type service struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewService(db *gorm.DB, rdb *redis.Client) Service {
	return &service{db: db, redis: rdb}
}

func (s *service) getCustomerCacheKey(ctx context.Context, customerID int64, p utils.Pagination) string {
	versionKey := fmt.Sprintf("order_version:user:%d", customerID)
	version, err := s.redis.Get(ctx, versionKey).Result()
	if err != nil || version == "" {
		version = "1"
	}
	return fmt.Sprintf("orders:customer:%d:v:%s:page:%d:limit:%d", customerID, version, p.Page, p.Limit)
}

func (s *service) PlaceOrder(ctx context.Context, req CreateOrderRequest) (OrderResponse, error) {
	var order Order

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order = Order{CustomerID: req.CustomerID}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

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
				ProductName:  p.Name,
				Quantity:     qty,
				PriceInCents: p.PriceInCents,
				Product:      p,
			})

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

		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		order.Items = items
		return nil
	})

	if err != nil {
		return OrderResponse{}, err
	}
	versionKey := fmt.Sprintf("order_version:user:%d", req.CustomerID)
	s.redis.Incr(ctx, versionKey)

	return toOrderResponse(order), nil
}

func (s *service) ListOrdersByCustomer(ctx context.Context, customerID int64, p utils.Pagination) ([]OrderResponse, error) {
	cacheKey := s.getCustomerCacheKey(ctx, customerID, p)
	var result []OrderResponse

	cachedData, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return result, nil
		}
	}

	var orders []Order
	err = s.db.WithContext(ctx).
		Scopes(utils.Paginate(p)).
		Preload("Items").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&orders).Error

	if err != nil {
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
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
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
