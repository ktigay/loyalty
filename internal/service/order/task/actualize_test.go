package task

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	dbmocks "github.com/ktigay/loyalty/internal/db/mocks"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/service/order/task/mocks"
)

type fields struct {
	statusGetter func(ctrl *gomock.Controller) StatusGetterInterface
	orderRepo    func(ctrl *gomock.Controller) OrderRepoInterface
	balanceRepo  func(ctrl *gomock.Controller) BalanceRepoInterface
	pgxTx        func(ctrl *gomock.Controller) db.TxFacadeInterface
}

func TestActualizeOrderTask_ActualizeOrdersStatus(t *testing.T) {
	var (
		order1 = entity.Order{
			OrderID: "123456",
			Accrual: func() *int64 {
				v := int64(10000)
				return &v
			}(),
		}
		order2 = entity.Order{
			OrderID: "123457",
		}
		orders3 = entity.Order{
			OrderID: "123458",
			Accrual: func() *int64 {
				v := int64(20000)
				return &v
			}(),
		}
		orders4 = entity.Order{
			OrderID: "123459",
			Accrual: func() *int64 {
				v := int64(30000)
				return &v
			}(),
		}
	)

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Transaction_Success_With_Partial_Update",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacadeInterface {
					tx := dbmocks.NewMockTxFacadeInterface(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				statusGetter: func(ctrl *gomock.Controller) StatusGetterInterface {
					statusGetter := mocks.NewMockStatusGetterInterface(ctrl)
					statusGetter.EXPECT().ReceiveStatus(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, o []entity.Order) (*[]entity.Order, error) {
							return &o, nil
						},
					)
					return statusGetter
				},
				orderRepo: func(ctrl *gomock.Controller) OrderRepoInterface {
					orderRepo := mocks.NewMockOrderRepoInterface(ctrl)
					orders := []entity.Order{
						order1,
						order2,
						orders3,
						orders4,
					}
					orderRepo.EXPECT().OrdersByStatus(gomock.Any(), gomock.Any()).Times(1).Return(&orders, nil)

					orderRepo.EXPECT().UpdateAll(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, o []entity.Order) (*[]entity.Order, error) {
							return &o, nil
						},
					)
					return orderRepo
				},
				balanceRepo: func(ctrl *gomock.Controller) BalanceRepoInterface {
					balanceRepo := mocks.NewMockBalanceRepoInterface(ctrl)
					// Обновляются заказы с Accrual != null и соответствующим статусом.
					balanceRepo.EXPECT().IncreaseCurrent(gomock.Any(), gomock.Any(), gomock.Any()).Times(3).Return(&entity.Balance{}, nil)
					return balanceRepo
				},
			},
		},
		{
			name: "Transaction_Success_With_Full_Update",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacadeInterface {
					tx := dbmocks.NewMockTxFacadeInterface(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				statusGetter: func(ctrl *gomock.Controller) StatusGetterInterface {
					statusGetter := mocks.NewMockStatusGetterInterface(ctrl)
					statusGetter.EXPECT().ReceiveStatus(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, o []entity.Order) (*[]entity.Order, error) {
							return &o, nil
						},
					)
					return statusGetter
				},
				orderRepo: func(ctrl *gomock.Controller) OrderRepoInterface {
					orderRepo := mocks.NewMockOrderRepoInterface(ctrl)
					orders := []entity.Order{
						order1,
						order2,
						orders3,
						orders4,
					}
					orderRepo.EXPECT().OrdersByStatus(gomock.Any(), gomock.Any()).Times(1).Return(&orders, nil)

					orderRepo.EXPECT().UpdateAll(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, o []entity.Order) (*[]entity.Order, error) {
							return &o, nil
						},
					)
					return orderRepo
				},
				balanceRepo: func(ctrl *gomock.Controller) BalanceRepoInterface {
					balanceRepo := mocks.NewMockBalanceRepoInterface(ctrl)
					// Обновляются заказы с Accrual != null и соответствующим статусом.
					balanceRepo.EXPECT().IncreaseCurrent(gomock.Any(), gomock.Any(), gomock.Any()).Times(3).Return(&entity.Balance{}, nil)
					return balanceRepo
				},
			},
		},
		{
			name: "Transaction_With_Rollback",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacadeInterface {
					tx := dbmocks.NewMockTxFacadeInterface(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				statusGetter: func(ctrl *gomock.Controller) StatusGetterInterface {
					statusGetter := mocks.NewMockStatusGetterInterface(ctrl)
					statusGetter.EXPECT().ReceiveStatus(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, o []entity.Order) (*[]entity.Order, error) {
							return &[]entity.Order{
								order1,
								order2,
								orders3,
								orders4,
							}, nil
						},
					)
					return statusGetter
				},
				orderRepo: func(ctrl *gomock.Controller) OrderRepoInterface {
					orderRepo := mocks.NewMockOrderRepoInterface(ctrl)
					orderRepo.EXPECT().UpdateAll(gomock.Any(), gomock.Any()).Times(1).Return(nil, fmt.Errorf("error"))
					orderRepo.EXPECT().OrdersByStatus(gomock.Any(), gomock.Any()).Times(1).Return(&[]entity.Order{}, nil)

					return orderRepo
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			o := service(ctrl, tt.fields)
			o.actualize(context.Background())
		})
	}
}

func service(ctrl *gomock.Controller, fields fields) *ActualizeOrderTask {
	var (
		statusGetter StatusGetterInterface
		orderRepo    OrderRepoInterface
		balanceRepo  BalanceRepoInterface
		tx           db.TxFacadeInterface
	)
	if fields.statusGetter == nil {
		statusGetter = mocks.NewMockStatusGetterInterface(ctrl)
	} else {
		statusGetter = fields.statusGetter(ctrl)
	}
	if fields.orderRepo == nil {
		orderRepo = mocks.NewMockOrderRepoInterface(ctrl)
	} else {
		orderRepo = fields.orderRepo(ctrl)
	}
	if fields.balanceRepo == nil {
		balanceRepo = mocks.NewMockBalanceRepoInterface(ctrl)
	} else {
		balanceRepo = fields.balanceRepo(ctrl)
	}
	if fields.pgxTx == nil {
		tx = dbmocks.NewMockTxFacadeInterface(ctrl)
	} else {
		tx = fields.pgxTx(ctrl)
	}
	s := &ActualizeOrderTask{
		statusGetter: statusGetter,
		orderRepo:    orderRepo,
		balanceRepo:  balanceRepo,
		pgxTx:        tx,
		logger:       log.MockLogger,
	}
	return s
}
