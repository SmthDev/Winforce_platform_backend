package minio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"platform/backend/internal/filelink"
)

var ErrObjectNotFound = errors.New("object not found")

type ObjectInfo struct {
	ContentType  string
	Size         int64
	ETag         string
	LastModified time.Time
}

type Options struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Links *filelink.Signer
}

type Client struct {
	mc    *miniogo.Client
	links *filelink.Signer
}

func New(opts Options) (*Client, error) {
	if opts.Links == nil {
		return nil, errors.New("minio: link signer is required")
	}

	mc, err := miniogo.New(opts.Endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure: opts.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &Client{mc: mc, links: opts.Links}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if _, err := c.mc.ListBuckets(ctx); err != nil {
		return fmt.Errorf("ping minio: %w", err)
	}
	return nil
}

func (c *Client) checkBucket(ctx context.Context, bucket string) error {
	exists, err := c.mc.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", bucket, err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", bucket)
	}
	return nil
}

func (c *Client) Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error {
	if err := c.checkBucket(ctx, bucket); err != nil {
		return err
	}
	if _, err := c.mc.PutObject(ctx, bucket, objectName, reader, size, miniogo.PutObjectOptions{ContentType: contentType}); err != nil {
		return fmt.Errorf("upload object %q: %w", objectName, err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, bucket, objectName string) error {
	if err := c.mc.RemoveObject(ctx, bucket, objectName, miniogo.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object %q: %w", objectName, err)
	}
	return nil
}

func (c *Client) Get(ctx context.Context, bucket, objectName string) (io.ReadCloser, ObjectInfo, error) {
	obj, err := c.mc.GetObject(ctx, bucket, objectName, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("get object %q: %w", objectName, err)
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		if miniogo.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return nil, ObjectInfo{}, fmt.Errorf("%w: %s/%s", ErrObjectNotFound, bucket, objectName)
		}
		return nil, ObjectInfo{}, fmt.Errorf("stat object %q: %w", objectName, err)
	}

	return obj, ObjectInfo{
		ContentType:  stat.ContentType,
		Size:         stat.Size,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}

func (c *Client) EnsureBucket(ctx context.Context, bucket string, public bool) error {
	if err := c.checkBucket(ctx, bucket); err != nil {
		return err
	}

	if public {
		policy := fmt.Sprintf(`{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Principal": {"AWS": ["*"]},
			"Action": ["s3:GetObject"],
			"Resource": ["arn:aws:s3:::%s/*"]
		}
	]
}`, bucket)
		if err := c.mc.SetBucketPolicy(ctx, bucket, policy); err != nil {
			return fmt.Errorf("set bucket policy %q: %w", bucket, err)
		}
	}

	return nil
}

func (c *Client) PublicURL(bucket, objectName string) string {
	return c.links.Public(bucket, objectName)
}

func (c *Client) SignedURL(bucket, objectName string, expiry time.Duration) (string, error) {
	return c.links.Signed(bucket, objectName, expiry)
}

func (c *Client) NormalizeURL(link, bucket string) string {
	return c.links.Normalize(link, bucket)
}
