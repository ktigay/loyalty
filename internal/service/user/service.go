package user

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = bcrypt.DefaultCost
)

var (
	ErrLoginOrPwdEmpty = errors.New("login or password is empty")
	ErrUserNotFound    = errors.New("user not found")
	ErrWrongPassword   = errors.New("wrong password")
)

//go:generate mockgen -destination=./mocks/mock_user.go -package=mocks github.com/ktigay/loyalty/internal/service/user Repository
type Repository interface {
	UserByID(ctx context.Context, uuid string) (*entity.User, error)
	UserByLogin(ctx context.Context, login string) (*entity.User, error)
	CreateUser(ctx context.Context, login, password string) (*entity.User, error)
}

//go:generate mockgen -destination=./mocks/mock_balance.go -package=mocks github.com/ktigay/loyalty/internal/service/user BalanceRepo
type BalanceRepo interface {
	Create(ctx context.Context, userUUID string) error
}

// Service Сервис для работы с пользовательскими данными.
type Service struct {
	userRepo    Repository
	balanceRepo BalanceRepo
	pgxTx       db.TxFacade
	logger      *slog.Logger
}

// UserByCredentials Возвращает пользователя по логину/паролю.
func (s *Service) UserByCredentials(ctx context.Context, login, password string) (*entity.User, error) {
	var (
		usr *entity.User
		err error
	)

	login = strings.TrimSpace(login)
	password = strings.TrimSpace(password)
	if login == "" || password == "" {
		return nil, ErrLoginOrPwdEmpty
	}

	if usr, err = s.userRepo.UserByLogin(ctx, login); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrWrongPassword
		}

		return nil, err
	}

	return usr, nil
}

// Create Создаёт пользователя.
func (s *Service) Create(ctx context.Context, login, password string) (*entity.User, error) {
	var (
		err          error
		hashedPasswd []byte
	)

	login = strings.TrimSpace(login)
	password = strings.TrimSpace(password)

	if login == "" || password == "" {
		return nil, ErrLoginOrPwdEmpty
	}

	hashedPasswd, err = bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}

	var newUsr *entity.User
	err = s.pgxTx.RunInTx(ctx, pgx.TxOptions{}, func(ctxWithTx context.Context) error {
		var txErr error
		if newUsr, txErr = s.userRepo.CreateUser(ctxWithTx, login, string(hashedPasswd)); txErr != nil {
			return txErr
		}

		if txErr = s.balanceRepo.Create(ctxWithTx, newUsr.UUID); txErr != nil {
			return txErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return newUsr, err
}

// GetUserByID Возвращает пользователя по uuid
func (s *Service) GetUserByID(ctx context.Context, uuid string) (*entity.User, error) {
	return s.userRepo.UserByID(ctx, uuid)
}

// New Конструктор.
func New(t db.TxFacade, u Repository, b BalanceRepo, l *slog.Logger) *Service {
	return &Service{
		pgxTx:       t,
		userRepo:    u,
		balanceRepo: b,
		logger:      l,
	}
}
