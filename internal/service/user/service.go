package user

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/entity"
	"github.com/ktigay/loyalty/internal/repository/balance"
	repo "github.com/ktigay/loyalty/internal/repository/user"
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

//go:generate mockgen -destination=./mocks/mock_user.go -package=mocks github.com/ktigay/loyalty/internal/service/user RepositoryInterface
type RepositoryInterface interface {
	UserByID(ctx context.Context, uuid string) (*entity.User, error)
	UserByLogin(ctx context.Context, login string) (*entity.User, error)
	CreateUser(ctx context.Context, login, password string) (*entity.User, error)
}

//go:generate mockgen -destination=./mocks/mock_balance.go -package=mocks github.com/ktigay/loyalty/internal/service/user BalanceRepoInterface
type BalanceRepoInterface interface {
	Create(ctx context.Context, userUUID string) error
}

// Service Сервис для работы с пользовательскими данными.
type Service struct {
	userRepo    RepositoryInterface
	balanceRepo BalanceRepoInterface
	pgxTx       db.TxFacadeInterface
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
	_ = s.pgxTx.RunInTx(ctx, pgx.TxOptions{}, func(ctxWithTx context.Context) error {
		if newUsr, err = s.userRepo.CreateUser(ctxWithTx, login, string(hashedPasswd)); err != nil {
			return err
		}

		if err = s.balanceRepo.Create(ctxWithTx, newUsr.UUID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return newUsr, nil
}

// GetUserByID Возвращает пользователя по uuid
func (s *Service) GetUserByID(ctx context.Context, uuid string) (*entity.User, error) {
	return s.userRepo.UserByID(ctx, uuid)
}

// New Конструктор.
func New(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{
		userRepo:    repo.New(pool, logger),
		balanceRepo: balance.New(pool, logger),
		pgxTx:       db.NewPgxTxFacade(pool),
		logger:      logger,
	}
}
