package order

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/handler/order/mocks"
	"github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/order"
	"github.com/stretchr/testify/assert"
)

type fields struct {
	orderSv func(ctrl *gomock.Controller) ServiceInterface
}

func TestHandler_CreateOrderHandler(t *testing.T) {
	var (
		UUID   = "uuid-1"
		order1 = "111113333"
	)
	type args struct {
		r []byte
	}
	type want struct {
		wantErr    bool
		wantStatus int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Request_New_Order_With_StatusAccepted",
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)

					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), entity.Number("111113333")).
						Times(1).Return(&entity.Order{}, nil)
					return orderSv
				},
			},
			args: args{
				r: []byte(order1),
			},
			want: want{
				wantStatus: http.StatusAccepted,
			},
		},
		{
			name: "Request_Exists_Order_With_StatusOK",
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)

					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), entity.Number("111113333")).
						Times(1).Return(nil, order.ErrOrderAlreadyExists)
					return orderSv
				},
			},
			args: args{
				r: []byte(order1),
			},
			want: want{
				wantStatus: http.StatusOK,
			},
		},
		{
			name: "Request_Order_With_Wrong_Number",
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)

					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), entity.Number("111113333")).
						Times(1).Return(nil, order.ErrWrongOrderNumber)
					return orderSv
				},
			},
			args: args{
				r: []byte(order1),
			},
			want: want{
				wantStatus: http.StatusUnprocessableEntity,
			},
		},
		{
			name: "Request_With_Wrong_User",
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)

					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), entity.Number("111113333")).
						Times(1).Return(nil, order.ErrWrongUser)
					return orderSv
				},
			},
			args: args{
				r: []byte(order1),
			},
			want: want{
				wantStatus: http.StatusConflict,
			},
		},
		{
			name: "Request_With_Unknown_Error",
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)

					orderSv.EXPECT().Create(gomock.Any(), gomock.Any(), entity.Number("111113333")).
						Times(1).Return(nil, errors.New("unknown error"))
					return orderSv
				},
			},
			args: args{
				r: []byte(order1),
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
				b.CreateOrderHandler(writer, request)
			}))
			defer srv.Close()

			resp, err := http.Post(srv.URL, "text/plain", bytes.NewReader(tt.args.r))
			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if (err != nil) != tt.want.wantErr {
				t.Errorf("CreateOrderHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			assert.Equal(t, tt.want.wantStatus, resp.StatusCode)
		})
	}
}

func TestHandler_GetOrdersHandler(t *testing.T) {
	var (
		UUID   = "uuid-1"
		order1 = "111113333"
		order2 = "111113334"
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
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)
					e := []entity.Order{
						{
							OrderID: order1,
							Accrual: func() *int64 {
								v := int64(100200)
								return &v
							}(),
							UploadedAt: func() time.Time {
								tm, _ := time.Parse("2006-01-02 15:04:05", "2025-07-08 17:09:47")
								return tm
							}(),
						},
						{
							OrderID: order2,
							UploadedAt: func() time.Time {
								tm, _ := time.Parse("2006-01-02 15:04:05", "2025-07-07 17:09:47")
								return tm
							}(),
						},
					}

					orderSv.EXPECT().OrdersByUser(gomock.Any(), gomock.Any()).
						Times(1).Return(&e, nil)
					return orderSv
				},
			},
			want: want{
				wantStatus:  http.StatusOK,
				contentType: "application/json",
				response:    `[{"accrual":1002,"number":"111113333","status":"","uploaded_at":"2025-07-08T17:09:47Z"},{"number":"111113334","status":"","uploaded_at":"2025-07-07T17:09:47Z"}]` + "\n",
			},
		},
		{
			name: "Request_With_OrdersByUser_Failed",
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)
					orderSv.EXPECT().OrdersByUser(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, errors.New("some error"))
					return orderSv
				},
			},
			want: want{
				wantStatus: http.StatusInternalServerError,
			},
		},
		{
			name: "Request_With_Empty_Response",
			fields: fields{
				orderSv: func(ctrl *gomock.Controller) ServiceInterface {
					orderSv := mocks.NewMockServiceInterface(ctrl)
					orderSv.EXPECT().OrdersByUser(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					return orderSv
				},
			},
			want: want{
				wantStatus: http.StatusOK,
				response:   `[]` + "\n",
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
				b.GetOrdersHandler(writer, request)
			}))
			defer srv.Close()

			resp, err := http.Get(srv.URL)
			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if (err != nil) != tt.want.wantErr {
				t.Errorf("GetOrdersHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if tt.want.contentType != "" {
				assert.Equal(t, tt.want.contentType, resp.Header.Get("Content-Type"))
			}
			assert.Equal(t, tt.want.wantStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("GetOrdersHandler() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			assert.Equal(t, tt.want.response, string(respBody))
		})
	}
}

func handler(ctrl *gomock.Controller, fields fields) *Handler {
	var orderSv ServiceInterface
	if fields.orderSv == nil {
		orderSv = mocks.NewMockServiceInterface(ctrl)
	} else {
		orderSv = fields.orderSv(ctrl)
	}

	return &Handler{
		orderSv: orderSv,
		logger:  log.MockLogger,
	}
}
