package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/handlers"
)

func SetupUserSubscriptionRoutes(r *gin.Engine, handler *handlers.UserSubscriptionHandler) {
	// Nested under user routes for better resource hierarchy
	user := r.Group("/user")
	{
		// Get all subscriptions for a user
		user.GET("/:userId/subscription", handler.GetUserSubscriptions)
		// Create/Assign a specific subscription to a user
		user.POST("/:userId/subscription/:subscriptionId", handler.Create)
		// Update a specific user's subscription
		user.PATCH("/:userId/subscription/:subscriptionId", handler.UpdateUserSubscription)
	}
}
