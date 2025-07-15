package entity

// AccrualOrder Сущность начисление.
type AccrualOrder struct {
	OrderID string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual"`
}
