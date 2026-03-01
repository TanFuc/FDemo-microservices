package domain

// SystemStatus represents normalized shipping status across all providers
type SystemStatus string

const (
	StatusPending   SystemStatus = "PENDING"
	StatusPicking   SystemStatus = "PICKING"
	StatusShipping  SystemStatus = "SHIPPING"
	StatusDelivered SystemStatus = "DELIVERED"
	StatusReturned  SystemStatus = "RETURNED"
	StatusCancelled SystemStatus = "CANCELLED"
)

// ProviderName represents supported shipping providers
type ProviderName string

const (
	ProviderMock        ProviderName = "MOCK"
	ProviderGHN         ProviderName = "GHN"
	ProviderGHTK        ProviderName = "GHTK"
	ProviderViettelPost ProviderName = "VIETTELPOST"
)

// IsValid checks if the provider name is valid
func (p ProviderName) IsValid() bool {
	switch p {
	case ProviderMock, ProviderGHN, ProviderGHTK, ProviderViettelPost:
		return true
	}
	return false
}

// StatusMapping maps provider-specific status to system status
var StatusMapping = map[ProviderName]map[string]SystemStatus{
	ProviderGHN: {
		"ready_to_pick":     StatusPending,
		"picking":           StatusPicking,
		"picked":            StatusPicking,
		"storing":           StatusShipping,
		"transporting":      StatusShipping,
		"sorting":           StatusShipping,
		"delivering":        StatusShipping,
		"delivered":         StatusDelivered,
		"delivery_fail":     StatusShipping,
		"waiting_to_return": StatusReturned,
		"return":            StatusReturned,
		"return_transporting": StatusReturned,
		"returned":          StatusReturned,
		"cancel":            StatusCancelled,
	},
	ProviderGHTK: {
		"-1": StatusCancelled,
		"1":  StatusPending,
		"2":  StatusPicking,
		"3":  StatusShipping,
		"4":  StatusShipping,
		"5":  StatusDelivered,
		"6":  StatusReturned,
		"7":  StatusShipping,
		"8":  StatusPicking,
		"9":  StatusShipping,
		"10": StatusShipping,
		"11": StatusShipping,
		"12": StatusShipping,
		"13": StatusReturned,
		"20": StatusReturned,
		"21": StatusReturned,
	},
	ProviderMock: {
		"pending":   StatusPending,
		"picking":   StatusPicking,
		"shipping":  StatusShipping,
		"delivered": StatusDelivered,
		"returned":  StatusReturned,
		"cancelled": StatusCancelled,
	},
	ProviderViettelPost: {
		"PHAT_THANH_CONG":      StatusDelivered,
		"PHAT_KHONG_THANH_CONG": StatusShipping,
		"DA_GUI_POST":          StatusShipping,
		"CHUYEN_HOAN":          StatusReturned,
		"HUY":                  StatusCancelled,
		"CHO_PHAT":             StatusPending,
		"DONG_GOI_CHO_CHUYEN":  StatusPending,
	},
}

// MapCarrierStatus converts provider-specific status to system status
func MapCarrierStatus(provider ProviderName, carrierStatus string) SystemStatus {
	if mapping, ok := StatusMapping[provider]; ok {
		if status, ok := mapping[carrierStatus]; ok {
			return status
		}
	}
	return StatusPending
}
