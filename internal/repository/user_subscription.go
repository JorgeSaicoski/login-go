package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/JorgeSaicoski/login-go/internal/models"
)

var (
	dbOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "status"},
	)

	dbDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "db_operation_duration_seconds",
			Help: "Duration of database operations in seconds",
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(dbOperations, dbDuration)
}

// Common errors
var (
	ErrNotFound          = errors.New("record not found")
	ErrInvalidInput      = errors.New("invalid input")
	ErrDatabaseOperation = errors.New("database operation failed")
)

type UserSubscriptionRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewUserSubscriptionRepository(db *gorm.DB, logger *zap.Logger) *UserSubscriptionRepository {
	return &UserSubscriptionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserSubscriptionRepository) CreateWithContext(ctx context.Context, us *models.UserSubscription) error {
	start := time.Now()
	defer func() {
		dbDuration.WithLabelValues("create_subscription").Observe(time.Since(start).Seconds())
	}()

	if us == nil {
		dbOperations.WithLabelValues("create_subscription", "failed").Inc()
		return ErrInvalidInput
	}

	us.CreatedAt = time.Now()
	us.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if subscription already exists
		var count int64
		if err := tx.Model(&models.UserSubscription{}).
			Where("user_id = ? AND subscription_id = ? AND end_date > ?",
				us.UserID, us.SubscriptionID, time.Now()).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return errors.New("active subscription already exists")
		}

		// Create new subscription
		if err := tx.Create(us).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to create user subscription",
			zap.Error(err),
			zap.Uint("user_id", us.UserID),
			zap.Uint("subscription_id", us.SubscriptionID),
		)
		dbOperations.WithLabelValues("create_subscription", "failed").Inc()
		return fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	dbOperations.WithLabelValues("create_subscription", "success").Inc()
	return nil
}

func (r *UserSubscriptionRepository) GetByIDWithContext(ctx context.Context, id uint) (*models.UserSubscription, error) {
	start := time.Now()
	defer func() {
		dbDuration.WithLabelValues("get_subscription").Observe(time.Since(start).Seconds())
	}()

	var us models.UserSubscription
	err := r.db.WithContext(ctx).
		Preload(clause.Associations).
		First(&us, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dbOperations.WithLabelValues("get_subscription", "not_found").Inc()
			return nil, ErrNotFound
		}
		r.logger.Error("failed to get user subscription",
			zap.Error(err),
			zap.Uint("id", id),
		)
		dbOperations.WithLabelValues("get_subscription", "failed").Inc()
		return nil, fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	dbOperations.WithLabelValues("get_subscription", "success").Inc()
	return &us, nil
}

func (r *UserSubscriptionRepository) GetByUserIDWithContext(ctx context.Context, userID uint) ([]models.UserSubscription, error) {
	start := time.Now()
	defer func() {
		dbDuration.WithLabelValues("get_user_subscriptions").Observe(time.Since(start).Seconds())
	}()

	var subscriptions []models.UserSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Subscription").
		Find(&subscriptions).Error

	if err != nil {
		r.logger.Error("failed to get user subscriptions",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		dbOperations.WithLabelValues("get_user_subscriptions", "failed").Inc()
		return nil, fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	dbOperations.WithLabelValues("get_user_subscriptions", "success").Inc()
	return subscriptions, nil
}

func (r *UserSubscriptionRepository) GetActiveByUserIDWithContext(ctx context.Context, userID uint) ([]models.UserSubscription, error) {
	start := time.Now()
	defer func() {
		dbDuration.WithLabelValues("get_active_subscriptions").Observe(time.Since(start).Seconds())
	}()

	var subscriptions []models.UserSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ? AND end_date > ?", userID, true, time.Now()).
		Preload("Subscription").
		Find(&subscriptions).Error

	if err != nil {
		r.logger.Error("failed to get active user subscriptions",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		dbOperations.WithLabelValues("get_active_subscriptions", "failed").Inc()
		return nil, fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	dbOperations.WithLabelValues("get_active_subscriptions", "success").Inc()
	return subscriptions, nil
}

func (r *UserSubscriptionRepository) UpdateWithContext(ctx context.Context, us *models.UserSubscription) error {
	start := time.Now()

	defer func() {
		dbDuration.WithLabelValues("update_subscription").Observe(time.Since(start).Seconds())
	}()

	if us == nil {
		dbOperations.WithLabelValues("update_subscription", "failed").Inc()
		return ErrInvalidInput
	}

	us.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Verify subscription exists and get current state
		var current models.UserSubscription
		if err := tx.First(&current, us.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		// Update subscription
		if err := tx.Save(us).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to update user subscription",
			zap.Error(err),
			zap.Uint("id", us.ID),
			zap.Uint("user_id", us.UserID),
		)
		dbOperations.WithLabelValues("update_subscription", "failed").Inc()
		return fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	dbOperations.WithLabelValues("update_subscription", "success").Inc()
	return nil
}

// Additional helper methods for database operations

func (r *UserSubscriptionRepository) CancelSubscription(ctx context.Context, id uint) error {
	start := time.Now()
	defer func() {
		dbDuration.WithLabelValues("cancel_subscription").Observe(time.Since(start).Seconds())
	}()

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.UserSubscription{}).
			Where("id = ? AND is_active = ?", id, true).
			Updates(map[string]interface{}{
				"is_active":  false,
				"updated_at": time.Now(),
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return ErrNotFound
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to cancel subscription",
			zap.Error(err),
			zap.Uint("id", id),
		)
		dbOperations.WithLabelValues("cancel_subscription", "failed").Inc()
		return fmt.Errorf("%w: %v", ErrDatabaseOperation, err)
	}

	dbOperations.WithLabelValues("cancel_subscription", "success").Inc()
	return nil
}
