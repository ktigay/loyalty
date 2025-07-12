package user

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/golang/mock/gomock"
	"github.com/ktigay/loyalty/internal/db"
	dbmocks "github.com/ktigay/loyalty/internal/db/mocks"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/service/user/mocks"
	"golang.org/x/crypto/bcrypt"
)

type fields struct {
	userRepo    func(ctrl *gomock.Controller) Repository
	balanceRepo func(ctrl *gomock.Controller) BalanceRepo
	pgxTx       func(ctrl *gomock.Controller) db.TxFacade
}

func TestService_Create(t *testing.T) {
	const (
		login    = "login"
		password = "password"
	)

	type args struct {
		ctx      context.Context
		login    string
		password string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *entity.User
		wantErr error
	}{
		{
			name:    "Login_Empty_Error",
			wantErr: ErrLoginOrPwdEmpty,
		},
		{
			name:    "Password_Empty_Error",
			wantErr: ErrLoginOrPwdEmpty,
			args: args{
				ctx:      context.Background(),
				login:    login,
				password: "   ",
			},
		},
		{
			name: "Transaction_Rollback_On_CreateUser_Error",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacade {
					tx := dbmocks.NewMockTxFacade(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				userRepo: func(ctrl *gomock.Controller) Repository {
					userRepo := mocks.NewMockRepository(ctrl)
					userRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, fmt.Errorf("use create error"))
					return userRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				login:    login,
				password: password,
			},
		},
		{
			name: "Transaction_Rollback_On_Balance_Create_Error",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacade {
					tx := dbmocks.NewMockTxFacade(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				userRepo: func(ctrl *gomock.Controller) Repository {
					userRepo := mocks.NewMockRepository(ctrl)
					var u entity.User
					userRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&u, nil)
					return userRepo
				},
				balanceRepo: func(ctrl *gomock.Controller) BalanceRepo {
					balanceRepo := mocks.NewMockBalanceRepo(ctrl)
					balanceRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(fmt.Errorf("balance create error"))
					return balanceRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				login:    login,
				password: password,
			},
		},
		{
			name: "Transaction_Success",
			fields: fields{
				pgxTx: func(ctrl *gomock.Controller) db.TxFacade {
					tx := dbmocks.NewMockTxFacade(ctrl)
					tx.EXPECT().RunInTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
						func(ctx context.Context, opts pgx.TxOptions, fn func(ctxWithTx context.Context) error) error {
							return fn(ctx)
						})
					return tx
				},
				userRepo: func(ctrl *gomock.Controller) Repository {
					userRepo := mocks.NewMockRepository(ctrl)
					var u entity.User
					userRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&u, nil)
					return userRepo
				},
				balanceRepo: func(ctrl *gomock.Controller) BalanceRepo {
					balanceRepo := mocks.NewMockBalanceRepo(ctrl)
					balanceRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)
					return balanceRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				login:    login,
				password: password,
			},
			want: &entity.User{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := service(ctrl, tt.fields)
			got, err := s.Create(tt.args.ctx, tt.args.login, tt.args.password)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Create() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_UserByCredentials(t *testing.T) {
	const (
		login    = "login"
		password = "password"
	)
	type args struct {
		ctx      context.Context
		login    string
		password string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *entity.User
		wantErr error
	}{
		{
			name:    "Login_Empty_Error",
			wantErr: ErrLoginOrPwdEmpty,
		},
		{
			name:    "Password_Empty_Error",
			wantErr: ErrLoginOrPwdEmpty,
			args: args{
				ctx:      context.Background(),
				login:    login,
				password: "   ",
			},
		},
		{
			name: "Success_Without_Errors",
			fields: fields{
				userRepo: func(ctrl *gomock.Controller) Repository {
					userRepo := mocks.NewMockRepository(ctrl)
					pwd, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
					u := entity.User{
						Login:    login,
						Password: string(pwd),
					}
					userRepo.EXPECT().UserByLogin(gomock.Any(), gomock.Eq(login)).Times(1).Return(&u, err)
					return userRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				login:    "login ",
				password: "  password",
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := service(ctrl, tt.fields)

			_, err := s.UserByCredentials(tt.args.ctx, tt.args.login, tt.args.password)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("UserByCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func service(ctrl *gomock.Controller, fields fields) *Service {
	var (
		userRepo    Repository
		balanceRepo BalanceRepo
		tx          db.TxFacade
	)
	if fields.userRepo == nil {
		userRepo = mocks.NewMockRepository(ctrl)
	} else {
		userRepo = fields.userRepo(ctrl)
	}
	if fields.balanceRepo == nil {
		balanceRepo = mocks.NewMockBalanceRepo(ctrl)
	} else {
		balanceRepo = fields.balanceRepo(ctrl)
	}
	if fields.pgxTx == nil {
		tx = dbmocks.NewMockTxFacade(ctrl)
	} else {
		tx = fields.pgxTx(ctrl)
	}
	s := &Service{
		userRepo:    userRepo,
		balanceRepo: balanceRepo,
		pgxTx:       tx,
		logger:      log.MockLogger,
	}
	return s
}
