package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/JorgeSaicoski/login-go/internal/models"
)

type SubscriptionRepository struct {
	DB *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db}
}

func (r *SubscriptionRepository) GetByID(id uint) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := r.DB.First(&subscription, id).Error; err != nil {
		return nil, errors.New("subscription not found")
	}
	return &subscription, nil
}

func (r *SubscriptionRepository) GetByName(name string) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := r.DB.Where("name = ?", name).First(&subscription).Error; err != nil {
		return nil, errors.New("subscription not found")
	}
	return &subscription, nil
}

func (r *SubscriptionRepository) Update(subscription *models.Subscription) error {
	if err := r.DB.Save(subscription).Error; err != nil {
		return errors.New("failed to update subscription")
	}
	return nil
}

func (r *SubscriptionRepository) GetDescription(id uint) (string, error) {
	sub, err := r.GetByID(id)
	if err != nil {
		return "", err
	}
	return sub.Description, nil
}
