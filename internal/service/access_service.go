package service

import (
	"context"
	"fmt"
)

type AccessRepository interface {
	IsAdmin(ctx context.Context, userID any) (isAdmin bool, found bool, err error)
}

type AccessService struct {
	repo AccessRepository
}

func NewAccess(repo AccessRepository) *AccessService {
	return &AccessService{repo: repo}
}

func (s *AccessService) IsAdmin(ctx context.Context, userID any) (bool, bool, error) {
	isAdmin, found, err := s.repo.IsAdmin(ctx, userID)
	if err != nil {
		return false, false, fmt.Errorf("check admin access: %w", err)
	}
	return isAdmin, found, nil
}
