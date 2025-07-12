package balance

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
)

var (
	insertQuery = `
	INSERT INTO balance ("user_uuid")
		VALUES ($1)`

	selectByUUIDQuery = `
		SELECT 
		    id, user_uuid, current, withdrawn, updated_at, created_at
		FROM balance WHERE user_uuid = $1
	`

	incCurrentQuery = `
		UPDATE balance
		SET 
			"current" = balance.current + $1,
			"updated_at" = NOW()
		WHERE
			user_uuid = $2
		RETURNING
		    id, user_uuid, current, withdrawn, updated_at, created_at
	`

	incWithdrawnQuery = `
		UPDATE balance
		SET 
			"current" = balance.current - $1,
			"withdrawn" = balance.withdrawn + $2,
			"updated_at" = NOW()
		WHERE
			user_uuid = $3
		RETURNING 
		    id, user_uuid, current, withdrawn, updated_at, created_at
	`
)

// Repository Репозиторий баланса.
type Repository struct {
	db     db.ConnWrapper
	logger *slog.Logger
}

// Create Создание записи.
func (r *Repository) Create(ctx context.Context, userUUID string) error {
	_, err := r.db.Connection(ctx).Exec(ctx, insertQuery, userUUID)

	return err
}

// Balance Баланс пользователя.
func (r *Repository) Balance(ctx context.Context, userUUID string) (*entity.Balance, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	return r.queryRow(c, selectByUUIDQuery, userUUID)
}

// IncreaseCurrent Зачисление баллов на баланс.
func (r *Repository) IncreaseCurrent(ctx context.Context, userUUID string, delta int64) (*entity.Balance, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	return r.queryRow(c, incCurrentQuery, delta, userUUID)
}

// IncreaseWithdraw Списание баллов с баланса.
func (r *Repository) IncreaseWithdraw(ctx context.Context, userUUID string, delta int64) (*entity.Balance, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	return r.queryRow(c, incWithdrawnQuery, delta, delta, userUUID)
}

func (r *Repository) queryRow(ctx context.Context, query string, args ...any) (*entity.Balance, error) {
	var (
		balance entity.Balance
		err     error
	)

	if err = r.fullScan(
		r.db.Connection(ctx).QueryRow(ctx, query, args...),
		&balance,
	); err != nil {
		return nil, err
	}
	return &balance, nil
}

func (r *Repository) fullScan(row pgx.Row, balance *entity.Balance) error {
	return row.Scan(
		&balance.ID,
		&balance.UserUUID,
		&balance.Current,
		&balance.Withdrawn,
		&balance.UpdatedAt,
		&balance.CreatedAt,
	)
}

// New Конструктор.
func New(db db.ConnWrapper, logger *slog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}
