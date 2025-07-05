package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/accrual"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/repository/balance"
	"github.com/ktigay/loyalty/internal/repository/order"
	ordersv "github.com/ktigay/loyalty/internal/service/order"
)

//go:generate mockgen -destination=./mocks/mock_order.go -package=mocks github.com/ktigay/loyalty/internal/service/order/task OrderRepoInterface
type OrderRepoInterface interface {
	OrdersByStatus(ctx context.Context, st ...entity.OrderStatus) (*[]entity.Order, error)
	UpdateAll(ctx context.Context, orders []entity.Order) (*[]entity.Order, error)
}

//go:generate mockgen -destination=./mocks/mock_balance.go -package=mocks github.com/ktigay/loyalty/internal/service/order/task BalanceRepoInterface
type BalanceRepoInterface interface {
	IncreaseCurrent(ctx context.Context, userUUID string, delta int64) (*entity.Balance, error)
}

//go:generate mockgen -destination=./mocks/mock_status.go -package=mocks github.com/ktigay/loyalty/internal/service/order/task StatusGetterInterface
type StatusGetterInterface interface {
	ReceiveStatus(ctx context.Context, orders []entity.Order) (*[]entity.Order, error)
}

// ActualizeOrderTask Структура для обновления статуса заказов.
type ActualizeOrderTask struct {
	statusGetter StatusGetterInterface
	orderRepo    OrderRepoInterface
	balanceRepo  BalanceRepoInterface
	pgxTx        db.TxFacadeInterface
	logger       *slog.Logger
	interval     time.Duration
}

// ActualizeOrdersStatus Актуализация данных не обработанных заказов.
func (o *ActualizeOrderTask) ActualizeOrdersStatus(ctx, exitCtx context.Context) {
	t := time.NewTicker(o.interval)
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
		ordersPtr   *[]entity.Order
		receivedPtr *[]entity.Order
		updatedPtr  *[]entity.Order
		cntOrders   int
		cntReceived int
		cntUpdated  int
		err         error
	)

	statusesForUpdate := []entity.OrderStatus{entity.NEW, entity.PROCESSING}

	o.logger.Debug("Actualizing order status")
	ordersPtr, err = o.orderRepo.OrdersByStatus(ctx, statusesForUpdate...)
	if err != nil {
		o.logger.Error("Failed to get orders", "err", err)
		return
	}

	if ordersPtr != nil {
		cntOrders = len(*ordersPtr)
		// Пытаемся получить новый статус из Accrual.
		if receivedPtr, err = o.statusGetter.ReceiveStatus(ctx, *ordersPtr); err != nil {
			o.logger.Error("Failed to update ordersPtr status", "err", err)
		}
	}
	if receivedPtr == nil || len(*receivedPtr) == 0 {
		o.logger.Debug("Received no orders")
		return
	}

	_ = o.pgxTx.RunInTx(ctx, pgx.TxOptions{}, func(ctxWithTx context.Context) error {
		received := *receivedPtr
		cntReceived = len(received)

		if updatedPtr, err = o.orderRepo.UpdateAll(ctxWithTx, received); err != nil {
			o.logger.Error("Failed to update orders", "err", err)
		}
		if updatedPtr != nil {
			cntUpdated = len(*updatedPtr)
			for _, r := range *updatedPtr {
				if r.Accrual == nil {
					continue
				}
				// Обновляем пользовательский баланс.
				if _, err = o.balanceRepo.IncreaseCurrent(ctxWithTx, r.UserUUID, *r.Accrual); err != nil {
					o.logger.Error("Failed to update user current", "err", err, "method", "IncreaseCurrent")
					return err
				}
			}
		}
		return nil
	})

	o.logger.Debug("Actualized orders finished",
		"orders", cntOrders,
		"received", cntReceived,
		"updated", cntUpdated,
	)
}

// NewActualizeOrderTask Конструктор.
func NewActualizeOrderTask(accrualHost string, interval int, pool *pgxpool.Pool, logger *slog.Logger) *ActualizeOrderTask {
	return &ActualizeOrderTask{
		statusGetter: ordersv.NewStatusGetterService(
			accrual.New(accrualHost, logger),
			logger,
		),
		orderRepo: order.New(
			pool,
			logger,
		),
		balanceRepo: balance.New(pool, logger),
		pgxTx:       db.NewPgxTxFacade(pool),
		logger:      logger,
		interval:    time.Duration(interval) * time.Second,
	}
}
