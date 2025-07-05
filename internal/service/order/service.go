package order

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/entity"
	repo "github.com/ktigay/loyalty/internal/repository/order"
)

var (
	ErrWrongOrderNumber   = errors.New("wrong order number")
	ErrWrongUser          = errors.New("wrong user order")
	ErrOrderAlreadyExists = errors.New("order already exists")
)

// AccrualClientInterface Интерфейс сервиса Accrual.
//
//go:generate mockgen -destination=./mocks/mock_accrual.go -package=mocks github.com/ktigay/loyalty/internal/service/order AccrualClientInterface
type AccrualClientInterface interface {
	OrdersStatus(ctx context.Context, orderIDs []string) (*[]entity.Accrual, error)
}

// HydratorInterface Интерфейс гидратора.
//
//go:generate mockgen -destination=./mocks/mock_hydrator.go -package=mocks github.com/ktigay/loyalty/internal/service/order HydratorInterface
type HydratorInterface interface {
	Hydrate(e []entity.Order, acc []entity.Accrual) (*[]entity.Order, error)
}

// RepositoryInterface Интерфейс репозитория.
//
//go:generate mockgen -destination=./mocks/mock_orderrepo.go -package=mocks github.com/ktigay/loyalty/internal/service/order RepositoryInterface
type RepositoryInterface interface {
	Order(ctx context.Context, orderID string) (*entity.Order, error)
	Create(ctx context.Context, userUUID, orderID string) (*entity.Order, error)
	UpdateAll(ctx context.Context, orders []entity.Order) (*[]entity.Order, error)
	OrdersByUser(ctx context.Context, userUUID string) (*[]entity.Order, error)
	OrdersByStatus(ctx context.Context, st ...entity.OrderStatus) (*[]entity.Order, error)
}

// Service Сервис заказов.
type Service struct {
	orderRepo RepositoryInterface
	logger    *slog.Logger
}

// Create Создает заказ.
func (s *Service) Create(ctx context.Context, userUUID string, orderID entity.Number) (*entity.Order, error) {
	if !orderID.IsValid() {
		return nil, ErrWrongOrderNumber
	}

	var (
		od  *entity.Order
		err error
	)

	if od, err = s.orderRepo.Order(ctx, string(orderID)); err != nil {
		return nil, err
	}

	if od != nil {
		if od.UserUUID != userUUID {
			return nil, ErrWrongUser
		}

		return nil, ErrOrderAlreadyExists
	}

	return s.orderRepo.Create(ctx, userUUID, string(orderID))
}

// OrdersByUser Заказы пользователя.
func (s *Service) OrdersByUser(ctx context.Context, userUUID string) (*[]entity.Order, error) {
	orders, err := s.orderRepo.OrdersByUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &[]entity.Order{}, nil
		}
		return nil, err
	}
	return orders, err
}

// OrdersByStatus Заказы по статусам.
func (s *Service) OrdersByStatus(ctx context.Context, st ...entity.OrderStatus) (*[]entity.Order, error) {
	return s.orderRepo.OrdersByStatus(ctx, st...)
}

// UpdateAll Сохранение заказов.
func (s *Service) UpdateAll(ctx context.Context, orders []entity.Order) (*[]entity.Order, error) {
	return s.orderRepo.UpdateAll(ctx, orders)
}

// New Конструктор.
func New(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{
		orderRepo: repo.New(pool, logger),
		logger:    logger,
	}
}

// StatusGetterService Сервис актуализации статусов заказов.
type StatusGetterService struct {
	client   AccrualClientInterface
	hydrator HydratorInterface
	logger   *slog.Logger
}

// ReceiveStatus Получение новых статусов заказов.
func (s *StatusGetterService) ReceiveStatus(ctx context.Context, orders []entity.Order) (*[]entity.Order, error) {
	var (
		updated *[]entity.Order
		acc     *[]entity.Accrual
		err     error
	)

	ids := make([]string, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.OrderID)
	}
	if acc, err = s.client.OrdersStatus(ctx, ids); err != nil {
		s.logger.Error("Getting orders info finished with", "err", err)
	}
	if acc == nil || len(*acc) == 0 {
		s.logger.Debug("Getting orders info finished with no accrual")
		return nil, nil
	}

	updated, err = s.hydrator.Hydrate(orders, *acc)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// NewStatusGetterService Конструктор.
func NewStatusGetterService(client AccrualClientInterface, logger *slog.Logger) *StatusGetterService {
	return &StatusGetterService{
		client:   client,
		hydrator: NewAccrualHydrator(),
		logger:   logger,
	}
}
