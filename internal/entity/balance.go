package entity

import "time"

// Balance Сущность баланс.
type Balance struct {
	ID        int64
	UserUUID  string
	Current   int64
	Withdrawn int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
