package entity

import (
	"strconv"
	"time"

	"github.com/ktigay/loyalty/internal/api"
)

// OrderStatus Статус заказа.
type OrderStatus string

const (
	NEW        OrderStatus = "NEW"
	PROCESSING OrderStatus = "PROCESSING"
	INVALID    OrderStatus = "INVALID"
	PROCESSED  OrderStatus = "PROCESSED"
)

// Order Сущность заказа.
type Order struct {
	ID          int64
	UserUUID    string
	OrderID     string
	Status      OrderStatus
	Accrual     *int64
	UploadedAt  time.Time
	ProcessedAt time.Time
}

// ToAPI В Api.
func (o Order) ToAPI() api.Order {
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

// Number Номер.
type Number string

// String В строку.
func (o Number) String() string {
	return string(o)
}

// IsValid Проверка на алгоритм Луна.
func (o Number) IsValid() bool {
	var (
		number int
		err    error
	)
	if number, err = strconv.Atoi(string(o)); err != nil {
		return false
	}
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int) int {
	var luhn int
	for i := 0; number > 0; i++ {
		cur := number % 10
		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}
		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
