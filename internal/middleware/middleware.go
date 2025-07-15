package middleware

import (
	"bytes"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	apphttp "github.com/ktigay/loyalty/internal/http"
	"github.com/ktigay/loyalty/internal/security"
)

// Auth Интерфейс сервиса аутентификации.
type Auth interface {
	GetIdentity(r *http.Request) (*entity.Identity, error)
}

// Middleware Мидлвар.
type Middleware struct {
	auth   Auth
	logger *slog.Logger
}

// SecurityMiddleware Обработка аутентификации пользователя.
func (m *Middleware) SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		needAuth := ctx.Value(api.NeedAuthScopes) != nil
		if !needAuth {
			next.ServeHTTP(w, r)
			return
		}

		var (
			userIdentity *entity.Identity
			err          error
		)
		if userIdentity, err = m.auth.GetIdentity(r); err != nil {
			log.Println(err)
			if errors.Is(err, security.ErrMissingIdentity) {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		r = r.WithContext(security.NewContextWithUserIdentity(r.Context(), userIdentity))

		next.ServeHTTP(w, r)
	})
}

// RecoverWrapMiddleware Обработка panic().
func (m *Middleware) RecoverWrapMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Error("Recovering from", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// WithLogging логирует запрос.
func (m *Middleware) WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rd := apphttp.ResponseData{
			Status: 0,
			Size:   0,
		}
		sw := apphttp.NewWriter(w, &rd)

		start := time.Now()

		b, err := io.ReadAll(r.Body)
		if err != nil && !errors.Is(err, io.EOF) {
			m.logger.Error("Error reading body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(b))

		m.logger.Info(
			"request",
			"requestURI", r.RequestURI,
			"method", r.Method,
			"body", string(b),
			"headers", r.Header,
		)

		next.ServeHTTP(sw, r)

		m.logger.Info(
			"response",
			"duration", time.Since(start),
			"status", rd.Status,
			"size", rd.Size,
			"body", string(rd.Body),
		)
	})
}

// New Конструктор.
func New(auth Auth, logger *slog.Logger) *Middleware {
	return &Middleware{
		auth:   auth,
		logger: logger,
	}
}
