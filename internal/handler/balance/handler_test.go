package balance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/handler/balance/mocks"
	"github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/withdraw"
	"github.com/stretchr/testify/assert"
)

type fields struct {
	orderSv    func(ctrl *gomock.Controller) OrderServiceInterface
	balanceSv  func(ctrl *gomock.Controller) ServiceInterface
	withdrawSv func(ctrl *gomock.Controller) WithdrawServiceInterface
}

func TestHandler_BalanceWithdrawHandler(t *testing.T) {
	var (
		UUID   = "uuid-1"
		order1 = "11112222"
	)
	type args struct {
		r           []byte
		contentType string
	}
	type want struct {
		wantErr     bool
		wantStatus  int
		contentType string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Request_With_StatusOK_With_Sum_Check",
			args: args{
				r: func() []byte {
					r, err := json.Marshal(api.PostUserBalanceWithdrawJSONRequestBody{
						Order: order1,
						Sum:   100.12,
					})
					if err != nil {
						t.Fatal(err)
					}
					return r
				}(),
				contentType: "application/json",
			},
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) OrderServiceInterface {
					orderSv := mocks.NewMockOrderServiceInterface(ctrl)
					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Eq(entity.Number("11112222"))).
						Times(1).Return(&entity.Order{}, nil)
					return orderSv
				},
				withdrawSv: func(ctrl *gomock.Controller) WithdrawServiceInterface {
					withdrawSv := mocks.NewMockWithdrawServiceInterface(ctrl)
					withdrawSv.EXPECT().MakeWithdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(int64(10012))).
						Times(1).Return(nil, nil)
					return withdrawSv
				},
			},
			want: want{
				wantStatus: http.StatusOK,
			},
		},
		{
			name: "Request_With_Wrong_Request_JSON",
			args: args{
				r: func() []byte {
					return []byte(`{"Order":"11112222",`)
				}(),
				contentType: "application/json",
			},
			want: want{
				wantStatus: http.StatusBadRequest,
			},
		},
		{
			name: "Request_Create_Error_Failed",
			args: args{
				r: func() []byte {
					return []byte(`{"Order":"11112222"}`)
				}(),
				contentType: "application/json",
			},
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) OrderServiceInterface {
					orderSv := mocks.NewMockOrderServiceInterface(ctrl)
					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Eq(entity.Number("11112222"))).
						Times(1).Return(nil, fmt.Errorf("some error"))
					return orderSv
				},
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
		{
			name: "Request_With_NotEnoughBalance",
			args: args{
				r: func() []byte {
					return []byte(`{"Order":"11112222"}`)
				}(),
				contentType: "application/json",
			},
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) OrderServiceInterface {
					orderSv := mocks.NewMockOrderServiceInterface(ctrl)
					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Eq(entity.Number("11112222"))).
						Times(1).Return(&entity.Order{}, nil)
					return orderSv
				},
				withdrawSv: func(ctrl *gomock.Controller) WithdrawServiceInterface {
					withdrawSv := mocks.NewMockWithdrawServiceInterface(ctrl)
					withdrawSv.EXPECT().MakeWithdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(nil, withdraw.ErrNotEnoughBalance)
					return withdrawSv
				},
			},
			want: want{
				wantStatus: http.StatusPaymentRequired,
			},
		},
		{
			name: "Request_With_Wrong_OrderNumber",
			args: args{
				r: func() []byte {
					return []byte(`{"Order":"11112222"}`)
				}(),
				contentType: "application/json",
			},
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) OrderServiceInterface {
					orderSv := mocks.NewMockOrderServiceInterface(ctrl)
					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Eq(entity.Number("11112222"))).
						Times(1).Return(&entity.Order{}, nil)
					return orderSv
				},
				withdrawSv: func(ctrl *gomock.Controller) WithdrawServiceInterface {
					withdrawSv := mocks.NewMockWithdrawServiceInterface(ctrl)
					withdrawSv.EXPECT().MakeWithdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(nil, withdraw.ErrWrongOrderNumber)
					return withdrawSv
				},
			},
			want: want{
				wantStatus: http.StatusUnprocessableEntity,
			},
		},
		{
			name: "Request_With_MakeWithdraw_Failed",
			args: args{
				r: func() []byte {
					return []byte(`{"Order":"11112222"}`)
				}(),
				contentType: "application/json",
			},
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) OrderServiceInterface {
					orderSv := mocks.NewMockOrderServiceInterface(ctrl)
					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Eq(entity.Number("11112222"))).
						Times(1).Return(&entity.Order{}, nil)
					return orderSv
				},
				withdrawSv: func(ctrl *gomock.Controller) WithdrawServiceInterface {
					withdrawSv := mocks.NewMockWithdrawServiceInterface(ctrl)
					withdrawSv.EXPECT().MakeWithdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Times(1).Return(nil, errors.New("some error"))
					return withdrawSv
				},
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			b := handler(ctrl, tt.fields)

			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				ctx := request.Context()
				request = request.WithContext(
					security.NewContextWithUserIdentity(ctx, entity.NewIdentity(UUID)),
				)
				b.BalanceWithdrawHandler(writer, request)
			}))
			defer srv.Close()

			resp, err := http.Post(srv.URL, tt.args.contentType, bytes.NewReader(tt.args.r))
			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if (err != nil) != tt.want.wantErr {
				t.Errorf("BalanceWithdrawHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if tt.want.contentType != "" {
				assert.Equal(t, tt.want.contentType, resp.Header.Get("Content-Type"))
			}
			assert.Equal(t, tt.want.wantStatus, resp.StatusCode)
		})
	}
}

func TestHandler_GetBalanceHandler(t *testing.T) {
	UUID := "uuid-1"
	type want struct {
		wantErr     bool
		wantStatus  int
		contentType string
		response    string
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "Request_With_StatusOK_With_Sum_Check",
			fields: fields{
				balanceSv: func(ctrl *gomock.Controller) ServiceInterface {
					balanceSv := mocks.NewMockServiceInterface(ctrl)
					e := entity.Balance{
						Current:   int64(20012),
						Withdrawn: int64(10000),
					}
					balanceSv.EXPECT().Balance(gomock.Any(), gomock.Any()).
						Times(1).Return(&e, nil)
					return balanceSv
				},
			},
			want: want{
				wantStatus:  http.StatusOK,
				contentType: "application/json",
				response:    `{"current":200.12,"withdrawn":100}` + "\n",
			},
		},
		{
			name: "Request_With_Get_Balance_Failed",
			fields: fields{
				balanceSv: func(ctrl *gomock.Controller) ServiceInterface {
					balanceSv := mocks.NewMockServiceInterface(ctrl)
					balanceSv.EXPECT().Balance(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, errors.New("some error"))
					return balanceSv
				},
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			b := handler(ctrl, tt.fields)

			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				ctx := request.Context()
				request = request.WithContext(
					security.NewContextWithUserIdentity(ctx, entity.NewIdentity(UUID)),
				)
				b.GetBalanceHandler(writer, request)
			}))
			defer srv.Close()

			resp, err := http.Get(srv.URL)
			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if (err != nil) != tt.want.wantErr {
				t.Errorf("GetBalanceHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if tt.want.contentType != "" {
				assert.Equal(t, tt.want.contentType, resp.Header.Get("Content-Type"))
			}
			assert.Equal(t, tt.want.wantStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("GetBalanceHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			assert.Equal(t, tt.want.response, string(respBody))
		})
	}
}

func TestHandler_GetWithdrawalsHandler(t *testing.T) {
	var (
		UUID   = "uuid-1"
		order1 = "111113333"
	)
	type want struct {
		wantErr     bool
		wantStatus  int
		contentType string
		response    string
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "Request_With_StatusOK",
			fields: fields{
				withdrawSv: func(ctrl *gomock.Controller) WithdrawServiceInterface {
					withdrawSv := mocks.NewMockWithdrawServiceInterface(ctrl)
					e := []entity.Withdrawal{
						{
							OrderID: order1,
							UUID:    UUID,
							Sum:     int64(20012),
							ProcessedAt: func() time.Time {
								tm, _ := time.Parse("2006-01-02 15:04:05", "2025-07-08 17:09:47")
								return tm
							}(),
						},
					}

					withdrawSv.EXPECT().Withdrawals(gomock.Any(), gomock.Any()).
						Times(1).Return(&e, nil)
					return withdrawSv
				},
			},
			want: want{
				wantStatus:  http.StatusOK,
				contentType: "application/json",
				response:    `[{"order":"111113333","processed_at":"2025-07-08T17:09:47Z","sum":200.12}]` + "\n",
			},
		},
		{
			name: "Request_GetWithdrawals_Failed",
			fields: fields{
				withdrawSv: func(ctrl *gomock.Controller) WithdrawServiceInterface {
					withdrawSv := mocks.NewMockWithdrawServiceInterface(ctrl)

					withdrawSv.EXPECT().Withdrawals(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, errors.New("some error"))
					return withdrawSv
				},
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
		{
			name: "Request_GetWithdrawals_Empty_Result",
			fields: fields{
				withdrawSv: func(ctrl *gomock.Controller) WithdrawServiceInterface {
					withdrawSv := mocks.NewMockWithdrawServiceInterface(ctrl)

					withdrawSv.EXPECT().Withdrawals(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					return withdrawSv
				},
			},
			want: want{
				wantStatus: http.StatusNoContent,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			b := handler(ctrl, tt.fields)

			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				ctx := request.Context()
				request = request.WithContext(
					security.NewContextWithUserIdentity(ctx, entity.NewIdentity(UUID)),
				)
				b.GetWithdrawalsHandler(writer, request)
			}))
			defer srv.Close()

			resp, err := http.Get(srv.URL)
			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if (err != nil) != tt.want.wantErr {
				t.Errorf("GetWithdrawalsHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if tt.want.contentType != "" {
				assert.Equal(t, tt.want.contentType, resp.Header.Get("Content-Type"))
			}
			assert.Equal(t, tt.want.wantStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("GetWithdrawalsHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			assert.Equal(t, tt.want.response, string(respBody))
		})
	}
}

func handler(ctrl *gomock.Controller, fields fields) *Handler {
	var (
		orderSv    OrderServiceInterface
		balanceSv  ServiceInterface
		withdrawSv WithdrawServiceInterface
	)
	if fields.orderSv == nil {
		orderSv = mocks.NewMockOrderServiceInterface(ctrl)
	} else {
		orderSv = fields.orderSv(ctrl)
	}
	if fields.balanceSv == nil {
		balanceSv = mocks.NewMockServiceInterface(ctrl)
	} else {
		balanceSv = fields.balanceSv(ctrl)
	}
	if fields.withdrawSv == nil {
		withdrawSv = mocks.NewMockWithdrawServiceInterface(ctrl)
	} else {
		withdrawSv = fields.withdrawSv(ctrl)
	}

	return &Handler{
		orderSv:    orderSv,
		balanceSv:  balanceSv,
		withdrawSv: withdrawSv,
		logger:     log.MockLogger,
	}
}
