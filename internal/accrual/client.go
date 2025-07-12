package accrual

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/ktigay/loyalty/internal/entity"
)

const (
	orderInfoPath = "/api/orders/"
)

type AccrualRequestTimeout time.Duration

// Accrual Структура для работы с сервисом Accrual.
type Accrual struct {
	endpoint string
	timeout  AccrualRequestTimeout
	logger   *slog.Logger
}

// GetOrder Данные по заказу из сервиса Accrual.
func (s *Accrual) GetOrder(ctx context.Context, orderID string) (*entity.AccrualOrder, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout))
	defer cancel()

	client := resty.New()
	url := s.endpoint + orderInfoPath + orderID

	e := entity.AccrualOrder{}
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

// New Конструктор.
func New(endpoint string, timeout AccrualRequestTimeout, logger *slog.Logger) *Accrual {
	return &Accrual{
		endpoint: strings.TrimRight(endpoint, "/"),
		timeout:  timeout,
		logger:   logger,
	}
}

// RequestError Ошибка при получении ответа от сервиса Accrual.
type RequestError struct {
	StatusCode int
	URL        string
	Message    string
}

// Error Метод интерфейса.
func (e RequestError) Error() string {
	msg := fmt.Sprintf("ошибка, %s url: %s, статус: %d", e.Message, e.URL, e.StatusCode)
	return msg
}
