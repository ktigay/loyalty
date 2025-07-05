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

	_ "github.com/golang/mock/mockgen/model"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ktigay/loyalty/internal/api"
	"github.com/ktigay/loyalty/internal/config"
	appdb "github.com/ktigay/loyalty/internal/db"
	"github.com/ktigay/loyalty/internal/handler/balance"
	"github.com/ktigay/loyalty/internal/handler/order"
	"github.com/ktigay/loyalty/internal/handler/user"
	applog "github.com/ktigay/loyalty/internal/log"
	"github.com/ktigay/loyalty/internal/middleware"
	"github.com/ktigay/loyalty/internal/security"
	"github.com/ktigay/loyalty/internal/service/order/task"
)

func main() {
	ctx := context.Background()
	exitCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	if pool, err = appdb.NewPgxPool(ctx, cfg.DatabaseDSN); err != nil {
		logger.Error("Failed to create connect to DB", "error", err)
		os.Exit(1)
	}
	if err = appdb.CreateStructure(ctx, pool); err != nil {
		logger.Error("Failed to create structure", "error", err)
		os.Exit(1)
	}

	auth := security.NewJWTWrapper(cfg.AuthSecret)

	userAuthHandler := user.NewAuthHandler(auth, pool, logger)
	balanceHandler := balance.New(pool, logger)
	orderHandler := order.New(pool, logger)

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
	midWare := middleware.New(auth)

	_, err = api.NewServerHandler(apiSrv, router, []api.MiddlewareFunc{
		midWare.SecurityMiddleware(),
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

	wg.Add(1)
	go func() {
		t := task.NewActualizeOrderTask(cfg.AccrualHost, cfg.ActualizeInterval, pool, logger)
		t.ActualizeOrdersStatus(ctx, exitCtx)
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
