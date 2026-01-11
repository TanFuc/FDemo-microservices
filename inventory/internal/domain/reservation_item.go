package domain

type ReservationItem struct {
	SkuID    string `json:"sku_id"`
	Quantity int    `json:"quantity"`
}
