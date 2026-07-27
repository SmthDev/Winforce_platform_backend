package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
)

type AvatarRepository interface {
	GetAvatarLink(ctx context.Context, userID any) (link string, found bool, err error)
	UpsertAvatarLink(ctx context.Context, userID any, avatarLink string) error
}

type AvatarStorage interface {
	Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, bucket, objectName string) error
	PublicURL(bucket, objectName string) string
}

type AvatarService struct {
	repo    AvatarRepository
	storage AvatarStorage
	log     *slog.Logger
}

func NewAvatar(repo AvatarRepository, storage AvatarStorage, log *slog.Logger) *AvatarService {
	return &AvatarService{repo: repo, storage: storage, log: log}
}

func (s *AvatarService) UploadAvatar(ctx context.Context, userID any, bucket, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	oldLink, hadOld, err := s.repo.GetAvatarLink(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get previous avatar link: %w", err)
	}

	if err := s.storage.Upload(ctx, bucket, objectName, reader, size, contentType); err != nil {
		return "", fmt.Errorf("upload avatar: %w", err)
	}

	link := s.storage.PublicURL(bucket, objectName)
	if err := s.repo.UpsertAvatarLink(ctx, userID, link); err != nil {
		return "", fmt.Errorf("save avatar link: %w", err)
	}

	if hadOld {
		if oldObjectName, ok := objectNameFromLink(oldLink, bucket); ok {
			if err := s.storage.Delete(ctx, bucket, oldObjectName); err != nil {
				s.log.Error("delete previous avatar failed", slog.Any("error", err), slog.Any("user_id", userID))
			}
		}
	}

	return link, nil
}

func objectNameFromLink(link, bucket string) (string, bool) {
	u, err := url.Parse(link)
	if err != nil {
		return "", false
	}

	prefix := "/" + bucket + "/"
	if !strings.HasPrefix(u.Path, prefix) {
		return "", false
	}

	return strings.TrimPrefix(u.Path, prefix), true
}
