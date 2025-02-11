package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/JorgeSaicoski/login-go/internal/models"
	"github.com/JorgeSaicoski/login-go/internal/repository"
)

// Metrics
var (
	subscriptionOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "subscription_operations_total",
			Help: "Total number of subscription operations",
		},
		[]string{"operation", "status"},
	)

	subscriptionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "subscription_operation_duration_seconds",
			Help:    "Duration of subscription operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(subscriptionOperations)
	prometheus.MustRegister(subscriptionDuration)
}

type UserSubscriptionHandler struct {
	repo        *repository.UserSubscriptionRepository
	mu          sync.RWMutex
	logger      *zap.Logger
	validator   *validator.Validate
	rateLimiter *rate.Limiter
}

type HandlerError struct {
	Status  int
	Message string
	Err     error
}

func (e *HandlerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func NewUserSubscriptionHandler(repo *repository.UserSubscriptionRepository, logger *zap.Logger) *UserSubscriptionHandler {
	return &UserSubscriptionHandler{
		repo:        repo,
		logger:      logger,
		validator:   validator.New(),
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 100), // 100 requests per second
	}
}

// validateSubscriptionDates ensures dates are valid
func (h *UserSubscriptionHandler) validateSubscriptionDates(start, end time.Time) error {
	if end.Before(start) {
		return &HandlerError{
			Status:  http.StatusBadRequest,
			Message: "End date must be after start date",
		}
	}
	if start.Before(time.Now().Add(-24 * time.Hour)) {
		return &HandlerError{
			Status:  http.StatusBadRequest,
			Message: "Start date cannot be in the past",
		}
	}
	return nil
}

// validateSubscriptionType ensures type is valid
func (h *UserSubscriptionHandler) validateSubscriptionType(subType models.SubscriptionType) error {
	if subType != models.Individual && subType != models.Enterprise {
		return &HandlerError{
			Status:  http.StatusBadRequest,
			Message: "Invalid subscription type",
		}
	}
	return nil
}

func (h *UserSubscriptionHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()
	defer func() {
		subscriptionDuration.WithLabelValues("create").Observe(time.Since(start).Seconds())
	}()

	// Rate limiting
	if !h.rateLimiter.Allow() {
		subscriptionOperations.WithLabelValues("create", "rate_limited").Inc()
		handleError(c, &HandlerError{Status: http.StatusTooManyRequests, Message: "Rate limit exceeded"})
		return
	}

	userID, subscriptionID, err := h.parseUserAndSubscriptionID(c)
	if err != nil {
		subscriptionOperations.WithLabelValues("create", "failed").Inc()
		handleError(c, err)
		return
	}

	var us models.UserSubscription
	if err := c.ShouldBindJSON(&us); err != nil {
		subscriptionOperations.WithLabelValues("create", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusBadRequest, Message: "Invalid request body", Err: err})
		return
	}

	// Validate subscription
	if err := h.validateSubscriptionType(us.Type); err != nil {
		subscriptionOperations.WithLabelValues("create", "failed").Inc()
		handleError(c, err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Set IDs and defaults
	us.UserID = userID
	us.SubscriptionID = subscriptionID
	us.IsActive = true

	now := time.Now()
	if us.StartDate.IsZero() {
		us.StartDate = now
	}
	if us.EndDate.IsZero() {
		us.EndDate = now.AddDate(1, 0, 0)
	}

	// Validate dates
	if err := h.validateSubscriptionDates(us.StartDate, us.EndDate); err != nil {
		subscriptionOperations.WithLabelValues("create", "failed").Inc()
		handleError(c, err)
		return
	}

	// Create with context
	if err := h.repo.CreateWithContext(ctx, &us); err != nil {
		h.logger.Error("failed to create subscription",
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		subscriptionOperations.WithLabelValues("create", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusInternalServerError, Message: "Failed to create subscription", Err: err})
		return
	}

	h.logger.Info("subscription created",
		zap.Uint("user_id", userID),
		zap.Uint("subscription_id", subscriptionID),
	)
	subscriptionOperations.WithLabelValues("create", "success").Inc()
	c.JSON(http.StatusCreated, us)
}

func (h *UserSubscriptionHandler) GetUserSubscriptions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	start := time.Now()
	defer func() {
		subscriptionDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		subscriptionOperations.WithLabelValues("get", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusBadRequest, Message: "Invalid user ID"})
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	subscriptions, err := h.repo.GetByUserIDWithContext(ctx, uint(userID))
	if err != nil {
		h.logger.Error("failed to get subscriptions",
			zap.Uint64("user_id", userID),
			zap.Error(err),
		)
		subscriptionOperations.WithLabelValues("get", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusInternalServerError, Message: "Failed to get subscriptions", Err: err})
		return
	}

	h.logger.Info("subscriptions retrieved",
		zap.Uint64("user_id", userID),
		zap.Int("count", len(subscriptions)),
	)
	subscriptionOperations.WithLabelValues("get", "success").Inc()
	c.JSON(http.StatusOK, subscriptions)
}

func (h *UserSubscriptionHandler) UpdateUserSubscription(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()
	defer func() {
		subscriptionDuration.WithLabelValues("update").Observe(time.Since(start).Seconds())
	}()

	if !h.rateLimiter.Allow() {
		subscriptionOperations.WithLabelValues("update", "rate_limited").Inc()
		handleError(c, &HandlerError{Status: http.StatusTooManyRequests, Message: "Rate limit exceeded"})
		return
	}

	userID, subscriptionID, err := h.parseUserAndSubscriptionID(c)
	if err != nil {
		subscriptionOperations.WithLabelValues("update", "failed").Inc()
		handleError(c, err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	currentUs, err := h.repo.GetByIDWithContext(ctx, subscriptionID)
	if err != nil {
		subscriptionOperations.WithLabelValues("update", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusNotFound, Message: "User subscription not found"})
		return
	}

	if currentUs.UserID != userID {
		subscriptionOperations.WithLabelValues("update", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusForbidden, Message: "Subscription does not belong to specified user"})
		return
	}

	var newUs models.UserSubscription
	if err := c.ShouldBindJSON(&newUs); err != nil {
		subscriptionOperations.WithLabelValues("update", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusBadRequest, Message: "Invalid request body", Err: err})
		return
	}

	if newUs.Type != "" {
		if err := h.validateSubscriptionType(newUs.Type); err != nil {
			subscriptionOperations.WithLabelValues("update", "failed").Inc()
			handleError(c, err)
			return
		}
	}

	// Update fields
	h.updateSubscriptionFields(currentUs, &newUs)

	// Validate dates if they were updated
	if !newUs.StartDate.IsZero() || !newUs.EndDate.IsZero() {
		if err := h.validateSubscriptionDates(currentUs.StartDate, currentUs.EndDate); err != nil {
			subscriptionOperations.WithLabelValues("update", "failed").Inc()
			handleError(c, err)
			return
		}
	}

	if err := h.repo.UpdateWithContext(ctx, currentUs); err != nil {
		h.logger.Error("failed to update subscription",
			zap.Uint("user_id", userID),
			zap.Uint("subscription_id", subscriptionID),
			zap.Error(err),
		)
		subscriptionOperations.WithLabelValues("update", "failed").Inc()
		handleError(c, &HandlerError{Status: http.StatusInternalServerError, Message: "Failed to update subscription", Err: err})
		return
	}

	h.logger.Info("subscription updated",
		zap.Uint("user_id", userID),
		zap.Uint("subscription_id", subscriptionID),
	)
	subscriptionOperations.WithLabelValues("update", "success").Inc()
	c.JSON(http.StatusOK, currentUs)
}

// Helper methods remain mostly unchanged but add context support
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

func handleError(c *gin.Context, err error) {
	if handlerErr, ok := err.(*HandlerError); ok {
		c.JSON(handlerErr.Status, gin.H{"error": handlerErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
