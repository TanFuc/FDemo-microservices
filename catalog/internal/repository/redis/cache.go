package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"microservices/catalog/internal/domain"
	"microservices/catalog/internal/repository"

	"github.com/redis/go-redis/v9"
)

var _ repository.CacheRepository = (*CacheRepository)(nil)

const (
	categoryPrefix     = "category:"
	productPrefix      = "product:"
	categoryTTL        = 10 * time.Minute
	productTTL         = 5 * time.Minute
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

func (r *CacheRepository) GetCategory(ctx context.Context, slug string) (*domain.Category, error) {
	key := fmt.Sprintf("%s%s", categoryPrefix, slug)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var category domain.Category
	if err := json.Unmarshal(data, &category); err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CacheRepository) SetCategory(ctx context.Context, category *domain.Category) error {
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

func (r *CacheRepository) GetProduct(ctx context.Context, slug string) (*domain.Product, error) {
	key := fmt.Sprintf("%s%s", productPrefix, slug)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var product domain.Product
	if err := json.Unmarshal(data, &product); err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *CacheRepository) SetProduct(ctx context.Context, product *domain.Product) error {
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
