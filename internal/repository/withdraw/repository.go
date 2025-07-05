package withdraw

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
)

var (
	insertQuery = `
		INSERT INTO withdrawals ("user_uuid", "order_id", "sum")
			VALUES ($1, $2, $3)
		RETURNING 
			"uuid", "user_uuid", "order_id", "sum", "created_at", "processed_at"
	`
	getWithdrawalsQuery = `
		SELECT 
			"uuid", "user_uuid", "order_id", "sum", "created_at", "processed_at"
		FROM withdrawals WHERE "user_uuid" = $1
		ORDER BY "created_at" DESC
	`
)

// Repository Репозиторий списаний.
type Repository struct {
	db     *db.ConnWrapper
	logger *slog.Logger
}

// Create Создаёт запись о списании баллов.
func (r *Repository) Create(ctx context.Context, userUUID, orderID string, sum int64) (*entity.Withdrawal, error) {
	var (
		w   entity.Withdrawal
		err error
	)

	if err = r.fullScan(
		r.db.Connection(ctx).QueryRow(ctx, insertQuery, userUUID, orderID, sum),
		&w,
	); err != nil {
		return nil, err
	}

	return &w, err
}

// GetWithdrawals Списания баллов.
func (r *Repository) GetWithdrawals(ctx context.Context, userUUID string) (*[]entity.Withdrawal, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	var (
		err         error
		withdrawals []entity.Withdrawal
		rows        pgx.Rows
	)

	if rows, err = r.db.Connection(ctx).Query(c, getWithdrawalsQuery, userUUID); err != nil {
		return nil, err
	}
	defer rows.Close()

	withdrawals = make([]entity.Withdrawal, 0)
	for rows.Next() {
		var w entity.Withdrawal
		if err = r.fullScan(rows, &w); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, w)
	}

	return &withdrawals, nil
}

func (r *Repository) fullScan(row pgx.Row, w *entity.Withdrawal) error {
	return row.Scan(
		&w.UUID,
		&w.UserUUID,
		&w.OrderID,
		&w.Sum,
		&w.CreatedAt,
		&w.ProcessedAt,
	)
}

// New Конструктор.
func New(conn db.ConnInterface, logger *slog.Logger) *Repository {
	return &Repository{
		db:     db.NewConnWrapper(conn),
		logger: logger,
	}
}
