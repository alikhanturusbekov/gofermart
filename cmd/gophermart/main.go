package main

import (
	"context"
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
)

func main() {
	if err := run(); err != nil {
		log.Printf("Error while starting the app: %s", err)

		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Setups worker
	orderProcessWorker := worker.NewOrderProcessWorker(repository, accrualClient, worker.DefaultBufferSize)
	go orderProcessWorker.Run(ctx)

	// Setups services
	authService := service.NewAuthService(repository, appConfig.AuthSecretKey)
	loyaltyService := service.NewLoyaltyService(repository, orderProcessWorker)

	// Setups handler and router
	mainHandler := handler.NewHandler(authService, loyaltyService)
	r := setupRouter(appConfig, mainHandler)

	// Serves the Application
	return http.ListenAndServe(appConfig.RunAddress, r)
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

		// Withdrawals
		r.Get("/api/user/withdrawals", handler.GetUserWithdrawals)
		r.Post("/api/user/withdraw", handler.Withdraw)
	})

	return r
}
