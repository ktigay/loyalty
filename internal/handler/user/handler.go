package user

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/service/user"
)

// AuthInterface Интерфейс сервиса аутентификации.
//
//go:generate mockgen -destination=./mocks/mock_auth.go -package=mocks github.com/ktigay/loyalty/internal/handler/user AuthInterface
type AuthInterface interface {
	SetIdentity(r *http.Request, w http.ResponseWriter, identity *entity.Identity) error
}

// ServiceInterface Интерфейс сервиса пользователей.
//
//go:generate mockgen -destination=./mocks/mock_service.go -package=mocks github.com/ktigay/loyalty/internal/handler/user ServiceInterface
type ServiceInterface interface {
	UserByCredentials(ctx context.Context, login, password string) (*entity.User, error)
	Create(ctx context.Context, login, password string) (*entity.User, error)
}

// AuthHandler Обработчик пользователей.
type AuthHandler struct {
	userSv ServiceInterface
	auth   AuthInterface
	logger *slog.Logger
}

// LoginHandler Обработчик авторизации.
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var auth api.PostUserLoginJSONRequestBody

	if err := json.NewDecoder(r.Body).Decode(&auth); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var (
		usr *entity.User
		err error
	)
	if usr, err = h.userSv.UserByCredentials(r.Context(), auth.Login, auth.Password); err != nil {
		if errors.Is(err, user.ErrUserNotFound) ||
			errors.Is(err, user.ErrWrongPassword) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	identity := entity.NewIdentity(usr.UUID)
	if err = h.auth.SetIdentity(r, w, identity); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RegisterHandler Обработчик регистрации.
func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	reg := api.PostUserRegisterJSONBody{}

	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var (
		usr *entity.User
		err error
	)
	if usr, err = h.userSv.Create(r.Context(), reg.Login, reg.Password); err != nil {
		h.logger.Warn("user not registered", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	identity := entity.NewIdentity(usr.UUID)
	if err = h.auth.SetIdentity(r, w, identity); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// NewAuthHandler Конструктор.
func NewAuthHandler(auth AuthInterface, pool *pgxpool.Pool, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		userSv: user.New(pool, logger),
		auth:   auth,
		logger: logger,
	}
}
