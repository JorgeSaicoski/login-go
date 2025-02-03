package main

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/config"
	"github.com/JorgeSaicoski/login-go/internal/handlers"
)

func main() {
	db := config.ConnectDatabase()
	userHandler := handlers.NewUserHandler(db)

	r := gin.Default()

	r.POST("/users", userHandler.Create)
	r.GET("/users", userHandler.List)
	r.GET("/users/:id", userHandler.Get)
	r.PUT("/users/:id", userHandler.Update)
	r.DELETE("/users/:id", userHandler.Delete)

	r.Run(":8080")
}
