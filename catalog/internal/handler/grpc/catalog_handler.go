package grpc

import (
	"context"
	"errors"

	"microservices/catalog/internal/model"
	"microservices/catalog/internal/service"
	"microservices/catalog/internal/service/impl"
	"microservices/catalog/pkg/pb"
)

type CatalogHandler struct {
	pb.UnimplementedCatalogServiceServer
	productService  service.ProductService
	categoryService service.CategoryService
	brandService    service.BrandService
	authService     service.AuthService
}

func NewCatalogHandler(
	productService service.ProductService,
	categoryService service.CategoryService,
	brandService service.BrandService,
	authService service.AuthService,
) *CatalogHandler {
	return &CatalogHandler{
		productService:  productService,
		categoryService: categoryService,
		brandService:    brandService,
		authService:     authService,
	}
}

// ========== Product Operations ==========

func (h *CatalogHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	product, err := h.productService.GetProduct(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return nil, errors.New("product not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return nil, errors.New("invalid product ID")
		}
		return nil, err
	}

	return &pb.GetProductResponse{
		Product: toProtoProduct(product),
	}, nil
}

func (h *CatalogHandler) GetProductBySlug(ctx context.Context, req *pb.GetProductBySlugRequest) (*pb.GetProductResponse, error) {
	product, err := h.productService.GetProductBySlug(ctx, req.Slug)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &pb.GetProductResponse{
		Product: toProtoProduct(product),
	}, nil
}

func (h *CatalogHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	createReq := &model.CreateProductRequest{
		ShopID:           req.ShopId,
		ShopName:         req.ShopName,
		Name:             req.Name,
		Description:      req.Description,
		ShortDescription: req.ShortDescription,
		CategoryID:       req.CategoryId,
		BrandID:          req.BrandId,
		Thumbnail:        req.Thumbnail,
		Images:           toModelImages(req.Images),
		BasePrice:        req.BasePrice,
		Currency:         req.Currency,
		Variations:       toModelVariations(req.Variations),
		Status:           req.Status,
		Visibility:       req.Visibility,
		CreatedBy:        req.CreatedBy,
	}

	product, err := h.productService.CreateProduct(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return &pb.CreateProductResponse{
		Product: toProtoProduct(product),
	}, nil
}

func (h *CatalogHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	updateReq := &model.UpdateProductRequest{
		Name:             req.Name,
		Description:      req.Description,
		ShortDescription: req.ShortDescription,
		CategoryID:       req.CategoryId,
		BrandID:          req.BrandId,
		Thumbnail:        req.Thumbnail,
		Images:           toModelImages(req.Images),
		BasePrice:        req.BasePrice,
		Variations:       toModelVariations(req.Variations),
		Status:           req.Status,
		Visibility:       req.Visibility,
		UpdatedBy:        req.UpdatedBy,
	}

	product, err := h.productService.UpdateProduct(ctx, req.Id, updateReq)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &pb.UpdateProductResponse{
		Product: toProtoProduct(product),
	}, nil
}

func (h *CatalogHandler) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	err := h.productService.DeleteProduct(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return &pb.DeleteProductResponse{Success: false}, nil
		}
		return nil, err
	}

	return &pb.DeleteProductResponse{Success: true}, nil
}

func (h *CatalogHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	page := int(req.Pagination.Page)
	limit := int(req.Pagination.Limit)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	filter := &model.ProductFilter{
		CategoryID: req.CategoryId,
		BrandID:    req.BrandId,
		Status:     req.Status,
		ShopID:     req.ShopId,
		Page:       page,
		Limit:      limit,
	}

	result, err := h.productService.ListProducts(ctx, filter)
	if err != nil {
		return nil, err
	}

	products := make([]*pb.Product, len(result.Products))
	for i, product := range result.Products {
		products[i] = toProtoProduct(&product)
	}

	return &pb.ListProductsResponse{
		Products: products,
		Pagination: &pb.PaginationInfo{
			Page:       int32(result.Page),
			Limit:      int32(result.Limit),
			Total:      result.Total,
			TotalPages: int32(result.TotalPages),
		},
	}, nil
}

// ========== Category Operations ==========

func (h *CatalogHandler) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.GetCategoryResponse, error) {
	category, err := h.categoryService.GetCategory(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	return &pb.GetCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (h *CatalogHandler) GetCategoryBySlug(ctx context.Context, req *pb.GetCategoryBySlugRequest) (*pb.GetCategoryResponse, error) {
	category, err := h.categoryService.GetCategoryBySlug(ctx, req.Slug)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	return &pb.GetCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (h *CatalogHandler) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	createReq := &model.CreateCategoryRequest{
		Name:        map[string]string{"en": req.Name},
		Description: map[string]string{"en": req.Description},
		ParentID:    req.ParentId,
		ImageURL:    req.ImageUrl,
		IconURL:     req.IconUrl,
	}

	category, err := h.categoryService.CreateCategory(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return &pb.CreateCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (h *CatalogHandler) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.UpdateCategoryResponse, error) {
	updateReq := &model.UpdateCategoryRequest{
		ParentID: req.ParentId,
		ImageURL: req.ImageUrl,
		IconURL:  req.IconUrl,
		Status:   req.Status,
	}
	if req.Name != "" {
		updateReq.Name = map[string]string{"en": req.Name}
	}
	if req.Description != "" {
		updateReq.Description = map[string]string{"en": req.Description}
	}

	category, err := h.categoryService.UpdateCategory(ctx, req.Id, updateReq)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	return &pb.UpdateCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (h *CatalogHandler) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.DeleteCategoryResponse, error) {
	err := h.categoryService.DeleteCategory(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return &pb.DeleteCategoryResponse{Success: false}, nil
		}
		return nil, err
	}

	return &pb.DeleteCategoryResponse{Success: true}, nil
}

func (h *CatalogHandler) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	result, err := h.categoryService.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	categories := make([]*pb.Category, len(result.Categories))
	for i, category := range result.Categories {
		categories[i] = toProtoCategory(&category)
	}

	return &pb.ListCategoriesResponse{
		Categories: categories,
		Pagination: &pb.PaginationInfo{
			Page:       int32(result.Page),
			Limit:      int32(result.Limit),
			Total:      result.Total,
			TotalPages: 1,
		},
	}, nil
}

// ========== Brand Operations ==========

func (h *CatalogHandler) GetBrand(ctx context.Context, req *pb.GetBrandRequest) (*pb.GetBrandResponse, error) {
	brand, err := h.brandService.GetBrand(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return nil, errors.New("brand not found")
		}
		return nil, err
	}

	return &pb.GetBrandResponse{
		Brand: toProtoBrand(brand),
	}, nil
}

func (h *CatalogHandler) GetBrandBySlug(ctx context.Context, req *pb.GetBrandBySlugRequest) (*pb.GetBrandResponse, error) {
	brand, err := h.brandService.GetBrandBySlug(ctx, req.Slug)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return nil, errors.New("brand not found")
		}
		return nil, err
	}

	return &pb.GetBrandResponse{
		Brand: toProtoBrand(brand),
	}, nil
}

func (h *CatalogHandler) CreateBrand(ctx context.Context, req *pb.CreateBrandRequest) (*pb.CreateBrandResponse, error) {
	createReq := &model.CreateBrandRequest{
		Name:        req.Name,
		Description: map[string]string{"en": req.Description},
		LogoURL:     req.LogoUrl,
		BannerURL:   req.BannerUrl,
		Website:     req.Website,
		Country:     req.Country,
	}

	brand, err := h.brandService.CreateBrand(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return &pb.CreateBrandResponse{
		Brand: toProtoBrand(brand),
	}, nil
}

func (h *CatalogHandler) UpdateBrand(ctx context.Context, req *pb.UpdateBrandRequest) (*pb.UpdateBrandResponse, error) {
	updateReq := &model.UpdateBrandRequest{
		Name:      req.Name,
		LogoURL:   req.LogoUrl,
		BannerURL: req.BannerUrl,
		Website:   req.Website,
		Country:   req.Country,
		Status:    req.Status,
	}
	if req.Description != "" {
		updateReq.Description = map[string]string{"en": req.Description}
	}
	if req.IsVerified {
		updateReq.IsVerified = &req.IsVerified
	}
	if req.IsFeatured {
		updateReq.IsFeatured = &req.IsFeatured
	}

	brand, err := h.brandService.UpdateBrand(ctx, req.Id, updateReq)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return nil, errors.New("brand not found")
		}
		return nil, err
	}

	return &pb.UpdateBrandResponse{
		Brand: toProtoBrand(brand),
	}, nil
}

func (h *CatalogHandler) DeleteBrand(ctx context.Context, req *pb.DeleteBrandRequest) (*pb.DeleteBrandResponse, error) {
	err := h.brandService.DeleteBrand(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return &pb.DeleteBrandResponse{Success: false}, nil
		}
		return nil, err
	}

	return &pb.DeleteBrandResponse{Success: true}, nil
}

func (h *CatalogHandler) ListBrands(ctx context.Context, req *pb.ListBrandsRequest) (*pb.ListBrandsResponse, error) {
	result, err := h.brandService.ListBrands(ctx)
	if err != nil {
		return nil, err
	}

	brands := make([]*pb.Brand, len(result.Brands))
	for i, brand := range result.Brands {
		brands[i] = toProtoBrand(&brand)
	}

	return &pb.ListBrandsResponse{
		Brands: brands,
		Pagination: &pb.PaginationInfo{
			Page:       int32(result.Page),
			Limit:      int32(result.Limit),
			Total:      result.Total,
			TotalPages: 1,
		},
	}, nil
}

// ========== Helper Functions ==========

func toProtoProduct(p *model.ProductResponse) *pb.Product {
	images := make([]*pb.ProductImage, len(p.Images))
	for i, img := range p.Images {
		images[i] = &pb.ProductImage{
			Url:       img.URL,
			Alt:       img.Alt,
			Position:  int32(img.Position),
			IsPrimary: img.IsPrimary,
		}
	}

	variations := make([]*pb.Variation, len(p.Variations))
	for i, v := range p.Variations {
		variations[i] = &pb.Variation{
			Sku:        v.SKU,
			Attributes: v.Attributes,
			Price:      v.Price.String(),
			Stock:      int32(v.Stock),
			Available:  int32(v.Available),
			IsActive:   v.IsActive,
		}
	}

	return &pb.Product{
		Id:               p.ID,
		ShopId:           p.ShopID,
		ShopName:         p.ShopName,
		Name:             p.Name,
		Slug:             p.Slug,
		Description:      p.Description,
		ShortDescription: p.ShortDescription,
		CategoryId:       p.CategoryID,
		CategoryPath:     p.CategoryPath,
		BrandId:          p.BrandID,
		BrandName:        p.BrandName,
		Thumbnail:        p.Thumbnail,
		Images:           images,
		BasePrice:        p.BasePrice,
		CompareAtPrice:   p.CompareAtPrice,
		MinPrice:         p.MinPrice,
		MaxPrice:         p.MaxPrice,
		Currency:         p.Currency,
		Variations:       variations,
		TotalStock:       int32(p.TotalStock),
		Status:           p.Status,
		Visibility:       p.Visibility,
		CreatedAt:        p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoCategory(c *model.CategoryResponse) *pb.Category {
	name := ""
	if n, ok := c.Name["en"]; ok {
		name = n
	}
	desc := ""
	if d, ok := c.Description["en"]; ok {
		desc = d
	}

	return &pb.Category{
		Id:           c.ID,
		Name:         name,
		Slug:         c.Slug,
		Description:  desc,
		ParentId:     c.ParentID,
		Path:         c.Path,
		Level:        int32(c.Level),
		Position:     int32(c.Position),
		ImageUrl:     c.ImageURL,
		IconUrl:      c.IconURL,
		Status:       c.Status,
		ProductCount: int64(c.ProductCount),
		CreatedAt:    c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoBrand(b *model.BrandResponse) *pb.Brand {
	desc := ""
	if d, ok := b.Description["en"]; ok {
		desc = d
	}

	return &pb.Brand{
		Id:           b.ID,
		Name:         b.Name,
		Slug:         b.Slug,
		Description:  desc,
		LogoUrl:      b.LogoURL,
		BannerUrl:    b.BannerURL,
		Website:      b.Website,
		Country:      b.Country,
		Status:       b.Status,
		IsVerified:   b.IsVerified,
		IsFeatured:   b.IsFeatured,
		ProductCount: int64(b.ProductCount),
		CreatedAt:    b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    b.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toModelImages(images []*pb.ProductImage) []model.ProductImage {
	result := make([]model.ProductImage, len(images))
	for i, img := range images {
		result[i] = model.ProductImage{
			URL:       img.Url,
			Alt:       img.Alt,
			Position:  int(img.Position),
			IsPrimary: img.IsPrimary,
		}
	}
	return result
}

func toModelVariations(variations []*pb.Variation) []model.Variation {
	result := make([]model.Variation, len(variations))
	for i, v := range variations {
		result[i] = model.Variation{
			SKU:        v.Sku,
			Attributes: v.Attributes,
			Stock:      int(v.Stock),
			Available:  int(v.Available),
			IsActive:   v.IsActive,
		}
	}
	return result
}
