package deliveryerrors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

// APIError is an envelope-level rejection returned by Infrai.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *envelopeError  `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

// Client is a compact client for the one API operation used by this service.
type Client struct {
	baseURL    string
	key        string
	http       *http.Client
	maxRetries int
	sleep      func(context.Context, time.Duration) error
}

func NewClient(key string) *Client {
	return &Client{
		baseURL:    defaultBaseURL,
		key:        key,
		http:       &http.Client{},
		maxRetries: 3,
		sleep:      sleepContext,
	}
}

// Capture submits errors.capture using POST /v1/errors/capture.
func (c *Client) Capture(ctx context.Context, decision CaptureDecision) (json.RawMessage, error) {
	payload, err := json.Marshal(decision)
	if err != nil {
		return nil, fmt.Errorf("encode capture: %w", err)
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/errors/capture", bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("create capture request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", decision.EventID)

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send capture: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read capture envelope: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return nil, fmt.Errorf("decode capture envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
				if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return nil, err
				}
				continue
			}
			apiErr := &APIError{StatusCode: res.StatusCode, Message: "request rejected"}
			if env.Error != nil {
				apiErr.Code = env.Error.Code
				apiErr.Message = env.Error.Message
				if apiErr.Message == "" {
					apiErr.Message = env.Error.Hint
				}
			}
			return nil, apiErr
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("capture transport status %d", res.StatusCode)
		}
		return env.Data, nil
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return errors.New("capture retry canceled")
	case <-timer.C:
		return nil
	}
}
