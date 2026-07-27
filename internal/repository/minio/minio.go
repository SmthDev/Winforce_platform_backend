package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	mc       *miniogo.Client
	endpoint string
	useSSL   bool
}

func New(endpoint, accessKey, secretKey string, useSSL bool) (*Client, error) {
	mc, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &Client{mc: mc, endpoint: endpoint, useSSL: useSSL}, nil
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

func (c *Client) Download(ctx context.Context, bucket, objectName string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, bucket, objectName, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("download object %q: %w", objectName, err)
	}
	return obj, nil
}

func (c *Client) Delete(ctx context.Context, bucket, objectName string) error {
	if err := c.mc.RemoveObject(ctx, bucket, objectName, miniogo.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object %q: %w", objectName, err)
	}
	return nil
}

func (c *Client) PresignedGetURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, bucket, objectName, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign object %q: %w", objectName, err)
	}
	return u.String(), nil
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
	scheme := "http"
	if c.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, c.endpoint, bucket, objectName)
}
