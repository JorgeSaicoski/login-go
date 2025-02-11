package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/JorgeSaicoski/login-go/internal/repository"
	"github.com/JorgeSaicoski/login-go/internal/services"
)

var (
	authHandlerOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_handler_operations_total",
			Help: "Total number of authentication handler operations",
		},
		[]string{"operation", "status"},
	)

	authHandlerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "auth_handler_duration_seconds",
			Help: "Duration of authentication handler operations in seconds",
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(authHandlerOperations, authHandlerDuration)
}

type AuthHandler struct {
	authService *services.AuthService
	userRepo    *repository.UserRepository
	logger      *zap.Logger
	validator   *validator.Validate
	rateLimiter *rate.Limiter
}

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8"`
}

func NewAuthHandler(authService *services.AuthService, userRepo *repository.UserRepository, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userRepo:    userRepo,
		logger:      logger,
		validator:   validator.New(),
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 10), // 10 login attempts per second
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	start := time.Now()
	defer func() {
		authHandlerDuration.WithLabelValues("login").Observe(time.Since(start).Seconds())
	}()

	// Rate limiting
	if !h.rateLimiter.Allow() {
		authHandlerOperations.WithLabelValues("login", "rate_limited").Inc()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authHandlerOperations.WithLabelValues("login", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		authHandlerOperations.WithLabelValues("login", "failed").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": err.Error()})
		return
	}

	// Sanitize inputs
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	user, token, err := h.authService.Login(ctx, req.Username, req.Password)
	if err != nil {
		h.logger.Warn("login failed",
			zap.String("username", req.Username),
			zap.Error(err),
		)
		authHandlerOperations.WithLabelValues("login", "failed").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Don't return password in response
	user.Password = ""

	h.logger.Info("successful login",
		zap.String("username", user.UsernameForLogin),
		zap.Uint("user_id", user.ID),
	)

	authHandlerOperations.WithLabelValues("login", "success").Inc()
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) ValidateToken(c *gin.Context) {
	start := time.Now()
	defer func() {
		authHandlerDuration.WithLabelValues("validate_token").Observe(time.Since(start).Seconds())
	}()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	token := c.GetHeader("Authorization")
	if token == "" {
		authHandlerOperations.WithLabelValues("validate_token", "failed").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no token provided"})
		return
	}

	// Remove 'Bearer ' prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := h.authService.ValidateToken(ctx, token)
	if err != nil {
		h.logger.Warn("token validation failed",
			zap.Error(err),
		)
		authHandlerOperations.WithLabelValues("validate_token", "failed").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	authHandlerOperations.WithLabelValues("validate_token", "success").Inc()
	c.JSON(http.StatusOK, claims)
}

// Middleware for protected routes
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		defer func() {
			authHandlerDuration.WithLabelValues("middleware").Observe(time.Since(start).Seconds())
		}()

		token := c.GetHeader("Authorization")
		if token == "" {
			authHandlerOperations.WithLabelValues("middleware", "failed").Inc()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no token provided"})
			return
		}

		// Remove 'Bearer ' prefix if present
		token = strings.TrimPrefix(token, "Bearer ")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		claims, err := h.authService.ValidateToken(ctx, token)
		if err != nil {
			h.logger.Warn("auth middleware: token validation failed",
				zap.Error(err),
			)
			authHandlerOperations.WithLabelValues("middleware", "failed").Inc()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Set user info in context for use in subsequent handlers
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		authHandlerOperations.WithLabelValues("middleware", "success").Inc()
		c.Next()
	}
}

// Helper method to get authenticated user ID from context
func GetAuthenticatedUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}
