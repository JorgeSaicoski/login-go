package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/models"
	"github.com/JorgeSaicoski/login-go/internal/repository"
)

type SubscriptionHandler struct {
	repo *repository.SubscriptionRepository
}

func NewSubscriptionHandler(repo *repository.SubscriptionRepository) *SubscriptionHandler {
	return &SubscriptionHandler{
		repo: repo,
	}
}

func (h *SubscriptionHandler) UpdateByID(c *gin.Context) {
	// Convert ID from string to uint
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// Get existing subscription using repository
	subscription, err := h.repo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	// Bind JSON request body to subscription struct
	var updateData models.Subscription
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	subscription.Name = updateData.Name
	subscription.Description = updateData.Description
	subscription.Price = updateData.Price

	// Use repository to save changes
	if err := h.repo.Update(subscription); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	// Convert ID from string to uint
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// Use repository to get subscription
	subscription, err := h.repo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}
