package products

import (
	"context"
	"ecom-api/internal/utils"
	"ecom-api/internal/worker"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var ErrProductNotFound = errors.New("product not found")

type Service interface {
	ListProducts(ctx context.Context, p utils.Pagination) ([]Product, error)
	FindProductByID(ctx context.Context, id int64) (Product, error)
	CreateProduct(ctx context.Context, p ProductParams) (Product, error)
	UpdateProduct(ctx context.Context, p ProductParams) (Product, error)
	DeleteProduct(ctx context.Context, id int64) error
}

type service struct {
	db         *gorm.DB
	redis      *redis.Client
	workerPool worker.Pool
}

func NewService(db *gorm.DB, rdb *redis.Client, wp worker.Pool) Service {
	return &service{db: db, redis: rdb, workerPool: wp}
}

func (s *service) ListProducts(ctx context.Context, p utils.Pagination) ([]Product, error) {
	cacheKey := fmt.Sprintf("products:page:%d:limit:%d", p.Page, p.Limit)
	var products []Product

	if cachedData, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(cachedData), &products); err == nil {
			return products, nil
		}
	}

	err := s.db.WithContext(ctx).
		Scopes(utils.Paginate(p)).
		Preload("Category").
		Preload("Images").
		Preload("Variants").
		Order("id DESC").
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	if productsJSON, err := json.Marshal(products); err == nil {
		s.redis.Set(ctx, cacheKey, productsJSON, 5*time.Minute)
	}

	return products, nil
}

func (s *service) FindProductByID(ctx context.Context, id int64) (Product, error) {
	cacheKey := fmt.Sprintf("product:%d", id)
	var product Product

	if cachedData, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(cachedData), &product); err == nil {
			return product, nil
		}
	}

	err := s.db.WithContext(ctx).
		Preload("Category").
		Preload("Images").
		Preload("Variants").
		First(&product, id).Error

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
	if p.Name == nil || len(p.Variants) == 0 {
		return Product{}, errors.New("product name and at least one variant are required")
	}

	var basePrice int32 = p.Variants[0].PriceInCents
	for _, v := range p.Variants {
		if v.PriceInCents < basePrice {
			basePrice = v.PriceInCents
		}
	}

	var newProduct Product

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		status := "active"
		if p.Status != nil {
			status = *p.Status
		}

		desc := ""
		if p.Description != nil {
			desc = *p.Description
		}

		product := Product{
			Name:        *p.Name,
			Description: desc,
			BasePrice:   basePrice,
			CategoryID:  p.CategoryID,
			Status:      status,
		}

		if err := tx.Create(&product).Error; err != nil {
			return err
		}

		for _, imgParam := range p.Images {
			img := ProductImage{
				ProductID: product.ID,
				ImageURL:  imgParam.ImageURL,
				IsPrimary: imgParam.IsPrimary,
			}
			if err := tx.Create(&img).Error; err != nil {
				return err
			}
		}

		for _, varParam := range p.Variants {
			variant := ProductVariant{
				ProductID:    product.ID,
				SKU:          varParam.SKU,
				Name:         varParam.Name,
				PriceInCents: varParam.PriceInCents,
				Stock:        varParam.Stock,
			}
			if err := tx.Create(&variant).Error; err != nil {
				return err
			}
		}

		tx.Preload("Category").Preload("Images").Preload("Variants").First(&newProduct, product.ID)
		return nil
	})

	if err != nil {
		return Product{}, err
	}

	s.invalidateCache()
	return newProduct, nil
}

func (s *service) UpdateProduct(ctx context.Context, p ProductParams) (Product, error) {
	var updatedProduct Product

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product Product
		if err := tx.First(&product, p.ID).Error; err != nil {
			return err
		}

		if p.Name != nil {
			product.Name = *p.Name
		}
		if p.Description != nil {
			product.Description = *p.Description
		}
		if p.CategoryID != nil {
			product.CategoryID = p.CategoryID
		}
		if p.Status != nil {
			product.Status = *p.Status
		}

		tx.Model(&product).Association("Images").Clear()
		tx.Model(&product).Association("Variants").Clear()

		var newImages []ProductImage
		for _, img := range p.Images {
			newImages = append(newImages, ProductImage{ImageURL: img.ImageURL, IsPrimary: img.IsPrimary})
		}
		product.Images = newImages

		var newVariants []ProductVariant
		for _, v := range p.Variants {
			newVariants = append(newVariants, ProductVariant{
				SKU: v.SKU, Name: v.Name, PriceInCents: v.PriceInCents, Stock: v.Stock,
			})
		}
		product.Variants = newVariants

		if len(newVariants) > 0 {
			basePrice := newVariants[0].PriceInCents
			for _, v := range newVariants {
				if v.PriceInCents < basePrice {
					basePrice = v.PriceInCents
				}
			}
			product.BasePrice = basePrice
		}

		if err := tx.Save(&product).Error; err != nil {
			return err
		}

		tx.Preload("Category").Preload("Images").Preload("Variants").First(&updatedProduct, product.ID)
		return nil
	})

	if err != nil {
		return Product{}, err
	}

	s.invalidateCacheSpecific(p.ID)
	return updatedProduct, nil
}

func (s *service) DeleteProduct(ctx context.Context, id int64) error {
	err := s.db.WithContext(ctx).Delete(&Product{}, id).Error
	if err == nil {
		s.invalidateCacheSpecific(id)
	}
	return err
}

func (s *service) invalidateCache() {
	s.workerPool.Enqueue(func(bgCtx context.Context) error {
		s.redis.Del(bgCtx, "products:all")
		return nil
	})
}

func (s *service) invalidateCacheSpecific(id int64) {
	s.workerPool.Enqueue(func(bgCtx context.Context) error {
		s.redis.Del(bgCtx, fmt.Sprintf("product:%d", id), "products:all")
		return nil
	})
}
