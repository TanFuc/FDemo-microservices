package grpc

import (
	"context"
	"errors"

	"microservices/template-service/internal/model"
	"microservices/template-service/internal/service"
	"microservices/template-service/internal/service/impl"
	"microservices/template-service/pkg/pb"
)

type ItemHandler struct {
	pb.UnimplementedTemplateServiceServer
	itemService service.ItemService
}

func NewItemHandler(itemService service.ItemService) *ItemHandler {
	return &ItemHandler{
		itemService: itemService,
	}
}

func (h *ItemHandler) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	item, err := h.itemService.GetItem(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrItemNotFound) {
			return nil, errors.New("item not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return nil, errors.New("invalid item ID")
		}
		return nil, err
	}

	return &pb.GetItemResponse{
		Item: toProtoItem(item),
	}, nil
}

func (h *ItemHandler) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	createReq := &model.CreateItemRequest{
		Name:        req.Name,
		Description: req.Description,
	}

	item, err := h.itemService.CreateItem(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return &pb.CreateItemResponse{
		Item: toProtoItem(item),
	}, nil
}

func (h *ItemHandler) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	page := int(req.Page)
	limit := int(req.Limit)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	result, err := h.itemService.ListItems(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.Item, len(result.Items))
	for i, item := range result.Items {
		items[i] = &pb.Item{
			Id:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			CreatedAt:   item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return &pb.ListItemsResponse{
		Items: items,
		Total: int32(result.Total),
	}, nil
}

func (h *ItemHandler) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	err := h.itemService.DeleteItem(ctx, req.Id)
	if err != nil {
		if errors.Is(err, impl.ErrItemNotFound) {
			return &pb.DeleteItemResponse{Success: false}, nil
		}
		return nil, err
	}

	return &pb.DeleteItemResponse{Success: true}, nil
}

func toProtoItem(item *model.ItemResponse) *pb.Item {
	return &pb.Item{
		Id:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
