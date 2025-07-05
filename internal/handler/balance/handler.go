package balance

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/balance"
	"github.com/ktigay/loyalty/internal/service/order"
	"github.com/ktigay/loyalty/internal/service/withdraw"
)

// OrderServiceInterface Интерфейс сервиса заказов.
//
//go:generate mockgen -destination=./mocks/mock_orderservice.go -package=mocks github.com/ktigay/loyalty/internal/handler/balance OrderServiceInterface
type OrderServiceInterface interface {
	Create(ctx context.Context, userUUID string, orderID entity.Number) (*entity.Order, error)
}

// ServiceInterface Интерфейс сервиса балансов.
//
//go:generate mockgen -destination=./mocks/mock_service.go -package=mocks github.com/ktigay/loyalty/internal/handler/balance ServiceInterface
type ServiceInterface interface {
	Balance(ctx context.Context, userUUID string) (*entity.Balance, error)
}

// WithdrawServiceInterface Интерфейс сервиса списаний.
//
//go:generate mockgen -destination=./mocks/mock_withdrawservice.go -package=mocks github.com/ktigay/loyalty/internal/handler/balance WithdrawServiceInterface
type WithdrawServiceInterface interface {
	MakeWithdraw(ctx context.Context, userUUID, orderID string, delta int64) (*entity.Withdrawal, error)
	Withdrawals(ctx context.Context, userUUID string) (*[]entity.Withdrawal, error)
}

// Handler Обработчик балансов.
type Handler struct {
	orderSv    OrderServiceInterface
	balanceSv  ServiceInterface
	withdrawSv WithdrawServiceInterface
	logger     *slog.Logger
}

// GetBalanceHandler Обработчик получения баланса.
func (b *Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      context.Context
		identity *entity.Identity
		bl       *entity.Balance
		err      error
	)

	ctx = r.Context()
	identity = security.UserIdentityFromCtx(ctx)

	if bl, err = b.balanceSv.Balance(ctx, identity.UUID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := api.Balance{
		Current:   float64(bl.Current) / 100,
		Withdrawn: float64(bl.Withdrawn) / 100,
	}
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		b.logger.Error("encode balance response", "err", err)
	}
}

// BalanceWithdrawHandler Обработчик создания списания.
func (b *Handler) BalanceWithdrawHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      context.Context
		reg      api.PostUserBalanceWithdrawJSONBody
		identity *entity.Identity
		err      error
	)
	ctx = r.Context()

	if err = json.NewDecoder(r.Body).Decode(&reg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	identity = security.UserIdentityFromCtx(ctx)

	// хоть это и операция баланса, но надо создать заказ.
	if _, err = b.orderSv.Create(ctx, identity.UUID, entity.Number(reg.Order)); err != nil {
		if !errors.Is(err, order.ErrOrderAlreadyExists) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	delta := int64(math.Round(reg.Sum * 100))

	if _, err = b.withdrawSv.MakeWithdraw(r.Context(), identity.UUID, reg.Order, delta); err != nil {
		switch {
		case errors.Is(err, withdraw.ErrNotEnoughBalance):
			w.WriteHeader(http.StatusPaymentRequired)
			return
		case errors.Is(err, withdraw.ErrWrongOrderNumber):
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// GetWithdrawalsHandler Обработчик получения списка списаний.
func (b *Handler) GetWithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx         context.Context
		identity    *entity.Identity
		err         error
		withdrawals *[]entity.Withdrawal
	)
	ctx = r.Context()

	identity = security.UserIdentityFromCtx(ctx)

	if withdrawals, err = b.withdrawSv.Withdrawals(r.Context(), identity.UUID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if withdrawals == nil || len(*withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := make([]api.Withdrawal, 0, len(*withdrawals))
	for _, wd := range *withdrawals {
		resp = append(resp, wd.ToAPI())
	}
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		b.logger.Error("Failed to encode response", "err", err)
	}
}

// New Конструктор.
func New(pool *pgxpool.Pool, logger *slog.Logger) *Handler {
	return &Handler{
		orderSv:    order.New(pool, logger),
		balanceSv:  balance.New(pool, logger),
		withdrawSv: withdraw.New(pool, logger),
		logger:     logger,
	}
}
