package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/models"
	"github.com/JorgeSaicoski/login-go/internal/repository"
)

type UserSubscriptionHandler struct {
	repo *repository.UserSubscriptionRepository
}

func NewUserSubscriptionHandler(repo *repository.UserSubscriptionRepository) *UserSubscriptionHandler {
	return &UserSubscriptionHandler{
		repo: repo,
	}
}

func (h *UserSubscriptionHandler) Create(c *gin.Context) {
	// Parse user ID and subscription ID from URL
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	subscriptionID, err := strconv.ParseUint(c.Param("subscriptionId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	// Bind subscription details from JSON body
	var us models.UserSubscription
	if err := c.ShouldBindJSON(&us); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set IDs from URL parameters
	us.UserID = uint(userID)
	us.SubscriptionID = uint(subscriptionID)

	// Set default values if not provided
	if us.StartDate.IsZero() {
		us.StartDate = time.Now()
	}
	if us.EndDate.IsZero() {
		us.EndDate = time.Now().AddDate(1, 0, 0) // Default to 1 year
	}
	us.IsActive = true

	if err := h.repo.Create(&us); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, us)
}

func (h *UserSubscriptionHandler) GetUserSubscriptions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	subscriptions, err := h.repo.GetByUserID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

func (h *UserSubscriptionHandler) UpdateUserSubscription(c *gin.Context) {
	// Get and validate user ID
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get and validate subscription ID
	subscriptionID, err := strconv.ParseUint(c.Param("subscriptionId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	currentUs, err := h.repo.GetByID(uint(subscriptionID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User subscription not found"})
		return
	}

	// Verify the subscription belongs to the specified user
	if currentUs.UserID != uint(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Subscription does not belong to specified user"})
		return
	}

	var newUs models.UserSubscription
	if err := c.ShouldBindJSON(&newUs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update only allowed fields while preserving others
	if newUs.Type != "" {
		currentUs.Type = newUs.Type
	}
	if newUs.CompanyName != "" {
		currentUs.CompanyName = newUs.CompanyName
	}
	if newUs.Role != "" {
		currentUs.Role = newUs.Role
	}
	if !newUs.StartDate.IsZero() {
		currentUs.StartDate = newUs.StartDate
	}
	if !newUs.EndDate.IsZero() {
		currentUs.EndDate = newUs.EndDate
	}
	currentUs.IsActive = newUs.IsActive

	// Perform the update
	if err := h.repo.Update(currentUs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, currentUs)
}
