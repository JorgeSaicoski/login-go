package models

import (
	"time"
)

type SubscriptionType string

const (
	Individual SubscriptionType = "individual"
	Enterprise SubscriptionType = "enterprise"
)

type UserSubscription struct {
	ID             uint             `json:"id" gorm:"primaryKey"`
	UserID         uint             `json:"user_id"`
	User           User             `json:"user" gorm:"foreignKey:UserID"`
	SubscriptionID uint             `json:"subscription_id"`
	Subscription   Subscription     `json:"subscription" gorm:"foreignKey:SubscriptionID"`
	Type           SubscriptionType `json:"type"`
	CompanyName    string           `json:"company_name,omitempty"`
	Role           string           `json:"role"`
	StartDate      time.Time        `json:"start_date"`
	EndDate        time.Time        `json:"end_date"`
	IsActive       bool             `json:"is_active"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}
