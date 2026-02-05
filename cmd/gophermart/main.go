package main

import (
	"context"
	"errors"
	"github.com/alikhanturusbekov/gofermart/internal/client"
	"github.com/alikhanturusbekov/gofermart/internal/handler"
	"github.com/alikhanturusbekov/gofermart/internal/middleware"
	"github.com/alikhanturusbekov/gofermart/internal/repository/postgres"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/alikhanturusbekov/gofermart/internal/setup"
	"github.com/alikhanturusbekov/gofermart/internal/worker"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Printf("Error while starting the app: %s", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Gets the application config
	appConfig, err := setup.LoadConfig()
	if err != nil {
		return err
	}

	// Prepares application database
	database, err := setup.PrepareDatabase(appConfig)
	if err != nil {
		return err
	}
	defer database.Close()

	// Setups repository
	repository := postgres.NewRepository(database)

	// Setups client
	accrualClient := client.NewClient(appConfig.AccrualSystemAddress, client.DefaultTimeout)

	// Setups workers
	orderProcessor := worker.NewOrderProcessWorker(
		repository,
		accrualClient,
		worker.DefaultBufferSize,
		worker.DefaultWorkers,
	)
	go orderProcessor.Run(ctx)

	// Setups services
	authService := service.NewAuthService(repository, appConfig.AuthSecretKey)
	loyaltyService := service.NewLoyaltyService(repository, orderProcessor)

	// Setups handler and router
	mainHandler := handler.NewHandler(authService, loyaltyService)
	r := setupRouter(appConfig, mainHandler)

	// Setups the HTTP server
	srv := &http.Server{
		Addr:    appConfig.RunAddress,
		Handler: r,
	}

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	// Wait for signal or server error
	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	// Graceful shutdown for server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	// Graceful shutdown for workers
	orderProcessor.Shutdown()

	log.Println("application stopped gracefully")
	return nil
}

func setupRouter(appConfig *setup.Config, handler *handler.Handler) *chi.Mux {
	r := chi.NewRouter()

	// Authentication
	r.Post("/api/user/register", handler.Register)
	r.Post("/api/user/login", handler.Login)

	// Requires authentication
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(appConfig.AuthSecretKey))

		// Orders
		r.Get("/api/user/orders", handler.GetUserOrders)
		r.Post("/api/user/orders", handler.UploadOrder)

		// Balance
		r.Get("/api/user/balance", handler.GetUserBalance)
		r.Post("/api/user/balance/withdraw", handler.Withdraw)

		// Withdrawals
		r.Get("/api/user/withdrawals", handler.GetUserWithdrawals)
	})

	return r
}
