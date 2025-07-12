package entity

import (
	"time"
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
