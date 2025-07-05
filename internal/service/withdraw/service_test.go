package withdraw

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	dbmocks "github.com/ktigay/loyalty/internal/db/mocks"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/service/withdraw/mocks"
)

type fields struct {
	withdrawRepo func(ctrl *gomock.Controller) RepositoryInterface
	balanceRepo  func(ctrl *gomock.Controller) BalanceRepoInterface
	orderRepo    func(ctrl *gomock.Controller) OrderRepoInterface
	pgxTx        func(ctrl *gomock.Controller) db.TxFacadeInterface
}

func TestService_MakeWithdraw(t *testing.T) {
	errBalanceForUpdate := fmt.Errorf("balance for update error")
	type args struct {
		ctx      context.Context
		userUUID string
		orderID  string
		delta    int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *entity.Withdrawal
		wantErr error
	}{
		{
			name: "OrderByUser_With_No_Result_Error",
			fields: fields{
				orderRepo: func(ctrl *gomock.Controller) OrderRepoInterface {
					orderRepo := mocks.NewMockOrderRepoInterface(ctrl)
					orderRepo.EXPECT().OrderByUser(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, pgx.ErrNoRows)
					return orderRepo
				},
			},
			wantErr: ErrWrongOrderNumber,
		},
		{
			name: "Transaction_Rollback_On_BalanceForUpdate_Error",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacadeInterface {
					tx := dbmocks.NewMockTxFacadeInterface(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				orderRepo: func(ctrl *gomock.Controller) OrderRepoInterface {
					orderRepo := mocks.NewMockOrderRepoInterface(ctrl)
					orderRepo.EXPECT().OrderByUser(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
					return orderRepo
				},
				balanceRepo: func(ctrl *gomock.Controller) BalanceRepoInterface {
					balanceRepo := mocks.NewMockBalanceRepoInterface(ctrl)
					balanceRepo.EXPECT().Balance(gomock.Any(), gomock.Any()).Times(1).Return(nil, errBalanceForUpdate)
					return balanceRepo
				},
			},
			wantErr: errBalanceForUpdate,
		},
		{
			name: "Increase_Withdraw_Success",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacadeInterface {
					tx := dbmocks.NewMockTxFacadeInterface(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				orderRepo: func(ctrl *gomock.Controller) OrderRepoInterface {
					orderRepo := mocks.NewMockOrderRepoInterface(ctrl)
					orderRepo.EXPECT().OrderByUser(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
					return orderRepo
				},
				balanceRepo: func(ctrl *gomock.Controller) BalanceRepoInterface {
					balanceRepo := mocks.NewMockBalanceRepoInterface(ctrl)
					b := entity.Balance{
						Current: 20100,
					}
					balanceRepo.EXPECT().Balance(gomock.Any(), gomock.Any()).Times(1).Return(&b, nil)
					balanceRepo.EXPECT().IncreaseWithdraw(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
					return balanceRepo
				},
				withdrawRepo: func(ctrl *gomock.Controller) RepositoryInterface {
					withdrawRepo := mocks.NewMockRepositoryInterface(ctrl)
					w := entity.Withdrawal{}
					withdrawRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&w, nil)
					return withdrawRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				userUUID: "user-uuid",
				orderID:  "order-id",
				delta:    20000,
			},
			want: &entity.Withdrawal{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := service(ctrl, tt.fields)
			got, err := s.MakeWithdraw(tt.args.ctx, tt.args.userUUID, tt.args.orderID, tt.args.delta)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("MakeWithdraw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeWithdraw() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_Withdrawals(t *testing.T) {
	type args struct {
		ctx      context.Context
		userUUID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *[]entity.Withdrawal
		wantErr bool
	}{
		{
			name: "ErrNoRows_Without_Error_In_Result",
			fields: fields{
				withdrawRepo: func(ctrl *gomock.Controller) RepositoryInterface {
					withdrawRepo := mocks.NewMockRepositoryInterface(ctrl)
					withdrawRepo.EXPECT().GetWithdrawals(gomock.Any(), gomock.Any()).Return(nil, pgx.ErrNoRows)
					return withdrawRepo
				},
			},
			want:    &[]entity.Withdrawal{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := service(ctrl, tt.fields)
			got, err := s.Withdrawals(tt.args.ctx, tt.args.userUUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Withdrawals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Withdrawals() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func service(ctrl *gomock.Controller, fields fields) *Service {
	var (
		withdrawRepo RepositoryInterface
		balanceRepo  BalanceRepoInterface
		orderRepo    OrderRepoInterface
		tx           db.TxFacadeInterface
	)
	if fields.withdrawRepo == nil {
		withdrawRepo = mocks.NewMockRepositoryInterface(ctrl)
	} else {
		withdrawRepo = fields.withdrawRepo(ctrl)
	}
	if fields.balanceRepo == nil {
		balanceRepo = mocks.NewMockBalanceRepoInterface(ctrl)
	} else {
		balanceRepo = fields.balanceRepo(ctrl)
	}
	if fields.orderRepo == nil {
		orderRepo = mocks.NewMockOrderRepoInterface(ctrl)
	} else {
		orderRepo = fields.orderRepo(ctrl)
	}
	if fields.pgxTx == nil {
		tx = dbmocks.NewMockTxFacadeInterface(ctrl)
	} else {
		tx = fields.pgxTx(ctrl)
	}
	s := &Service{
		withdrawRepo: withdrawRepo,
		balanceRepo:  balanceRepo,
		orderRepo:    orderRepo,
		pgxTx:        tx,
		logger:       log.MockLogger,
	}
	return s
}
