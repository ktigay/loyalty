package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dbKey int

const (
	txInContext dbKey = iota
)

// TxFacade Интерфейс для работы с транзакциями.
//
//go:generate mockgen -destination=./mocks/mock_tx.go -package=mocks github.com/ktigay/loyalty/internal/db TxFacade
type TxFacade interface {
	RunInTx(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error
}

// PgxConn Интерфейс pgx для работы с транзакциями.
//
//go:generate mockgen -destination=./mocks/mock_conn.go -package=mocks github.com/ktigay/loyalty/internal/db PgxConn
type PgxConn interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// PgxTxFacade Структура для работы с транзакциями.
type PgxTxFacade struct {
	pgxConn PgxConn
}

// RunInTx Выполняет функцию внутри транзакции.
func (t PgxTxFacade) RunInTx(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
	tx, err := t.pgxConn.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil {
			err = errors.Join(err, rollbackErr)
		}
	}()

	ctxWithTx := txWithContext(ctx, tx)
	if err = fn(ctxWithTx); err == nil {
		err = tx.Commit(ctx)
	}

	return err
}

// NewPgxTxFacade Конструктор.
func NewPgxTxFacade(pool *pgxpool.Pool) *PgxTxFacade {
	return &PgxTxFacade{pgxConn: pool}
}

func txWithContext(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txInContext, tx)
}

// TxFromContext Транзакция из контекста.
func TxFromContext(ctx context.Context) pgx.Tx {
	pgTx, _ := ctx.Value(txInContext).(pgx.Tx)
	return pgTx
}

// TxConnWrapper Обёртка для работы с БД.
type TxConnWrapper struct {
	db Conn
}

// Connection Возвращает [Conn].
// Если есть транзакция, то возвращается транзакция. Иначе обычное соединение к БД.
func (c *TxConnWrapper) Connection(ctx context.Context) Conn {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return c.db
}

// NewTxConnWrapper Конструктор.
func NewTxConnWrapper(conn Conn) *TxConnWrapper {
	return &TxConnWrapper{
		db: conn,
	}
}
