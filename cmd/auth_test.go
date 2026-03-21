package cmd

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCLIAuthBrowserURL(t *testing.T) {
	tests := []struct {
		name       string
		apiBaseURL string
		want       string
	}{
		{
			name:       "default podwise URL",
			apiBaseURL: "https://podwise.ai/api",
			want:       "https://podwise.ai/auth/cli?confirm_code=abc123",
		},
		{
			name:       "custom prefixed path",
			apiBaseURL: "https://example.com/podwise/api",
			want:       "https://example.com/podwise/auth/cli?confirm_code=abc123",
		},
		{
			name:       "api root with trailing slash",
			apiBaseURL: "https://example.com/api/",
			want:       "https://example.com/auth/cli?confirm_code=abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cliBrowserAuthURL(tt.apiBaseURL, "abc123")
			if err != nil {
				t.Fatalf("cliBrowserAuthURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("cliBrowserAuthURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInitCLIAuthRetriesServerErrors(t *testing.T) {
	var attempts atomic.Int32
	client := api.New("https://podwise.ai/api", "", api.WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempt := attempts.Add(1)
			if req.URL.Path != "/api/no-auth/cli/auth/init" {
				t.Fatalf("unexpected path %q", req.URL.Path)
			}
			if attempt < 3 {
				return jsonResponse(http.StatusInternalServerError, `{"error":"server_error","message":"boom"}`), nil
			}
			return jsonResponse(http.StatusOK, `"abc123"`), nil
		}),
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	got, err := initCLIAuth(ctx, client)
	if err != nil {
		t.Fatalf("initCLIAuth() error = %v", err)
	}
	if got != "abc123" {
		t.Fatalf("initCLIAuth() = %q, want %q", got, "abc123")
	}
	if attempts.Load() != 3 {
		t.Fatalf("initCLIAuth() attempts = %d, want 3", attempts.Load())
	}
}

func TestPollCLIAuthAuthorized(t *testing.T) {
	var polls atomic.Int32
	client := api.New("https://podwise.ai/api", "", api.WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.URL.Query().Get("confirmCode"); got != "abc123" {
				t.Fatalf("confirmCode = %q, want %q", got, "abc123")
			}

			poll := polls.Add(1)
			if poll < 3 {
				return jsonResponse(http.StatusOK, `{"status":"pending"}`), nil
			}
			return jsonResponse(http.StatusOK, `{"status":"authorized","accessToken":"token-123"}`), nil
		}),
	}))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := pollCLIAuth(ctx, client, "abc123", 5*time.Millisecond)
	if err != nil {
		t.Fatalf("pollCLIAuth() error = %v", err)
	}
	if got != "token-123" {
		t.Fatalf("pollCLIAuth() = %q, want %q", got, "token-123")
	}
}

func TestPollCLIAuthExpired(t *testing.T) {
	client := api.New("https://podwise.ai/api", "", api.WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusNotFound, `{"error":"Not found or expired"}`), nil
		}),
	}))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := pollCLIAuth(ctx, client, "abc123", 5*time.Millisecond)
	if err == nil {
		t.Fatal("pollCLIAuth() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("pollCLIAuth() error = %q, want expiry message", err)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Request:    &http.Request{},
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestAuthContextError(t *testing.T) {
	err := authContextError(context.DeadlineExceeded, "authorization")
	if err == nil || !strings.Contains(err.Error(), "timed out after 2m 0s") {
		t.Fatalf("authContextError() = %v", err)
	}
}
