package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/JorgeSaicoski/login-go/internal/models"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	return GetByID[models.User](r.DB, id)
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.DB.Where("username_for_login = ?", username).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}
