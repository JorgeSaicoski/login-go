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
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionRepo)
	userRepo := repository.NewUserRepository(db)
	userHandler := handlers.NewUserHandler(userRepo)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService, userRepo)

	r := gin.Default()

	routes.SetupSubscriptionRoutes(r, subscriptionHandler)
	routes.SetupUserRoutes(r, userHandler)
	routes.SetupAuthRoutes(r, authHandler)

	r.Run(":8080")
}
