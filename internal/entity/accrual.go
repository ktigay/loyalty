package entity

// Accrual Сущность начисление.
type Accrual struct {
	OrderID string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual"`
}
