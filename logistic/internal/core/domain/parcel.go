package domain

// Parcel represents a package to be shipped
type Parcel struct {
	Name       string  `json:"name"`
	Quantity   int     `json:"quantity"`
	WeightGram int     `json:"weight_gram"`
	Value      float64 `json:"value"` // Insurance value in VND
}

// TotalWeight calculates total weight of parcels
func TotalWeight(parcels []Parcel) int {
	total := 0
	for _, p := range parcels {
		total += p.WeightGram * p.Quantity
	}
	return total
}

// TotalValue calculates total value of parcels
func TotalValue(parcels []Parcel) float64 {
	var total float64
	for _, p := range parcels {
		total += p.Value * float64(p.Quantity)
	}
	return total
}
