package order

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/service/order/mocks"
	"github.com/stretchr/testify/assert"
)

func TestService_Create(t *testing.T) {
	type fields struct {
		orderRepo func(ctrl *gomock.Controller) *mocks.MockRepository
	}
	type args struct {
		ctx      context.Context
		userUUID string
		orderID  entity.Number
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *entity.Order
		wantErr error
	}{
		{
			name: "Wrong_Order_Number",
			args: args{
				ctx:      context.Background(),
				userUUID: "user-uuid",
				orderID:  entity.Number("111111"),
			},
			want:    nil,
			wantErr: ErrWrongOrderNumber,
		},
		{
			name: "Correct_Order_Number",
			fields: fields{
				orderRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
					orderRepo := mocks.NewMockRepository(ctrl)
					orderRepo.EXPECT().
						Order(gomock.Any(), gomock.Any()).
						Times(1)
					orderRepo.EXPECT().
						Create(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1)

					return orderRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				userUUID: "user-uuid",
				orderID:  entity.Number("9278923470"),
			},
			want: nil,
		},
		{
			name: "Order_Already_Exists_Error",
			fields: fields{
				orderRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
					orderRepo := mocks.NewMockRepository(ctrl)
					orderRepo.EXPECT().
						Order(gomock.Any(), gomock.Any()).
						Times(1).Return(
						&entity.Order{
							UserUUID: "user-uuid",
							OrderID:  "9278923470",
						}, nil)

					orderRepo.EXPECT().
						Create(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(0)

					return orderRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				userUUID: "user-uuid",
				orderID:  entity.Number("9278923470"),
			},
			want:    nil,
			wantErr: ErrOrderAlreadyExists,
		},
		{
			name: "Wrong_User_Error",
			fields: fields{
				orderRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
					orderRepo := mocks.NewMockRepository(ctrl)
					orderRepo.EXPECT().
						Order(gomock.Any(), gomock.Any()).
						Times(1).Return(
						&entity.Order{
							UserUUID: "user-uuid2",
							OrderID:  "1278923470",
						}, nil)

					orderRepo.EXPECT().
						Create(gomock.Any(), gomock.Any(), gomock.Any()).
						Times(0)

					return orderRepo
				},
			},
			args: args{
				ctx:      context.Background(),
				userUUID: "user-uuid",
				orderID:  entity.Number("9278923470"),
			},
			want:    nil,
			wantErr: ErrWrongUser,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var orderRepo *mocks.MockRepository
			if tt.fields.orderRepo == nil {
				orderRepo = mocks.NewMockRepository(ctrl)
				orderRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			} else {
				orderRepo = tt.fields.orderRepo(ctrl)
			}

			s := &Service{
				orderRepo: orderRepo,
				logger:    log.MockLogger,
			}
			got, err := s.Create(tt.args.ctx, tt.args.userUUID, tt.args.orderID)

			if tt.wantErr != nil {
				assert.Same(t, tt.wantErr, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Create() got = %v, want %v", got, tt.want)
			}
		})
	}
}
