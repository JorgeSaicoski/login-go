package repository

import (
	"errors"

	"gorm.io/gorm"
)

// GetByID fetches a record from the database by ID.
func GetByID[T any](db *gorm.DB, id uint) (*T, error) {
	var entity T
	if err := db.First(&entity, id).Error; err != nil {
		return nil, errors.New("record not found")
	}
	return &entity, nil
}
