package balance

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"

	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/order"
	"github.com/ktigay/loyalty/internal/service/withdraw"
)

// OrderService Интерфейс сервиса заказов.
//
//go:generate mockgen -destination=./mocks/mock_orderservice.go -package=mocks github.com/ktigay/loyalty/internal/handler/balance OrderService
type OrderService interface {
	Create(ctx context.Context, userUUID string, orderID entity.Number) (*entity.Order, error)
}

// Service Интерфейс сервиса балансов.
//
//go:generate mockgen -destination=./mocks/mock_service.go -package=mocks github.com/ktigay/loyalty/internal/handler/balance Service
type Service interface {
	Balance(ctx context.Context, userUUID string) (*entity.Balance, error)
}

// WithdrawService Интерфейс сервиса списаний.
//
//go:generate mockgen -destination=./mocks/mock_withdrawservice.go -package=mocks github.com/ktigay/loyalty/internal/handler/balance WithdrawService
type WithdrawService interface {
	MakeWithdraw(ctx context.Context, userUUID, orderID string, delta int64) (*entity.Withdrawal, error)
	Withdrawals(ctx context.Context, userUUID string) ([]entity.Withdrawal, error)
}

// Handler Обработчик балансов.
type Handler struct {
	orderSv    OrderService
	balanceSv  Service
	withdrawSv WithdrawService
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
		withdrawals []entity.Withdrawal
	)
	ctx = r.Context()

	identity = security.UserIdentityFromCtx(ctx)

	if withdrawals, err = b.withdrawSv.Withdrawals(r.Context(), identity.UUID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := make([]api.Withdrawal, 0, len(withdrawals))
	for _, wd := range withdrawals {
		resp = append(resp, withdrawalToAPI(wd))
	}
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		b.logger.Error("Failed to encode response", "err", err)
	}
}

// New Конструктор.
func New(o OrderService, b Service, w WithdrawService, logger *slog.Logger) *Handler {
	return &Handler{
		orderSv:    o,
		balanceSv:  b,
		withdrawSv: w,
		logger:     logger,
	}
}
