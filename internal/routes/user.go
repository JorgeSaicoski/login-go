package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/models"
)

func SetupSubscriptionRoutes(r *gin.Engine, subscriptionHandler *models.SubscriptionHandler) {
	subscription := r.Group("/subscription")
	{
		subscription.GET("/:id", subscriptionHandler.GetByID)
		subscription.PATCH("/:id", subscriptionHandler.UpdateByID)
	}
}
