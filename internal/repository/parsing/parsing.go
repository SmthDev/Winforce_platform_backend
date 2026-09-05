package parsing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"platform/backend/internal/models"
)

const (
	parsePath = "/api/v1/parse"
	fileField = "file"
	maxErrorBody = 256
)

var (
	ErrNotConfigured = errors.New("parsing api is not configured")
	ErrUnavailable = errors.New("parsing api unavailable")
	ErrBadFile = errors.New("parsing api rejected the file")
	ErrNotRecognized = errors.New("receipt not recognized")
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func New(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
	}
}

// 
func (c *Client) ParseReceipt(ctx context.Context, fileName string, file io.Reader) (models.ParsedReceipt, []byte, error) {
	if c.baseURL == "" {
		return models.ParsedReceipt{}, nil, ErrNotConfigured
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		return models.ParsedReceipt{}, nil, fmt.Errorf("build parse request: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return models.ParsedReceipt{}, nil, fmt.Errorf("build parse request: %w", err)
	}
	if err := writer.Close(); err != nil {
		return models.ParsedReceipt{}, nil, fmt.Errorf("build parse request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+parsePath, &body)
	if err != nil {
		return models.ParsedReceipt{}, nil, fmt.Errorf("build parse request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return models.ParsedReceipt{}, nil, fmt.Errorf("%w: call parse endpoint: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ParsedReceipt{}, nil, fmt.Errorf("%w: read parse response: %v", ErrUnavailable, err)
	}

	if resp.StatusCode != http.StatusOK {
		return models.ParsedReceipt{}, nil, statusError(resp.StatusCode, payload)
	}

	var parsed models.ParsedReceipt
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return models.ParsedReceipt{}, nil, fmt.Errorf("%w: decode parse response: %v", ErrUnavailable, err)
	}
	return parsed, payload, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c.baseURL == "" {
		return ErrNotConfigured
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: ping parsing api: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: ping parsing api: status %d", ErrUnavailable, resp.StatusCode)
	}
	return nil
}


func statusError(status int, payload []byte) error {
	detail := strings.TrimSpace(string(payload))
	if len(detail) > maxErrorBody {
		detail = detail[:maxErrorBody]
	}

	switch status {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrBadFile, detail)
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %s", ErrNotRecognized, detail)
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: parsing api rejected the api key: %s", ErrUnavailable, detail)
	default:
		return fmt.Errorf("%w: unexpected status %d: %s", ErrUnavailable, status, detail)
	}
}
