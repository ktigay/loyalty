package order

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/entity"
)

var (
	ErrWrongOrderNumber   = errors.New("wrong order number")
	ErrWrongUser          = errors.New("wrong user order")
	ErrOrderAlreadyExists = errors.New("order already exists")
	ErrOrdersNotFound     = errors.New("orders not found")
)

// OrdersGetter Интерфейс сервиса AccrualOrder.
//
//go:generate mockgen -destination=./mocks/mock_accrual.go -package=mocks github.com/ktigay/loyalty/internal/service/order OrdersGetter
type OrdersGetter interface {
	GetOrders(ctx context.Context, ids ...string) ([]entity.AccrualOrder, error)
}

// Hydrator Интерфейс гидратора.
//
//go:generate mockgen -destination=./mocks/mock_hydrator.go -package=mocks github.com/ktigay/loyalty/internal/service/order Hydrator
type Hydrator interface {
	Hydrate(e []entity.Order, acc []entity.AccrualOrder) ([]entity.Order, error)
}

// Repository Интерфейс репозитория.
//
//go:generate mockgen -destination=./mocks/mock_orderrepo.go -package=mocks github.com/ktigay/loyalty/internal/service/order Repository
type Repository interface {
	Order(ctx context.Context, orderID string) (*entity.Order, error)
	Create(ctx context.Context, userUUID, orderID string) (*entity.Order, error)
	UpdateAll(ctx context.Context, orders []entity.Order) ([]entity.Order, error)
	OrdersByUser(ctx context.Context, userUUID string) ([]entity.Order, error)
	OrdersByStatus(ctx context.Context, st ...entity.OrderStatus) ([]entity.Order, error)
}

// Service Сервис заказов.
type Service struct {
	orderRepo Repository
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
func (s *Service) OrdersByUser(ctx context.Context, userUUID string) ([]entity.Order, error) {
	orders, err := s.orderRepo.OrdersByUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrdersNotFound
		}
		return nil, err
	}
	return orders, err
}

// OrdersByStatus Заказы по статусам.
func (s *Service) OrdersByStatus(ctx context.Context, st ...entity.OrderStatus) ([]entity.Order, error) {
	return s.orderRepo.OrdersByStatus(ctx, st...)
}

// UpdateAll Сохранение заказов.
func (s *Service) UpdateAll(ctx context.Context, orders []entity.Order) ([]entity.Order, error) {
	return s.orderRepo.UpdateAll(ctx, orders)
}

// New Конструктор.
func New(o Repository, l *slog.Logger) *Service {
	return &Service{
		orderRepo: o,
		logger:    l,
	}
}

// StatusGetterService Сервис актуализации статусов заказов.
type StatusGetterService struct {
	client   OrdersGetter
	hydrator Hydrator
	logger   *slog.Logger
}

// ReceiveStatus Получение новых статусов заказов.
func (s *StatusGetterService) ReceiveStatus(ctx context.Context, orders []entity.Order) ([]entity.Order, error) {
	var (
		updated []entity.Order
		acc     []entity.AccrualOrder
		err     error
	)

	ids := make([]string, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.OrderID)
	}
	if acc, err = s.client.GetOrders(ctx, ids...); err != nil {
		s.logger.Error("Getting orders info finished with", "err", err)
		return nil, err
	}

	updated, err = s.hydrator.Hydrate(orders, acc)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// NewStatusGetterService Конструктор.
func NewStatusGetterService(c OrdersGetter, h Hydrator, l *slog.Logger) *StatusGetterService {
	return &StatusGetterService{
		client:   c,
		hydrator: h,
		logger:   l,
	}
}
