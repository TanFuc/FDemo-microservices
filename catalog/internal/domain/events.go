package domain

const (
	SubjectProductCreated = "catalog.product.created"
	SubjectProductUpdated = "catalog.product.updated"
	SubjectProductDeleted = "catalog.product.deleted"

	EventTypeCreated = "created"
	EventTypeUpdated = "updated"
	EventTypeDeleted = "deleted"
)

// ProductEvent represents the event payload for NATS publishing
// This is the internal domain event - the publisher transforms it to the wire format
type ProductEvent struct {
	EventType string   `json:"eventType"`
	ProductID string   `json:"productId,omitempty"`
	Product   *Product `json:"product,omitempty"`
}
