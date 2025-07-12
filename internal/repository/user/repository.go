package user

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
)

var (
	insertQuery = `
	INSERT INTO users ("login", "password")
		VALUES ($1, $2) RETURNING "uuid", "login", "password", "updated_at", "created_at"`

	selectByUUIDQuery = `
	SELECT "uuid", "login", "password", "updated_at", "created_at" FROM users WHERE "uuid" = $1`

	selectByLoginQuery = `
	SELECT "uuid", "login", "password", "updated_at", "created_at" FROM users WHERE "login" = $1`
)

// Repository Репозиторий пользователи.
type Repository struct {
	db     db.ConnWrapper
	logger *slog.Logger
}

// UserByID Пользователь по uuid.
func (r *Repository) UserByID(ctx context.Context, uuid string) (*entity.User, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	var (
		user entity.User
		err  error
	)
	if err = r.fullScan(
		r.db.Connection(ctx).QueryRow(c, selectByUUIDQuery, uuid),
		&user,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

// UserByLogin Пользователь по логину.
func (r *Repository) UserByLogin(ctx context.Context, login string) (*entity.User, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	var (
		user entity.User
		err  error
	)
	if err = r.fullScan(
		r.db.Connection(ctx).QueryRow(c, selectByLoginQuery, login),
		&user,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser Создаёт запись о пользователе.
func (r *Repository) CreateUser(ctx context.Context, login, password string) (*entity.User, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	var (
		newUser entity.User
		err     error
	)

	if err = r.fullScan(
		r.db.Connection(ctx).QueryRow(c, insertQuery, login, password),
		&newUser,
	); err != nil {
		return nil, err
	}

	return &newUser, nil
}

func (r *Repository) fullScan(row pgx.Row, user *entity.User) error {
	return row.Scan(
		&user.UUID,
		&user.Login,
		&user.Password,
		&user.UpdatedAt,
		&user.CreatedAt,
	)
}

// New Конструктор.
func New(db db.ConnWrapper, logger *slog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}
