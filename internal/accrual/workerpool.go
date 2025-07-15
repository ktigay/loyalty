package accrual

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/ktigay/loyalty/internal/entity"
)

var ErrOrdersNotFound = errors.New("orders not found")

// Client Интерфейс клиента для работы с Accrual.
//
//go:generate mockgen -destination=./mocks/mock_client.go -package=mocks github.com/ktigay/loyalty/internal/accrual Client
type Client interface {
	GetOrder(ctx context.Context, orderID string) (*entity.AccrualOrder, error)
}

// WorkerPoolClient Клиент с worker pool.
type WorkerPoolClient struct {
	client       Client
	maxRateLimit int
	logger       *slog.Logger
	ch           chan string
	respCh       chan result
	done         chan struct{}
}

// GetOrders Возвращает заказы из Accrual.
func (w *WorkerPoolClient) GetOrders(_ context.Context, ids ...string) ([]entity.AccrualOrder, error) {
	go func() {
		for _, id := range ids {
			select {
			case <-w.done:
				return
			default:
				w.ch <- id
			}
		}
	}()

	w.logger.Debug("GetOrders start")
	var err error
	resp := make([]entity.AccrualOrder, 0)
	for i := 0; i < len(ids); i++ {
		r, ok := <-w.respCh
		if !ok {
			return resp, errors.New("response channel closed")
		}
		if r.err != nil {
			w.logger.Debug("error in response", "error", r.err)
			err = r.err
		}
		if r.entity != nil {
			resp = append(resp, *r.entity)
		}
	}

	w.logger.Debug("GetOrders end")

	if len(resp) == 0 {
		return nil, ErrOrdersNotFound
	}

	return resp, err
}

// Run Запускает воркеры.
func (w *WorkerPoolClient) Run(ctx context.Context) {
	defer close(w.ch)
	defer close(w.respCh)

	var wg sync.WaitGroup
	for i := 1; i <= w.maxRateLimit; i++ {
		wg.Add(1)
		go func() {
			w.worker(ctx, i, w.ch, w.respCh)
			wg.Done()
		}()
	}
	wg.Wait()

	close(w.done)
}

func (w *WorkerPoolClient) worker(ctx context.Context, thread int, jobs <-chan string, respCh chan<- result) {
	w.logger.Debug("start worker", "thread", thread)
loop:
	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("context done", "thread", thread)
			break loop
		case orderID := <-jobs:
			resp, err := w.client.GetOrder(ctx, orderID)

			var reqErr RequestError
			if err != nil && (!errors.As(err, &reqErr) || (reqErr.StatusCode >= 300 || reqErr.StatusCode < 200)) {
				respCh <- result{
					entity: resp,
					err:    err,
				}
			} else {
				respCh <- result{
					entity: resp,
					err:    nil,
				}
			}
		}
	}
	w.logger.Debug("finish worker", "thread", thread)
}

// NewWorkerPoolClient Конструктор.
func NewWorkerPoolClient(c Client, r int, l *slog.Logger) *WorkerPoolClient {
	ch := make(chan string, r)
	respCh := make(chan result)
	done := make(chan struct{})

	return &WorkerPoolClient{
		client:       c,
		maxRateLimit: r,
		logger:       l,
		ch:           ch,
		respCh:       respCh,
		done:         done,
	}
}

type result struct {
	err    error
	entity *entity.AccrualOrder
}
