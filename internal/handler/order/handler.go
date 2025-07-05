package order

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/order"
)

// ServiceInterface Интерфейс сервиса заказов.
//
//go:generate mockgen -destination=./mocks/mock_service.go -package=mocks github.com/ktigay/loyalty/internal/handler/order ServiceInterface
type ServiceInterface interface {
	Create(ctx context.Context, userUUID string, orderID entity.Number) (*entity.Order, error)
	OrdersByUser(ctx context.Context, userUUID string) (*[]entity.Order, error)
}

// Handler Структура обработки заказов.
type Handler struct {
	orderSv ServiceInterface
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
		err      error
		identity *entity.Identity
		ctx      context.Context
		orders   *[]entity.Order
	)

	ctx = r.Context()
	identity = security.UserIdentityFromCtx(ctx)

	if orders, err = o.orderSv.OrdersByUser(ctx, identity.UUID); err != nil {
		o.logger.Error("Failed to get orders", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if orders == nil {
		orders = &[]entity.Order{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := make([]api.Order, 0, len(*orders))
	for _, ord := range *orders {
		resp = append(resp, ord.ToAPI())
	}

	if err = json.NewEncoder(w).Encode(resp); err != nil {
		o.logger.Error("Failed to encode response", "err", err)
	}
}

// New Конструктор.
func New(pool *pgxpool.Pool, logger *slog.Logger) *Handler {
	return &Handler{
		orderSv: order.New(pool, logger),
		logger:  logger,
	}
}
