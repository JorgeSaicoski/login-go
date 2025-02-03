package config

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/JorgeSaicoski/login-go/internal/models"
)

func ConnectDatabase() *gorm.DB {
	dsn := "host=db user=postgres password=yourpassword dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	db.AutoMigrate(&models.User{})
	return db
}
