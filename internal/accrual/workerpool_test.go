package accrual

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/ktigay/loyalty/internal/accrual/mocks"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/log"
)

func TestWorkerPool_GetOrders(t *testing.T) {
	ids := []string{
		"123456", "123457", "123458", "123459", "123460",
	}
	type fields struct {
		clientFn     func(ctrl *gomock.Controller) Client
		maxRateLimit int
	}
	type args struct {
		ids []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []entity.AccrualOrder
		wantErr bool
	}{
		{
			name: "Success_Single_Tread",
			fields: fields{
				clientFn: func(ctrl *gomock.Controller) Client {
					c := mocks.NewMockClient(ctrl)
					c.EXPECT().GetOrder(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
						func(_ context.Context, orderID string) (*entity.AccrualOrder, error) {
							return &entity.AccrualOrder{OrderID: orderID}, nil
						})
					return c
				},
				maxRateLimit: 1,
			},
			args: args{
				ids: []string{
					"123456", "123457", "123458",
				},
			},
			want: []entity.AccrualOrder{
				{
					OrderID: "123456",
				},
				{
					OrderID: "123457",
				},
				{
					OrderID: "123458",
				},
			},
			wantErr: false,
		},
		{
			name: "Success_Multi_Tread",
			fields: fields{
				clientFn: func(ctrl *gomock.Controller) Client {
					c := mocks.NewMockClient(ctrl)
					c.EXPECT().GetOrder(gomock.Any(), gomock.Any()).Times(5).DoAndReturn(
						func(_ context.Context, orderID string) (*entity.AccrualOrder, error) {
							return &entity.AccrualOrder{OrderID: orderID}, nil
						})
					return c
				},
				maxRateLimit: 3,
			},
			args: args{
				ids: ids,
			},
			want: []entity.AccrualOrder{
				{
					OrderID: "123456",
				},
				{
					OrderID: "123457",
				},
				{
					OrderID: "123458",
				},
				{
					OrderID: "123459",
				},
				{
					OrderID: "123460",
				},
			},
			wantErr: false,
		},
		{
			name: "GetOrder_Without_StatusOK",
			fields: fields{
				clientFn: func(ctrl *gomock.Controller) Client {
					c := mocks.NewMockClient(ctrl)
					cnt := 0
					ch := make(chan struct{})
					c.EXPECT().GetOrder(gomock.Any(), gomock.Any()).Times(5).DoAndReturn(
						func(_ context.Context, orderID string) (*entity.AccrualOrder, error) {
							if orderID == "123458" {
								<-ch
								return nil, RequestError{
									StatusCode: 202,
								}
							}
							defer func() {
								if cnt == 4 {
									ch <- struct{}{}
								}
							}()
							cnt++
							return &entity.AccrualOrder{OrderID: orderID}, nil
						})
					return c
				},
				maxRateLimit: 3,
			},
			args: args{
				ids: ids,
			},
			want: []entity.AccrualOrder{
				{
					OrderID: "123456",
				},
				{
					OrderID: "123457",
				},
				{
					OrderID: "123459",
				},
				{
					OrderID: "123460",
				},
			},
			wantErr: false,
		},
		{
			name: "GetOrder_With_Critical_Error",
			fields: fields{
				clientFn: func(ctrl *gomock.Controller) Client {
					c := mocks.NewMockClient(ctrl)
					c.EXPECT().GetOrder(gomock.Any(), gomock.Any()).MinTimes(3).MaxTimes(5).DoAndReturn(
						func(_ context.Context, orderID string) (*entity.AccrualOrder, error) {
							if orderID == "123458" || orderID == "123459" {
								return nil, RequestError{
									StatusCode: 500,
								}
							}
							return &entity.AccrualOrder{OrderID: orderID}, nil
						})
					return c
				},
				maxRateLimit: 2,
			},
			args: args{
				ids: ids,
			},
			want: []entity.AccrualOrder{
				{
					OrderID: "123456",
				},
				{
					OrderID: "123457",
				},
				{
					OrderID: "123460",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			w := WorkerPoolClient{
				client:       tt.fields.clientFn(ctrl),
				maxRateLimit: tt.fields.maxRateLimit,
				logger:       log.MockLogger,
				ch:           make(chan string, tt.fields.maxRateLimit),
				respCh:       make(chan result),
				done:         make(chan struct{}),
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				w.Run(ctx)
				wg.Done()
			}()

			got, err := w.GetOrders(ctx, tt.args.ids...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			sort.Slice(got, func(i, j int) bool { return got[i].OrderID < got[j].OrderID })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].OrderID < tt.want[j].OrderID })

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("GetOrders() got = %v, want %v, diff %v", got, tt.want, diff)
			}

			cancel()
			wg.Wait()
		})
	}
}
