package balance

import (
	"context"
	"log/slog"

	"github.com/ktigay/loyalty/internal/entity"
)

// Repository Интерфейс репозитория.
type Repository interface {
	Balance(ctx context.Context, userUUID string) (*entity.Balance, error)
}

// Service Сервис баланса.
type Service struct {
	balanceRepo Repository
	logger      *slog.Logger
}

// Balance Возвращает баланс пользователя.
func (s *Service) Balance(ctx context.Context, userUUID string) (*entity.Balance, error) {
	return s.balanceRepo.Balance(ctx, userUUID)
}

// New Конструктор.
func New(b Repository, l *slog.Logger) *Service {
	return &Service{
		balanceRepo: b,
		logger:      l,
	}
}
