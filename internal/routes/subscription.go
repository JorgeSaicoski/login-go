package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/handlers"
)

func SetupSubscriptionRoutes(r *gin.Engine, subscriptionHandler *handlers.SubscriptionHandler) {
	subscription := r.Group("/subscription")
	{
		subscription.GET("/:id", subscriptionHandler.GetByID)
		subscription.PATCH("/:id", subscriptionHandler.UpdateByID)
	}
}
