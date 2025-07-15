package entity

// Identity Идентификатор пользователя.
type Identity struct {
	UUID string
}

// NewIdentity Конструктор.
func NewIdentity(uuid string) *Identity {
	return &Identity{
		UUID: uuid,
	}
}
