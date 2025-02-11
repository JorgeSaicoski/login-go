package main

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/config"
	"github.com/JorgeSaicoski/login-go/internal/handlers"
	"github.com/JorgeSaicoski/login-go/internal/repository"
	"github.com/JorgeSaicoski/login-go/internal/routes"
	"github.com/JorgeSaicoski/login-go/internal/services"
)

func main() {
	db := config.ConnectDatabase()

	// Initialize repositories
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	userRepo := repository.NewUserRepository(db)
	userSubscriptionRepo := repository.NewUserSubscriptionRepository(db)

	// Initialize handlers
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionRepo)
	userHandler := handlers.NewUserHandler(userRepo)
	userSubscriptionHandler := handlers.NewUserSubscriptionHandler(userSubscriptionRepo)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService, userRepo)

	r := gin.Default()

	// Setup routes
	routes.SetupSubscriptionRoutes(r, subscriptionHandler)
	routes.SetupUserRoutes(r, userHandler)
	routes.SetupUserSubscriptionRoutes(r, userSubscriptionHandler)
	routes.SetupAuthRoutes(r, authHandler)

	r.Run(":8080")
}
