package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/handlers"
)

func SetupUserSubscriptionRoutes(r *gin.Engine, handler *handlers.UserSubscriptionHandler) {
	userSub := r.Group("/user-subscription")
	{
		userSub.POST("/", handler.Create)
		userSub.GET("/user/:userId", handler.GetUserSubscriptions)
	}
}
