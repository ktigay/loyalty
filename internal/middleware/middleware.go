package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/security"
)

// AuthInterface Интерфейс сервиса аутентификации.
type AuthInterface interface {
	GetIdentity(r *http.Request) (*entity.Identity, error)
}

// Middleware Мидлвар.
type Middleware struct {
	auth AuthInterface
}

// SecurityMiddleware Обработка аутентификации пользователя.
func (m *Middleware) SecurityMiddleware() api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
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
}

// New Конструктор.
func New(auth AuthInterface) *Middleware {
	return &Middleware{
		auth: auth,
	}
}
