package main

import (
	"database/sql"
	"github.com/alikhanturusbekov/gofermart/internal/handler"
	"github.com/alikhanturusbekov/gofermart/internal/repository/postgres"
	"github.com/alikhanturusbekov/gofermart/internal/service"
	"github.com/alikhanturusbekov/gofermart/internal/setup"
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

	// Setups the Router
	r := setupRouter(database, appConfig)

	// Serves the Application
	return http.ListenAndServe(appConfig.RunAddress, r)
}

func setupRouter(database *sql.DB, appConfig *setup.Config) *chi.Mux {
	r := chi.NewRouter()

	repository := postgres.NewRepository(database)
	authenticationService := service.NewAuthenticationService(repository, appConfig.AuthenticationSecretKey)
	mainHandler := handler.NewHandler(authenticationService)

	// Authentication
	r.Post("/api/user/register", mainHandler.Register)
	r.Post("/api/user/login", mainHandler.Login)

	return r
}
