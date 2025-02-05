package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/handlers"
)

func SetupUserRoutes(r *gin.Engine, userHandler *handlers.UserHandler) {
	user := r.Group("/user")
	{
		user.GET("/:id", userHandler.GetByID)
		user.PATCH("/:id", userHandler.UpdateByID)
	}
}
