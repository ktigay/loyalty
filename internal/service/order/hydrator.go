package order

import (
	"math"

	"github.com/ktigay/loyalty/internal/entity"
)

// AccrualHydrator Гидратор для обновления статуса заказов.
type AccrualHydrator struct{}

// Hydrate Гидрирует сущности заказов данными из сервиса accrual.
func (a AccrualHydrator) Hydrate(orders []entity.Order, acc []entity.Accrual) (*[]entity.Order, error) {
	ln := len(orders)

	accMap := make(map[string]*entity.Order, ln)
	for idx, o := range orders {
		accMap[o.OrderID] = &orders[idx]
	}

	for _, order := range acc {
		ao, ok := accMap[order.OrderID]
		if !ok {
			continue
		}
		ao.Status = mapStatus(order.Status)

		v := order.Accrual
		if v != nil {
			n := int64(math.Round(*order.Accrual * 100))
			ao.Accrual = &n
		}
	}

	return &orders, nil
}

func mapStatus(accStatus string) entity.OrderStatus {
	switch accStatus {
	case "INVALID":
		return entity.INVALID
	case "PROCESSING":
		return entity.PROCESSING
	case "PROCESSED":
		return entity.PROCESSED
	}
	return entity.NEW
}

// NewAccrualHydrator Конструктор.
func NewAccrualHydrator() *AccrualHydrator {
	return &AccrualHydrator{}
}
