package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/JorgeSaicoski/login-go/config"
	"github.com/JorgeSaicoski/login-go/internal/handlers"
	"github.com/JorgeSaicoski/login-go/internal/repository"
	"github.com/JorgeSaicoski/login-go/internal/routes"
	"github.com/JorgeSaicoski/login-go/internal/services"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize database
	db := config.ConnectDatabase()
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("failed to get database instance", zap.Error(err))
	}

	// Initialize repositories
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	userRepo := repository.NewUserRepository(db, logger)
	userSubscriptionRepo := repository.NewUserSubscriptionRepository(db, logger)

	// Initialize handlers
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionRepo)
	userHandler := handlers.NewUserHandler(userRepo, logger)
	userSubscriptionHandler := handlers.NewUserSubscriptionHandler(userSubscriptionRepo, logger)
	healthHandler := handlers.NewHealthHandler(db)

	// Initialize auth service with configuration
	authConfig := services.AuthConfig{
		PrivateKeyPath: "path/to/private.pem", // Update with actual path
		PublicKeyPath:  "path/to/public.pem",  // Update with actual path
		TokenExpiry:    24 * time.Hour,
	}
	authService, err := services.NewAuthService(userRepo, logger, authConfig)
	if err != nil {
		logger.Fatal("failed to initialize auth service", zap.Error(err))
	}
	authHandler := handlers.NewAuthHandler(authService, userRepo, logger)

	// Initialize router
	r := gin.Default()

	// Setup routes
	routes.SetupSubscriptionRoutes(r, subscriptionHandler)
	routes.SetupUserRoutes(r, userHandler)
	routes.SetupUserSubscriptionRoutes(r, userSubscriptionHandler)
	routes.SetupAuthRoutes(r, authHandler)

	// Health check routes
	r.GET("/health", healthHandler.Check)
	r.GET("/ready", healthHandler.Check)

	// Initialize server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting server", zap.String("port", "8080"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	// Close database connection
	if err := sqlDB.Close(); err != nil {
		logger.Error("error closing database connection", zap.Error(err))
	}

	logger.Info("server exited properly")
}
