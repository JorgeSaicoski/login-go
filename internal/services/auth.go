package services

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/JorgeSaicoski/login-go/internal/models"
	"github.com/JorgeSaicoski/login-go/internal/repository"
)

var (
	authOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_operations_total",
			Help: "Total number of authentication operations",
		},
		[]string{"operation", "status"},
	)

	authDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "auth_operation_duration_seconds",
			Help: "Duration of authentication operations in seconds",
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(authOperations, authDuration)
}

type AuthService struct {
	userRepo    *repository.UserRepository
	logger      *zap.Logger
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	tokenExpiry time.Duration
}

type AuthConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	TokenExpiry    time.Duration
}

func NewAuthService(userRepo *repository.UserRepository, logger *zap.Logger, config AuthConfig) (*AuthService, error) {
	privateKey, err := loadPrivateKey(config.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	publicKey, err := loadPublicKey(config.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return &AuthService{
		userRepo:    userRepo,
		logger:      logger,
		privateKey:  privateKey,
		publicKey:   publicKey,
		tokenExpiry: config.TokenExpiry,
	}, nil
}

func (s *AuthService) GenerateToken(ctx context.Context, user *models.User) (string, error) {
	start := time.Now()
	defer func() {
		authDuration.WithLabelValues("generate_token").Observe(time.Since(start).Seconds())
	}()

	if user == nil || user.ID == 0 {
		authOperations.WithLabelValues("generate_token", "failed").Inc()
		return "", errors.New("invalid user")
	}

	now := time.Now()
	claims := &models.Claims{
		UserID:   user.ID,
		Username: user.UsernameForLogin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "login-go",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		s.logger.Error("failed to sign token",
			zap.Error(err),
			zap.Uint("user_id", user.ID),
		)
		authOperations.WithLabelValues("generate_token", "failed").Inc()
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	authOperations.WithLabelValues("generate_token", "success").Inc()
	return signedToken, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenStr string) (*models.Claims, error) {
	start := time.Now()
	defer func() {
		authDuration.WithLabelValues("validate_token").Observe(time.Since(start).Seconds())
	}()

	if tokenStr == "" {
		authOperations.WithLabelValues("validate_token", "failed").Inc()
		return nil, errors.New("empty token")
	}

	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		s.logger.Warn("token validation failed",
			zap.Error(err),
		)
		authOperations.WithLabelValues("validate_token", "failed").Inc()
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		authOperations.WithLabelValues("validate_token", "failed").Inc()
		return nil, errors.New("invalid token")
	}

	authOperations.WithLabelValues("validate_token", "success").Inc()
	return claims, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*models.User, string, error) {
	start := time.Now()
	defer func() {
		authDuration.WithLabelValues("login").Observe(time.Since(start).Seconds())
	}()

	if username == "" || password == "" {
		authOperations.WithLabelValues("login", "failed").Inc()
		return nil, "", errors.New("username and password are required")
	}

	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		s.logger.Warn("login failed: user not found",
			zap.String("username", username),
		)
		authOperations.WithLabelValues("login", "failed").Inc()
		return nil, "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.logger.Warn("login failed: invalid password",
			zap.String("username", username),
		)
		authOperations.WithLabelValues("login", "failed").Inc()
		return nil, "", errors.New("invalid credentials")
	}

	token, err := s.GenerateToken(ctx, user)
	if err != nil {
		authOperations.WithLabelValues("login", "failed").Inc()
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("successful login",
		zap.String("username", username),
		zap.Uint("user_id", user.ID),
	)

	authOperations.WithLabelValues("login", "success").Inc()
	return user, token, nil
}

// Helper functions for loading keys
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return key, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return key, nil
}
