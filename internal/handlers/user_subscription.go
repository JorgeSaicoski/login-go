package handlers

import (
	"net/http"
	"strconv"

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
	var us models.UserSubscription
	if err := c.ShouldBindJSON(&us); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
