package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

var (
	userHandlerOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_handler_operations_total",
			Help: "Total number of user handler operations",
		},
		[]string{"operation", "status"},
	)

	userHandlerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "user_handler_duration_seconds",
			Help: "Duration of user handler operations in seconds",
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(userHandlerOperations, userHandlerDuration)
}

type UserHandler struct {
	repo        *repository.UserRepository
	logger      *zap.Logger
	validator   *validator.Validate
	rateLimiter *rate.Limiter
	mu          sync.RWMutex
}

type CreateUserRequest struct {
	Name             string `json:"name" validate:"required,min=2,max=100"`
	UsernameForLogin string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email            string `json:"email" validate:"required,email"`
	Password         string `json:"password" validate:"required,min=8,max=100"`
}

type UpdateUserRequest struct {
	Name  string `json:"name" validate:"omitempty,min=2,max=100"`
	Email string `json:"email" validate:"omitempty,email"`
}

func NewUserHandler(repo *repository.UserRepository, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		repo:        repo,
		logger:      logger,
		validator:   validator.New(),
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 50),
	}
}

func (h *UserHandler) Create(c *gin.Context) {
	start := time.Now()
	defer func() {
		userHandlerDuration.WithLabelValues("create").Observe(time.Since(start).Seconds())
	}()

	if !h.rateLimiter.Allow() {
		userHandlerOperations.WithLabelValues("create", "rate_limited").Inc()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		userHandlerOperations.WithLabelValues("create", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		userHandlerOperations.WithLabelValues("create", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": err.Error()})
		return
	}

	// Sanitize inputs
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.UsernameForLogin = strings.TrimSpace(strings.ToLower(req.UsernameForLogin))

	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if username or email already exists
	if _, err := h.repo.GetByUsername(req.UsernameForLogin); err == nil {
		userHandlerOperations.WithLabelValues("create", "failed").Inc()
		c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
		return
	}

	if _, err := h.repo.GetByEmail(req.Email); err == nil {
		userHandlerOperations.WithLabelValues("create", "failed").Inc()
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	user := &models.User{
		Name:             req.Name,
		UsernameForLogin: req.UsernameForLogin,
		Email:            req.Email,
		Password:         req.Password,
	}

	if err := h.repo.CreateWithContext(ctx, user); err != nil {
		h.logger.Error("failed to create user",
			zap.Error(err),
			zap.String("username", req.UsernameForLogin),
		)
		userHandlerOperations.WithLabelValues("create", "failed").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	h.logger.Info("user created",
		zap.String("username", user.UsernameForLogin),
		zap.Uint("user_id", user.ID),
	)

	// Don't return the password
	user.Password = ""

	userHandlerOperations.WithLabelValues("create", "success").Inc()
	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	start := time.Now()
	defer func() {
		userHandlerDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		userHandlerOperations.WithLabelValues("get", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID format"})
		return
	}

	// Check if user is requesting their own data
	authUserID, exists := GetAuthenticatedUserID(c)
	if !exists || authUserID != uint(id) {
		userHandlerOperations.WithLabelValues("get", "unauthorized").Inc()
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized access"})
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	user, err := h.repo.GetByIDWithContext(ctx, uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			userHandlerOperations.WithLabelValues("get", "not_found").Inc()
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		h.logger.Error("failed to get user",
			zap.Error(err),
			zap.Uint64("user_id", id),
		)
		userHandlerOperations.WithLabelValues("get", "failed").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	// Don't return the password
	user.Password = ""

	userHandlerOperations.WithLabelValues("get", "success").Inc()
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateByID(c *gin.Context) {
	start := time.Now()
	defer func() {
		userHandlerDuration.WithLabelValues("update").Observe(time.Since(start).Seconds())
	}()

	if !h.rateLimiter.Allow() {
		userHandlerOperations.WithLabelValues("update", "rate_limited").Inc()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		userHandlerOperations.WithLabelValues("update", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID format"})
		return
	}

	// Check if user is updating their own data
	authUserID, exists := GetAuthenticatedUserID(c)
	if !exists || authUserID != uint(id) {
		userHandlerOperations.WithLabelValues("update", "unauthorized").Inc()
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized access"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		userHandlerOperations.WithLabelValues("update", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		userHandlerOperations.WithLabelValues("update", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": err.Error()})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	user, err := h.repo.GetByIDWithContext(ctx, uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			userHandlerOperations.WithLabelValues("update", "not_found").Inc()
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		h.logger.Error("failed to get user for update",
			zap.Error(err),
			zap.Uint64("user_id", id),
		)
		userHandlerOperations.WithLabelValues("update", "failed").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	// Update fields if provided
	if req.Name != "" {
		user.Name = strings.TrimSpace(req.Name)
	}
	if req.Email != "" {
		newEmail := strings.TrimSpace(strings.ToLower(req.Email))
		if newEmail != user.Email {
			// Check if new email is already in use
			if _, err := h.repo.GetByEmail(newEmail); err == nil {
				userHandlerOperations.WithLabelValues("update", "failed").Inc()
				c.JSON(http.StatusConflict, gin.H{"error": "email already in use"})
				return
			}
			user.Email = newEmail
		}
	}

	if err := h.repo.UpdateWithContext(ctx, user); err != nil {
		h.logger.Error("failed to update user",
			zap.Error(err),
			zap.Uint("user_id", user.ID),
		)
		userHandlerOperations.WithLabelValues("update", "failed").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	h.logger.Info("user updated",
		zap.Uint("user_id", user.ID),
	)

	// Don't return the password
	user.Password = ""

	userHandlerOperations.WithLabelValues("update", "success").Inc()
	c.JSON(http.StatusOK, user)
}
