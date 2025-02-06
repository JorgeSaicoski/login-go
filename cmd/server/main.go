package main

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/config"
	"github.com/JorgeSaicoski/login-go/internal/handlers"
	"github.com/JorgeSaicoski/login-go/internal/repository"
	"github.com/JorgeSaicoski/login-go/internal/routes"
)

func main() {
	db := config.ConnectDatabase()
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionRepo)
	userRepo := repository.NewUserRepository(db)
	userHandler := handlers.NewUserHandler(userRepo)

	r := gin.Default()

	routes.SetupSubscriptionRoutes(r, subscriptionHandler)
	routes.SetupUserRoutes(r, userHandler)

	r.Run(":8080")
}
