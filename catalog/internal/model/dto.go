package model

import "time"

// ========== Product DTOs ==========

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	ShopID           string                 `json:"shopId" validate:"required"`
	ShopName         string                 `json:"shopName" validate:"required"`
	Name             string                 `json:"name" validate:"required,min=1,max=255"`
	Description      string                 `json:"description" validate:"required"`
	ShortDescription string                 `json:"shortDescription,omitempty"`
	CategoryID       string                 `json:"categoryId" validate:"required"`
	BrandID          string                 `json:"brandId,omitempty"`
	Thumbnail        string                 `json:"thumbnail" validate:"required"`
	Images           []ProductImage         `json:"images,omitempty"`
	BasePrice        string                 `json:"basePrice" validate:"required"`
	Currency         string                 `json:"currency,omitempty"`
	Variations       []Variation            `json:"variations,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	Status           string                 `json:"status,omitempty"`
	Visibility       string                 `json:"visibility,omitempty"`
	CreatedBy        string                 `json:"createdBy" validate:"required"`
}

// UpdateProductRequest represents the request to update a product
type UpdateProductRequest struct {
	Name             string                 `json:"name,omitempty"`
	Description      string                 `json:"description,omitempty"`
	ShortDescription string                 `json:"shortDescription,omitempty"`
	CategoryID       string                 `json:"categoryId,omitempty"`
	BrandID          string                 `json:"brandId,omitempty"`
	Thumbnail        string                 `json:"thumbnail,omitempty"`
	Images           []ProductImage         `json:"images,omitempty"`
	BasePrice        string                 `json:"basePrice,omitempty"`
	Variations       []Variation            `json:"variations,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	Status           string                 `json:"status,omitempty"`
	Visibility       string                 `json:"visibility,omitempty"`
	UpdatedBy        string                 `json:"updatedBy,omitempty"`
}

// ProductResponse represents the response for a product
type ProductResponse struct {
	ID               string         `json:"id"`
	ShopID           string         `json:"shopId"`
	ShopName         string         `json:"shopName"`
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	Description      string         `json:"description"`
	ShortDescription string         `json:"shortDescription,omitempty"`
	CategoryID       string         `json:"categoryId"`
	CategoryPath     string         `json:"categoryPath"`
	BrandID          string         `json:"brandId,omitempty"`
	BrandName        string         `json:"brandName,omitempty"`
	Thumbnail        string         `json:"thumbnail"`
	Images           []ProductImage `json:"images"`
	BasePrice        string         `json:"basePrice"`
	CompareAtPrice   string         `json:"compareAtPrice,omitempty"`
	MinPrice         string         `json:"minPrice"`
	MaxPrice         string         `json:"maxPrice"`
	Currency         string         `json:"currency"`
	Variations       []Variation    `json:"variations"`
	TotalStock       int            `json:"totalStock"`
	Status           string         `json:"status"`
	Visibility       string         `json:"visibility"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// ProductFilter represents filter options for listing products
type ProductFilter struct {
	CategoryID string `json:"categoryId,omitempty"`
	BrandID    string `json:"brandId,omitempty"`
	Status     string `json:"status,omitempty"`
	ShopID     string `json:"shopId,omitempty"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// ListProductsResponse represents paginated product response
type ListProductsResponse struct {
	Products   []ProductResponse `json:"products"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"totalPages"`
}

// ToResponse converts Product entity to ProductResponse DTO
func (p *Product) ToResponse() *ProductResponse {
	resp := &ProductResponse{
		ID:               p.ID.Hex(),
		ShopID:           p.ShopID,
		ShopName:         p.ShopName,
		Name:             p.Name,
		Slug:             p.Slug,
		Description:      p.Description,
		ShortDescription: p.ShortDescription,
		CategoryID:       p.CategoryID.Hex(),
		CategoryPath:     p.CategoryPath,
		Thumbnail:        p.Thumbnail,
		Images:           p.Images,
		BasePrice:        p.BasePrice.String(),
		CompareAtPrice:   p.CompareAtPrice.String(),
		MinPrice:         p.MinPrice.String(),
		MaxPrice:         p.MaxPrice.String(),
		Currency:         p.Currency,
		Variations:       p.Variations,
		TotalStock:       p.TotalStock,
		Status:           p.Status,
		Visibility:       p.Visibility,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
	if !p.BrandID.IsZero() {
		resp.BrandID = p.BrandID.Hex()
	}
	resp.BrandName = p.BrandName
	return resp
}

// ========== Category DTOs ==========

// CreateCategoryRequest represents the request to create a category
type CreateCategoryRequest struct {
	Name        map[string]string `json:"name" validate:"required"`
	Description map[string]string `json:"description,omitempty"`
	ParentID    string            `json:"parentId,omitempty"`
	ImageURL    string            `json:"imageUrl,omitempty"`
	IconURL     string            `json:"iconUrl,omitempty"`
}

// UpdateCategoryRequest represents the request to update a category
type UpdateCategoryRequest struct {
	Name        map[string]string `json:"name,omitempty"`
	Description map[string]string `json:"description,omitempty"`
	ParentID    string            `json:"parentId,omitempty"`
	ImageURL    string            `json:"imageUrl,omitempty"`
	IconURL     string            `json:"iconUrl,omitempty"`
	Status      string            `json:"status,omitempty"`
}

// CategoryResponse represents the response for a category
type CategoryResponse struct {
	ID           string            `json:"id"`
	Name         map[string]string `json:"name"`
	Slug         string            `json:"slug"`
	Description  map[string]string `json:"description,omitempty"`
	ParentID     string            `json:"parentId,omitempty"`
	Path         string            `json:"path"`
	Level        int               `json:"level"`
	Position     int               `json:"position"`
	ImageURL     string            `json:"imageUrl,omitempty"`
	IconURL      string            `json:"iconUrl,omitempty"`
	Status       string            `json:"status"`
	ProductCount int               `json:"productCount"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// ListCategoriesResponse represents paginated category response
type ListCategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
}

// ToResponse converts Category entity to CategoryResponse DTO
func (c *Category) ToResponse() *CategoryResponse {
	resp := &CategoryResponse{
		ID:           c.ID.Hex(),
		Name:         c.Name,
		Slug:         c.Slug,
		Description:  c.Description,
		Path:         c.Path,
		Level:        c.Level,
		Position:     c.Position,
		ImageURL:     c.ImageURL,
		IconURL:      c.IconURL,
		Status:       c.Status,
		ProductCount: c.ProductCount,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
	if c.ParentID != nil {
		resp.ParentID = c.ParentID.Hex()
	}
	return resp
}

// ========== Brand DTOs ==========

// CreateBrandRequest represents the request to create a brand
type CreateBrandRequest struct {
	Name        string            `json:"name" validate:"required,min=1,max=255"`
	Description map[string]string `json:"description,omitempty"`
	LogoURL     string            `json:"logoUrl,omitempty"`
	BannerURL   string            `json:"bannerUrl,omitempty"`
	Website     string            `json:"website,omitempty"`
	Country     string            `json:"country,omitempty"`
}

// UpdateBrandRequest represents the request to update a brand
type UpdateBrandRequest struct {
	Name        string            `json:"name,omitempty"`
	Description map[string]string `json:"description,omitempty"`
	LogoURL     string            `json:"logoUrl,omitempty"`
	BannerURL   string            `json:"bannerUrl,omitempty"`
	Website     string            `json:"website,omitempty"`
	Country     string            `json:"country,omitempty"`
	Status      string            `json:"status,omitempty"`
	IsVerified  *bool             `json:"isVerified,omitempty"`
	IsFeatured  *bool             `json:"isFeatured,omitempty"`
}

// BrandResponse represents the response for a brand
type BrandResponse struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"`
	Description  map[string]string `json:"description,omitempty"`
	LogoURL      string            `json:"logoUrl,omitempty"`
	BannerURL    string            `json:"bannerUrl,omitempty"`
	Website      string            `json:"website,omitempty"`
	Country      string            `json:"country,omitempty"`
	Status       string            `json:"status"`
	IsVerified   bool              `json:"isVerified"`
	IsFeatured   bool              `json:"isFeatured"`
	ProductCount int               `json:"productCount"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// ListBrandsResponse represents paginated brand response
type ListBrandsResponse struct {
	Brands []BrandResponse `json:"brands"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// ToResponse converts Brand entity to BrandResponse DTO
func (b *Brand) ToResponse() *BrandResponse {
	return &BrandResponse{
		ID:           b.ID.Hex(),
		Name:         b.Name,
		Slug:         b.Slug,
		Description:  b.Description,
		LogoURL:      b.LogoURL,
		BannerURL:    b.BannerURL,
		Website:      b.Website,
		Country:      b.Country,
		Status:       b.Status,
		IsVerified:   b.IsVerified,
		IsFeatured:   b.IsFeatured,
		ProductCount: b.ProductCount,
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
}
