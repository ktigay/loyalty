package order

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/order"
)

// Service Интерфейс сервиса заказов.
//
//go:generate mockgen -destination=./mocks/mock_service.go -package=mocks github.com/ktigay/loyalty/internal/handler/order Service
type Service interface {
	Create(ctx context.Context, userUUID string, orderID entity.Number) (*entity.Order, error)
	OrdersByUser(ctx context.Context, userUUID string) ([]entity.Order, error)
}

// Handler Структура обработки заказов.
type Handler struct {
	orderSv Service
	logger  *slog.Logger
}

// CreateOrderHandler Обработчик создания заказа.
func (o *Handler) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	var (
		orderID  []byte
		err      error
		identity *entity.Identity
		ctx      context.Context
	)

	ctx = r.Context()
	identity = security.UserIdentityFromCtx(ctx)

	if orderID, err = io.ReadAll(r.Body); err != nil {
		o.logger.Error("Failed to read order id", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		if err = r.Body.Close(); err != nil {
			o.logger.Error("Failed to close request body", "err", err)
		}
	}()

	if _, err = o.orderSv.Create(ctx, identity.UUID, entity.Number(orderID)); err != nil {
		o.logger.Error("Failed to create order", "err", err)

		switch {
		case errors.Is(err, order.ErrOrderAlreadyExists):
			w.WriteHeader(http.StatusOK)
			return
		case errors.Is(err, order.ErrWrongOrderNumber):
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		case errors.Is(err, order.ErrWrongUser):
			w.WriteHeader(http.StatusConflict)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// GetOrdersHandler Обработчик получения списка заказов.
func (o *Handler) GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	var (
		identity *entity.Identity
		ctx      context.Context
		orders   []entity.Order
	)

	ctx = r.Context()
	identity = security.UserIdentityFromCtx(ctx)

	ordersByUser, err := o.orderSv.OrdersByUser(ctx, identity.UUID)
	if err != nil && !errors.Is(err, order.ErrOrdersNotFound) {
		o.logger.Error("Failed to get orders", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ordersByUser != nil {
		orders = ordersByUser
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := make([]api.Order, 0, len(orders))
	for _, ord := range orders {
		resp = append(resp, orderToAPI(ord))
	}

	if err = json.NewEncoder(w).Encode(resp); err != nil {
		o.logger.Error("Failed to encode response", "err", err)
	}
}

// New Конструктор.
func New(o Service, logger *slog.Logger) *Handler {
	return &Handler{
		orderSv: o,
		logger:  logger,
	}
}
