package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/accrual"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
	ordersv "github.com/ktigay/loyalty/internal/service/order"
)

//go:generate mockgen -destination=./mocks/mock_order.go -package=mocks github.com/ktigay/loyalty/internal/service/order/task OrderRepo
type OrderRepo interface {
	OrdersByStatus(ctx context.Context, st ...entity.OrderStatus) ([]entity.Order, error)
	UpdateAll(ctx context.Context, orders []entity.Order) ([]entity.Order, error)
}

//go:generate mockgen -destination=./mocks/mock_balance.go -package=mocks github.com/ktigay/loyalty/internal/service/order/task BalanceRepo
type BalanceRepo interface {
	IncreaseCurrent(ctx context.Context, userUUID string, delta int64) (*entity.Balance, error)
}

//go:generate mockgen -destination=./mocks/mock_status.go -package=mocks github.com/ktigay/loyalty/internal/service/order/task StatusGetter
type StatusGetter interface {
	ReceiveStatus(ctx context.Context, orders []entity.Order) ([]entity.Order, error)
}

type ActualizeInterval time.Duration

// ActualizeOrderTask Структура для обновления статуса заказов.
type ActualizeOrderTask struct {
	statusGetter StatusGetter
	orderRepo    OrderRepo
	balanceRepo  BalanceRepo
	pgxTx        db.TxFacade
	logger       *slog.Logger
	interval     ActualizeInterval
}

// ActualizeOrdersStatus Актуализация данных не обработанных заказов.
func (o *ActualizeOrderTask) ActualizeOrdersStatus(ctx, exitCtx context.Context) {
	t := time.NewTicker(time.Duration(o.interval))
	defer t.Stop()

	for {
		select {
		case <-exitCtx.Done():
			t.Stop()
			o.logger.Info("Exiting ActualizeOrdersStatus")
			return
		case <-t.C:
			o.actualize(ctx)
		}
	}
}

func (o *ActualizeOrderTask) actualize(ctx context.Context) {
	var (
		orders      []entity.Order
		received    []entity.Order
		updated     []entity.Order
		cntOrders   int
		cntReceived int
		cntUpdated  int
		err         error
	)

	o.logger.Debug("Actualizing order status")
	if orders, err = o.orderRepo.OrdersByStatus(ctx, entity.New, entity.Processing); err != nil {
		o.logger.Error("Failed to get orders", "err", err)
		return
	}

	cntOrders = len(orders)
	// Пытаемся получить новый статус из AccrualOrder.
	if received, err = o.statusGetter.ReceiveStatus(ctx, orders); err != nil {
		o.logger.Error("Failed to update orders status", "err", err)
		return
	}

	err = o.pgxTx.RunInTx(ctx, pgx.TxOptions{}, func(ctxWithTx context.Context) error {
		var txErr error
		cntReceived = len(received)

		if updated, txErr = o.orderRepo.UpdateAll(ctxWithTx, received); txErr != nil {
			o.logger.Error("Failed to update orders", "err", txErr)
		}

		cntUpdated = len(updated)
		for _, r := range updated {
			if r.Accrual == nil {
				continue
			}
			// Обновляем пользовательский баланс.
			if _, txErr = o.balanceRepo.IncreaseCurrent(ctxWithTx, r.UserUUID, *r.Accrual); txErr != nil {
				o.logger.Error("Failed to update user current", "err", txErr, "method", "IncreaseCurrent")
				return txErr
			}
		}
		return nil
	})
	if err != nil {
		o.logger.Error("Error while updating orders", "err", err)
	}

	o.logger.Debug("Actualized orders finished",
		"orders", cntOrders,
		"received", cntReceived,
		"updated", cntUpdated,
	)
}

// NewActualizeOrderTask Конструктор.
func NewActualizeOrderTask(
	wp *accrual.WorkerPoolClient,
	h ordersv.Hydrator,
	t db.TxFacade,
	o OrderRepo,
	b BalanceRepo,
	interval ActualizeInterval,
	l *slog.Logger,
) *ActualizeOrderTask {
	return &ActualizeOrderTask{
		statusGetter: ordersv.NewStatusGetterService(wp, h, l),
		pgxTx:        t,
		orderRepo:    o,
		balanceRepo:  b,
		logger:       l,
		interval:     interval,
	}
}
