package models

import "golang.org/x/crypto/bcrypt"

type User struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	Name             string         `json:"name"`
	UsernameForLogin string         `json:"username"`
	Email            string         `json:"email"`
	Password         string         `json:"password"`
	Products         []Subscription `json:"products" gorm:"many2many:user_products;"`
}

func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
