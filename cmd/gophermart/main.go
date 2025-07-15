package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/golang/mock/mockgen/model"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/accrual"
	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/config"
	appdb "github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/handler/balance"
	"github.com/ktigay/loyalty/internal/handler/order"
	"github.com/ktigay/loyalty/internal/handler/user"
	applog "github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/middleware"
	balancerepo "github.com/ktigay/loyalty/internal/repository/balance"
	orderrepo "github.com/ktigay/loyalty/internal/repository/order"
	userrepo "github.com/ktigay/loyalty/internal/repository/user"
	withdrawrepo "github.com/ktigay/loyalty/internal/repository/withdraw"
	"github.com/ktigay/loyalty/internal/security"
	balancesv "github.com/ktigay/loyalty/internal/service/balance"
	ordersv "github.com/ktigay/loyalty/internal/service/order"
	"github.com/ktigay/loyalty/internal/service/order/task"
	usersv "github.com/ktigay/loyalty/internal/service/user"
	withdrawsv "github.com/ktigay/loyalty/internal/service/withdraw"
)

func main() {
	var (
		cfg    *config.Config
		logger *slog.Logger
		pool   *pgxpool.Pool
		err    error
	)

	if cfg, err = config.New(os.Args[1:]); err != nil {
		log.Fatalf("can't parse flags: %v", err)
	}

	logger = applog.New(cfg.LogLevel)
	logger.Debug("config loaded", "config", cfg)

	ctx := context.Background()

	if pool, err = appdb.NewPgxPool(ctx, cfg.DatabaseDSN); err != nil {
		log.Fatalf("Failed to create connect to DB: %v", err)
	}
	if err = appdb.CreateSchema(ctx, pool); err != nil {
		log.Fatalf("Failed to create structure: %v", err)
	}

	auth := security.NewJWTWrapper(cfg.AuthSecret)

	var (
		txFacade  = appdb.NewPgxTxFacade(pool)
		dbWrapper = appdb.NewTxConnWrapper(pool)

		userRepo     = userrepo.New(dbWrapper, logger)
		balanceRepo  = balancerepo.New(dbWrapper, logger)
		withdrawRepo = withdrawrepo.New(dbWrapper, logger)
		orderRepo    = orderrepo.New(dbWrapper, logger)

		userSv     = usersv.New(txFacade, userRepo, balanceRepo, logger)
		balanceSv  = balancesv.New(balanceRepo, logger)
		withdrawSv = withdrawsv.New(txFacade, withdrawRepo, balanceRepo, orderRepo, logger)
		orderSv    = ordersv.New(orderRepo, logger)

		userAuthHandler = user.New(auth, userSv, pool, logger)
		balanceHandler  = balance.New(orderSv, balanceSv, withdrawSv, logger)
		orderHandler    = order.New(orderSv, logger)
	)

	apiSrv := api.Server{
		PostUserLoginHandler:           userAuthHandler.LoginHandler,
		PostUserRegisterHandler:        userAuthHandler.RegisterHandler,
		GetUserBalanceHandler:          balanceHandler.GetBalanceHandler,
		PostUserBalanceWithdrawHandler: balanceHandler.BalanceWithdrawHandler,
		PostUserOrdersHandler:          orderHandler.CreateOrderHandler,
		GetUserOrdersHandler:           orderHandler.GetOrdersHandler,
		GetUserWithdrawsHandler:        balanceHandler.GetWithdrawalsHandler,
	}

	router := mux.NewRouter()
	midWare := middleware.New(auth, logger)

	_, err = api.NewServerHandler(apiSrv, router, []api.MiddlewareFunc{
		midWare.WithLogging,
		midWare.RecoverWrapMiddleware,
		midWare.SecurityMiddleware,
	})
	if err != nil {
		logger.Error("Failed to register API", "error", err)
	}

	httpSrv := http.Server{
		Handler: router,
		Addr:    cfg.ServerHost,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	exitCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err = httpSrv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				logger.Debug("http server stopped")
			} else {
				logger.Error("can't start http server", "error", err)
				stop()
			}
		}
		wg.Done()
	}()

	workerPool := accrual.NewWorkerPoolClient(
		accrual.New(
			cfg.AccrualHost,
			accrual.AccrualRequestTimeout(time.Duration(cfg.AccrualTimeout)*time.Second),
			logger,
		),
		cfg.AccrualMaxRateLimit,
		logger,
	)
	wg.Add(1)
	go func() {
		workerPool.Run(exitCtx)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		t := task.NewActualizeOrderTask(
			workerPool,
			ordersv.NewAccrualHydrator(),
			txFacade,
			orderRepo,
			balanceRepo,
			task.ActualizeInterval(time.Duration(cfg.AccrualActualizeInterval)*time.Second),
			logger,
		)

		t.ActualizeOrdersStatus(exitCtx)
		wg.Done()
	}()

	go func() {
		<-exitCtx.Done()

		logger.Debug("http server shutting down")
		if err = httpSrv.Shutdown(context.Background()); err != nil {
			logger.Error("can't shutdown http server", "error", err)
		}
	}()

	wg.Wait()
	logger.Debug("server shutdown gracefully")
}
