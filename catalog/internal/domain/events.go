package domain

const (
	SubjectProductCreated = "catalog.product.created"
	SubjectProductUpdated = "catalog.product.updated"
)

type ProductEvent struct {
	Product *Product `json:"product"`
}
