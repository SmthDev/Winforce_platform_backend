package service

import (
 "context"
 "platform/backend/internal/models"
)

type UserRepository interface {
	UpdateName(ctx context.Context, userID any, firstName, lastName string) error
	GetUserProfile(ctx context.Context, userID any) (models.Profile, error)
	ListUsers(ctx context.Context) ([]models.Profile, error)
}

type ProfileService struct {
	userRepo UserRepository
}

func NewProfile(userRepo UserRepository) *ProfileService {
	return &ProfileService{userRepo: userRepo}
}

func (s *ProfileService) UpdateName(ctx context.Context, userID any, firstName, lastName string) error {
	return s.userRepo.UpdateName(ctx, userID, firstName, lastName)
}

func (s *ProfileService) GetUserProfile(ctx context.Context, userID any) (models.Profile, error) {
	return s.userRepo.GetUserProfile(ctx, userID)
}

func (s *ProfileService) ListUsers(ctx context.Context) ([]models.Profile, error) {
	return s.userRepo.ListUsers(ctx)
}
