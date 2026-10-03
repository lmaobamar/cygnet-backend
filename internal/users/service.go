package users

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/lmaobamar/cygnet-backend/internal/models"
)

var (
	ErrHandleTaken        = errors.New("handle taken")
	ErrEmailTaken         = errors.New("email already in use")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound           = errors.New("user not found")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, handle, displayName, email, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	handle = strings.ToLower(strings.TrimSpace(handle))
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = handle
	}

	user := models.User{
		Handle:       handle,
		DisplayName:  displayName,
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, s.whichConflict(ctx, handle)
		}
		return nil, err
	}
	return &user, nil
}

// whichConflict runs after a duplicate-key error to see which field clashed.
func (s *Service) whichConflict(ctx context.Context, handle string) error {
	var n int64
	if err := s.db.WithContext(ctx).Model(&models.User{}).
		Where("handle = ?", handle).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrHandleTaken
	}
	return ErrEmailTaken
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

func (s *Service) ByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
