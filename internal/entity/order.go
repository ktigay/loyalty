package entity

import (
	"strconv"
	"time"
)

// OrderStatus Статус заказа.
type OrderStatus string

const (
	New        OrderStatus = "NEW"
	Processing OrderStatus = "PROCESSING"
	Invalid    OrderStatus = "INVALID"
	Processed  OrderStatus = "PROCESSED"
)

// Order Сущность заказа.
type Order struct {
	ID          int64
	UserUUID    string
	OrderID     string
	Status      OrderStatus
	StatusPrev  OrderStatus
	Accrual     *int64
	UploadedAt  time.Time
	ProcessedAt time.Time
}

func (o Order) StatusChanged() bool {
	return o.Status != "" && o.Status != o.StatusPrev
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
