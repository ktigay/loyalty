package entity

import (
	"time"
)

// User Сущность пользователь.
type User struct {
	UUID      string
	Login     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
