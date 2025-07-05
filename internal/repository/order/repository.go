package order

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"unsafe"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
)

var (
	insertQuery = `
		INSERT INTO orders ("user_uuid", "order_id", "status")
			VALUES ($1, $2, $3) RETURNING "id", "user_uuid", "order_id", "status", "accrual", "uploaded_at", "processed_at"`

	updateQuery = `
		UPDATE orders 
		SET 
		    "status" = $1, "accrual" = $2, "processed_at" = NOW()
		WHERE "order_id" = $3
		RETURNING "id", "user_uuid", "order_id", "status", "accrual", "uploaded_at", "processed_at"`

	selectByIDQuery = `
		SELECT "id", "user_uuid", "order_id", "status", "accrual", "uploaded_at", "processed_at" FROM orders WHERE "order_id" = $1`

	selectByIDsQueryList = `
		SELECT "order_id" FROM orders WHERE "order_id" = ANY($1::varchar[]) AND "status" = ANY($2::order_status_type[])`

	selectByUserAndIDQuery = `
		SELECT "id", "user_uuid", "order_id", "status", "accrual", "uploaded_at", "processed_at" FROM orders WHERE "user_uuid" = $1 AND "order_id" = $2`

	selectByUserQuery = `
		SELECT "id", "user_uuid", "order_id", "status", "accrual", "uploaded_at", "processed_at" FROM orders WHERE "user_uuid" = $1 ORDER BY "uploaded_at" DESC`

	selectByStatusQuery = `
		SELECT "id", "user_uuid", "order_id", "status", "accrual", "uploaded_at", "processed_at" FROM orders WHERE "status" = ANY($1::order_status_type[]) ORDER BY "uploaded_at"`
)

// Repository Репозиторий заказов.
type Repository struct {
	db     *db.ConnWrapper
	logger *slog.Logger
}

// Order Заказ по orderID.
func (r *Repository) Order(ctx context.Context, orderID string) (*entity.Order, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	var order entity.Order
	err := r.fullScan(
		r.db.Connection(ctx).QueryRow(c, selectByIDQuery, orderID),
		&order,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &order, nil
}

// OrdersList Заказы по orderID.
func (r *Repository) OrdersList(ctx context.Context, orderIDs []string, st ...entity.OrderStatus) ([]string, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	statuses := *(*[]string)(unsafe.Pointer(&st))
	rows, err := r.db.Connection(ctx).Query(
		c,
		selectByIDsQueryList,
		"{"+strings.Join(orderIDs, ",")+"}",
		"{"+strings.Join(statuses, ",")+"}",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]string, 0)
	for rows.Next() {
		var order string
		if err = rows.Scan(&order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return orders, nil
}

// OrderByUser Заказ по userUUID и orderID.
func (r *Repository) OrderByUser(ctx context.Context, userUUID, orderID string) (*entity.Order, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	return r.queryRow(c, selectByUserAndIDQuery, userUUID, orderID)
}

// OrdersByUser Заказы по userUUID.
func (r *Repository) OrdersByUser(ctx context.Context, userUUID string) (*[]entity.Order, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	rows, err := r.db.Connection(ctx).Query(c, selectByUserQuery, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]entity.Order, 0)

	for rows.Next() {
		var order entity.Order
		err = r.fullScan(
			rows,
			&order,
		)
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return &orders, nil
}

// OrdersByStatus Заказы по статусам.
func (r *Repository) OrdersByStatus(ctx context.Context, st ...entity.OrderStatus) (*[]entity.Order, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	statuses := *(*[]string)(unsafe.Pointer(&st))
	rows, err := r.db.Connection(ctx).Query(c, selectByStatusQuery, "{"+strings.Join(statuses, ",")+"}")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]entity.Order, 0)

	for rows.Next() {
		var order entity.Order
		err = r.fullScan(
			rows,
			&order,
		)
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return &orders, nil
}

// Create Создаёт заказ.
func (r *Repository) Create(ctx context.Context, userUUID, orderID string) (*entity.Order, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	return r.queryRow(c, insertQuery, userUUID, orderID, entity.NEW)
}

// Update Обновляет заказ.
func (r *Repository) Update(ctx context.Context, order entity.Order) (*entity.Order, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	return r.queryRow(c, updateQuery, order.Status, order.Accrual, order.OrderID)
}

// UpdateAll Обновляет заказы.
func (r *Repository) UpdateAll(ctx context.Context, orders []entity.Order) (*[]entity.Order, error) {
	c, cancel := context.WithTimeout(ctx, db.RequestTimeout)
	defer cancel()

	var (
		err       error
		newOrders []entity.Order
	)

	batch := &pgx.Batch{}
	for _, order := range orders {
		batch.Queue(updateQuery, order.Status, order.Accrual, order.OrderID)
	}

	results := r.db.Connection(ctx).SendBatch(c, batch)
	defer func() {
		if err = results.Close(); err != nil {
			r.logger.Warn("pgx.BatchResults close with", "err", err)
		}
	}()

	newOrders = make([]entity.Order, 0, len(orders))
	for i := 0; i < len(orders); i++ {
		row := results.QueryRow()

		newOrder := entity.Order{}
		err = r.fullScan(row, &newOrder)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		newOrders = append(newOrders, newOrder)
	}

	return &newOrders, nil
}

func (r *Repository) queryRow(ctx context.Context, query string, args ...any) (*entity.Order, error) {
	var (
		order entity.Order
		err   error
	)
	if err = r.fullScan(
		r.db.Connection(ctx).QueryRow(ctx, query, args...),
		&order,
	); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *Repository) fullScan(row pgx.Row, order *entity.Order) error {
	return row.Scan(
		&order.ID,
		&order.UserUUID,
		&order.OrderID,
		&order.Status,
		&order.Accrual,
		&order.UploadedAt,
		&order.ProcessedAt,
	)
}

// New Конструктор.
func New(conn db.ConnInterface, logger *slog.Logger) *Repository {
	return &Repository{
		db:     db.NewConnWrapper(conn),
		logger: logger,
	}
}
