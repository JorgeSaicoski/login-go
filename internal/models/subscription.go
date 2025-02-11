package models

import "time"

type Subscription struct {
	ID          uint               `json:"id" gorm:"primaryKey"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Price       float64            `json:"price"`
	Users       []UserSubscription `json:"users" gorm:"foreignKey:SubscriptionID"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}
