package accrual

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/ktigay/loyalty/internal/entity"
)

const (
	orderInfoPath = "/api/orders/"
	rateLimit     = 3
	timeout       = 1 * time.Second
)

// Client Структура для работы с сервисом Accrual.
type Client struct {
	endpoint string
	logger   *slog.Logger
}

// OrderStatus Данные по заказу из сервиса Accrual.
func (s *Client) OrderStatus(ctx context.Context, orderID string) (*entity.Accrual, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := resty.New()
	url := s.endpoint + orderInfoPath + orderID

	e := entity.Accrual{}
	resp, err := client.R().
		SetContext(ctx).
		SetResult(&e).
		Get(url)
	if err != nil {
		return nil, RequestError{
			StatusCode: 0,
			URL:        url,
			Message:    err.Error(),
		}
	}

	code := resp.StatusCode()
	if code == http.StatusOK {
		return &e, nil
	}

	return nil, RequestError{
		StatusCode: code,
		URL:        url,
	}
}

// OrdersStatus Данные по заказам из сервиса Accrual.
func (s *Client) OrdersStatus(ctx context.Context, orderIDs []string) (*[]entity.Accrual, error) {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	ln := len(orderIDs)
	ch := make(chan string, ln)

	rate := rateLimit
	if rateLimit > ln {
		rate = ln
	}

	respCh := make(chan result)

	var wg sync.WaitGroup
	for i := 1; i <= rate; i++ {
		wg.Add(1)
		go func() {
			s.worker(ctxCancel, i, ch, respCh)
			wg.Done()
		}()
	}

	// закрываем канал respCh только после того как отработают все воркеры.
	go func() {
		wg.Wait()
		close(respCh)
	}()

	for _, orderID := range orderIDs {
		ch <- orderID
	}
	close(ch)

	var (
		reqErr      RequestError
		criticalErr error
	)
	resp := make([]entity.Accrual, 0)
	for r := range respCh {
		if r.err != nil {
			s.logger.Debug("error in response", "error", r.err)

			// критическая ошибка, завершаем выполнение.
			if errors.As(r.err, &reqErr); reqErr.StatusCode >= 300 || reqErr.StatusCode < 200 {
				cancel()
				criticalErr = r.err
			}
		}
		if r.entity != nil {
			resp = append(resp, *r.entity)
		}
	}

	return &resp, criticalErr
}

// New Конструктор.
func New(endpoint string, logger *slog.Logger) *Client {
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		logger:   logger,
	}
}

func (s *Client) worker(ctx context.Context, thread int, jobs <-chan string, respCh chan<- result) {
	s.logger.Debug("start worker", "thread", thread)
	for orderID := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			resp, err := s.OrderStatus(ctx, orderID)

			respCh <- result{
				entity: resp,
				err:    err,
			}
		}
	}
	s.logger.Debug("finish worker", "thread", thread)
}

type result struct {
	err    error
	entity *entity.Accrual
}

// RequestError Ошибка при получении ответа от сервиса Accrual.
type RequestError struct {
	StatusCode int
	URL        string
	Message    string
}

// Error Метод интерфейса.
func (e RequestError) Error() string {
	msg := fmt.Sprintf("Ошибка, %s url: %s, статус: %d", e.Message, e.URL, e.StatusCode)
	return msg
}
