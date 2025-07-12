package balance

import (
	"time"

	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
)

func withdrawalToAPI(w entity.Withdrawal) api.Withdrawal {
	return api.Withdrawal{
		Order:       w.OrderID,
		ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
		Sum:         float64(w.Sum) / 100,
	}
}
