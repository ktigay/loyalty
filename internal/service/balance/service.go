package balance

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/entity"
	repo "github.com/ktigay/loyalty/internal/repository/balance"
)

// RepositoryInterface Интерфейс репозитория.
type RepositoryInterface interface {
	Balance(ctx context.Context, userUUID string) (*entity.Balance, error)
}

// Service Сервис баланса.
type Service struct {
	balanceRepo RepositoryInterface
	pool        *pgxpool.Pool
	logger      *slog.Logger
}

// Balance Возвращает баланс пользователя.
func (s *Service) Balance(ctx context.Context, userUUID string) (*entity.Balance, error) {
	return s.balanceRepo.Balance(ctx, userUUID)
}

// New Конструктор.
func New(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{
		balanceRepo: repo.New(pool, logger),
		pool:        pool,
		logger:      logger,
	}
}
