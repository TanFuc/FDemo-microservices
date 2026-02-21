package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"microservices/catalog/internal/model"
	"microservices/catalog/internal/repository"

	"github.com/redis/go-redis/v9"
)

var _ repository.CacheRepository = (*CacheRepository)(nil)

const (
	categoryPrefix = "category:"
	productPrefix  = "product:"
	brandPrefix    = "brand:"
	categoryTTL    = 10 * time.Minute
	productTTL     = 5 * time.Minute
	brandTTL       = 10 * time.Minute
)

type CacheRepository struct {
	client *redis.Client
}

func NewCacheRepository(client *redis.Client) *CacheRepository {
	return &CacheRepository{client: client}
}

func Connect(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// Product cache operations

func (r *CacheRepository) GetProduct(ctx context.Context, slug string) (*model.Product, error) {
	key := fmt.Sprintf("%s%s", productPrefix, slug)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var product model.Product
	if err := json.Unmarshal(data, &product); err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *CacheRepository) SetProduct(ctx context.Context, product *model.Product) error {
	key := fmt.Sprintf("%s%s", productPrefix, product.Slug)
	data, err := json.Marshal(product)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, productTTL).Err()
}

func (r *CacheRepository) DeleteProduct(ctx context.Context, slug string) error {
	key := fmt.Sprintf("%s%s", productPrefix, slug)
	return r.client.Del(ctx, key).Err()
}

// Category cache operations

func (r *CacheRepository) GetCategory(ctx context.Context, slug string) (*model.Category, error) {
	key := fmt.Sprintf("%s%s", categoryPrefix, slug)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var category model.Category
	if err := json.Unmarshal(data, &category); err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CacheRepository) SetCategory(ctx context.Context, category *model.Category) error {
	key := fmt.Sprintf("%s%s", categoryPrefix, category.Slug)
	data, err := json.Marshal(category)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, categoryTTL).Err()
}

func (r *CacheRepository) DeleteCategory(ctx context.Context, slug string) error {
	key := fmt.Sprintf("%s%s", categoryPrefix, slug)
	return r.client.Del(ctx, key).Err()
}

// Brand cache operations

func (r *CacheRepository) GetBrand(ctx context.Context, slug string) (*model.Brand, error) {
	key := fmt.Sprintf("%s%s", brandPrefix, slug)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var brand model.Brand
	if err := json.Unmarshal(data, &brand); err != nil {
		return nil, err
	}
	return &brand, nil
}

func (r *CacheRepository) SetBrand(ctx context.Context, brand *model.Brand) error {
	key := fmt.Sprintf("%s%s", brandPrefix, brand.Slug)
	data, err := json.Marshal(brand)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, brandTTL).Err()
}

func (r *CacheRepository) DeleteBrand(ctx context.Context, slug string) error {
	key := fmt.Sprintf("%s%s", brandPrefix, slug)
	return r.client.Del(ctx, key).Err()
}
