package domain

// ContactInfo represents sender or receiver contact details
type ContactInfo struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	WardCode    string `json:"ward_code"`
	DistrictID  int    `json:"district_id"`
	ProvinceID  int    `json:"province_id"`
}
