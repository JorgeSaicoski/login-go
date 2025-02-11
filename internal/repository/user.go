package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/JorgeSaicoski/login-go/internal/models"
)

var (
	userDBOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_db_operations_total",
			Help: "Total number of user database operations",
		},
		[]string{"operation", "status"},
	)

	userDBDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "user_db_duration_seconds",
			Help: "Duration of user database operations in seconds",
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(userDBOperations, userDBDuration)
}

var (
	ErrDuplicateEntry = errors.New("duplicate entry")
)

type UserRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewUserRepository(db *gorm.DB, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) CreateWithContext(ctx context.Context, user *models.User) error {
	start := time.Now()
	defer func() {
		userDBDuration.WithLabelValues("create").Observe(time.Since(start).Seconds())
	}()

	if user == nil {
		userDBOperations.WithLabelValues("create", "failed").Inc()
		return ErrInvalidInput
	}

	// Hash password before saving
	if err := user.HashPassword(); err != nil {
		r.logger.Error("failed to hash password",
			zap.Error(err),
		)
		userDBOperations.WithLabelValues("create", "failed").Inc()
		return fmt.Errorf("failed to hash password: %w", err)
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check for existing username
		var count int64
		if err := tx.Model(&models.User{}).
			Where("username_for_login = ?", user.UsernameForLogin).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDuplicateEntry
		}

		// Check for existing email
		if err := tx.Model(&models.User{}).
			Where("email = ?", user.Email).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDuplicateEntry
		}

		// Create user
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to create user",
			zap.Error(err),
			zap.String("username", user.UsernameForLogin),
		)
		userDBOperations.WithLabelValues("create", "failed").Inc()
		return fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	userDBOperations.WithLabelValues("create", "success").Inc()
	return nil
}

func (r *UserRepository) GetByIDWithContext(ctx context.Context, id uint) (*models.User, error) {
	start := time.Now()

	defer func() {
		userDBDuration.WithLabelValues("get_by_id").Observe(time.Since(start).Seconds())
	}()

	var user models.User
	err := r.db.WithContext(ctx).
		First(&user, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			userDBOperations.WithLabelValues("get_by_id", "not_found").Inc()
			return nil, ErrNotFound
		}
		r.logger.Error("failed to get user by id",
			zap.Error(err),
			zap.Uint("id", id),
		)
		userDBOperations.WithLabelValues("get_by_id", "failed").Inc()
		return nil, fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	userDBOperations.WithLabelValues("get_by_id", "success").Inc()
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	start := time.Now()

	defer func() {
		userDBDuration.WithLabelValues("get_by_username").Observe(time.Since(start).Seconds())
	}()

	var user models.User
	err := r.db.Where("username_for_login = ?", username).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			userDBOperations.WithLabelValues("get_by_username", "not_found").Inc()
			return nil, ErrNotFound
		}
		r.logger.Error("failed to get user by username",
			zap.Error(err),
			zap.String("username", username),
		)
		userDBOperations.WithLabelValues("get_by_username", "failed").Inc()
		return nil, fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	userDBOperations.WithLabelValues("get_by_username", "success").Inc()
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	start := time.Now()
	defer func() {
		userDBDuration.WithLabelValues("get_by_email").Observe(time.Since(start).Seconds())
	}()

	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			userDBOperations.WithLabelValues("get_by_email", "not_found").Inc()
			return nil, ErrNotFound
		}
		r.logger.Error("failed to get user by email",
			zap.Error(err),
			zap.String("email", email),
		)
		userDBOperations.WithLabelValues("get_by_email", "failed").Inc()
		return nil, fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	userDBOperations.WithLabelValues("get_by_email", "success").Inc()
	return &user, nil
}

func (r *UserRepository) UpdateWithContext(ctx context.Context, user *models.User) error {
	start := time.Now()
	defer func() {
		userDBDuration.WithLabelValues("update").Observe(time.Since(start).Seconds())
	}()

	if user == nil {
		userDBOperations.WithLabelValues("update", "failed").Inc()
		return ErrInvalidInput
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if email is already in use by another user
		var count int64
		if err := tx.Model(&models.User{}).
			Where("email = ? AND id != ?", user.Email, user.ID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDuplicateEntry
		}

		// Update user
		if err := tx.Save(user).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to update user",
			zap.Error(err),
			zap.Uint("id", user.ID),
		)
		userDBOperations.WithLabelValues("update", "failed").Inc()
		return fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	userDBOperations.WithLabelValues("update", "success").Inc()
	return nil
}

func (r *UserRepository) DeleteWithContext(ctx context.Context, id uint) error {
	start := time.Now()
	defer func() {
		userDBDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
	}()

	result := r.db.WithContext(ctx).Delete(&models.User{}, id)
	if result.Error != nil {
		r.logger.Error("failed to delete user",
			zap.Error(result.Error),
			zap.Uint("id", id),
		)
		userDBOperations.WithLabelValues("delete", "failed").Inc()
		return fmt.Errorf("%w: %v", ErrDatabaseOperation, result.Error)
	}

	if result.RowsAffected == 0 {
		userDBOperations.WithLabelValues("delete", "not_found").Inc()
		return ErrNotFound
	}

	userDBOperations.WithLabelValues("delete", "success").Inc()
	return nil
}

// Additional helper methods

func (r *UserRepository) Login(username, password string) (*models.User, error) {
	start := time.Now()
	defer func() {
		userDBDuration.WithLabelValues("login").Observe(time.Since(start).Seconds())
	}()

	user, err := r.GetByUsername(username)
	if err != nil {
		userDBOperations.WithLabelValues("login", "failed").Inc()
		return nil, errors.New("invalid credentials")
	}

	if err := user.CheckPassword(password); err != nil {
		userDBOperations.WithLabelValues("login", "failed").Inc()
		return nil, errors.New("invalid credentials")
	}

	userDBOperations.WithLabelValues("login", "success").Inc()
	return user, nil
}
