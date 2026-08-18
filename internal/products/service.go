package products

import (
	"context"
	"ecom-api/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrProductNotFound = errors.New("product not found")

type Service interface {
	ListProducts(ctx context.Context, p utils.Pagination) ([]Product, error)
	FindProductsByID(ctx context.Context, id int64) (Product, error)
	CreateProduct(ctx context.Context, p ProductParams) (Product, error)
	UpdateProduct(ctx context.Context, p ProductParams) (Product, error)
}

type service struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewService(db *gorm.DB, rdb *redis.Client) Service {
	return &service{db: db, redis: rdb}
}

func (s *service) ListProducts(ctx context.Context, p utils.Pagination) ([]Product, error) {
	cacheKey := fmt.Sprintf("products:page:%d:limit:%d", p.Page, p.Limit)
	var products []Product

	cachedData, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &products); err == nil {
			return products, nil
		}
	}

	err = s.db.WithContext(ctx).
		Scopes(utils.Paginate(p)).
		Order("id ASC").
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	if productsJSON, err := json.Marshal(products); err == nil {
		s.redis.Set(ctx, cacheKey, productsJSON, 5*time.Minute)
	}

	return products, nil
}

func (s *service) FindProductsByID(ctx context.Context, id int64) (Product, error) {
	cacheKey := fmt.Sprintf("product:%d", id)
	var product Product

	cachedData, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &product); err == nil {
			return product, nil
		}
	} else if err != redis.Nil {
		fmt.Printf("redis error: %v\n", err)
	}

	err = s.db.WithContext(ctx).First(&product, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Product{}, ErrProductNotFound
	} else if err != nil {
		return Product{}, err
	}

	if productJSON, err := json.Marshal(product); err == nil {
		s.redis.Set(ctx, cacheKey, productJSON, 1*time.Hour)
	}

	return product, nil
}

func (s *service) CreateProduct(ctx context.Context, p ProductParams) (Product, error) {
	if p.ProductName == nil || p.PriceInCents == nil || p.Quantity == nil {
		return Product{}, errors.New("product_name, price_in_cents, and quantity are required")
	}

	product := Product{
		Name:         *p.ProductName,
		PriceInCents: *p.PriceInCents,
		Quantity:     *p.Quantity,
	}
	err := s.db.WithContext(ctx).Create(&product).Error
	if err != nil {
		return Product{}, err
	}

	s.redis.Del(ctx, "products:all")

	return product, nil
}

func (s *service) UpdateProduct(ctx context.Context, p ProductParams) (Product, error) {
	updates := map[string]interface{}{}
	if p.ProductName != nil {
		updates["name"] = *p.ProductName
	}
	if p.PriceInCents != nil {
		updates["price_in_cents"] = *p.PriceInCents
	}
	if p.Quantity != nil {
		updates["quantity"] = *p.Quantity
	}
	if len(updates) == 0 {
		return Product{}, errors.New("no fields to update")
	}

	var product Product
	err := s.db.WithContext(ctx).
		Clauses(clause.Returning{}).
		Model(&product).
		Where("id = ?", p.ID).
		Updates(updates).Error

	if err != nil {
		return Product{}, err
	}
	if product.ID == 0 {
		return Product{}, ErrProductNotFound
	}

	cacheKeySpecific := fmt.Sprintf("product:%d", p.ID)
	cacheKeyAll := "products:all"
	s.redis.Del(ctx, cacheKeySpecific, cacheKeyAll)

	return product, nil
}
