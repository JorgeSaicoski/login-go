package models

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Subscription struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type SubscriptionHandler struct {
	DB *gorm.DB
}

func NewSubscriptionHandler(db *gorm.DB) *SubscriptionHandler {
	return &SubscriptionHandler{DB: db}
}

func (h *SubscriptionHandler) UpdateByID(c *gin.Context) {
	// Get subscription ID from URL parameter
	id := c.Param("id")

	// Find existing subscription
	var subscription Subscription
	if err := h.DB.First(&subscription, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	// Bind JSON request body to subscription struct
	var updateData Subscription
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	subscription.Name = updateData.Name
	subscription.Description = updateData.Description
	subscription.Price = updateData.Price

	// Save changes to database
	if err := h.DB.Save(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}

	// Return updated subscription
	c.JSON(http.StatusOK, subscription)
}

func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	var subscription Subscription

	if err := h.DB.First(&subscription, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}
