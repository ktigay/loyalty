package order

import (
	"fmt"
	"testing"

	"github.com/ktigay/loyalty/internal/entity"
	"github.com/stretchr/testify/assert"
)

func TestAccrualHydrator_OrdersWithAccrual(t *testing.T) {
	type args struct {
		orders []entity.Order
		acc    []entity.AccrualOrder
	}
	tests := []struct {
		name    string
		args    args
		want    []entity.Order
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Hydrate_Orders_With_Accrual",
			args: args{
				orders: []entity.Order{
					{
						ID:       4,
						UserUUID: "uuid-1",
						OrderID:  "5577189519503182",
					},
					{
						ID:       3,
						UserUUID: "uuid-1",
						OrderID:  "79927398713",
					},
					{
						ID:       2,
						UserUUID: "uuid-2",
						OrderID:  "4929972884676289",
					},
					{
						ID:       1,
						UserUUID: "uuid-1",
						OrderID:  "4532733309529845",
					},
					{
						ID:       5,
						UserUUID: "uuid-1",
						Status:   entity.New,
						OrderID:  "4026843483168683",
					},
				},
				acc: []entity.AccrualOrder{
					{
						OrderID: "4929972884676289",
						Status:  "NEW",
						Accrual: func() *float64 {
							v := 100.0
							return &v
						}(),
					},
					{
						OrderID: "4532733309529845",
						Status:  "PROCESSING",
						Accrual: nil,
					},
					{
						OrderID: "5499078785968242",
						Status:  "INVALID",
					},
					{
						OrderID: "79927398713",
						Status:  "PROCESSED",
						Accrual: func() *float64 {
							v := 120.22
							return &v
						}(),
					},
					{
						OrderID: "4026843483168683",
						Status:  "PROCESSED",
					},
				},
			},
			want: []entity.Order{
				{
					ID:       4,
					UserUUID: "uuid-1",
					OrderID:  "5577189519503182",
				},
				{
					ID:       3,
					UserUUID: "uuid-1",
					OrderID:  "79927398713",
					Status:   "PROCESSED",
					Accrual: func() *int64 {
						v := int64(12022)
						return &v
					}(),
				},
				{
					ID:       2,
					UserUUID: "uuid-2",
					OrderID:  "4929972884676289",
					Status:   "NEW",
					Accrual: func() *int64 {
						v := int64(10000)
						return &v
					}(),
				},
				{
					ID:       1,
					UserUUID: "uuid-1",
					OrderID:  "4532733309529845",
					Status:   "PROCESSING",
				},
				{
					ID:         5,
					UserUUID:   "uuid-1",
					OrderID:    "4026843483168683",
					Status:     "PROCESSED",
					StatusPrev: entity.New,
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AccrualHydrator{}
			got, err := a.Hydrate(tt.args.orders, tt.args.acc)
			if !tt.wantErr(t, err, fmt.Sprintf("Hydrate(%v, %v)", tt.args.orders, tt.args.acc)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Hydrate(%v, %v)", tt.args.orders, tt.args.acc)
		})
	}
}
