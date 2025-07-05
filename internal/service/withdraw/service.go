package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/repository/balance"
	"github.com/ktigay/loyalty/internal/repository/order"
	repo "github.com/ktigay/loyalty/internal/repository/withdraw"
)

var (
	ErrNotEnoughBalance = errors.New("not enough balance")
	ErrWrongOrderNumber = errors.New("wrong order number")
)

// RepositoryInterface Интерфейс репозитория списаний.
//
//go:generate mockgen -destination=./mocks/mock_withdraw.go -package=mocks github.com/ktigay/loyalty/internal/service/withdraw RepositoryInterface
type RepositoryInterface interface {
	Create(ctx context.Context, userUUID, orderID string, sum int64) (*entity.Withdrawal, error)
	GetWithdrawals(ctx context.Context, userUUID string) (*[]entity.Withdrawal, error)
}

// BalanceRepoInterface Интерфейс репозитория балансов.
//
//go:generate mockgen -destination=./mocks/mock_balance.go -package=mocks github.com/ktigay/loyalty/internal/service/withdraw BalanceRepoInterface
type BalanceRepoInterface interface {
	Balance(ctx context.Context, userUUID string) (*entity.Balance, error)
	IncreaseWithdraw(ctx context.Context, userUUID string, delta int64) (*entity.Balance, error)
}

// OrderRepoInterface Интерфейс репозитория пользователя.
//
//go:generate mockgen -destination=./mocks/mock_order.go -package=mocks github.com/ktigay/loyalty/internal/service/withdraw OrderRepoInterface
type OrderRepoInterface interface {
	OrderByUser(ctx context.Context, userUUID, orderID string) (*entity.Order, error)
}

// Service Сервис списаний.
type Service struct {
	withdrawRepo RepositoryInterface
	balanceRepo  BalanceRepoInterface
	orderRepo    OrderRepoInterface
	pgxTx        db.TxFacadeInterface
	pool         *pgxpool.Pool
	logger       *slog.Logger
}

// MakeWithdraw Проводит списание.
func (s *Service) MakeWithdraw(ctx context.Context, userUUID, orderID string, delta int64) (*entity.Withdrawal, error) {
	var (
		withdraw *entity.Withdrawal
		respErr  error
	)
	if _, respErr = s.orderRepo.OrderByUser(ctx, userUUID, orderID); respErr != nil {
		if errors.Is(respErr, pgx.ErrNoRows) {
			return nil, ErrWrongOrderNumber
		}
		return nil, respErr
	}
	respErr = s.pgxTx.RunInTx(ctx, pgx.TxOptions{}, func(ctxWithTx context.Context) error {
		var (
			err error
			bl  *entity.Balance
		)

		if bl, err = s.balanceRepo.Balance(ctxWithTx, userUUID); err != nil {
			return err
		}
		if bl.Current < delta {
			return ErrNotEnoughBalance
		}
		if _, err = s.balanceRepo.IncreaseWithdraw(ctxWithTx, userUUID, delta); err != nil {
			return err
		}
		if withdraw, err = s.withdrawRepo.Create(ctxWithTx, userUUID, orderID, delta); err != nil {
			return err
		}
		return nil
	})
	return withdraw, respErr
}

// Withdrawals Возвращает списания пользователя.
func (s *Service) Withdrawals(ctx context.Context, userUUID string) (*[]entity.Withdrawal, error) {
	withdrawals, err := s.withdrawRepo.GetWithdrawals(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &[]entity.Withdrawal{}, nil
		}
		return nil, err
	}
	return withdrawals, nil
}

// New Конструктор.
func New(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{
		withdrawRepo: repo.New(pool, logger),
		balanceRepo:  balance.New(pool, logger),
		orderRepo:    order.New(pool, logger),
		pgxTx:        db.NewPgxTxFacade(pool),
		pool:         pool,
		logger:       logger,
	}
}
