package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
)

var (
	ErrNotEnoughBalance = errors.New("not enough balance")
	ErrWrongOrderNumber = errors.New("wrong order number")
)

// Repository Интерфейс репозитория списаний.
//
//go:generate mockgen -destination=./mocks/mock_withdraw.go -package=mocks github.com/ktigay/loyalty/internal/service/withdraw Repository
type Repository interface {
	Create(ctx context.Context, userUUID, orderID string, sum int64) (*entity.Withdrawal, error)
	GetWithdrawals(ctx context.Context, userUUID string) ([]entity.Withdrawal, error)
}

// BalanceRepo Интерфейс репозитория балансов.
//
//go:generate mockgen -destination=./mocks/mock_balance.go -package=mocks github.com/ktigay/loyalty/internal/service/withdraw BalanceRepo
type BalanceRepo interface {
	Balance(ctx context.Context, userUUID string) (*entity.Balance, error)
	IncreaseWithdraw(ctx context.Context, userUUID string, delta int64) (*entity.Balance, error)
}

// OrderRepo Интерфейс репозитория пользователя.
//
//go:generate mockgen -destination=./mocks/mock_order.go -package=mocks github.com/ktigay/loyalty/internal/service/withdraw OrderRepo
type OrderRepo interface {
	OrderByUser(ctx context.Context, userUUID, orderID string) (*entity.Order, error)
}

// Service Сервис списаний.
type Service struct {
	withdrawRepo Repository
	balanceRepo  BalanceRepo
	orderRepo    OrderRepo
	pgxTx        db.TxFacade
	logger       *slog.Logger
}

// MakeWithdraw Проводит списание.
func (s *Service) MakeWithdraw(ctx context.Context, userUUID, orderID string, delta int64) (withdraw *entity.Withdrawal, err error) {
	if _, err = s.orderRepo.OrderByUser(ctx, userUUID, orderID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWrongOrderNumber
		}
		return nil, err
	}
	err = s.pgxTx.RunInTx(ctx, pgx.TxOptions{}, func(ctxWithTx context.Context) error {
		var bl *entity.Balance

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
	return
}

// Withdrawals Возвращает списания пользователя.
func (s *Service) Withdrawals(ctx context.Context, userUUID string) ([]entity.Withdrawal, error) {
	withdrawals, err := s.withdrawRepo.GetWithdrawals(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []entity.Withdrawal{}, nil
		}
		return nil, err
	}
	return withdrawals, nil
}

// New Конструктор.
func New(t db.TxFacade, w Repository, b BalanceRepo, o OrderRepo, l *slog.Logger) *Service {
	return &Service{
		pgxTx:        t,
		withdrawRepo: w,
		balanceRepo:  b,
		orderRepo:    o,
		logger:       l,
	}
}
