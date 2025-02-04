package main

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/config"
	"github.com/JorgeSaicoski/login-go/internal/models"
	"github.com/JorgeSaicoski/login-go/internal/routes"
)

func main() {
	db := config.ConnectDatabase()
	subscriptionHandler := models.NewSubscriptionHandler(db)
	userHandler := models.NewUserHandler(db)

	r := gin.Default()

	routes.SetupSubscriptionRoutes(r, subscriptionHandler)
	routes.SetupUserRoutes(r, userHandler)

	r.Run(":8080")
}
