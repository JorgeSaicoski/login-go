// handlers/user.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/JorgeSaicoski/login-go/models"
)

type UserHandler struct {
	DB *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{DB: db}
}

func (h *UserHandler) Create(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.DB.Create(&user)
	c.JSON(200, user)
}

func (h *UserHandler) List(c *gin.Context) {
	var users []models.User
	h.DB.Find(&users)
	c.JSON(200, users)
}

func (h *UserHandler) Get(c *gin.Context) {
	var user models.User
	if err := h.DB.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}
	c.JSON(200, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	var user models.User
	if err := h.DB.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.DB.Save(&user)
	c.JSON(200, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.DB.Delete(&models.User{}, c.Param("id")).Error; err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "User deleted"})
}
