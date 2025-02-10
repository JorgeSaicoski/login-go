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

func (r *UserRepository) Update(user *models.User) error {
	if err := r.DB.Save(user).Error; err != nil {
		return errors.New("failed to update user")
	}
	return nil
}

func (r *UserRepository) Create(user *models.User) error {
	// Hash the password before saving
	if err := user.HashPassword(); err != nil {
		return errors.New("failed to hash password")
	}

	// Check if user with same email already exists
	if _, err := r.GetByEmail(user.Email); err == nil {
		return errors.New("user with this email already exists")
	}

	// Check if username already exists
	if _, err := r.GetByUsername(user.UsernameForLogin); err == nil {
		return errors.New("username already taken")
	}

	// Create the user
	if err := r.DB.Create(user).Error; err != nil {
		return errors.New("failed to create user")
	}

	return nil
}

func (r *UserRepository) Login(username, password string) (*models.User, error) {
	// Get user by username
	user, err := r.GetByUsername(username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check password
	if err := user.CheckPassword(password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}
