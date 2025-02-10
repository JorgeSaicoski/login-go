package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/handlers"
)

func SetupAuthRoutes(r *gin.Engine, authHandler *handlers.AuthHandler) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/validate", authHandler.ValidateToken)
	}
}
