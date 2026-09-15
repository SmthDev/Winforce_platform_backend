package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"platform/backend/internal/filelink"
	miniorepo "platform/backend/internal/repository/minio"
)

type FileAccess int

const (
	AccessPublic FileAccess = iota
	AccessSigned
)

var (
	ErrFileNotFound  = errors.New("file not found")
	ErrFileForbidden = errors.New("file link is not valid")
	ErrBadObjectName = errors.New("object name is not valid")
)

type FileStorage interface {
	Get(ctx context.Context, bucket, objectName string) (io.ReadCloser, miniorepo.ObjectInfo, error)
}

type FileLinkVerifier interface {
	Verify(bucket, objectName, expires, signature string) error
}

type FileRequest struct {
	Bucket     string
	ObjectName string
	Expires    string
	Signature  string
}

type FileService struct {
	storage FileStorage
	links   FileLinkVerifier
	buckets map[string]FileAccess
	log     *slog.Logger
}

func NewFile(storage FileStorage, links FileLinkVerifier, buckets map[string]FileAccess, log *slog.Logger) *FileService {
	return &FileService{storage: storage, links: links, buckets: buckets, log: log}
}

func (s *FileService) Open(ctx context.Context, req FileRequest) (io.ReadCloser, miniorepo.ObjectInfo, FileAccess, error) {
	access, known := s.buckets[req.Bucket]
	if !known {
		return nil, miniorepo.ObjectInfo{}, access, fmt.Errorf("%w: unknown bucket %q", ErrFileNotFound, req.Bucket)
	}

	if !validObjectName(req.ObjectName) {
		return nil, miniorepo.ObjectInfo{}, access, fmt.Errorf("%w: %q", ErrBadObjectName, req.ObjectName)
	}

	if access == AccessSigned {
		if err := s.links.Verify(req.Bucket, req.ObjectName, req.Expires, req.Signature); err != nil {
			if errors.Is(err, filelink.ErrNoSecret) {
				return nil, miniorepo.ObjectInfo{}, access, fmt.Errorf("verify file link: %w", err)
			}
			return nil, miniorepo.ObjectInfo{}, access, fmt.Errorf("%w: %v", ErrFileForbidden, err)
		}
	}

	body, info, err := s.storage.Get(ctx, req.Bucket, req.ObjectName)
	if err != nil {
		if errors.Is(err, miniorepo.ErrObjectNotFound) {
			return nil, miniorepo.ObjectInfo{}, access, fmt.Errorf("%w: %v", ErrFileNotFound, err)
		}
		return nil, miniorepo.ObjectInfo{}, access, fmt.Errorf("open file: %w", err)
	}

	return body, info, access, nil
}

func (s *FileService) CacheControl(access FileAccess) string {
	if access == AccessPublic {
		return "public, max-age=" + immutableMaxAge + ", immutable"
	}
	return "private, no-store"
}

const immutableMaxAge = "31536000"

func validObjectName(objectName string) bool {
	if objectName == "" || len(objectName) > 1024 {
		return false
	}
	if strings.ContainsAny(objectName, "\x00\n\r\\") {
		return false
	}

	for segment := range strings.SplitSeq(objectName, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
