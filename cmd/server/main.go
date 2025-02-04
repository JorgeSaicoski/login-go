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

	r := gin.Default()

	routes.SetupSubscriptionRoutes(r, subscriptionHandler)

	r.Run(":8080")
}
