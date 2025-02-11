package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/JorgeSaicoski/login-go/internal/models"
)

type UserSubscriptionRepository struct {
	DB *gorm.DB
}

func NewUserSubscriptionRepository(db *gorm.DB) *UserSubscriptionRepository {
	return &UserSubscriptionRepository{DB: db}
}

func (r *UserSubscriptionRepository) Create(us *models.UserSubscription) error {
	us.CreatedAt = time.Now()
	us.UpdatedAt = time.Now()

	if err := r.DB.Create(us).Error; err != nil {
		return errors.New("failed to create user subscription")
	}
	return nil
}

func (r *UserSubscriptionRepository) GetByID(id uint) (*models.UserSubscription, error) {
	var us models.UserSubscription
	if err := r.DB.Preload("User").Preload("Subscription").First(&us, id).Error; err != nil {
		return nil, errors.New("user subscription not found")
	}
	return &us, nil
}

func (r *UserSubscriptionRepository) GetByUserID(userID uint) ([]models.UserSubscription, error) {
	var subscriptions []models.UserSubscription
	if err := r.DB.Where("user_id = ?", userID).Preload("Subscription").Find(&subscriptions).Error; err != nil {
		return nil, errors.New("failed to get user subscriptions")
	}
	return subscriptions, nil
}

func (r *UserSubscriptionRepository) GetActiveByUserID(userID uint) ([]models.UserSubscription, error) {
	var subscriptions []models.UserSubscription
	if err := r.DB.Where("user_id = ? AND is_active = ? AND end_date > ?", userID, true, time.Now()).
		Preload("Subscription").Find(&subscriptions).Error; err != nil {
		return nil, errors.New("failed to get active user subscriptions")
	}
	return subscriptions, nil
}

func (r *UserSubscriptionRepository) Update(us *models.UserSubscription) error {
	us.UpdatedAt = time.Now()
	if err := r.DB.Save(us).Error; err != nil {
		return errors.New("failed to update user subscription")
	}
	return nil
}
