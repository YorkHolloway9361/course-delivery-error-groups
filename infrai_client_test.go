package deliveryerrors

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCaptureReadsBusinessEnvelopeBeforeStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("Idempotency-Key") != "delivery-evt-1042" {
			t.Fatalf("idempotency key = %q", r.Header.Get("Idempotency-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"data":null,"error":{"message":"review delivery input"},"metadata":{}}`))
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.baseURL = server.URL
	_, err := client.Capture(context.Background(), CaptureDecision{EventID: "delivery-evt-1042"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "review delivery input" {
		t.Fatalf("APIError = %#v", apiErr)
	}
}

func TestCaptureRetriesRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"data":null,"error":{"message":"retry later"},"metadata":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"event_id":"evt"},"error":null,"metadata":{}}`))
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.baseURL = server.URL
	var waited time.Duration
	client.sleep = func(_ context.Context, delay time.Duration) error { waited = delay; return nil }
	_, err := client.Capture(context.Background(), CaptureDecision{EventID: "delivery-evt-1042"})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || waited != 2*time.Second {
		t.Fatalf("attempts = %d, waited = %s", attempts, waited)
	}
}
