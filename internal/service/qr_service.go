package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"platform/backend/internal/models"
)

type QRRepository interface {
	CreateQRCode(ctx context.Context, userID any, targetLink, qrLink string) (models.QRCode, error)
	ListQRCodes(ctx context.Context, userID any) ([]models.QRCode, error)
}

type QRStorage interface {
	Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, bucket, objectName string) error
	PublicURL(bucket, objectName string) string
	NormalizeURL(link, bucket string) string
}

type QRService struct {
	repo    QRRepository
	storage QRStorage
	log     *slog.Logger
}

func NewQR(repo QRRepository, storage QRStorage, log *slog.Logger) *QRService {
	return &QRService{repo: repo, storage: storage, log: log}
}

func (s *QRService) SaveQRCode(ctx context.Context, userID any, targetLink, bucket, objectName string, reader io.Reader, size int64, contentType string) (models.QRCode, error) {
	if err := s.storage.Upload(ctx, bucket, objectName, reader, size, contentType); err != nil {
		return models.QRCode{}, fmt.Errorf("upload qr code: %w", err)
	}

	qrLink := s.storage.PublicURL(bucket, objectName)
	qr, err := s.repo.CreateQRCode(ctx, userID, targetLink, qrLink)
	if err != nil {
		if delErr := s.storage.Delete(ctx, bucket, objectName); delErr != nil {
			s.log.Error("delete orphaned qr object failed",
				slog.Any("error", delErr),
				slog.String("object_name", objectName),
				slog.Any("user_id", userID),
			)
		}
		return models.QRCode{}, fmt.Errorf("save qr code link: %w", err)
	}

	return qr, nil
}

func (s *QRService) ListQRCodes(ctx context.Context, userID any, bucket string) ([]models.QRCode, error) {
	codes, err := s.repo.ListQRCodes(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list qr codes: %w", err)
	}
	for i := range codes {
		codes[i].QRLink = s.storage.NormalizeURL(codes[i].QRLink, bucket)
	}
	return codes, nil
}
