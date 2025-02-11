package handlers

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/JorgeSaicoski/login-go/internal/models"
	"github.com/JorgeSaicoski/login-go/internal/repository"
)

type UserSubscriptionHandler struct {
	repo *repository.UserSubscriptionRepository
	mu   sync.RWMutex // Protects concurrent operations
}

func NewUserSubscriptionHandler(repo *repository.UserSubscriptionRepository) *UserSubscriptionHandler {
	return &UserSubscriptionHandler{
		repo: repo,
	}
}

// parseUserAndSubscriptionID extracts and validates user and subscription IDs from the context
func (h *UserSubscriptionHandler) parseUserAndSubscriptionID(c *gin.Context) (uint, uint, error) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return 0, 0, &HandlerError{Status: http.StatusBadRequest, Message: "Invalid user ID"}
	}

	subscriptionID, err := strconv.ParseUint(c.Param("subscriptionId"), 10, 32)
	if err != nil {
		return 0, 0, &HandlerError{Status: http.StatusBadRequest, Message: "Invalid subscription ID"}
	}

	return uint(userID), uint(subscriptionID), nil
}

// HandlerError standardizes error responses
type HandlerError struct {
	Status  int
	Message string
}

func (h *HandlerError) Error() string {
	return h.Message
}

// handleError standardizes error responses
func handleError(c *gin.Context, err error) {
	if handlerErr, ok := err.(*HandlerError); ok {
		c.JSON(handlerErr.Status, gin.H{"error": handlerErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (h *UserSubscriptionHandler) Create(c *gin.Context) {
	userID, subscriptionID, err := h.parseUserAndSubscriptionID(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var us models.UserSubscription
	if err := c.ShouldBindJSON(&us); err != nil {
		handleError(c, &HandlerError{Status: http.StatusBadRequest, Message: err.Error()})
		return
	}

	h.mu.Lock() // Lock for write operation
	defer h.mu.Unlock()

	us.UserID = userID
	us.SubscriptionID = subscriptionID
	us.IsActive = true

	// Set default dates if not provided
	now := time.Now()
	if us.StartDate.IsZero() {
		us.StartDate = now
	}
	if us.EndDate.IsZero() {
		us.EndDate = now.AddDate(0, 1, 0)
	}

	if err := h.repo.Create(&us); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, us)
}

func (h *UserSubscriptionHandler) GetUserSubscriptions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		handleError(c, &HandlerError{Status: http.StatusBadRequest, Message: "Invalid user ID"})
		return
	}

	h.mu.RLock() // Lock for read operation
	defer h.mu.RUnlock()

	subscriptions, err := h.repo.GetByUserID(uint(userID))
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

func (h *UserSubscriptionHandler) UpdateUserSubscription(c *gin.Context) {
	userID, subscriptionID, err := h.parseUserAndSubscriptionID(c)
	if err != nil {
		handleError(c, err)
		return
	}

	h.mu.Lock() // Lock for write operation
	defer h.mu.Unlock()

	currentUs, err := h.repo.GetByID(subscriptionID)
	if err != nil {
		handleError(c, &HandlerError{Status: http.StatusNotFound, Message: "User subscription not found"})
		return
	}

	if currentUs.UserID != userID {
		handleError(c, &HandlerError{Status: http.StatusForbidden, Message: "Subscription does not belong to specified user"})
		return
	}

	var newUs models.UserSubscription
	if err := c.ShouldBindJSON(&newUs); err != nil {
		handleError(c, &HandlerError{Status: http.StatusBadRequest, Message: err.Error()})
		return
	}

	// Update fields using a helper method
	h.updateSubscriptionFields(currentUs, &newUs)

	if err := h.repo.Update(currentUs); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, currentUs)
}

// updateSubscriptionFields updates only the allowed fields
func (h *UserSubscriptionHandler) updateSubscriptionFields(current, new *models.UserSubscription) {
	if new.Type != "" {
		current.Type = new.Type
	}
	if new.CompanyName != "" {
		current.CompanyName = new.CompanyName
	}
	if new.Role != "" {
		current.Role = new.Role
	}
	if !new.StartDate.IsZero() {
		current.StartDate = new.StartDate
	}
	if !new.EndDate.IsZero() {
		current.EndDate = new.EndDate
	}
	current.IsActive = new.IsActive
}
