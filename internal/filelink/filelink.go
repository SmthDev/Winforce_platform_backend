package filelink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultPathPrefix = "/api/v1/files"

const (
	QueryExpires   = "expires"
	QuerySignature = "sig"
)

var (
	ErrNoSecret     = errors.New("file link secret is not configured")
	ErrBadSignature = errors.New("file link signature is invalid")
	ErrExpired      = errors.New("file link has expired")
)

type Signer struct {
	baseURL string
	secret  []byte
	now     func() time.Time
}

func NewSigner(baseURL string, secret []byte) *Signer {
	return &Signer{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		now:     time.Now,
	}
}

func (s *Signer) Public(bucket, objectName string) string {
	return s.baseURL + "/" + url.PathEscape(bucket) + "/" + escapePath(objectName)
}

func (s *Signer) Signed(bucket, objectName string, ttl time.Duration) (string, error) {
	if len(s.secret) == 0 {
		return "", ErrNoSecret
	}

	expires := s.now().Add(ttl).Unix()
	query := url.Values{
		QueryExpires:   {strconv.FormatInt(expires, 10)},
		QuerySignature: {s.sign(bucket, objectName, expires)},
	}
	return s.Public(bucket, objectName) + "?" + query.Encode(), nil
}

func (s *Signer) Verify(bucket, objectName, expires, signature string) error {
	if len(s.secret) == 0 {
		return ErrNoSecret
	}
	if expires == "" || signature == "" {
		return ErrBadSignature
	}

	expiresAt, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return ErrBadSignature
	}
	if !hmac.Equal([]byte(signature), []byte(s.sign(bucket, objectName, expiresAt))) {
		return ErrBadSignature
	}
	if s.now().Unix() > expiresAt {
		return ErrExpired
	}
	return nil
}

func (s *Signer) Normalize(link, bucket string) string {
	objectName, ok := ObjectName(link, bucket)
	if !ok {
		return link
	}
	return s.Public(bucket, objectName)
}

func ObjectName(link, bucket string) (string, bool) {
	u, err := url.Parse(link)
	if err != nil {
		return "", false
	}

	marker := "/" + bucket + "/"
	index := strings.LastIndex(u.Path, marker)
	if index < 0 {
		return "", false
	}

	objectName := u.Path[index+len(marker):]
	if objectName == "" {
		return "", false
	}
	return objectName, true
}

func (s *Signer) sign(bucket, objectName string, expires int64) string {
	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "%s/%s?%d", bucket, objectName, expires)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func escapePath(objectName string) string {
	segments := strings.Split(objectName, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
