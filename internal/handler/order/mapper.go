package order

import (
	"time"

	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
)

func orderToAPI(o entity.Order) api.Order {
	var accrual *float64
	if o.Accrual != nil {
		v := float64(*o.Accrual) / 100
		accrual = &v
	}
	return api.Order{
		Accrual:    accrual,
		Number:     o.OrderID,
		Status:     api.OrderStatus(o.Status),
		UploadedAt: o.UploadedAt.Format(time.RFC3339),
	}
}
