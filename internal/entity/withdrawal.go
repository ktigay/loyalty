package entity

import (
	"time"

	"github.com/ktigay/loyalty/internal/api"
)

// Withdrawal Структура списание баллов.
type Withdrawal struct {
	UUID        string
	UserUUID    string
	OrderID     string
	Sum         int64
	CreatedAt   time.Time
	ProcessedAt time.Time
}

// ToAPI В Api.
func (w Withdrawal) ToAPI() api.Withdrawal {
	return api.Withdrawal{
		Order:       w.OrderID,
		ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
		Sum:         float64(w.Sum) / 100,
	}
}
